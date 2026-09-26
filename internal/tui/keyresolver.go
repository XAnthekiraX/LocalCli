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

// Describir reproduce el literal tal como lo ve el usuario en la ayuda de
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
	Kmap    *Keymap
	Estado  EstadoResolver
	Pendiente []Secuencia // candidatas que empiezan por la líder pulsada
}

// NuevoKeyResolver deja el resolver en NORMAL con el mapa dado.
func NuevoKeyResolver(km *Keymap) *KeyResolver {
	return &KeyResolver{Kmap: km, Estado: EstadoNormal}
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
// LEADER. Reglas de SPEC-KEYBINDS:
//
//   - Con un modal o el panel de aprobaciones abierto, las teclas simples van al
//     componente (Contexto != Vista): el resolver no interrumpe su navegación.
//   - La líder funciona desde cualquier contexto: es el escape hatch.
//   - En LEADER, la segunda tecla resuelve la secuencia; si no forma ningún
//     binding, se descarta todo y se vuelve a NORMAL sin emitir (la líder y su
//     intento no se escriben).
//   - Pulsar la líder otra vez reinicia la espera.
//   - `esc` cancela la espera.
//   - Con el input enfocado, las letras sueltas no activan acciones: el texto
//     manda (por eso los atajos de fábrica usan modificadores o la líder).
func (r *KeyResolver) Resolver(m tea.KeyMsg, ctx Contexto) (Accion, tea.Cmd) {
	nombre := m.String()

	// 1. Dentro de un modal o del panel de aprobaciones: manda el componente.
	//    Solo la líder sigue funcionando desde ahí.
	if ctx == ContextoModal || ctx == ContextoAprobaciones {
		if nombre == r.Kmap.Lider {
			return r.entrarEnLider()
		}
		return AccionNinguna, nil
	}

	// 2. Si estaba esperando la segunda tecla, se combina.
	if r.Estado == EstadoLider {
		return r.combinar(nombre)
	}

	// 3. Estado NORMAL: la líder abre la secuencia.
	if nombre == r.Kmap.Lider {
		return r.entrarEnLider()
	}

	// 4. Binding simple del contexto. Con el input enfocado, una letra suelta
	//    es texto, no atajo.
	if ctx == ContextoInput && len([]rune(nombre)) == 1 {
		return AccionNinguna, nil
	}
	for _, e := range r.Kmap.Entradas {
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
	for _, e := range r.Kmap.Entradas {
		for _, sec := range e.Secuencias {
			if sec.Paso2 != "" && sec.Paso1 == r.Kmap.Lider {
				candidatas = append(candidatas, sec)
			}
		}
	}
	r.Estado = EstadoLider
	r.Pendiente = candidatas
	return AccionNinguna, CmdLiderEsperando(r.Kmap.TimeoutMs())
}

// combinar resuelve la segunda tecla de la secuencia.
func (r *KeyResolver) combinar(nombre string) (Accion, tea.Cmd) {
	// Esc cancela la espera y vuelve a NORMAL sin emitir.
	if nombre == "esc" {
		r.Cancelar()
		return AccionNinguna, nil
	}
	// La líder otra vez reinicia la espera desde cero.
	if nombre == r.Kmap.Lider {
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
	for _, e := range r.Kmap.Entradas {
		for _, sec := range e.Secuencias {
			if sec == querida {
				return e.Accion
			}
		}
	}
	return AccionNinguna
}
