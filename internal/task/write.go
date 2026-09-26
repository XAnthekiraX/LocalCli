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

// EscribirArchivo escribe `contenido` en `ruta` de forma atómica y sin salirse
// de `ai/tasks/`.
//
// Dos garantías que un os.WriteFile no da:
//
//	A atomicidad. Si el proceso muere entre el truncate y el último write, un
//	  WriteFile deja un MAIN-TASKS.md a medio escribir: se pierde la tabla
//	  entera y con ella el estado de todas las tareas. Con temp+rename el
//	  archivo viejo o el nuevo están siempre completos; nunca un híbrido.
//	B la frontera. `ruta` se resuelve contra symlinks antes de comprobar, así
//	  que un enlace dentro de ai/tasks/ que apunte fuera no sirve de puerta
//	  atrás. Este módulo escribe el TODO del proyecto, nada más; el resto del
//	  árbol lo escriben fileops (T-B008) con su propia aprobación.
//
// La aprobación previa vive en el flujo; aquí solo se garantiza que lo que se
// escribe cae dentro del directorio del TODO.
func EscribirArchivo(raiz, ruta string, contenido []byte, perm os.FileMode) error {
	abs, err := filepath.Abs(ruta)
	if err != nil {
		return fmt.Errorf("task: ruta %s: %w", ruta, err)
	}
	if err := comprobarFronteraTasks(raiz, abs); err != nil {
		return err
	}

	dir := filepath.Dir(abs)
	// El temp tiene que vivir en el mismo directorio que el destino: rename
	// entre sistemas de archivos distintos no es atómico.
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(abs)+".tmp-*")
	if err != nil {
		return fmt.Errorf("task: crear temporal en %s: %w", dir, err)
	}
	tmpName := tmp.Name()
	// Cualquier fallo posterior tiene que dejar el directorio como estaba: sin
	// esto un error a mitad de escritura acumula archivos .tmp.
	defer os.Remove(tmpName)

	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		return fmt.Errorf("task: permisos de %s: %w", tmpName, err)
	}
	if _, err := tmp.Write(contenido); err != nil {
		tmp.Close()
		return fmt.Errorf("task: escribir %s: %w", tmpName, err)
	}
	// Sin fsync, el rename puede quedar en la caché y un corte de luz devuelve
	// el archivo viejo, o peor, el nuevo a medias.
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("task: sincronizar %s: %w", tmpName, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("task: cerrar %s: %w", tmpName, err)
	}
	if err := os.Rename(tmpName, abs); err != nil {
		return fmt.Errorf("task: renombrar a %s: %w", abs, err)
	}
	// Sincronizar la entrada de directorio hace durable el rename.
	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		_ = d.Close()
	}
	return nil
}

// comprobarFronteraTasks devuelve error si `abs` no está dentro de
// <raiz>/ai/tasks/. Se comparan rutas ya resueltas para que un symlink no
// sirva de atajo, y en minúsculas porque el nombre del directorio es fijo pero
// el sistema de archivos puede no distinguir mayúsculas.
func comprobarFronteraTasks(raiz, abs string) error {
	base, err := filepath.Abs(raiz)
	if err != nil {
		return fmt.Errorf("task: raíz %s: %w", raiz, err)
	}
	permitido := filepath.Join(base, "ai", "tasks")
	resueltaPermitido, err := filepath.EvalSymlinks(permitido)
	if err != nil {
		return fmt.Errorf("task: no existe %s: %w", permitido, err)
	}
	// El archivo puede no existir todavía (se crea): se resuelve el directorio.
	resuelta := abs
	if r, err := filepath.EvalSymlinks(abs); err == nil {
		resuelta = r
	} else if r, err := filepath.EvalSymlinks(filepath.Dir(abs)); err == nil {
		resuelta = filepath.Join(r, filepath.Base(abs))
	}
	if !dentroDe(resuelta, resueltaPermitido) {
		return fmt.Errorf("task: %s está fuera de ai/tasks/; este módulo solo escribe el TODO", abs)
	}
	return nil
}

// dentroDe dice si `ruta` es `dir` o está bajo ella, con separadores correctos:
// un prefijo de texto plano daría por bueno `ai/tasks-secreto/x.md`.
func dentroDe(ruta, dir string) bool {
	if strings.EqualFold(ruta, dir) {
		return true
	}
	// filepath.Rel distingue mayúsculas, así que AI/TASKS/x.md daría ../TASKS
	// y sería rechazado aunque sea el mismo directorio. Se normaliza antes.
	rel, err := filepath.Rel(strings.ToLower(dir), strings.ToLower(ruta))
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// CambiarEstado lee el archivo del TODO dado, cambia el estado del elemento
// `id` y lo reescribe. Devuelve el contenido nuevo.
func CambiarEstado(raiz, ruta, id string, estado Estado) ([]byte, error) {
	b, err := os.ReadFile(ruta)
	if err != nil {
		return nil, fmt.Errorf("task: leer %s: %w", ruta, err)
	}
	nuevo, err := ActualizarEstadoEnContenido(b, id, estado)
	if err != nil {
		return nil, err
	}
	if err := EscribirArchivo(raiz, ruta, nuevo, 0o644); err != nil {
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
		if err == nil && contieneFila(b, id, capa) {
			if _, err := CambiarEstado(raiz, mainPath, id, estado); err != nil {
				return "", err
			}
			return mainPath, nil
		}
	}
	ruta, err := ArchivoTareaPara(raiz, capa, id)
	if err != nil {
		return "", err
	}
	if _, err := CambiarEstado(raiz, ruta, id, estado); err != nil {
		return "", err
	}
	return ruta, nil
}

// contieneFila informa si el contenido tiene ese ID en una fila de datos de
// una tabla de elementos.
//
// La capa se compara de verdad. ParseTabla recibe la capa del ARCHIVO, no la
// del elemento, así que la fija sin comprobarla: pedir T-F001 en un archivo de
// backend daba un no-encontrado aunque la fila existiera, y dar "existe" para
// un ID de otra capa habría hecho que CambiarEstadoElemento escribiera en el
// archivo equivocado en vez de avisar de que la tarea no está.
func contieneFila(contenido []byte, id string, capa Capa) bool {
	deID, ok := CapaDeID(id)
	if !ok || deID != capa {
		return false
	}
	elems, err := ParseTabla(contenido, capa)
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
