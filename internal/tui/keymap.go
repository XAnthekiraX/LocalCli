// keymap.go — T-B014-07 (atajos) actualizado por T-F012-01/-05: el mapa central.
//
// Fuente de verdad: specs/SPEC-KEYBINDS.md (§Acción: IDs estables con sus atajos
// por defecto; §Binding: cero, uno o varios literales por acción, prefijo
// `<leader>`, duplicado en el mismo contexto rechazado; §Tecla líder: ctrl+x por
// defecto, timeout 2000 ms configurable entre 500 y 10000 ms) y
// SPEC-INTERFAZ-ATAJOS ("Un atajo no puede quedar asignado a dos acciones").
//
// El mapa es un dato, no un switch enterrado: se valida (sin duplicados) y se
// puede sustituir por otro sin tocar la vista. Los componentes consumen
// acciones ya resueltas; quién traduce teclas a acciones es el KeyResolver
// (keyresolver.go).
package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Accion es lo que hace un atajo, ya resuelto. El valor cero significa "ninguna
// acción"; los identificadores estables del JSON viven en nombreDeAccion.
type Accion int

const (
	AccionNinguna Accion = iota
	// AccionSalir es app_exit: global, siempre activa (también con modales).
	AccionSalir
	// AccionPanel abre o cierra el panel de datos (panel_toggle).
	AccionPanel
	// AccionSelector abre el modal de sesiones (session_picker, <leader>l).
	AccionSelector
	// AccionSesionNueva crea una sesión y la deja activa (session_new,
	// <leader>n): solo desde la vista principal (SPEC-KEYBINDS).
	AccionSesionNueva
	// AccionEliminarSesion borra la sesión resaltada en el modal de sesiones
	// (session_delete, ctrl+d). Si está trabajando, pide confirmación. El
	// literal es el mismo que panel_toggle, pero en ámbitos distintos: el
	// modal captura la tecla y la vista de abajo no la ve.
	AccionEliminarSesion
	// AccionRazonamiento muestra u oculta el razonamiento (reasoning_toggle).
	AccionRazonamiento
	// AccionAprobaciones abre o cierra el panel de aprobaciones (approvals_toggle).
	AccionAprobaciones
	// AccionAprobar / AccionDeclinar resuelven la línea seleccionada (approve/decline).
	AccionAprobar
	AccionDeclinar
	// AccionPausar detiene la cola tras el elemento actual (pausar).
	AccionPausar
	// AccionCancelar cancela el trabajo en curso con confirmación (cancel).
	AccionCancelar
	// AccionCerrarSelector cierra lo abierto sin cambiar nada (dismiss).
	AccionCerrarSelector
	// AccionAyuda abre el modal de atajos (command_palette, ctrl+p).
	AccionAyuda
	// AccionEnviar envía la petición (send); solo aplica con el input enfocado.
	AccionEnviar
	// AccionModalModelos abre el modal de modelos (model_picker, <leader>m).
	AccionModalModelos
	// AccionCiclarAgente alterna plan ↔ build (agent_cycle, tab): nunca dentro
	// de un modal ni escribiendo.
	AccionCiclarAgente
	// AccionSubir / AccionBajar navegan la lista del componente con el foco.
	AccionSubir
	AccionBajar
	// AccionChatSubir / AccionChatBajar / AccionChatPaginaArriba /
	// AccionChatPaginaAbajo recorren el historial del chat (chat_scroll_up,
	// chat_scroll_down, chat_page_up, chat_page_down). Son de la vista: en el
	// modal, up/down siguen siendo AccionSubir/AccionBajar.
	AccionChatSubir
	AccionChatBajar
	AccionChatPaginaArriba
	AccionChatPaginaAbajo
)

// LíderPorDefecto y TimeoutPorDefecto son los valores de fábrica de la
// configuración (SPEC-KEYBINDS §Tecla líder, §Configuración).
const (
	LíderPorDefecto     = "ctrl+x"
	TimeoutPorDefectoMs = 2000
	TimeoutMínimoMs     = 500
	TimeoutMáximoMs     = 10000
)

