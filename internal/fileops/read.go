package fileops

// read.go — T-B008-02: leer un archivo dentro de la frontera.
//
// Fuente de verdad: ai/docs/backend/02-interfaces/dto/TOOLS-DTO.md §3
// (`leer_archivo` devuelve "el contenido del archivo", string) y
// ai/docs/backend/03-security/SECURITY.md §5 ("El contenido de los archivos
// que se entregan se lee antes, y lo que sale se registra").
//
// TOOLS-DTO no define paginación ni metadatos para `leer_archivo`: la respuesta
// es el contenido. Sí hay un límite defensivo, porque la lectura la pide el
// modelo y no puede cargar sin fin en memoria; un archivo que lo supera se
// rechaza en vez de truncarse en silencio (VALIDATION.md: nada de fallos
// silenciosos).

import (
	"context"
	"os"
	"strings"

	"localcli/internal/tools"
)

// LimiteLecturaBytes es el tamaño máximo que se lee de un golpe. No lo fija la
// documentación; es defensivo: evita que un archivo enorme llene la memoria del
// harness por una petición del modelo.
const LimiteLecturaBytes = 2 << 20 // 2 MiB

// LeerArchivo devuelve el contenido de un archivo del proyecto. Es el camino de
// quien lee sin poder pedir permiso (el nodo de contexto): una ruta de fuera no
// se toca.
func LeerArchivo(proyecto, ruta string) (tools.RespuestaLeerArchivo, error) {
	return (&Ops{Proyecto: proyecto}).LeerArchivo(context.Background(), ruta)
}

// extensionesImagen son las que la vista usa para adjuntar una imagen al turno
// (internal/tui/adjuntos.go). Aquí sirven para no leer un binario: la imagen ya
// viaja en el mensaje.
var extensionesImagen = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true, ".bmp": true,
}

// esRutaDeImagen mira solo la extensión, como el detector de adjuntos.
func esRutaDeImagen(ruta string) bool {
	i := strings.LastIndex(ruta, ".")
	if i < 0 {
		return false
	}
	return extensionesImagen[strings.ToLower(ruta[i:])]
}

// LeerArchivo devuelve el contenido de un archivo. Si la ruta sale de la carpeta
// del proyecto, pide permiso antes de leerla (SPEC-ARCHIVOS §Reglas). Una imagen
// no se lee: ya viaja adjunta en el turno, así que se devuelve una corrección en
// vez de un binario —y nunca se pide permiso por ella—.
func (o *Ops) LeerArchivo(ctx context.Context, ruta string) (tools.RespuestaLeerArchivo, error) {
	if esRutaDeImagen(ruta) {
		return tools.RespuestaLeerArchivo{}, nuevoError(CodigoArgumentosInvalidos,
			"«"+ruta+"» es una imagen: viaja adjunta al turno, no hace falta leerla")
	}
	abs, err := o.rutaDeLectura(ctx, "leer_archivo", ruta)
	if err != nil {
		return tools.RespuestaLeerArchivo{}, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return tools.RespuestaLeerArchivo{}, nuevoError(CodigoArgumentosInvalidos,
			"no se pudo leer "+ruta+": "+err.Error())
	}
	if info.IsDir() {
		return tools.RespuestaLeerArchivo{}, nuevoError(CodigoArgumentosInvalidos,
			ruta+" es una carpeta, no un archivo")
	}
	if info.Size() > LimiteLecturaBytes {
		return tools.RespuestaLeerArchivo{}, nuevoError(CodigoArgumentosInvalidos,
			"el archivo "+ruta+" supera el límite de lectura")
	}
	datos, err := os.ReadFile(abs)
	if err != nil {
		return tools.RespuestaLeerArchivo{}, nuevoError(CodigoArgumentosInvalidos,
			"no se pudo leer "+ruta+": "+err.Error())
	}
	// El contenido va tal cual: leer no normaliza lo que el archivo tiene.
	return tools.RespuestaLeerArchivo{Contenido: string(datos)}, nil
}
