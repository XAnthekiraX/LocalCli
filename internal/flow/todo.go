package flow

// todo.go — T-B010-03: generar el TODO de la ejecución al detectar trabajo.
//
// Fuente de verdad: [[specs/SPEC-COLA-TAREAS]] ("el TODO es la cola"; se crea
// al detectar una petición ordenada, con los elementos que van a seguir, en
// orden) y [[specs/SPEC-CICLO-TRABAJO]] §Reglas ("la acción de cada elemento
// coincide con la de su tarea principal").
//
// El TODO vive en archivos y su formato es el que `task` sabe leer: una tabla
// Markdown en `ai/tasks/<capa>/MAIN-TASKS.md` con las columnas que reconoce
// `task.ParseTabla`. Por eso se escribe a través de `task.EscribirArchivo` (que
// garantiza la frontera de `ai/tasks/` y la atomicidad) y no con os.WriteFile.

import (
	"fmt"
	"path/filepath"
	"strings"

	"localcli/internal/task"
)

// ElementoTODO es un elemento a escribir en el TODO. Su acción la determina la
// entrada del ciclo (crear/actualizar/eliminar/verificar).
type ElementoTODO struct {
	ID          string
	Accion      task.Accion
	Descripcion string
	DependeDe   []string
	Documentos  []string
}

// GenerarTODO escribe el MAIN-TASKS.md de la capa con los elementos dados, en
// el orden recibido, y devuelve la ruta del archivo. Todos los elementos nacen
// pendientes.
func GenerarTODO(raiz string, capa task.Capa, elementos []ElementoTODO) (string, error) {
	if !task.EsCapa(string(capa)) {
		return "", fmt.Errorf("flow: capa inválida %q", capa)
	}
	if len(elementos) == 0 {
		return "", fmt.Errorf("flow: no hay elementos para el TODO de %s", capa)
	}

	var b strings.Builder
	b.WriteString("# MAIN-TASKS — " + string(capa) + "\n\n")
	b.WriteString("| ID | Acción | Tarea | Dep | Estado | Documentos |\n")
	b.WriteString("|----|--------|-------|-----|--------|------------|\n")
	for _, e := range elementos {
		if !task.IDValido(e.ID) {
			return "", fmt.Errorf("flow: id de elemento inválido %q", e.ID)
		}
		if !task.EsAccion(string(e.Accion)) {
			return "", fmt.Errorf("flow: acción inválida %q en %s", e.Accion, e.ID)
		}
		desc := strings.TrimSpace(e.Descripcion)
		if desc == "" {
			desc = e.ID
		}
		b.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %s |\n",
			e.ID, e.Accion, desc, celdaLista(e.DependeDe), task.EstadoPendiente, celdaWiki(e.Documentos)))
	}

	ruta := filepath.Join(raiz, "ai", "tasks", string(capa), task.NombreMainTasks)
	if err := task.EscribirArchivo(raiz, ruta, []byte(b.String()), 0o644); err != nil {
		return "", err
	}
	return ruta, nil
}

// celdaLista pinta una lista de dependencias con el marcador de vacío que
// entiende el parser ("—").
func celdaLista(xs []string) string {
	if len(xs) == 0 {
		return "—"
	}
	return strings.Join(xs, ", ")
}

// celdaWiki pinta las rutas de contexto como enlaces wiki, para que
// `task.ParseTabla` las recoja como documentos. Referenciar, no incrustar.
func celdaWiki(xs []string) string {
	if len(xs) == 0 {
		return "—"
	}
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		x = strings.TrimSpace(x)
		if x == "" {
			continue
		}
		if !strings.HasPrefix(x, "[[") {
			x = "[[" + x + "]]"
		}
		out = append(out, x)
	}
	if len(out) == 0 {
		return "—"
	}
	return strings.Join(out, " ")
}