// DescripcionDeAccion da la ayuda legible de cada acción. Es la fuente de la
// lista de atajos (la ayuda clásica `?` queda sustituida por el modal de
// command_palette, SPEC-KEYBINDS §Acción), incluida «pausar», que existe como
// acción pero no trae tecla de fábrica.
var DescripcionDeAccion = map[Accion]string{
	AccionEnviar:           "enviar la petición",
	AccionSalir:            "salir",
	AccionPanel:            "abrir o cerrar el panel de datos",
	AccionSelector:         "modal de sesiones",
	AccionSesionNueva:      "crear una sesión nueva",
	AccionEliminarSesion:   "eliminar la sesión resaltada",
	AccionRazonamiento:     "mostrar u ocultar el razonamiento",
	AccionAprobaciones:     "abrir o cerrar el panel de aprobaciones",
	AccionAprobar:          "aprobar la propuesta seleccionada",
	AccionDeclinar:         "declinar la propuesta seleccionada",
	AccionPausar:           "pausar la cola en curso",
	AccionCancelar:         "cancelar el trabajo en curso (pide confirmación)",
	AccionCerrarSelector:   "cerrar lo abierto",
	AccionAyuda:            "modal de atajos de teclado",
	AccionModalModelos:     "modal de modelos",
	AccionCiclarAgente:     "cambiar de agente",
	AccionSubir:            "subir en la lista",
	AccionBajar:            "bajar en la lista",
	AccionChatSubir:        "subir por el historial del chat",
	AccionChatBajar:        "bajar por el historial del chat",
	AccionChatPaginaArriba: "página arriba en el historial",
	AccionChatPaginaAbajo:  "página abajo en el historial",
}

// Atajo une una acción con sus literales: cero (deshabilitada), uno o varios
// (SPEC-KEYBINDS §Binding). Tecla conserva el primer literal por compatibilidad
// con quien solo consume un literal suelto.
type Atajo struct {
	Accion      Accion
	Secuencias  []Secuencia
	Descripcion string
}

