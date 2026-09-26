package tui

// Tests de T-F010: el modelo raíz y su integración. Una prueba, una regla
// (frontend/05-quality/TESTING.md §3):
//
//	T-F010-04 → un atajo reasignado dispara la misma acción (tui_test.go)
//	T-F010-05 → los eventos del motor llegan a la vista por el canal
//	T-F010-06 → la escucha se arma en Init, se re-arma tras cada evento y
//	            no abre suscripciones nuevas
//	T-F010-07 → ? abre la ayuda con la lista de atajos; se cierra sin efectos
//	T-F010-09 → la vista compone chat, entrada, panel y línea de aviso

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"localcli/internal/session"
)

// --- T-F010-05/06: la escucha del canal del motor ------------------------------

func TestLaEscuchaSeArmaUnaVezYSeRearmaTrasCadaEvento(t *testing.T) {
	p := &puertoStub{}
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	cmd := a.Init()
	if p.suscripciones != 1 {
		t.Fatalf("Init arma la escucha una vez: %d", p.suscripciones)
	}

	// Dos eventos publicados en el canal, cada uno procesado con el comando
	// que el anterior devolvió: exactamente lo que hace el bucle de Bubble
	// Tea. Ninguno abre una suscripción nueva.
	eventos := []Evento{
		{Nombre: EventoEtapaIniciada, Datos: map[string]string{"etapa": "plan"}},
		{Nombre: EventoEtapaTerminada, Datos: map[string]string{"etapa": "plan"}},
	}
	for _, e := range eventos {
		p.canal <- e
		msg := cmd()
		var nueva tea.Cmd
		_, nueva = a.Update(msg)
		if nueva == nil {
			t.Fatal("tras cada evento la escucha queda re-armada")
		}
		cmd = nueva
	}
	if p.suscripciones != 1 {
		t.Errorf("no se re-suscribe en cada evento: %d", p.suscripciones)
	}
	v := sinEstilo(a.View())
	if !strings.Contains(v, "etapa iniciada: plan") || !strings.Contains(v, "etapa terminada: plan") {
		t.Errorf("los dos eventos llegaron a la vista: %s", v)
	}
}

// --- T-F010-07: la ayuda ---------------------------------------------------------

