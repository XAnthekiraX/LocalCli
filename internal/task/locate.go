package task

// locate.go — T-B004-03: localizar los archivos del TODO bajo ai/tasks/<capa>/.
//
// Estructura documentada ([[backend/BACKEND]] [49] y SPEC-CICLO-TRABAJO):
//
//	ai/tasks/
//	  backend/    MAIN-TASKS.md + NNN-task-<modulo>.md
//	  frontend/   idem
//	  database/   idem
//
// El prefijo numérico NNN (tres dígitos) identifica la tarea grande de la
// capa; MAIN-TASKS.md es el índice. La capa se deduce del nombre del
// directorio que contiene los archivos.

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// NombreMainTasks es el archivo índice del TODO de cada capa.
const NombreMainTasks = "MAIN-TASKS.md"

// nombreTareaArchivoRE reconoce los nombres NNN-task-*.md de las tareas pequeñas.
var nombreTareaArchivoRE = regexp.MustCompile(`^([0-9]{3})-task-[a-z0-9-]+\.md$`)

// CapaDesdeRuta deduce la capa ("frontend" | "backend" | "database") a partir
// de una ruta bajo ai/tasks/. Devuelve error si la ruta no pertenece al árbol
// del TODO o la carpeta no es una capa conocida.
func CapaDesdeRuta(ruta string) (Capa, error) {
	rutaNorm := filepath.ToSlash(filepath.Clean(ruta))
	i := strings.LastIndex(rutaNorm, "ai/tasks/")
	if i < 0 {
		return "", fmt.Errorf("task: %s no está bajo ai/tasks/", ruta)
	}
	resto := strings.TrimPrefix(rutaNorm[i+len("ai/tasks/"):], "/")
	partes := strings.Split(resto, "/")
	if len(partes) < 2 || partes[0] == "" {
		return "", fmt.Errorf("task: %s no indica una capa (falta ai/tasks/<capa>/)", ruta)
	}
	if !EsCapa(partes[0]) {
		return "", fmt.Errorf("task: capa desconocida %q en %s", partes[0], ruta)
	}
	return Capa(partes[0]), nil
}

// prefijoIDRE captura el NNN del ID: son los tres dígitos que siguen a la
// letra de capa ("T-B004" o "T-B004-02"); el sufijo -NN tiene solo dos.
var prefijoIDRE = regexp.MustCompile(`[A-Za-z][0-9]?([0-9]{3})`)

// PrefijoDeID extrae el número NNN del ID de un elemento: "T-B004-02" ⇒ "004",
// "T-B004" ⇒ "004". Acepta IDs de cualquier capa (la letra de la capa no
// interviene en el prefijo).
func PrefijoDeID(id string) (string, bool) {
	m := prefijoIDRE.FindStringSubmatch(id)
	if m == nil {
		return "", false
	}
	return m[1], true
}

// ArchivosTODO lista los archivos del TODO de una capa dentro de raiz
// (p. ej. el directorio del proyecto), ordenados por prefijo NNN ascendente.
// Incluye MAIN-TASKS.md primero (es el índice) seguido de los NNN-task-*.md.
func ArchivosTODO(raiz string, capa Capa) ([]string, error) {
	if !EsCapa(string(capa)) {
		return nil, fmt.Errorf("task: capa inválida %q", capa)
	}
	dir := filepath.Join(raiz, "ai", "tasks", string(capa))
	entradas, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("task: leer %s: %w", dir, err)
	}
	var tasks []string
	for _, e := range entradas {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if n == NombreMainTasks {
			continue
		}
		if nombreTareaArchivoRE.MatchString(n) {
			tasks = append(tasks, filepath.Join(dir, n))
		}
	}
	sort.Slice(tasks, func(i, j int) bool {
		return filepath.Base(tasks[i]) < filepath.Base(tasks[j])
	})
	mainPath := filepath.Join(dir, NombreMainTasks)
	if _, err := os.Stat(mainPath); err != nil {
		return nil, fmt.Errorf("task: %s no contiene %s: %w", dir, NombreMainTasks, err)
	}
	return append([]string{mainPath}, tasks...), nil
}

// ArchivoTareaPara devuelve la ruta del archivo NNN-task-*.md de la capa que
// corresponde al ID dado (por su prefijo NNN). Error si no existe ninguno.
func ArchivoTareaPara(raiz string, capa Capa, id string) (string, error) {
	prefijo, ok := PrefijoDeID(id)
	if !ok {
		return "", fmt.Errorf("task: id %q no tiene prefijo NNN", id)
	}
	dir := filepath.Join(raiz, "ai", "tasks", string(capa))
	entradas, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("task: leer %s: %w", dir, err)
	}
	for _, e := range entradas {
		if e.IsDir() {
			continue
		}
		m := nombreTareaArchivoRE.FindStringSubmatch(e.Name())
		if m != nil && m[1] == prefijo {
			return filepath.Join(dir, e.Name()), nil
		}
	}
	return "", fmt.Errorf("task: no hay archivo %s-task-*.md en %s", prefijo, dir)
}

// CargarElementos parsea todos los archivos del TODO de una capa y devuelve
// sus elementos (índice primero, luego las tareas pequeñas por prefijo).
func CargarElementos(raiz string, capa Capa) ([]Elemento, error) {
	archivos, err := ArchivosTODO(raiz, capa)
	if err != nil {
		return nil, err
	}
	var todos []Elemento
	for _, a := range archivos {
		b, err := os.ReadFile(a)
		if err != nil {
			return nil, fmt.Errorf("task: leer %s: %w", a, err)
		}
		elems, err := ParseTabla(b, capa)
		if err != nil {
			return nil, fmt.Errorf("task: %s: %w", a, err)
		}
		todos = append(todos, elems...)
	}
	return todos, nil
}
