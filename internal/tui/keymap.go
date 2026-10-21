package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"localcli/internal/tui/components/modals"
)

// Accion es lo que hace un atajo, ya resuelto. El valor cero significa «ninguna
// acción»; los identificadores estables del JSON viven en nombreDeAccion.
type Accion int

const (
	AccionNinguna Accion = iota
	// AccionSalir es app_exit: global, siempre activa (también con modales).
	AccionSalir
	// AccionEnviar envía la petición (send); resuelve con el input enfocado.
	AccionEnviar
	// AccionPanel abre o cierra el panel de datos (panel_toggle).
	AccionPanel
	// AccionSelector abre el modal de sesiones (session_picker, <leader>l).
	AccionSelector
	// AccionSesionNueva crea una sesión y la deja activa (session_new, <leader>n).
	AccionSesionNueva
	// AccionEliminarSesion borra la sesión resaltada en el modal de sesiones
	// (session_delete, ctrl+d). El literal es el mismo que panel_toggle, pero en
	// ámbitos distintos: el modal captura la tecla y la vista de abajo no la ve.
	AccionEliminarSesion
	// AccionEliminarModelo pide confirmación para borrar el modelo resaltado en
	// el modal de modelos (model_delete, ctrl+d). Comparte literal con
	// panel_toggle y session_delete, pero en su propio ámbito.
	AccionEliminarModelo
	// AccionRazonamiento muestra u oculta el razonamiento (reasoning_toggle).
	AccionRazonamiento
	// AccionAprobaciones abre o cierra el panel de aprobaciones (approvals_toggle).
	AccionAprobaciones
	// AccionAprobar / AccionDeclinar resuelven la línea seleccionada (approve/decline).
	AccionAprobar
	AccionDeclinar
	// AccionCancelar cancela el trabajo en curso con confirmación (cancel).
	AccionCancelar
	// AccionCerrarModal cierra lo abierto sin cambiar nada (dismiss).
	AccionCerrarModal
	// AccionAyuda abre el modal de atajos (command_palette, ctrl+p).
	AccionAyuda
	// AccionModalModelos abre el modal de modelos (model_picker, <leader>m).
	AccionModalModelos
	// AccionDescargarModal abre el modal de descarga de modelos desde una URL
	// (model_download, <leader>d).
	AccionDescargarModal
	// AccionCiclarAgente cambia de agente (agent_cycle, tab): nunca dentro de un
	// modal.
	AccionCiclarAgente
	// AccionSubir / AccionBajar navegan la lista del modal abierto.
	AccionSubir
	AccionBajar
	// AccionChatSubir / AccionChatBajar / AccionChatPaginaArriba /
	// AccionChatPaginaAbajo recorren el historial del chat. Son de la vista: en
	// el modal, up/down siguen siendo AccionSubir/AccionBajar.
	AccionChatSubir
	AccionChatBajar
	AccionChatPaginaArriba
	AccionChatPaginaAbajo
	// AccionConfirmar aplica lo resaltado en el modal y lo cierra (confirm). Es
	// la misma tecla que AccionEnviar, pero en el ámbito del modal.
	AccionConfirmar
)

// Valores de fábrica de la configuración (SPEC-KEYBINDS §Tecla líder, §Configuración).
const (
	LiderPorDefecto     = "ctrl+x"
	TimeoutPorDefectoMs = 2000
	TimeoutMinimoMs     = 500
	TimeoutMaximoMs     = 10000
)

