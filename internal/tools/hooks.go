package tools

// hooks.go — T-B024-07: los tres puntos de enganche de `tools`.
//
// Fuente de verdad: ai/docs/backend/02-interfaces/TOOLS.md §10 (auditoría y
// métricas sin instrumentar trece handlers; adaptación del catálogo a un modelo
// concreto sin tocar el catálogo) y DECISIONS.md ("Tres hooks en `tools` para
// auditar, medir y adaptar el catálogo").
//
// Son opcionales por diseño: sin ninguno, la capa universal funciona igual. Se
// conectan en el cableado y el catálogo no sabe quién los instala.

// Hooks es el conjunto de puntos de enganche. Todos pueden ser nil.
type Hooks struct {
	// AntesDeEjecutar se llama justo antes del handler, con los argumentos ya
	// validados. Es el sitio de auditoría y métricas.
	AntesDeEjecutar func(nombre string, args any, meta map[string]any)
	// DespuesDeEjecutar se llama al terminar, con el resultado y el error (si
	// lo hubo). Un hook no puede abortar una ejecución ya hecha.
	DespuesDeEjecutar func(nombre string, r Resultado, err error, meta map[string]any)
	// DefinirHerramienta puede ajustar nombre, descripción y esquema antes de
	// que lleguen al modelo. Es el punto de extensión para adaptar el catálogo
	// a un modelo concreto. Devolver cadena vacía y nil deja todo como está.
	DefinirHerramienta func(nombre, descripcion string, e *Esquema) (string, *Esquema)
}
