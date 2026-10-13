package panel

import "localcli/internal/tui/render"

// Ancho es el ancho fijo del sidebar cuando está abierto. Si la terminal no da
// para este ancho más la columna principal mínima, el sidebar cede: nunca el
// chat.
const Ancho = 42

// AnchoMinimoChat es el ancho que la columna principal conserva siempre.
const AnchoMinimoChat = 20

// UmbralContexto es el porcentaje a partir del cual se avisa de que el contexto
// se acerca a su límite (SPEC-PANEL-CONTEXTO §Flujos alternativos).
const UmbralContexto = 80

// Estados de un paso de la lista del agente ([[database/01-schema/ENUMS]]
// §Estado de paso). La vista no inventa estados: pinta los que le llegan.
const (
	Pendiente  = "pendiente"
	EnProgreso = "en_progreso"
	Completada = "completada"
	Cancelada  = "cancelada"
)

// Tarea es un paso de la lista del agente, tal como llega en `todo_actualizada`:
// su contenido y su estado. No lleva prioridad porque el panel no la muestra.
type Tarea struct {
	Contenido string
	Estado    string
}

// Datos son los nueve datos del panel, ya resueltos por quien los conoce: la
// sesión activa, su contexto, el TODO y la cola, las aprobaciones pendientes, el
// agente, y las tres filas del pie (git, ruta y firma del harness).
type Datos struct {
	// Sesión activa: su nombre y su estado.
	Sesion string
	Estado string

	// CONTEXTO: los tokens del chat que forman el contexto de la sesión —no el
	// consumo del turno—, el límite sobre el que se mide y si ese límite es el
	// tope del harness porque el motor no declaró su ventana. El total es una
	// estimación —no hay tokenizador exacto por mensaje— y se marca como tal.
	ContextoTokens   int
	TokensEstimados  bool
	LimiteTokens     int
	LimiteDelHarness bool

	// TODO: el elemento en curso y cuántos quedan.
	ElementoActual     string
	ElementosRestantes int

	// LISTA DE TAREAS: los pasos de la sesión activa.
	Tareas []Tarea

	// ESTADO: la capa y la cola y el agente activo.
	Capa          string
	TareasGrandes int
	Aprobaciones  int
	Agente        string

	// Pie: se lee al arrancar y no se vuelve a consultar. Sin rama, el proyecto
	// no tiene git inicializado —o no hay git— y la fila lo dice.
	Ruta       string
	GitRama    string
	GitCambios int

	// Firma del harness.
	Proyecto string
	Version  string
}

// Model es el sidebar: sus datos y si está plegado. Arranca desplegado, que es
// como se pinta por defecto.
type Model struct {
	Datos
	plegado bool
}

// Nuevo deja el sidebar visible por defecto, con la firma del harness: es el
// único dato del panel que no depende de la sesión ni del proyecto.
func Nuevo() Model {
	return Model{Datos: Datos{Proyecto: render.Nombre, Version: render.Version}}
}

// Abierto dice si el sidebar se pinta.
func (m Model) Abierto() bool { return !m.plegado }

// Alternar pliega o despliega el sidebar (`Ctrl+D`). Ni plegarlo ni desplegarlo
// interrumpe nada: es solo vista.
func (m *Model) Alternar() { m.plegado = !m.plegado }

// PorcentajeContexto devuelve qué parte del límite ocupa el contexto del chat (0
// si no hay límite conocido). Es una función pura.
func (m Model) PorcentajeContexto() int {
	if m.LimiteTokens <= 0 {
		return 0
	}
	pc := m.ContextoTokens * 100 / m.LimiteTokens
	if pc > 100 {
		return 100
	}
	return pc
}

// ContextoApretado dice si toca avisar de que el contexto se está llenando: se
// avisa antes de que la petición falle (SPEC-PANEL-CONTEXTO §Flujos
// alternativos).
func (m Model) ContextoApretado() bool { return m.PorcentajeContexto() >= UmbralContexto }
