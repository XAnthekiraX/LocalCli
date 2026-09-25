package task

// model.go — T-B004-01: el tipo Elemento con los 7 campos del frontmatter.
//
// Fuente de verdad: ai/docs/backend/DECISIONS.md [28]. El frontmatter de un
// elemento del TODO tiene exactamente: id, capa, accion, estado, depende_de,
// bloqueada_por y documentos. El contexto se referencia por rutas, no se
// incrusta (evita una segunda copia que diverge del original).
//
// Regla dura del mismo punto: `bloqueada_por` solo aparece con
// `estado = bloqueada`. La validación vive en validate.go (T-B004-05); aquí
// solo se fijan los vocabularios cerrados para que el resto del módulo
// (parse, write, deps, queue) compare contra constantes, no contra literales.

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Accion es la acción declarada por el ciclo de trabajo
// ([[specs/SPEC-CICLO-TRABAJO]]): qué hará la implementación, no qué se borra.
type Accion string

const (
	AccionCrear      Accion = "crear"
	AccionActualizar Accion = "actualizar"
	AccionEliminar   Accion = "eliminar"
)

// EsAccion valida que el texto pertenezca al catálogo cerrado de acciones.
func EsAccion(s string) bool {
	switch Accion(s) {
	case AccionCrear, AccionActualizar, AccionEliminar:
		return true
	}
	return false
}

// Estado es el estado de una tarea ([[backend-executor SKILL]], estados de
// las tareas): pendiente → en_progreso → completada, y bloqueada si falta
// información.
type Estado string

const (
	EstadoPendiente  Estado = "pendiente"
	EstadoEnProgreso Estado = "en_progreso"
	EstadoCompletada Estado = "completada"
	EstadoBloqueada  Estado = "bloqueada"
)

// EsEstado valida que el texto pertenezca al catálogo cerrado de estados.
func EsEstado(s string) bool {
	switch Estado(s) {
	case EstadoPendiente, EstadoEnProgreso, EstadoCompletada, EstadoBloqueada:
		return true
	}
	return false
}

// Capa es la capa del proyecto a la que pertenece el elemento. Los archivos
// del TODO viven bajo ai/tasks/<capa>/ (BACKEND.md [49]); las capas
// documentadas son frontend, backend y database.
type Capa string

const (
	CapaFrontend Capa = "frontend"
	CapaBackend  Capa = "backend"
	CapaDatabase Capa = "database"
)

// EsCapa valida que el texto pertenezca al catálogo de capas del proyecto.
func EsCapa(s string) bool {
	switch Capa(s) {
	case CapaFrontend, CapaBackend, CapaDatabase:
		return true
	}
	return false
}

// Elemento es un elemento del TODO: una fila de MAIN-TASKS.md (tarea grande)
// o de un NNN-task-*.md (tarea pequeña), con los 7 campos del frontmatter
// aprobado en [[backend/DECISIONS]] [28].
type Elemento struct {
	ID           string   `json:"id"`                      // p. ej. "T-B004" o "T-B004-01"
	Capa         Capa     `json:"capa"`                    // frontend | backend | database
	Accion       Accion   `json:"accion"`                  // crear | actualizar | eliminar
	Estado       Estado   `json:"estado"`                  // pendiente | en_progreso | completada | bloqueada
	DependeDe    []string `json:"depende_de"`              // IDs que deben completarse antes; nil ⇒ ninguna
	BloqueadaPor []string `json:"bloqueada_por,omitempty"` // solo válido con Estado == EstadoBloqueada
	Documentos   []string `json:"documentos"`              // rutas del contexto, referenciado no incrustado
}

// NuevoElemento construye un Elemento normalizando las listas: cadenas vacías
// se descartan y una lista sin elementos queda nil, para que el round-trip
// JSON y el parseo de tablas ("—" / "ninguna") no introduzcan diferencias.
func NuevoElemento(id string, capa Capa, accion Accion, estado Estado, dependeDe, bloqueadaPor, documentos []string) Elemento {
	return Elemento{
		ID:           strings.TrimSpace(id),
		Capa:         capa,
		Accion:       accion,
		Estado:       estado,
		DependeDe:    normalizarLista(dependeDe),
		BloqueadaPor: normalizarLista(bloqueadaPor),
		Documentos:   normalizarLista(documentos),
	}
}

// normalizarLista recorta, descarta vacíos y devuelve nil para lista vacía.
func normalizarLista(xs []string) []string {
	if len(xs) == 0 {
		return nil
	}
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		if x = strings.TrimSpace(x); x != "" {
			out = append(out, x)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// CamposDevueltos verifica que los 7 nombres del frontmatter aprobado estén
// presentes en la codificación JSON del elemento (prueba de contrato con
// DECISIONS.md [28]).
var CamposDevueltos = []string{"id", "capa", "accion", "estado", "depende_de", "bloqueada_por", "documentos"}

// AJSON serializa el elemento a JSON (round-trip con FromJSON).
func (e Elemento) AJSON() ([]byte, error) { return json.Marshal(e) }

// FromJSON reconstruye un elemento desde su JSON. Rechaza vocabularios fuera
// de los catálogos cerrados: un elemento inválido nunca entra al TODO.
func FromJSON(b []byte) (Elemento, error) {
	var crudo struct {
		ID           string   `json:"id"`
		Capa         string   `json:"capa"`
		Accion       string   `json:"accion"`
		Estado       string   `json:"estado"`
		DependeDe    []string `json:"depende_de"`
		BloqueadaPor []string `json:"bloqueada_por"`
		Documentos   []string `json:"documentos"`
	}
	if err := json.Unmarshal(b, &crudo); err != nil {
		return Elemento{}, fmt.Errorf("task: decodificar elemento: %w", err)
	}
	e := NuevoElemento(crudo.ID, Capa(crudo.Capa), Accion(crudo.Accion), Estado(crudo.Estado),
		crudo.DependeDe, crudo.BloqueadaPor, crudo.Documentos)
	if err := e.validarVocabulario(); err != nil {
		return Elemento{}, err
	}
	return e, nil
}

// validarVocabulario comprueba id no vacío y los catálogos cerrados de capa,
// acción y estado. La regla de bloqueada_por (T-B004-05) vive en validate.go.
func (e Elemento) validarVocabulario() error {
	if e.ID == "" {
		return fmt.Errorf("task: elemento sin id")
	}
	if !EsCapa(string(e.Capa)) {
		return fmt.Errorf("task: %s: capa inválida %q", e.ID, e.Capa)
	}
	if !EsAccion(string(e.Accion)) {
		return fmt.Errorf("task: %s: acción inválida %q", e.ID, e.Accion)
	}
	if !EsEstado(string(e.Estado)) {
		return fmt.Errorf("task: %s: estado inválido %q", e.ID, e.Estado)
	}
	return nil
}
