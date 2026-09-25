package task

// write.go — T-B004-04: escritura quirúrgica del estado de un elemento.
//
// Regla dura (SPEC-CICLO-TRABAJO y la skill backend-executor): el executor
// cambia ÚNICAMENTE la celda Estado de la fila correspondiente; el resto del
// archivo (cabecera, otras filas, prosa, referencias) debe quedar byte a byte
// igual salvo esa celda. Por eso no se regenera la tabla desde los Elementos
// parseados: se localiza la fila por su columna ID y se reemplaza solo la
// celda Estado in situ, conservando el espaciado original.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ActualizarEstadoEnContenido devuelve contenido con la celda Estado de la
// fila cuyo ID es `id` cambiado a `estado`. Error si la fila no existe, si
// el estado no pertenece al catálogo cerrado o si la tabla no tiene columna
// Estado. El resto del archivo no se toca.
func ActualizarEstadoEnContenido(contenido []byte, id string, estado Estado) ([]byte, error) {
	if !EsEstado(string(estado)) {
		return nil, fmt.Errorf("task: estado inválido %q", estado)
	}
	lineas := strings.Split(strings.ReplaceAll(string(contenido), "\r\n", "\n"), "\n")
	crlf := strings.Contains(string(contenido), "\r\n")

	var cabecera []string
	cambiadas := 0
	for i, ln := range lineas {
		t := strings.TrimSpace(ln)
		if !strings.HasPrefix(t, "|") {
			cabecera = nil
			continue
		}
		celdas := dividirFila(t)
		if esSeparadora(celdas) {
			continue
		}
		if cabecera == nil {
			if indiceColumna(normalizarCabecera(celdas), "id") < 0 {
				continue
			}
			cabecera = normalizarCabecera(celdas)
			continue
		}
		if indiceColumna(normalizarCabecera(celdas), "id") < 0 && len(celdas) != len(cabecera) {
			cabecera = nil
			continue
		}
		if limpiarCelda(celdaEn(cabecera, celdas, "id")) != id {
			continue
		}
		iEst := indiceColumna(cabecera, "estado")
		if iEst < 0 || iEst >= len(celdas) {
			return nil, fmt.Errorf("task: la tabla de %s no tiene columna Estado", id)
		}
		nuevas := make([]string, len(celdas))
		copy(nuevas, celdas)
		nuevas[iEst] = string(estado)
		lineas[i] = unirFila(nuevas)
		cambiadas++
	}
	if cambiadas == 0 {
		return nil, fmt.Errorf("task: no encontré la fila %q en el archivo", id)
	}
	out := strings.Join(lineas, "\n")
	if crlf {
		out = strings.ReplaceAll(out, "\n", "\r\n")
	}
	return []byte(out), nil
}

// unirFila reconstruye una fila Markdown "| a | b |" con padding de una
// celda alrededor de cada valor (el formato canónico de los archivos del TODO).
func unirFila(celdas []string) string {
	parts := make([]string, len(celdas))
	for i, c := range celdas {
		parts[i] = " " + strings.TrimSpace(c) + " "
	}
	return "|" + strings.Join(parts, "|") + "|"
}

// celdaEn devuelve la celda de `celdas` cuya columna (por alias normalizado)
// coincide, o "".
func celdaEn(cabecera, celdas []string, alias ...string) string {
	i := indiceColumna(cabecera, alias...)
	if i < 0 || i >= len(celdas) {
		return ""
	}
	return celdas[i]
}

// EscribirArchivo aplica un cambio de contenido respetando la frontera de
// rutas básicas: solo escrituras atómicas simples (read-modify-write).
// La aprobación de cambios vive en fileops (T-B008); aquí el módulo task
// escribe exclusivamente dentro de ai/tasks/ tras que el flujo lo autorice.
func EscribirArchivo(ruta string, contenido []byte, perm os.FileMode) error {
	if err := os.WriteFile(ruta, contenido, perm); err != nil {
		return fmt.Errorf("task: escribir %s: %w", ruta, err)
	}
	return nil
}

// CambiarEstado lee el archivo del TODO dado, cambia el estado del elemento
// `id` y lo reescribe. Devuelve el contenido nuevo.
func CambiarEstado(ruta, id string, estado Estado) ([]byte, error) {
	b, err := os.ReadFile(ruta)
	if err != nil {
		return nil, fmt.Errorf("task: leer %s: %w", ruta, err)
	}
	nuevo, err := ActualizarEstadoEnContenido(b, id, estado)
	if err != nil {
		return nil, err
	}
	if err := EscribirArchivo(ruta, nuevo, 0o644); err != nil {
		return nil, err
	}
	return nuevo, nil
}

// CambiarEstadoElemento localiza el archivo correcto para el ID (MAIN si es
// tarea grande sin sufijo -NN propio de fila… pero ambas tablas conviven) y
// aplica el cambio. Busca primero en MAIN-TASKS.md (filas de tareas grandes)
// y si no está ahí, en el NNN-task-*.md que corresponde a su prefijo.
func CambiarEstadoElemento(raiz string, capa Capa, id string, estado Estado) (string, error) {
	mainPath := filepath.Join(raiz, "ai", "tasks", string(capa), NombreMainTasks)
	if _, err := os.Stat(mainPath); err == nil {
		b, err := os.ReadFile(mainPath)
		if err == nil && contieneFila(b, id) {
			if _, err := CambiarEstado(mainPath, id, estado); err != nil {
				return "", err
			}
			return mainPath, nil
		}
	}
	ruta, err := ArchivoTareaPara(raiz, capa, id)
	if err != nil {
		return "", err
	}
	if _, err := CambiarEstado(ruta, id, estado); err != nil {
		return "", err
	}
	return ruta, nil
}

// contieneFila informa si alguna fila de datos de las tablas del contenido
// tiene ese ID exacto en su columna ID.
func contieneFila(contenido []byte, id string) bool {
	elems, err := ParseTabla(contenido, CapaBackend) // la capa no afecta a la búsqueda por ID
	if err != nil {
		return false
	}
	for _, e := range elems {
		if e.ID == id {
			return true
		}
	}
	return false
}