// DescripcionDeAccion da la ayuda legible de cada acción; es la fuente del modal
// de atajos.
var DescripcionDeAccion = map[Accion]string{
	AccionSalir:            "salir",
	AccionEnviar:           "enviar la petición",
	AccionPanel:            "abrir o cerrar el panel de datos",
	AccionSelector:         "modal de sesiones",
	AccionSesionNueva:      "crear una sesión nueva",
	AccionEliminarSesion:   "eliminar la sesión resaltada",
	AccionEliminarModelo:   "eliminar el modelo resaltado (pide confirmación)",
	AccionRazonamiento:     "mostrar u ocultar el razonamiento",
	AccionAprobaciones:     "abrir o cerrar el panel de aprobaciones",
	AccionAprobar:          "aprobar la propuesta seleccionada",
	AccionDeclinar:         "declinar la propuesta seleccionada",
	AccionCancelar:         "cancelar el trabajo en curso (pide confirmación)",
	AccionCerrarModal:      "cerrar lo abierto",
	AccionAyuda:            "modal de atajos de teclado",
	AccionModalModelos:     "modal de modelos",
	AccionDescargarModal:   "descargar un modelo desde una URL",
	AccionCiclarAgente:     "cambiar de agente",
	AccionSubir:            "subir en la lista",
	AccionBajar:            "bajar en la lista",
	AccionChatSubir:        "subir por el historial del chat",
	AccionChatBajar:        "bajar por el historial del chat",
	AccionChatPaginaArriba: "página arriba en el historial",
	AccionChatPaginaAbajo:  "página abajo en el historial",
	AccionConfirmar:        "aplicar lo resaltado",
}

// ordenDeCategorias fija el orden de los grupos del modal de atajos: de lo
// general a lo particular, para que la tabla se lea de un vistazo.
func ordenDeCategorias() []string {
	return []string{"General", "Chat", "Vista", "Modales", "Aprobaciones", "Entrada"}
}

// categoriaDeAtajo reparte cada acción en su grupo. Una acción nueva cae en
// «Otras», que se pinta al final para que se note que falta clasificarla.
func categoriaDeAtajo(a Accion) string {
	switch a {
	case AccionSalir, AccionAyuda, AccionModalModelos, AccionDescargarModal, AccionSelector:
		return "General"
	case AccionChatSubir, AccionChatBajar, AccionChatPaginaArriba, AccionChatPaginaAbajo:
		return "Chat"
	case AccionPanel, AccionSesionNueva, AccionRazonamiento, AccionAprobaciones, AccionCancelar, AccionCiclarAgente:
		return "Vista"
	case AccionEliminarSesion, AccionEliminarModelo, AccionCerrarModal, AccionSubir, AccionBajar, AccionConfirmar:
		return "Modales"
	case AccionAprobar, AccionDeclinar:
		return "Aprobaciones"
	case AccionEnviar:
		return "Entrada"
	}
	return "Otras"
}

// FilasDeAtajos compone la tabla del modal de atajos desde el mapa vigente: por
// acción, sus literales con la líder ya expandida y su descripción, en el orden
// de las categorías. El modal no conoce el mapa: recibe las filas hechas.
func FilasDeAtajos(entradas []Atajo, lider string) []modals.Fila {
	var filas []modals.Fila
	for _, grupo := range ordenDeCategorias() {
		for _, e := range entradas {
			if categoriaDeAtajo(e.Accion) != grupo {
				continue
			}
			filas = append(filas, modals.Fila{
				Grupo:  grupo,
				Tecla:  teclasDeAtajo(e, lider),
				Accion: e.Descripcion,
			})
		}
	}
	return filas
}

// teclasDeAtajo junta los literales de una acción con la líder expandida —la
// combinación real (`ctrl+x m`), no el marcador—; sin ninguno, la acción está
// deshabilitada y la tabla lo dice.
func teclasDeAtajo(a Atajo, lider string) string {
	if lider == "" {
		lider = LiderPorDefecto
	}
	literales := make([]string, 0, len(a.Secuencias))
	for _, sec := range a.Secuencias {
		if sec.Paso2 != "" {
			literales = append(literales, lider+" "+sec.Paso2)
			continue
		}
		literales = append(literales, sec.Paso1)
	}
	if len(literales) == 0 {
		return "(deshabilitada)"
	}
	return strings.Join(literales, ", ")
}

