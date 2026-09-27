// keyresolver.go — T-F012: el KeyResolver con tecla líder, timeout y contexto.
//
// Fuente de verdad: specs/SPEC-KEYBINDS.md (estados NORMAL/LEADER, formato de
// bindings con `<leader>`, timeout configurable 500–10000 ms, resolución por
// contexto modal → input → vista → global, y "los componentes no comparan
// strings de tecla: solo consumen acciones").
//
// El resolver es un dato con estado mínimo, no un bucle ni una goroutine: el
// temporizador de expiración lo pone Bubble Tea (`tea.Cmd` con `time.AfterFunc`
// que produce `LiderExpiradoMsg`), y la app descarta el mensaje si para
// entonces el estado ya cambió. Así no hay bloqueos ni esperas propias.
package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Contexto es qué zona de la pantalla tiene el foco en este instante. La
// resolución empieza por aquí (SPEC-KEYBINDS §Resolución por contexto).
type Contexto int

const (
	// ContextoVista: ninguna zona especial reclama el teclado.
	ContextoVista Contexto = iota
	// ContextoInput: se está escribiendo; las letras sueltas son texto, nunca atajo.
	ContextoInput
	// ContextoModal: un modal abierto captura su navegación (flechas, enter, esc).
	ContextoModal
	// ContextoAprobaciones: el panel de aprobaciones abierto (a/d resuelven).
	ContextoAprobaciones
)

// EstadoResolver es la máquina explícita del resolver: NORMAL esperando tecla,
// LEADER esperando la segunda tecla de la secuencia. No hay más estados
// (SPEC-KEYBINDS §Estado explícito del resolver).
type EstadoResolver int

const (
	EstadoNormal EstadoResolver = iota
	EstadoLider
)

// Ambito es la zona donde una acción resuelve: la columna «Contexto» de la tabla
// de SPEC-KEYBINDS §Acción, que el resolver usa para filtrar qué bindings
// aplican según qué componente tiene el foco.
type Ambito int

const (
	// AmbitoVista: la vista principal (toggles, cancelar, enviar).
	AmbitoVista Ambito = iota
	// AmbitoGlobal: siempre, también con un modal abierto.
	AmbitoGlobal
	// AmbitoModal: solo con una lista abierta (navegar, cerrar).
	AmbitoModal
	// AmbitoAprobaciones: solo con el panel de aprobaciones con el foco.
	AmbitoAprobaciones
	// AmbitoEntrada: `enter` resuelve en todas partes porque su significado lo
	// da el contexto: con un modal abierto aplica lo resaltado y en el resto
	// envía la petición (INTERFACES §4).
	AmbitoEntrada
)

// ambitoDeAccion dice dónde resuelve cada acción. La lista es abierta: una
// acción nueva nace en AmbitoVista, que es el contexto menos sorprendente.
func ambitoDeAccion(a Accion) Ambito {
	switch a {
	case AccionSalir, AccionModalModelos, AccionSelector, AccionAyuda:
		return AmbitoGlobal
	case AccionCerrarSelector, AccionSubir, AccionBajar, AccionEliminarSesion:
		return AmbitoModal
	case AccionAprobar, AccionDeclinar:
		return AmbitoAprobaciones
	case AccionEnviar:
		return AmbitoEntrada
	}
	return AmbitoVista
}

// resuelveConFoco dice si una acción resuelve con ese foco. Con un modal
// abierto solo responden sus propias teclas y las globales; ninguna de la vista
// de abajo llega (SPEC-KEYBINDS §Acción de app_exit y §Resolución por
// contexto). El panel de aprobaciones no es un modal: sus toggles de vista
// siguen vivos, porque el panel se abre y se cierra sin detener el trabajo
// (SPEC-INTERFAZ-ATAJOS).
func resuelveConFoco(amb Ambito, ctx Contexto) bool {
	if amb == AmbitoEntrada {
		// `enter` es la misma tecla con dos significados, y el contexto los
		// separa: con un modal abierto aplica lo resaltado, en cualquier otro
		// sitio envía la petición (INTERFACES §4, las dos filas de `Enter`).
		return true
	}
	switch ctx {
	case ContextoModal:
		return amb == AmbitoGlobal || amb == AmbitoModal
	case ContextoVista:
		return amb != AmbitoModal
	}
	return true
}

