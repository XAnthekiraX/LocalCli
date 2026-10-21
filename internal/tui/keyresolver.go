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
	// ContextoModalSesiones: el modal de sesiones añade `session_delete` (ctrl+d).
	ContextoModalSesiones
	// ContextoModalModelos: el modal de modelos añade `model_delete` (ctrl+d).
	ContextoModalModelos
	// ContextoAprobaciones: el panel de aprobaciones con el foco (a/d resuelven).
	ContextoAprobaciones
)

// EstadoResolver es la máquina explícita del resolver: NORMAL esperando tecla,
// LIDER esperando la segunda tecla de la secuencia.
type EstadoResolver int

const (
	EstadoNormal EstadoResolver = iota
	EstadoLider
)

// Ambito es la zona donde una acción resuelve: la columna «Contexto» de la tabla
// de SPEC-KEYBINDS §Acción. El resolver la usa para filtrar qué bindings aplican
// según qué componente tiene el foco.
type Ambito int

const (
	// AmbitoVista: la vista principal y la bienvenida (toggles, cancelar,
	// historial, cambiar de agente).
	AmbitoVista Ambito = iota
	// AmbitoGlobal: siempre, también con un modal abierto.
	AmbitoGlobal
	// AmbitoModal: solo con una lista abierta (navegar, confirmar, cerrar).
	AmbitoModal
	// AmbitoModalSesiones: solo dentro del modal de sesiones (`session_delete`).
	AmbitoModalSesiones
	// AmbitoModalModelos: solo dentro del modal de modelos (`model_delete`).
	AmbitoModalModelos
	// AmbitoAprobaciones: solo con el panel de aprobaciones con el foco.
	AmbitoAprobaciones
	// AmbitoEntrada: `send`, con el input enfocado (no con un modal abierto).
	AmbitoEntrada
)

// ambitoDeAccion dice dónde resuelve cada acción. La lista es abierta: una
// acción nueva nace en AmbitoVista, que es el contexto menos sorprendente.
func ambitoDeAccion(a Accion) Ambito {
	switch a {
	case AccionSalir, AccionModalModelos, AccionDescargarModal, AccionSelector, AccionAyuda:
		return AmbitoGlobal
	case AccionCerrarModal, AccionSubir, AccionBajar, AccionConfirmar:
		return AmbitoModal
	case AccionEliminarSesion:
		return AmbitoModalSesiones
	case AccionEliminarModelo:
		return AmbitoModalModelos
	case AccionAprobar, AccionDeclinar:
		return AmbitoAprobaciones
	case AccionEnviar:
		return AmbitoEntrada
	}
	return AmbitoVista
}

// esModalContexto dice si el contexto es el de un modal abierto, sea cual sea.
func esModalContexto(ctx Contexto) bool {
	return ctx == ContextoModal || ctx == ContextoModalSesiones || ctx == ContextoModalModelos
}

// resuelveConFoco dice si una acción resuelve con ese foco. Con un modal abierto
// solo responden sus propias teclas y las globales; ninguna de la vista de abajo
// llega. El ámbito de entrada solo resuelve sin modal (el contexto da el
// significado de `enter`: enviar en la entrada, confirmar en el modal).
func resuelveConFoco(amb Ambito, ctx Contexto) bool {
	switch amb {
	case AmbitoGlobal:
		return true
	case AmbitoEntrada:
		return !esModalContexto(ctx)
	case AmbitoModal:
		return esModalContexto(ctx)
	case AmbitoModalSesiones:
		return ctx == ContextoModalSesiones
	case AmbitoModalModelos:
		return ctx == ContextoModalModelos
	case AmbitoAprobaciones:
		return ctx == ContextoAprobaciones
	}
	// AmbitoVista: nunca con un modal abierto.
	return !esModalContexto(ctx)
}

// LiderExpiradoMsg llega cuando pasó leader_timeout_ms sin segunda tecla. La app
// lo entrega al resolver para volver a NORMAL y quitar el indicador «lider ».
type LiderExpiradoMsg struct{}

// CmdLiderEsperando devuelve el comando que, tras `duran`, produce el mensaje de
// expiración. Se arma al entrar en LIDER; si la secuencia se resuelve antes, el
// mensaje sobrante se ignora porque el estado ya no es LIDER.
func CmdLiderEsperando(duran time.Duration) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(duran)
		return LiderExpiradoMsg{}
	}
}