// nombreDeAccion es el identificador estable de cada acción (tabla de
// SPEC-KEYBINDS §Acción); es también la clave con la que viaja en el JSON.
func nombreDeAccion(a Accion) string {
	switch a {
	case AccionSalir:
		return "app_exit"
	case AccionEnviar:
		return "send"
	case AccionPanel:
		return "panel_toggle"
	case AccionSelector:
		return "session_picker"
	case AccionSesionNueva:
		return "session_new"
	case AccionEliminarSesion:
		return "session_delete"
	case AccionEliminarModelo:
		return "model_delete"
	case AccionRazonamiento:
		return "reasoning_toggle"
	case AccionAprobaciones:
		return "approvals_toggle"
	case AccionAprobar:
		return "approve"
	case AccionDeclinar:
		return "decline"
	case AccionCancelar:
		return "cancel"
	case AccionCerrarModal:
		return "dismiss"
	case AccionAyuda:
		return "command_palette"
	case AccionModalModelos:
		return "model_picker"
	case AccionDescargarModal:
		return "model_download"
	case AccionCiclarAgente:
		return "agent_cycle"
	case AccionSubir:
		return "up"
	case AccionBajar:
		return "down"
	case AccionChatSubir:
		return "chat_scroll_up"
	case AccionChatBajar:
		return "chat_scroll_down"
	case AccionChatPaginaArriba:
		return "chat_page_up"
	case AccionChatPaginaAbajo:
		return "chat_page_down"
	case AccionConfirmar:
		return "confirm"
	}
	return ""
}

// OrdenDeAcciones fija el orden estable del mapa para listar y validar: sin él,
// recorrer el map daría errores y ayuda irreproducibles.
func OrdenDeAcciones() []Accion {
	return []Accion{
		AccionSalir, AccionEnviar, AccionPanel, AccionSelector, AccionSesionNueva,
		AccionEliminarSesion, AccionEliminarModelo, AccionRazonamiento, AccionAprobaciones,
		AccionAprobar, AccionDeclinar, AccionCancelar, AccionCerrarModal,
		AccionAyuda, AccionModalModelos, AccionDescargarModal, AccionCiclarAgente, AccionSubir,
		AccionBajar, AccionChatSubir, AccionChatBajar, AccionChatPaginaArriba,
		AccionChatPaginaAbajo, AccionConfirmar,
	}
}

// Atajo une una acción con sus literales: cero (deshabilitada), uno o varios
// (SPEC-KEYBINDS §Binding).
type Atajo struct {
	Accion      Accion
	Secuencias  []Secuencia
	Descripcion string
}

// NuevoAtajo construye un atajo a partir de sus literales, expandiendo el
// prefijo `<leader>` con la tecla líder dada. Un literal inválido (vacío,
// `<leader>` suelto, prefijo desconocido) es un error claro: se reporta al
// cargar o al guardar la configuración, no se adivina.
func NuevoAtajo(lider string, accion Accion, descripcion string, literales ...string) (Atajo, error) {
	a := Atajo{Accion: accion, Descripcion: descripcion}
	for _, lit := range literales {
		sec, err := AnalizarSecuencia(lit, lider)
		if err != nil {
			return Atajo{}, err
		}
		a.Secuencias = append(a.Secuencias, sec)
	}
	return a, nil
}