// LiderExpiradoMsg llega cuando pasó leader_timeout_ms sin segunda tecla. Es un
// mensaje interno de la vista: la app lo entrega al resolver para volver a
// NORMAL y quitar el indicador «lider ».
type LiderExpiradoMsg struct{}

// CmdLiderEsperando devuelve el comando que, tras `duran`, produce el mensaje
// de expiración. Se arma al entrar en LEADER; si la secuencia se resuelve antes,
// el mensaje sobrante se ignora porque el estado ya no es LEADER.
func CmdLiderEsperando(duran time.Duration) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(duran)
		return LiderExpiradoMsg{}
	}
}

// Secuencia separa un literal en sus dos partes: la tecla que inicia y la que
// termina. Un literal simple ("ctrl+p") tiene Paso2 vacío; uno con prefijo de
// líder ("<leader>m") tiene Paso1 = ctrl+x (la líder expandida) y Paso2 = "m".
type Secuencia struct {
	Paso1 string
	Paso2 string
}

// Describir reproduce el literal tal como lo ve el usuario en el modal de
// atajos: "<leader>m" o "ctrl+p".
func (s Secuencia) Describir() string {
	if s.Paso2 == "" {
		return s.Paso1
	}
	return "<leader>" + s.Paso2
}

// AnalizarSecuencia convierte un literal del mapa en una secuencia, expandiendo
// el prefijo `<leader>` con la tecla líder configurada. Los errores son claros
// porque se reportan al cargar/guardar la configuración: literal vacío, `<leader>`
// suelto sin segunda tecla o prefijo desconocido.
func AnalizarSecuencia(literal, lider string) (Secuencia, error) {
	l := strings.ToLower(strings.TrimSpace(literal))
	if l == "" {
		return Secuencia{}, fmt.Errorf("tui: un atajo no puede ser literal vacío")
	}
	if strings.HasPrefix(l, "<leader>") {
		resto := strings.TrimPrefix(l, "<leader>")
		if resto == "" {
			return Secuencia{}, fmt.Errorf("tui: el literal %q necesita una tecla tras <leader>", literal)
		}
		return Secuencia{Paso1: lider, Paso2: resto}, nil
	}
	if strings.HasPrefix(l, "<") {
		return Secuencia{}, fmt.Errorf("tui: prefijo desconocido en el literal %q", literal)
	}
	return Secuencia{Paso1: l}, nil
}

// KeyResolver traduce pulsaciones de Bubble Tea a acciones resueltas. Vive en
// la app raíz: los componentes reciben acciones, nunca teclas (T-F012-06).
type KeyResolver struct {
	Kmap      *Keymap
	Estado    EstadoResolver
	Pendiente []Secuencia // candidatas que empiezan por la líder pulsada

	// esperar es el temporizador de la espera de líder. Es una función para que
	// las pruebas inyecten un reloj propio y no dependan del tiempo real
	// (TESTING §3: «no hay sleep ni esperas fijas»).
	esperar func(time.Duration) tea.Cmd
}

// NuevoKeyResolver deja el resolver en NORMAL con el mapa dado.
func NuevoKeyResolver(km *Keymap) *KeyResolver {
	return &KeyResolver{Kmap: km, Estado: EstadoNormal, esperar: CmdLiderEsperando}
}

// EsperandoLeader dice si hay una secuencia a medias (para pintar el indicador
// «lider » en la barra de estado, T-F012-06).
func (r *KeyResolver) EsperandoLeader() bool { return r.Estado == EstadoLider }

// Cancelar vuelve a NORMAL sin emitir nada: lo usa `esc` durante la espera y el
// mensaje de timeout.
func (r *KeyResolver) Cancelar() {
	r.Estado = EstadoNormal
	r.Pendiente = nil
}