func TestLaAyudaListaLosAtajosYSeCierraSinEfectos(t *testing.T) {
	p := &puertoStub{}
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})

	pulsa(t, a, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if !a.AyudaAbierta {
		t.Fatal("? abre la ayuda")
	}
	v := a.View()
	for _, atajo := range a.Atajos {
		if !strings.Contains(v, atajo.Tecla) || !strings.Contains(v, atajo.Descripcion) {
			t.Errorf("la ayuda debe listar %q con su acción: %s", atajo.Tecla, atajo.Descripcion)
		}
	}
	// Cualquier tecla la cierra y no deja rastro: no escribe en la entrada.
	pulsa(t, a, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if a.AyudaAbierta {
		t.Error("cualquier tecla cierra la ayuda")
	}
	if a.Entrada.Texto() != "" {
		t.Errorf("cerrar la ayuda no escribe: %q", a.Entrada.Texto())
	}
}

// --- T-F010-09: la composición de la vista ----------------------------------------

func TestLaVistaComponeChatEntradaPanelYAviso(t *testing.T) {
	p := &puertoStub{}
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	a.Chat.AñadirUsuario("una pregunta de prueba")
	a.Chat.Token("una respuesta de prueba")
	a.Panel.Aprobaciones = 2

	// Panel cerrado: chat a todo el ancho y el aviso como única línea fuera.
	a.Panel.Abierto = false
	v := sinEstilo(a.View())
	for _, esperado := range []string{
		"una pregunta de prueba",
		"una respuesta de prueba",
		"Escribe tu petición…",
		"2 aprobaciones esperando tu decisión",
	} {
		if !strings.Contains(v, esperado) {
			t.Errorf("falta %q en la vista cerrado el panel: %s", esperado, v)
		}
	}

	// Panel abierto: el panel está a la derecha y el aviso no se repite fuera.
	a.Panel.Abierto = true
	v = sinEstilo(a.View())
	if !strings.Contains(v, "PANEL") || !strings.Contains(v, "2 esperando decisión") {
		t.Errorf("con el panel abierto se ve el panel con sus datos: %s", v)
	}
	if strings.Contains(v, "esperando tu decisión") {
		t.Errorf("con el panel abierto el aviso no se repite fuera: %s", v)
	}
}

// --- T-F011-03: recorridos de componente ---------------------------------------

func TestRecorridoBienvenidaEnvioYChat(t *testing.T) {
	p := &puertoStub{}
	a := Nuevo(p)
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	if a.Vista != VistaBienvenida {
		t.Fatal("el recorrido empieza en la bienvenida")
	}
	escribe(t, a, "primera petición")
	ejecuta(t, a, pulsa(t, a, tea.KeyMsg{Type: tea.KeyEnter}))

	if a.Vista != VistaPrincipal {
		t.Fatal("el envío abre la interfaz principal")
	}
	if n := strings.Count(sinEstilo(a.View()), "primera petición"); n != 1 {
		t.Errorf("el primer mensaje se ve una sola vez: %d", n)
	}
	if len(a.Chat.Mensajes()) != 1 {
		t.Fatalf("es el primer mensaje del chat: %+v", a.Chat.Mensajes())
	}
	// El recorrido sigue: la segunda petición entra en el mismo chat.
	escribe(t, a, "segunda")
	ejecuta(t, a, pulsa(t, a, tea.KeyMsg{Type: tea.KeyEnter}))
	if len(a.Chat.Mensajes()) != 2 {
		t.Errorf("el chat acumula los intercambios: %+v", a.Chat.Mensajes())
	}
	if len(p.enviados) != 2 {
		t.Errorf("cada petición salió una vez: %v", p.enviados)
	}
}

func TestRecorridoCambioDeSesiónEnVivo(t *testing.T) {
	h := nuevoArnes(t)
	h.puerto.sesiones = []session.Sesion{
		{ID: "s1", Nombre: "primera"},
		{ID: "s2", Nombre: "segunda", Estado: session.EstadoTrabajando},
	}
	h.puerto.historial = []MensajeHistorial{{Rol: "user", Texto: "del pasado"}}

	h.tecla("ctrl+s")
	h.veSiContiene("segunda")
	h.veSiContiene("trabajando")
	h.tecla("down")
	cmd := pulsa(h.t, h.app, tea.KeyMsg{Type: tea.KeyEnter})
	ejecuta(h.t, h.app, cmd)

	if h.app.Panel.SesionID != "s2" {
		t.Fatalf("la activa pasa a la elegida: %q", h.app.Panel.SesionID)
	}
	if h.app.Selector.Abierto {
		t.Error("el selector desaparece al elegir")
	}
	// El historial de la elegida llega y se pinta, y nada se detiene.
	h.veSiContiene("del pasado")
	if len(h.puerto.cancelado) != 0 || len(h.puerto.pausadas) != 0 {
		t.Errorf("cambiar no cancela ni pausa nada: %v, %v", h.puerto.cancelado, h.puerto.pausadas)
	}
}

func TestRecorridoRazonamientoOcultable(t *testing.T) {
	h := nuevoArnes(t)
	h.evento(EventoToken, map[string]string{"texto": "pienso ", "razonamiento": "true"})
	h.evento(EventoToken, map[string]string{"texto": "respondo"})
	h.veSiContiene("pienso")
	h.veSiContiene("respondo")

	h.tecla("ctrl+r")
	if strings.Contains(h.ve(), "pienso") {
		t.Error("oculto no se pinta")
	}
	// Mientras está oculto sigue llegando y la vista no pierde nada.
	h.evento(EventoToken, map[string]string{"texto": "y sigo", "razonamiento": "true"})
	h.tecla("ctrl+r")
	h.veSiContiene("pienso y sigo")
	h.veSiContiene("respondo")
}

func TestRecorridoPanelPlegable(t *testing.T) {
	h := nuevoArnes(t)
	h.escribe("texto a salvo")

	h.tecla("ctrl+d")
	h.veSiContiene("PANEL")
	h.veSiContiene("Proyecto")
	h.tecla("ctrl+d")
	if strings.Contains(h.ve(), "PANEL") {
		t.Error("cerrado no se pinta")
	}
	// Abrir y cerrar no pierde lo escrito.
	h.veSiContiene("texto a salvo")
}