// MapasPorDefecto describe los bindings de fábrica por acción (tabla de
// SPEC-KEYBINDS §Acción). Las secuencias con `<leader>` se expanden con la líder
// configurada al construir el mapa. Las acciones del modal de motores y de la
// pausa no están en el alcance de esta reconstrucción de la TUI.
func MapasPorDefecto() map[Accion][]string {
	return map[Accion][]string{
		AccionSalir:            {"ctrl+c"},
		AccionEnviar:           {"enter"},
		AccionPanel:            {"ctrl+d"},
		AccionSelector:         {"<leader>l"},
		AccionSesionNueva:      {"<leader>n"},
		AccionEliminarSesion:   {"ctrl+d"},
		AccionEliminarModelo:   {"ctrl+d"},
		AccionRazonamiento:     {"ctrl+r"},
		AccionAprobaciones:     {"ctrl+a"},
		AccionAprobar:          {"a"},
		AccionDeclinar:         {"d"},
		AccionCancelar:         {"esc", "ctrl+f"},
		AccionCerrarModal:      {"esc"},
		AccionAyuda:            {"ctrl+p"},
		AccionModalModelos:     {"<leader>m"},
		AccionDescargarModal:   {"<leader>d"},
		AccionCiclarAgente:     {"tab"},
		AccionSubir:            {"up"},
		AccionBajar:            {"down"},
		AccionChatSubir:        {"up"},
		AccionChatBajar:        {"down"},
		AccionChatPaginaArriba: {"pgup"},
		AccionChatPaginaAbajo:  {"pgdown"},
		AccionConfirmar:        {"enter"},
	}
}

// Keymap es el mapa completo: líder, timeout y atajos por acción. Es el dato que
// consume el KeyResolver; nacer roto no es opción, así que todo constructor
// valida antes de devolver.
type Keymap struct {
	lider     string
	timeoutMs int
	entradas  []Atajo // en el orden de OrdenDeAcciones
}

// NuevoKeymap construye el mapa desde los literales de cada acción, expande la
// líder y valida: duplicados, literales inválidos y timeout fuera de rango se
// rechazan con mensaje claro.
func NuevoKeymap(lider string, timeoutMs int, porAccion map[Accion][]string) (*Keymap, error) {
	if strings.TrimSpace(lider) == "" {
		lider = LiderPorDefecto
	}
	if timeoutMs == 0 {
		timeoutMs = TimeoutPorDefectoMs
	}
	if timeoutMs < TimeoutMinimoMs || timeoutMs > TimeoutMaximoMs {
		return nil, fmt.Errorf("tui: el timeout de la líder debe estar entre %d y %d ms (recibido %d)",
			TimeoutMinimoMs, TimeoutMaximoMs, timeoutMs)
	}
	km := &Keymap{lider: strings.ToLower(strings.TrimSpace(lider)), timeoutMs: timeoutMs}
	for _, accion := range OrdenDeAcciones() {
		at, err := NuevoAtajo(km.lider, accion, DescripcionDeAccion[accion], porAccion[accion]...)
		if err != nil {
			return nil, fmt.Errorf("tui: la acción %s trae un literal inválido: %w", nombreDeAccion(accion), err)
		}
		km.entradas = append(km.entradas, at)
	}
	if err := km.validar(); err != nil {
		return nil, err
	}
	return km, nil
}

// KeymapPorDefecto devuelve el mapa de fábrica (ctrl+x, 2000 ms). Es una copia
// nueva: cambiar el resultado no cambia los mapas del programa.
func KeymapPorDefecto() *Keymap {
	km, err := NuevoKeymap(LiderPorDefecto, TimeoutPorDefectoMs, MapasPorDefecto())
	if err != nil {
		// El mapa de fábrica es válido por construcción; si deja de serlo, es un
		// error de programación y conviene verlo pronto.
		panic(err)
	}
	return km
}

// Lider devuelve la tecla líder configurada.
func (k *Keymap) Lider() string { return k.lider }

// Timeout devuelve la ventana de espera de la segunda tecla.
func (k *Keymap) Timeout() time.Duration { return time.Duration(k.timeoutMs) * time.Millisecond }

// TimeoutMs devuelve la ventana en milisegundos.
func (k *Keymap) TimeoutMs() int { return k.timeoutMs }

// Entradas devuelve los atajos del mapa, en orden estable.
func (k *Keymap) Entradas() []Atajo { return k.entradas }

// ConLiderYTimeout devuelve una copia del mapa con otra líder y otro timeout,
// reexpandiendo los literales guardados (que pueden traer `<leader>`) y
// revalidando el resultado.
func (k *Keymap) ConLiderYTimeout(lider string, timeoutMs int) (*Keymap, error) {
	porAccion := map[Accion][]string{}
	for _, e := range k.entradas {
		porAccion[e.Accion] = literalesDe(e)
	}
	return NuevoKeymap(lider, timeoutMs, porAccion)
}

