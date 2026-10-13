package fileops

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"localcli/internal/tools"
)

// MaxCoincidencias acota cuántas coincidencias devuelve una búsqueda.
const MaxCoincidencias = 200

// LimiteArchivoBuscado es el tamaño máximo de un archivo cuyo contenido se
// inspecciona al buscar: más allá no se lee entero.
const LimiteArchivoBuscado = 2 * 1024 * 1024

// largoFragmento acota el trozo de línea que se devuelve por coincidencia.
const largoFragmento = 200

// Coincidencia es una línea que casa en `buscar_en_archivos`.
type Coincidencia struct {
	Archivo   string `json:"archivo"`
	Linea     int    `json:"linea"`
	Fragmento string `json:"fragmento"`
}

// BuscarArchivosHandler es el `Ejecutar` de buscar_archivos: recorre el proyecto
// y devuelve las rutas cuyo nombre contiene el patrón.
func (o *Operaciones) BuscarArchivosHandler() tools.Ejecutar {
	return func(ctx context.Context, args any, c tools.Contexto) (tools.Resultado, error) {
		peticion, ok := args.(tools.PeticionBuscarArchivos)
		if !ok {
			return corregible(tools.CodBadArgs + ": argumentos inesperados")
		}
		patron := strings.TrimSpace(peticion.Patron)
		if patron == "" {
			return corregible(tools.CodBadArgs + ": el `patron` no puede estar vacío")
		}

		rutas, truncado := o.buscarNombres(patron)
		salida, _ := json.Marshal(rutas)
		return tools.Resultado{
			Salida:   string(salida),
			Truncado: truncado,
			Meta:     map[string]any{"medida": pluralCoincidencias(len(rutas))},
		}, nil
	}
}

// BuscarEnArchivosHandler es el `Ejecutar` de buscar_en_archivos: busca el
// patrón dentro de los archivos de texto, opcionalmente acotado a una ruta.
func (o *Operaciones) BuscarEnArchivosHandler() tools.Ejecutar {
	return func(ctx context.Context, args any, c tools.Contexto) (tools.Resultado, error) {
		peticion, ok := args.(tools.PeticionBuscarEnArchivos)
		if !ok {
			return corregible(tools.CodBadArgs + ": argumentos inesperados")
		}
		patron := strings.TrimSpace(peticion.Patron)
		if patron == "" {
			return corregible(tools.CodBadArgs + ": el `patron` no puede estar vacío")
		}
		base := strings.TrimSpace(peticion.Ruta)
		if base == "" {
			base = "."
		}
		abs, codigo := Resolver(o.Raiz, base)
		if codigo != "" {
			return corregible(codigo)
		}
		if err := ValidarAcceso(ctx, o.Raiz, abs, peticion.Motivo, c); err != nil {
			return resultado(err)
		}
		if _, err := os.Stat(abs); err != nil {
			return corregible("la ruta no existe: " + Canonica(o.Raiz, abs) + ". Verifica la ruta o usa listar_carpeta para localizarla")
		}

		coincidencias, truncado := o.buscarContenido(abs, patron)
		salida, _ := json.Marshal(coincidencias)
		return tools.Resultado{
			Salida:   string(salida),
			Truncado: truncado,
			Meta:     map[string]any{"medida": pluralCoincidencias(len(coincidencias))},
		}, nil
	}
}

// buscarNombres recorre el proyecto y devuelve, ordenadas, las rutas canónicas
// cuyo nombre contiene el patrón (sin distinguir mayúsculas).
func (o *Operaciones) buscarNombres(patron string) ([]string, bool) {
	objetivo := strings.ToLower(patron)
	var encontradas []string
	truncado := false
	_ = filepath.WalkDir(o.Raiz, func(ruta string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if ruta != o.Raiz && saltarCarpeta(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.Contains(strings.ToLower(d.Name()), objetivo) {
			if len(encontradas) >= MaxCoincidencias {
				truncado = true
				return filepath.SkipAll
			}
			encontradas = append(encontradas, Canonica(o.Raiz, ruta))
		}
		return nil
	})
	sort.Strings(encontradas)
	return encontradas, truncado
}

// buscarContenido recorre `base` (archivo o carpeta) y devuelve las líneas que
// contienen el patrón en los archivos de texto que caben.
func (o *Operaciones) buscarContenido(base, patron string) ([]Coincidencia, bool) {
	objetivo := strings.ToLower(patron)
	var encontradas []Coincidencia
	truncado := false
	_ = filepath.WalkDir(base, func(ruta string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if ruta != base && saltarCarpeta(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := d.Info()
		if err != nil || info.Size() > LimiteArchivoBuscado {
			return nil
		}
		datos, err := os.ReadFile(ruta)
		if err != nil || !esTexto(datos) {
			return nil
		}
		for i, linea := range partirLineas(datos) {
			if !strings.Contains(strings.ToLower(linea), objetivo) {
				continue
			}
			if len(encontradas) >= MaxCoincidencias {
				truncado = true
				return filepath.SkipAll
			}
			encontradas = append(encontradas, Coincidencia{
				Archivo:   Canonica(o.Raiz, ruta),
				Linea:     i + 1,
				Fragmento: recortarFragmento(strings.TrimSpace(linea)),
			})
		}
		return nil
	})
	return encontradas, truncado
}

// carpetasIgnoradas son las carpetas que nunca se recorren al buscar: los
// sistemas de control de versiones y las dependencias. El resto —incluidas las
// ocultas como `.agents` o `.localcli`— sí se inspecciona, porque contienen
// contenido del proyecto que el agente necesita encontrar.
var carpetasIgnoradas = map[string]bool{
	".git":         true,
	".hg":          true,
	".svn":         true,
	"node_modules": true,
}

// saltarCarpeta dice si una carpeta no se recorre.
func saltarCarpeta(nombre string) bool {
	return carpetasIgnoradas[nombre]
}

// recortarFragmento acota el trozo de línea que se devuelve.
func recortarFragmento(linea string) string {
	return cortarUTF8(linea, largoFragmento)
}

func pluralCoincidencias(n int) string {
	if n == 1 {
		return "1 coincidencia"
	}
	return itoa(n) + " coincidencias"
}
