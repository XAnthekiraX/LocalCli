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
	"os"

	"localcli/internal/tools"
)

// LimiteLecturaBytes es el tamaño máximo que se lee de un golpe. No lo fija la
// documentación; es defensivo: evita que un archivo enorme llene la memoria del
// harness por una petición del modelo.
const LimiteLecturaBytes = 2 << 20 // 2 MiB

// LeerArchivo devuelve el contenido de un archivo del proyecto.
func LeerArchivo(proyecto, ruta string) (tools.RespuestaLeerArchivo, error) {
	abs, err := Resolver(proyecto, ruta)
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