// validar comprueba el mapa completo: timeout en rango, acciones conocidas y
// únicas, y ningún literal repetido EN EL MISMO ÁMBITO. El ámbito global está
// presente en todos los contextos, así que su literal no puede compartirse con
// nadie.
func (k *Keymap) validar() error {
	if k.timeoutMs < TimeoutMinimoMs || k.timeoutMs > TimeoutMaximoMs {
		return fmt.Errorf("tui: el timeout de la líder debe estar entre %d y %d ms (recibido %d)",
			TimeoutMinimoMs, TimeoutMaximoMs, k.timeoutMs)
	}
	type visto struct {
		lit    string
		ambito Ambito
		accion Accion
	}
	var vistos []visto
	puestas := map[Accion]bool{}
	for _, e := range k.entradas {
		if e.Accion == AccionNinguna {
			return fmt.Errorf("tui: hay un atajo sin acción")
		}
		if nombreDeAccion(e.Accion) == "" {
			return fmt.Errorf("tui: acción desconocida en el mapa (%d)", int(e.Accion))
		}
		if puestas[e.Accion] {
			return fmt.Errorf("tui: la acción %s aparece dos veces en el mapa", nombreDeAccion(e.Accion))
		}
		puestas[e.Accion] = true
		amb := ambitoDeAccion(e.Accion)
		for _, sec := range e.Secuencias {
			lit := sec.Describir()
			if lit == "" {
				return fmt.Errorf("tui: la acción %s tiene un literal vacío", nombreDeAccion(e.Accion))
			}
			for _, v := range vistos {
				pisa := v.ambito == amb || v.ambito == AmbitoGlobal || amb == AmbitoGlobal
				if v.lit != lit || !pisa {
					continue
				}
				return fmt.Errorf("tui: el literal %q está asignado a dos acciones en el mismo contexto (%s y %s)",
					lit, nombreDeAccion(v.accion), nombreDeAccion(e.Accion))
			}
			vistos = append(vistos, visto{lit: lit, ambito: amb, accion: e.Accion})
		}
	}
	return nil
}

// BuscarAccion devuelve la acción cuyo literal exacto coincide, si la hay.
func BuscarAccion(entradas []Atajo, literal string) (Accion, bool) {
	l := strings.ToLower(strings.TrimSpace(literal))
	for _, e := range entradas {
		for _, sec := range e.Secuencias {
			if sec.Describir() == l {
				return e.Accion, true
			}
		}
	}
	return AccionNinguna, false
}

// EsTeclaDeEdicion dice si una pulsación edita texto y debe llegar al componente
// con el foco antes de resolverse como acción: letras sueltas, espacio, borrado,
// flechas y home/end (con sus equivalentes ctrl+b/ctrl+e). SPEC-KEYBINDS
// §Resolución por contexto: «las teclas de edición que no son atajo llegan al
// input y mueven el cursor». Enter, tab y los modificadores de acción se
// resuelven por el mapa.
func EsTeclaDeEdicion(m tea.KeyMsg) bool {
	switch m.Type {
	case tea.KeyRunes, tea.KeySpace, tea.KeyBackspace, tea.KeyDelete,
		tea.KeyLeft, tea.KeyRight, tea.KeyHome, tea.KeyEnd,
		tea.KeyCtrlB, tea.KeyCtrlE:
		return true
	}
	return false
}

// literalesDe devuelve los literales de una acción tal como van en el JSON: la
// secuencia con líder se guarda con su prefijo `<leader>`, no con la líder ya
// expandida, para que cambiar de líder en el archivo no rompa nada.
func literalesDe(e Atajo) []string {
	literales := []string{}
	for _, sec := range e.Secuencias {
		literales = append(literales, sec.Describir())
	}
	return literales
}