// Secuencia separa un literal en sus dos partes: la tecla que inicia y la que
// termina. Un literal simple («ctrl+p») tiene Paso2 vacío; uno con prefijo de
// líder («<leader>m») tiene Paso1 = ctrl+x (la líder expandida) y Paso2 = «m».
type Secuencia struct {
	Paso1 string
	Paso2 string
}

// Describir reproduce el literal tal como lo ve el usuario: «<leader>m» o «ctrl+p».
func (s Secuencia) Describir() string {
	if s.Paso2 == "" {
		return s.Paso1
	}
	return "<leader>" + s.Paso2
}

// AnalizarSecuencia convierte un literal del mapa en una secuencia, expandiendo
// el prefijo `<leader>` con la tecla líder configurada. Los errores son claros
// porque se reportan al cargar/guardar la configuración.
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

// KeyResolver traduce pulsaciones de Bubble Tea a acciones resueltas. Vive en la
// app raíz: los componentes reciben acciones, nunca teclas.
type KeyResolver struct {
	Kmap      *Keymap
	Estado    EstadoResolver
	Pendiente []Secuencia // candidatas que empiezan por la líder pulsada

	// ctx guarda el contexto en el que se abrió la espera de líder: las
	// secuencias de un ámbito concreto solo resuelven ahí.
	ctx Contexto

	// esperar es el temporizador de la espera de líder. Es una función para que
	// las pruebas inyecten un reloj propio y no dependan del tiempo real.
	esperar func(time.Duration) tea.Cmd
}

// NuevoKeyResolver deja el resolver en NORMAL con el mapa dado.
func NuevoKeyResolver(km *Keymap) *KeyResolver {
	return &KeyResolver{Kmap: km, Estado: EstadoNormal, esperar: CmdLiderEsperando}
}

// EsperandoLider dice si hay una secuencia a medias (para pintar el indicador).
func (r *KeyResolver) EsperandoLider() bool { return r.Estado == EstadoLider }

// Cancelar vuelve a NORMAL sin emitir nada: lo usa `esc` durante la espera y el
// mensaje de timeout.
func (r *KeyResolver) Cancelar() {
	r.Estado = EstadoNormal
	r.Pendiente = nil
}

// Resolver procesa una pulsación y devuelve la acción resultante (posible cero)
// y el comando del temporizador de líder, que la app debe encadenar si entra en
// LIDER. Sigue el orden de SPEC-KEYBINDS §Resolución por contexto.
func (r *KeyResolver) Resolver(m tea.KeyMsg, ctx Contexto) (Accion, tea.Cmd) {
	nombre := strings.ToLower(strings.TrimSpace(m.String()))

	if r.Estado == EstadoLider {
		return r.combinar(nombre)
	}
	if nombre == r.Kmap.Lider() {
		return r.entrarEnLider(ctx)
	}
	if ctx == ContextoInput && len([]rune(nombre)) == 1 {
		// Con el input enfocado, una letra suelta es texto, nunca atajo.
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

// entrarEnLider guarda las secuencias candidatas y arranca el temporizador. Solo
// recoge las secuencias que resuelven en el contexto actual.
func (r *KeyResolver) entrarEnLider(ctx Contexto) (Accion, tea.Cmd) {
	var candidatas []Secuencia
	for _, e := range r.Kmap.Entradas() {
		if !resuelveConFoco(ambitoDeAccion(e.Accion), ctx) {
			continue
		}
		for _, sec := range e.Secuencias {
			if sec.Paso2 != "" && sec.Paso1 == r.Kmap.Lider() {
				candidatas = append(candidatas, sec)
			}
		}
	}
	r.Estado = EstadoLider
	r.ctx = ctx
	r.Pendiente = candidatas
	esperar := r.esperar
	if esperar == nil {
		esperar = CmdLiderEsperando
	}
	return AccionNinguna, esperar(r.Kmap.Timeout())
}

// combinar resuelve la segunda tecla de la secuencia.
func (r *KeyResolver) combinar(nombre string) (Accion, tea.Cmd) {
	// Esc cancela la espera y vuelve a NORMAL sin emitir.
	if nombre == "esc" {
		r.Cancelar()
		return AccionNinguna, nil
	}
	// La líder otra vez reinicia la espera desde cero, en el mismo contexto.
	if nombre == r.Kmap.Lider() {
		return r.entrarEnLider(r.ctx)
	}
	for _, sec := range r.Pendiente {
		if sec.Paso2 == nombre {
			accion := r.accionDe(sec)
			r.Cancelar()
			return accion, nil
		}
	}
	// Ninguna secuencia válida: se descarta todo y se vuelve a NORMAL. Ninguna
	// tecla se pierde hacia el editor.
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
