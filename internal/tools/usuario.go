package tools

// usuario.go — T-B024-08 y T-B024-09: las herramientas del usuario.
//
// Fuente de verdad: ai/docs/specs/SPEC-TOOLS.md §Herramientas del usuario,
// ai/docs/backend/02-interfaces/TOOLS.md §9 y
// ai/docs/backend/04-infrastructure/CONFIGURATION.md §4 ("Herramientas del
// usuario": dónde viven y qué declaran).
//
// No son código: son declaraciones que LocalCli ejecuta. El handler es un
// adaptador fino sobre `exec`, así que hereda sin código nuevo el aislamiento
// de Landlock, el límite de tiempo y el recorte. `tools` no importa `exec`: el
// runner se inyecta en el cableado (DECISIONS.md).

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Valores admitidos del campo `modo`.
const (
	ModoLee     = "lee"
	ModoEscribe = "escribe"
)

// DirHerramientas es la carpeta del proyecto donde el usuario declara las
// suyas, relativa a la raíz.
const DirHerramientas = ".localcli/tools"

// HerramientaUsuario es el contrato JSON de una herramienta del usuario.
type HerramientaUsuario struct {
	Nombre          string   `json:"nombre"`
	Descripcion     string   `json:"descripcion"`
	Modo            string   `json:"modo,omitempty"`
	Equipo          []string `json:"equipo"`
	TimeoutSegundos int      `json:"timeout_segundos,omitempty"`
}

// Timeout devuelve el límite propio de la herramienta, o cero (el de la
// terminal) si no lo declara.
func (h HerramientaUsuario) Timeout() time.Duration {
	if h.TimeoutSegundos <= 0 {
		return 0
	}
	return time.Duration(h.TimeoutSegundos) * time.Second
}

// Runner ejecuta un equipo declarado por el usuario como subproceso argv, sin
// shell. Lo implementa `exec.Ejecutor`: no reimplementa la terminal, la usa.
type Runner interface {
	EjecutarEquipo(ctx context.Context, equipo []string, entrada []byte, timeout time.Duration) (RespuestaEjecutarComando, error)
}

// DirHerramientasDe devuelve la carpeta de herramientas de un proyecto.
func DirHerramientasDe(proyecto string) string {
	return filepath.Join(proyecto, filepath.FromSlash(DirHerramientas))
}

// CargarHerramientasUsuario lee los `.json` de una carpeta de herramientas.
// Devuelve las declaraciones válidas y los avisos de las que se han saltado.
// Un archivo inválido se ignora y el arranque continúa: no tumba nada.
func CargarHerramientasUsuario(dir string) ([]HerramientaUsuario, []string) {
	entradas, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, []string{"no se pudo leer " + dir + ": " + err.Error()}
	}
	nombres := make([]string, 0, len(entradas))
	for _, e := range entradas {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		nombres = append(nombres, e.Name())
	}
	sort.Strings(nombres)

	var validas []HerramientaUsuario
	var avisos []string
	vistas := map[string]bool{}
	for _, nombre := range nombres {
		ruta := filepath.Join(dir, nombre)
		datos, err := os.ReadFile(ruta)
		if err != nil {
			avisos = append(avisos, "herramienta ignorada ("+nombre+"): no se pudo leer")
			continue
		}
		var h HerramientaUsuario
		if err := json.Unmarshal(datos, &h); err != nil {
			avisos = append(avisos, "herramienta ignorada ("+nombre+"): JSON inválido: "+err.Error())
			continue
		}
		h.Nombre = strings.TrimSpace(h.Nombre)
		h.Descripcion = strings.TrimSpace(h.Descripcion)
		if err := validarUsuario(h); err != nil {
			avisos = append(avisos, "herramienta ignorada ("+nombre+"): "+err.Error())
			continue
		}
		if Existe(h.Nombre) {
			avisos = append(avisos, "herramienta ignorada ("+nombre+"): `"+h.Nombre+"` choca con una incluida")
			continue
		}
		if vistas[h.Nombre] {
			avisos = append(avisos, "herramienta ignorada ("+nombre+"): `"+h.Nombre+"` ya está declarada")
			continue
		}
		vistas[h.Nombre] = true
		validas = append(validas, h)
	}
	return validas, avisos
}