// Tecla devuelve el literal del primer binding, tal como lo veía el mapa
// anterior de fábrica ("ctrl+p"); vacío si la acción está deshabilitada.
func (a Atajo) Tecla() string {
	if len(a.Secuencias) == 0 {
		return ""
	}
	return a.Secuencias[0].Describir()
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
// SPEC-KEYBINDS §Acción). Las secuencias con `<leader>` se expanden con la
// líder configurada al construir el mapa. `pausar` aparece con lista vacía: la
// acción existe y el usuario le puede asignar un literal.
func MapasPorDefecto() map[Accion][]string {
	return map[Accion][]string{
		AccionEnviar:         {"enter"},
		AccionSalir:          {"ctrl+c"},
		AccionPanel:          {"ctrl+d"},
		AccionSelector:       {"<leader>l"},
		AccionSesionNueva:    {"<leader>n"},
		AccionEliminarSesion: {"ctrl+d"},
		AccionRazonamiento:   {"ctrl+r"},
		AccionAprobaciones:   {"ctrl+a"},
		AccionCancelar:       {"ctrl+f"},
		AccionAyuda:          {"ctrl+p"},
		// dismiss pertenece al modal: con una lista abierta, esc cierra lo
		// abierto y no llega a la vista de abajo.
		AccionCerrarSelector: {"esc"},
		AccionModalModelos:   {"<leader>m"},
		AccionCiclarAgente:   {"tab"},
		AccionSubir:          {"up"},
		AccionBajar:          {"down"},
		// El historial del chat se recorre con las flechas y el paginado. `up`
		// y `down` no chocan con AccionSubir/AccionBajar (ámbito del modal):
		// sin modal, la vista no las ve y el chat las usa.
		AccionChatSubir:        {"up"},
		AccionChatBajar:        {"down"},
		AccionChatPaginaArriba: {"pgup"},
		AccionChatPaginaAbajo:  {"pgdown"},
		AccionAprobar:          {"a"},
		AccionDeclinar:         {"d"},
		AccionPausar:           {},
	}
}

// EsEntradaDeTexto dice si una pulsación es texto puro que debe llegar a la
// línea de entrada antes de cualquier resolución de atajos: letras sueltas,
// dígitos, símbolos, espacios, retroceso y esc. Los modificadores (ctrl+/alt+)
// nunca son texto para bubbles, así que un literal reasignado como «ctrl+a»
// sigue disparando su acción aunque coincida con una letra. Enter, tab y las
// flechas se resuelven por el mapa; esc se comporta como texto para que el
// modal de sesiones lo use cuando está abierto (la máquina de líder lo cancela
// igual durante una espera).
func EsEntradaDeTexto(m tea.KeyMsg) bool {
	switch m.Type {
	case tea.KeyRunes, tea.KeySpace, tea.KeyBackspace, tea.KeyEsc:
		return true
	}
	return false
}

// OrdenDeAcciones fija el orden estable del mapa para listar y validar: sin él,
// recorrer el map daría errores y ayuda irreproducibles.
func OrdenDeAcciones() []Accion {
	return []Accion{
		AccionEnviar, AccionSalir, AccionPanel, AccionSelector, AccionSesionNueva,
		AccionEliminarSesion, AccionRazonamiento,
		AccionAprobaciones, AccionAprobar, AccionDeclinar, AccionPausar,
		AccionCancelar, AccionCerrarSelector, AccionAyuda,
		AccionModalModelos, AccionCiclarAgente, AccionSubir, AccionBajar,
		AccionChatSubir, AccionChatBajar, AccionChatPaginaArriba, AccionChatPaginaAbajo,
	}
}

// Keymap es el mapa completo: líder, timeout de líder y atajos por acción. Es
// el dato que consume el KeyResolver; nacer roto no es opción, así que todo
// constructor valida antes de devolver.
type Keymap struct {
	lider     string
	timeoutMs int
	entradas  []Atajo // en el orden de OrdenDeAcciones
}

// NuevoKeymap construye el mapa desde los literales de cada acción, expande la
// líder y valida: duplicados, literales inválidos y timeout fuera de rango se
// rechazan con mensaje claro (T-F012-05).
func NuevoKeymap(lider string, timeoutMs int, porAccion map[Accion][]string) (*Keymap, error) {
	if strings.TrimSpace(lider) == "" {
		lider = LíderPorDefecto
	}
	if timeoutMs == 0 {
		timeoutMs = TimeoutPorDefectoMs
	}
	if timeoutMs < TimeoutMínimoMs || timeoutMs > TimeoutMáximoMs {
		return nil, fmt.Errorf("tui: el timeout de la líder debe estar entre %d y %d ms (recibido %d)",
			TimeoutMínimoMs, TimeoutMáximoMs, timeoutMs)
	}
	km := &Keymap{lider: strings.ToLower(strings.TrimSpace(lider)), timeoutMs: timeoutMs}
	for _, accion := range OrdenDeAcciones() {
		desc := DescripcionDeAccion[accion]
		at, err := NuevoAtajo(km.lider, accion, desc, porAccion[accion]...)
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
	km, err := NuevoKeymap(LíderPorDefecto, TimeoutPorDefectoMs, MapasPorDefecto())
	if err != nil {
		// El mapa de fábrica es válido por construcción; si deja de serlo,
		// es un error de programación y conviene verlo pronto.
		panic(err)
	}
	return km
}

// Lider devuelve la tecla líder configurada.
func (k *Keymap) Lider() string { return k.lider }

// TimeoutMs devuelve la ventana de espera de la segunda tecla.
func (k *Keymap) TimeoutMs() time.Duration { return time.Duration(k.timeoutMs) * time.Millisecond }

// Entradas devuelve los atajos del mapa, en orden estable.
func (k *Keymap) Entradas() []Atajo { return k.entradas }

// SetEntradas sustituye los atajos conservando líder y timeout, y valida el
// resultado: un mapa inválido se rechaza entero.
func (k *Keymap) SetEntradas(entradas []Atajo) error {
	nuevo := &Keymap{lider: k.lider, timeoutMs: k.timeoutMs, entradas: entradas}
	if err := nuevo.validar(); err != nil {
		return err
	}
	k.entradas = entradas
	return nil
}

// ConLíderYTimeout devuelve una copia del mapa con otra líder y otro timeout,
// reexpandiendo los literales guardados (que pueden traer `<leader>`) y
// revalidando el resultado.
func (k *Keymap) ConLíderYTimeout(lider string, timeoutMs int) (*Keymap, error) {
	porAccion := map[Accion][]string{}
	for _, e := range k.entradas {
		var literales []string
		for _, sec := range e.Secuencias {
			literales = append(literales, sec.Describir())
		}
		porAccion[e.Accion] = literales
	}
	return NuevoKeymap(lider, timeoutMs, porAccion)
}

// validar comprueba el mapa completo: timeout en rango, acciones conocidas y
// únicas, y ningún literal repetido EN EL MISMO ÁMBITO (SPEC-KEYBINDS §Reglas
// de negocio: "Un literal no puede pertenecer a dos acciones en un mismo
// contexto"). Los ámbitos viven en keyresolver.go: `ctrl+d` es panel en la
// vista y eliminar en el modal de sesiones sin pisarse, porque con un modal
// abierto la vista de abajo no recibe teclas. El ámbito global está presente
// en todos los contextos, así que su literal no puede compartirse con nadie.
func (k *Keymap) validar() error {
	if k.timeoutMs < TimeoutMínimoMs || k.timeoutMs > TimeoutMáximoMs {
		return fmt.Errorf("tui: el timeout de la líder debe estar entre %d y %d ms (recibido %d)",
			TimeoutMínimoMs, TimeoutMáximoMs, k.timeoutMs)
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

// ValidarAtajos rechaza literales duplicados, acciones sin identidad y repeticiones
// de acción en el mapa (SPEC-INTERFAZ-ATAJOS: "Un atajo no puede quedar asignado
// a dos acciones"). Una lista vacía de literales no es un error: deshabilita.
func ValidarAtajos(entradas []Atajo) error {
	return (&Keymap{lider: LíderPorDefecto, timeoutMs: TimeoutPorDefectoMs, entradas: entradas}).validar()
}

// AcciónDe devuelve la acción dueña de un literal exacto ("ctrl+p", "<leader>m").
// La segunda devolución distingue "no hay binding" de "hay y no hace nada".
func AcciónDe(entradas []Atajo, literal string) (Accion, bool) {
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

// AyudaAtajos lista los atajos con lo que hace cada uno, en el orden del mapa.
// Las acciones deshabilitadas (lista vacía) también se listan, marcadas.
func AyudaAtajos(entradas []Atajo) string {
	var b strings.Builder
	for _, a := range entradas {
		b.WriteString(lineaDeAtajo(a) + "\n")
	}
	return b.String()
}

// lineaDeAtajo compone la fila «tecla(s) + descripción» de un atajo para el
// listado de ayuda (AyudaAtajos). El modal de atajos compone sus filas con el
// ancho calculado por grupo (keysmodal.go, filaDeAtajo).
func lineaDeAtajo(a Atajo) string {
	return fmt.Sprintf("%-16s %s", teclasDeAtajo(a), a.Descripcion)
}

// --- puentes de compatibilidad -------------------------------------------------
//
// El paquete nació (T-B014/T-F001/T-F010) con un mapa plano de Atajo{Tecla,...}.
// T-F012-01 lo migró a bindings múltiples; estos helpers mantienen viva la API
// antigua mientras las vistas se enrutan por acciones.

// AtajosPorDefecto devuelve el mapa de fábrica en forma de lista de atajos,
// alineado con la tabla de SPEC-KEYBINDS §Acción. Es una copia: cambiar el
// resultado no cambia los atajos del programa.
func AtajosPorDefecto() []Atajo {
	return KeymapPorDefecto().Entradas()
}

// AccionDe es el alias ASCII de AcciónDe para llamadas antiguas.
func AccionDe(entradas []Atajo, tecla string) (Accion, bool) {
	return AcciónDe(entradas, tecla)
}

// buscarAtajo localiza la entrada de una acción (creándola si falta) para las
// mutaciones del arnés de pruebas.
func buscarAtajo(entradas []Atajo, accion Accion) ([]Atajo, int) {
	for i, e := range entradas {
		if e.Accion == accion {
			return entradas, i
		}
	}
	return append(entradas, Atajo{Accion: accion, Descripcion: DescripcionDeAccion[accion]}), len(entradas)
}

// fijarLiteral deja la acción con exactamente este literal (reemplazando los
// que tuviera), preservando el resto del mapa.
func fijarLiteral(entradas []Atajo, accion Accion, literal string) []Atajo {
	entradas, i := buscarAtajo(entradas, accion)
	sec, err := AnalizarSecuencia(literal, LíderPorDefecto)
	if err != nil {
		return entradas
	}
	entradas[i].Secuencias = []Secuencia{sec}
	return entradas
}

// deshabilitar quita todos los literales de una acción (lista vacía = acción
// apagada, T-F012-05).
func deshabilitar(entradas []Atajo, accion Accion) []Atajo {
	entradas, i := buscarAtajo(entradas, accion)
	entradas[i].Secuencias = nil
	return entradas
}

// ordenarPorNombre estabiliza el recorrido de un map de Go: sin esto, el orden
// de las claves aleatorias de Go haría irreproducibles errores y salidas.
func ordenarPorNombre[N ~int | string](ks []N) []N {
	sort.Slice(ks, func(i, j int) bool { return ks[i] < ks[j] })
	return ks
}
