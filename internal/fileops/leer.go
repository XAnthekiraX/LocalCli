package fileops

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"unicode/utf8"

	"localcli/internal/tools"
)

const (
	// LimiteArchivoCompleto es el tamaño hasta el que un archivo se devuelve entero.
	LimiteArchivoCompleto = 64 * 1024
	// LimiteRespuesta es el máximo de bytes de una respuesta.
	LimiteRespuesta = 32 * 1024
)

// RespuestaLectura es lo que ve el modelo al leer un archivo.
type RespuestaLectura struct {
	Ruta           string `json:"ruta"`
	Contenido      string `json:"contenido"`
	LineaInicio    int    `json:"linea_inicio"`
	LineaFin       int    `json:"linea_fin"`
	TotalLineas    int    `json:"total_lineas"`
	Truncado       bool   `json:"truncado"`
	SiguienteLinea int    `json:"siguiente_linea,omitempty"`
}

// LeerHandler es el `Ejecutar` de leer_archivo.
func (o *Operaciones) LeerHandler() tools.Ejecutar {
	return func(ctx context.Context, args any, c tools.Contexto) (tools.Resultado, error) {
		peticion, ok := args.(tools.PeticionLeerArchivo)
		if !ok {
			return corregible(tools.CodBadArgs + ": argumentos inesperados")
		}
		return o.leer(ctx, peticion, c)
	}
}

func (o *Operaciones) leer(ctx context.Context, p tools.PeticionLeerArchivo, c tools.Contexto) (tools.Resultado, error) {
	if p.InicioLinea < 0 || p.FinLinea < 0 || (p.InicioLinea > 0 && p.FinLinea > 0 && p.InicioLinea > p.FinLinea) {
		return corregible(tools.CodBadArgs + ": rango de líneas inválido")
	}
	abs, codigo := Resolver(o.Raiz, p.Ruta)
	if codigo != "" {
		return corregible(codigo)
	}
	if err := ValidarAcceso(ctx, o.Raiz, abs, p.Motivo, c); err != nil {
		return resultado(err)
	}

	info, err := os.Stat(abs)
	if err != nil {
		return corregible("el archivo no existe: " + Canonica(o.Raiz, abs) + ". Verifica la ruta o usa buscar_archivos o listar_carpeta para localizarlo")
	}
	if info.IsDir() {
		return corregible("la ruta es un directorio, no un archivo: " + Canonica(o.Raiz, abs))
	}
	datos, err := os.ReadFile(abs)
	if err != nil {
		return corregible("no se pudo leer el archivo: " + Canonica(o.Raiz, abs))
	}
	if !esTexto(datos) {
		if tipo := tipoImagen(datos); tipo != "" {
			return corregible("`" + Canonica(o.Raiz, abs) + "` es una imagen (" + tipo + "); no se lee como texto. " +
				"Pídele al usuario que la adjunte al mensaje para que el modelo la vea")
		}
		return corregible("el archivo no es texto legible: " + Canonica(o.Raiz, abs))
	}

	respuesta, err := recortarLectura(datos, p.InicioLinea, p.FinLinea)
	if err != nil {
		return corregible(err.Error())
	}
	respuesta.Ruta = Canonica(o.Raiz, abs)

	lineas := respuesta.LineaFin - respuesta.LineaInicio + 1
	salida, _ := json.Marshal(respuesta)
	meta := map[string]any{"medida": fmt.Sprintf("%d líneas", lineas), "unidades": lineas}
	if respuesta.Truncado {
		meta["rango"] = fmt.Sprintf("%d-%d", respuesta.LineaInicio, respuesta.LineaFin)
	}
	return tools.Resultado{Salida: string(salida), Truncado: respuesta.Truncado, Meta: meta}, nil
}

// recortarLectura aplica el rango sobre el archivo original y lo acota al tope
// por respuesta. Un archivo pequeño sin rango se devuelve entero.
func recortarLectura(datos []byte, inicio, fin int) (RespuestaLectura, error) {
	lineas := partirLineas(datos)
	total := len(lineas)
	if inicio == 0 {
		inicio = 1
	}
	if fin == 0 || fin > total {
		fin = total
	}
	if inicio > total {
		return RespuestaLectura{}, fmt.Errorf("%s: la línea inicial %d supera el total de %d", tools.CodBadArgs, inicio, total)
	}
	if inicio == 1 && fin == total && len(datos) <= LimiteArchivoCompleto {
		return RespuestaLectura{
			Contenido: strings.Join(lineas, "\n"), LineaInicio: 1, LineaFin: total, TotalLineas: total,
		}, nil
	}

	var b strings.Builder
	ultima := inicio - 1
	truncado := false
	for i := inicio; i <= fin; i++ {
		linea := lineas[i-1]
		if b.Len() == 0 && len(linea) > LimiteRespuesta {
			b.WriteString(cortarUTF8(linea, LimiteRespuesta))
			ultima = i
			truncado = true
			break
		}
		anexo := len(linea)
		if b.Len() > 0 {
			anexo++
		}
		if b.Len()+anexo > LimiteRespuesta {
			truncado = true
			break
		}
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(linea)
		ultima = i
	}

	respuesta := RespuestaLectura{
		Contenido: b.String(), LineaInicio: inicio, LineaFin: ultima, TotalLineas: total, Truncado: truncado,
	}
	if truncado {
		respuesta.SiguienteLinea = ultima + 1
	}
	return respuesta, nil
}

func partirLineas(datos []byte) []string {
	texto := strings.ReplaceAll(string(datos), "\r\n", "\n")
	lineas := strings.Split(texto, "\n")
	if len(lineas) > 0 && lineas[len(lineas)-1] == "" {
		lineas = lineas[:len(lineas)-1]
	}
	return lineas
}

func esTexto(datos []byte) bool {
	return utf8.Valid(datos) && !bytes.ContainsRune(datos, 0)
}

// tipoImagen reconoce una imagen por su contenido para dar un mensaje útil en vez
// de «no es texto legible».
func tipoImagen(datos []byte) string {
	if len(datos) == 0 {
		return ""
	}
	muestra := datos
	if len(muestra) > 512 {
		muestra = muestra[:512]
	}
	switch tipo := http.DetectContentType(muestra); tipo {
	case "image/png", "image/jpeg", "image/gif", "image/webp", "image/bmp":
		return tipo
	}
	return ""
}

func cortarUTF8(s string, tope int) string {
	if len(s) <= tope {
		return s
	}
	corte := tope
	for corte > 0 && !utf8.RuneStart(s[corte]) {
		corte--
	}
	return s[:corte]
}