// Resolver procesa una pulsación y devuelve la acción resultante (posible cero)
// y el comando del temporizador de líder, que la app debe encadenar si entra en
// LEADER. Sigue el orden de SPEC-KEYBINDS §Resolución por contexto:
//
//  1. Una secuencia a medias manda: en LEADER, la tecla se combina.
//  2. La líder abre la secuencia, desde cualquier contexto: es el escape hatch.
//  3. Modal abierto: solo sus teclas (navegar, cerrar) y las globales.
//  4. Input enfocado: una letra suelta es texto, nunca un atajo.
//  5. Vista principal: cualquier binding del mapa; gana la primera coincidencia.
func (r *KeyResolver) Resolver(m tea.KeyMsg, ctx Contexto) (Accion, tea.Cmd) {
	nombre := strings.ToLower(strings.TrimSpace(m.String()))

	if r.Estado == EstadoLider {
		return r.combinar(nombre)
	}
	if nombre == r.Kmap.Lider() {
		return r.entrarEnLider()
	}
	if ctx == ContextoInput && len([]rune(nombre)) == 1 {
		return AccionNinguna, nil
	}
	for _, e := range r.Kmap.Entradas() {
		if !resuelveConFoco(ambitoDeAccion(e.Accion), ctx) {
			continue
		}
		for _, sec := range e.Secuencias {
			if sec.Paso2 != "" {
				continue // las secuencias requieren la líder: paso 2
			}
			if sec.Paso1 == nombre {
				return e.Accion, nil
			}
		}
	}
	return AccionNinguna, nil
}

// entrarEnLider guarda las secuencias candidatas y arranca el temporizador.
func (r *KeyResolver) entrarEnLider() (Accion, tea.Cmd) {
	var candidatas []Secuencia
	for _, e := range r.Kmap.Entradas() {
		for _, sec := range e.Secuencias {
			if sec.Paso2 != "" && sec.Paso1 == r.Kmap.Lider() {
				candidatas = append(candidatas, sec)
			}
		}
	}
	r.Estado = EstadoLider
	r.Pendiente = candidatas
	esperar := r.esperar
	if esperar == nil {
		esperar = CmdLiderEsperando
	}
	return AccionNinguna, esperar(r.Kmap.TimeoutMs())
}

// combinar resuelve la segunda tecla de la secuencia.
func (r *KeyResolver) combinar(nombre string) (Accion, tea.Cmd) {
	// Esc cancela la espera y vuelve a NORMAL sin emitir.
	if nombre == "esc" {
		r.Cancelar()
		return AccionNinguna, nil
	}
	// La líder otra vez reinicia la espera desde cero.
	if nombre == r.Kmap.Lider() {
		return r.entrarEnLider()
	}
	for _, sec := range r.Pendiente {
		if sec.Paso2 == nombre {
			accion := r.accionDe(sec)
			r.Cancelar()
			return accion, nil
		}
	}
	// Ninguna secuencia válida: se descarta todo y se vuelve a NORMAL. Las
	// teclas no llegan al editor (SPEC-KEYBINDS §Tecla líder).
	r.Cancelar()
	return AccionNinguna, nil
}

// accionDe busca la acción dueña de una secuencia completa.
func (r *KeyResolver) accionDe(querida Secuencia) Accion {
	for _, e := range r.Kmap.Entradas() {
		for _, sec := range e.Secuencias {
			if sec == querida {
				return e.Accion
			}
		}
	}
	return AccionNinguna
}

// ResolverSimple es la búsqueda plana del mapa —literal exacto a acción— sin
// máquina de estados de líder. La usan las pruebas de reasignación heredadas
// (T-F001/T-F010: "un mapa reasignado dispara la misma acción con otra tecla")
// y cualquier consumidor que ya tenga el nombre de tecla resuelto. Las
// secuencias con líder nunca casan aquí: requieren el paso 2 de Resolver.
func ResolverSimple(entradas []Atajo, nombre string) (Accion, bool) {
	nombre = strings.ToLower(strings.TrimSpace(nombre))
	for _, e := range entradas {
		for _, sec := range e.Secuencias {
			if sec.Paso2 == "" && sec.Paso1 == nombre {
				return e.Accion, true
			}
		}
	}
	return AccionNinguna, false
}