// validarUsuario comprueba los campos obligatorios y el modo. `escribe` se
// rechaza con el motivo: la única vía sancionada para escribir son las
// herramientas de archivo, que solo tiene `build` y que pasan por aprobación.
func validarUsuario(h HerramientaUsuario) error {
	if h.Nombre == "" {
		return errUsuario("falta el campo obligatorio `nombre`")
	}
	if h.Descripcion == "" {
		return errUsuario("falta el campo obligatorio `descripcion`")
	}
	if len(h.Equipo) == 0 {
		return errUsuario("falta el campo obligatorio `equipo` (o está vacío)")
	}
	for _, a := range h.Equipo {
		if strings.TrimSpace(a) == "" {
			return errUsuario("`equipo` tiene un argumento vacío")
		}
	}
	switch strings.TrimSpace(h.Modo) {
	case "", ModoLee:
	default:
		if h.Modo == ModoEscribe {
			return errUsuario("`modo: escribe` no se admite: la única vía de escritura son las herramientas de archivo, que solo tiene `build`")
		}
		return errUsuario("`modo` desconocido: " + h.Modo)
	}
	return nil
}

// NuevaHerramientaUsuario construye la entrada del registro para una
// declaración válida. Siempre es de lectura: su vista para el modelo es un
// objeto genérico, y su ejecución pasa por aprobación sin excepción.
func NuevaHerramientaUsuario(h HerramientaUsuario, r Runner) Herramienta {
	return Herramienta{
		Nombre:      h.Nombre,
		Descripcion: h.Descripcion,
		Categoria:   CatUsuario,
		Modo:        Lee,
		Esquema:     EsquemaObjeto(),
		Ejecutar:    ejecutarUsuario(h, r),
	}
}

// ejecutarUsuario envuelve el equipo declarado: pide aprobación SIEMPRE —el
// comando lo escribió el usuario, no la herramienta, así que no puede estar en
// la lista blanca— e inyecta los argumentos por la entrada estándar como un
// único JSON, nunca en la línea de comandos.
func ejecutarUsuario(h HerramientaUsuario, r Runner) Ejecutar {
	return func(ctx context.Context, args any, c Contexto) (Resultado, error) {
		if c.Ask == nil {
			return Resultado{Error: "la herramienta `" + h.Nombre + "` necesita aprobación y no hay forma de pedirla"}, nil
		}
		d, err := c.Ask(ctx, Solicitud{Descripcion: "ejecutar la herramienta `" + h.Nombre + "` (`" + strings.Join(h.Equipo, " ") + "`)"})
		if err != nil {
			return Resultado{}, err
		}
		if !d.Aprobada {
			return Resultado{Error: "el usuario declinó ejecutar la herramienta `" + h.Nombre + "`"}, nil
		}
		if r == nil {
			return Resultado{Error: "no hay ejecutor conectado para la herramienta `" + h.Nombre + "`"}, nil
		}
		datos, mErr := json.Marshal(args)
		if mErr != nil {
			datos = []byte("{}")
		}
		resp, err := r.EjecutarEquipo(ctx, h.Equipo, datos, h.Timeout())
		if err != nil {
			return Resultado{}, err
		}
		return Resultado{
			Salida:   salidaUsuario(resp),
			Truncado: resp.Truncado,
			Meta:     map[string]any{"codigo": resp.Codigo, "termino": resp.Termino},
		}, nil
	}
}

// salidaUsuario concatena la salida y la de error, igual que hace la terminal:
// el modelo no necesita saber de dónde vino.
func salidaUsuario(r RespuestaEjecutarComando) string {
	var b strings.Builder
	if s := strings.TrimRight(r.Salida, "\n"); s != "" {
		b.WriteString(s)
	}
	if e := strings.TrimRight(r.Error, "\n"); e != "" {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(e)
	}
	if !r.Termino {
		b.WriteString("\n[el ejecutable no terminó]")
	}
	return b.String()
}

// HuellaHerramienta es un identificador estable de una declaración, para
// trazabilidad. No incluye la ruta del archivo.
func HuellaHerramienta(h HerramientaUsuario) string {
	suma := sha256.Sum256([]byte(h.Nombre + "\x00" + strings.Join(h.Equipo, "\x00")))
	return hex.EncodeToString(suma[:8])
}

func errUsuario(msg string) error {
	return &ErrorHerramienta{Codigo: "E_BAD_ARGS", Mensaje: msg, sentinel: ErrArgumentosInvalidos}
}
