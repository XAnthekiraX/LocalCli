package tui

// Tests de T-F010: el modelo raíz y su integración. Una prueba, una regla
// (frontend/05-quality/TESTING.md §3):
//
//	T-F010-04 → un atajo reasignado dispara la misma acción (tui_test.go)
//	T-F010-05 → los eventos del motor llegan a la vista por el canal
//	T-F010-06 → la escucha se arma en Init, se re-arma tras cada evento y
//	            no abre suscripciones nuevas
//	T-F010-07 → el modal de atajos (Ctrl+P) lista el keymap vigente; Esc lo
//	            cierra sin efectos (T-F014-03 lo tirelesse)
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
		// Init devuelve el comando de la escucha: el bucle reparte su mensaje
		// uno a uno, y la prueba hace lo mismo en vez de entregar el comando
		// entero a Update.
		nueva := entrega(t, a, cmd)
		if nueva == nil {
			t.Fatal("tras cada evento la escucha queda re-armada")
		}
		cmd = nueva
	}
	if p.suscripciones != 1 {
		t.Errorf("no se re-suscribe en cada evento: %d", p.suscripciones)
	}
	v := sinEstilo(a.View())
	if !strings.Contains(v, "[Sub Proceso] plan") {
		t.Errorf("los eventos de etapa llegaron a la vista: %s", v)
	}
	if strings.Contains(v, "etapa terminada") {
		t.Errorf("la etapa terminada no deja línea propia: %s", v)
	}
}

// --- T-F010-07: el modal de atajos -------------------------------------------

// entrega corre el comando y entrega a Update cada mensaje que produce,
// deshaciendo el Batch igual que el bucle de Bubble Tea. Devuelve el último
// comando no vacío, que es el que el bucle encadenaría.
func entrega(t *testing.T, a *App, cmd tea.Cmd) tea.Cmd {
	t.Helper()
	var ultimo tea.Cmd
	for _, msg := range mensajesDe(cmd) {
		var siguiente tea.Cmd
		_, siguiente = a.Update(msg)
		if siguiente != nil {
			ultimo = siguiente
		}
	}
	return ultimo
}

// mensajesDe ejecuta un comando y devuelve sus mensajes, recursively si el
// comando era un lote.
func mensajesDe(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if msg == nil {
		return nil
	}
	if lote, ok := msg.(tea.BatchMsg); ok {
		var msgs []tea.Msg
		for _, sub := range lote {
			msgs = append(msgs, mensajesDe(sub)...)
		}
		return msgs
	}
	return []tea.Msg{msg}
}

// T-F014-03: la ayuda clásica (`?`) quedó sustituida por el modal de atajos
// (`command_palette`, Ctrl+P). Se abre desde cualquier vista, lista el keymap
// vigente y se cierra con Esc, sin aplicar nada.
func TestElModalDeAtajosListaLosAtajosYSeCierraConEsc(t *testing.T) {
	p := &puertoStub{}
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})

	tecla(t, a, tea.KeyCtrlP)
	if !a.AtajosModal.Abierto {
		t.Fatal("ctrl+p abre el modal de atajos")
	}
	v := a.View()
	for _, atajo := range a.Atajos {
		if !strings.Contains(v, atajo.Tecla()) || !strings.Contains(v, atajo.Descripcion) {
			t.Errorf("el modal debe listar %q con su acción: %s", atajo.Tecla(), atajo.Descripcion)
		}
	}
	// Esc lo cierra y no deja rastro: no escribe en la entrada.
	pulsa(t, a, tea.KeyMsg{Type: tea.KeyEsc})
	if a.AtajosModal.Abierto {
		t.Error("esc cierra el modal de atajos")
	}
	if a.Entrada.Texto() != "" {
		t.Errorf("cerrar el modal no escribe: %q", a.Entrada.Texto())
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
	if !strings.Contains(v, "CONTEXTO") || !strings.Contains(v, "2 esperando decisión") {
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

	h.modalDeSesiones()
	h.veSiContiene("segunda")
	h.veSiContiene("trabajando")
	h.tecla("down")
	cmd := pulsa(h.t, h.app, tea.KeyMsg{Type: tea.KeyEnter})
	ejecuta(h.t, h.app, cmd)

	if h.app.Panel.SesionID != "s2" {
		t.Fatalf("la activa pasa a la elegida: %q", h.app.Panel.SesionID)
	}
	if h.app.Sesiones.Abierto {
		t.Error("el modal desaparece al elegir")
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
	// Por defecto el razonamiento no se vuelca: se ve la respuesta y el indicador.
	h.veSiContiene("respondo")
	if strings.Contains(h.ve(), "pienso") {
		t.Error("el razonamiento no se vuelca sin revelarlo")
	}

	// Ctrl+R revela el texto y sigue acumulando sin perder nada.
	h.tecla("ctrl+r")
	h.veSiContiene("pienso")
	h.evento(EventoToken, map[string]string{"texto": "y sigo", "razonamiento": "true"})
	h.veSiContiene("pienso y sigo")
	h.veSiContiene("respondo")

	// Ocultarlo de nuevo lo retira sin borrarlo.
	h.tecla("ctrl+r")
	if strings.Contains(h.ve(), "pienso") {
		t.Error("oculto no se pinta")
	}
}

func TestRecorridoPanelPlegable(t *testing.T) {
	h := nuevoArnes(t)
	h.escribe("texto a salvo")

	// El sidebar arranca visible: el recorrido va de plegar a desplegar.
	h.tecla("ctrl+d")
	if strings.Contains(h.ve(), "CONTEXTO") {
		t.Error("cerrado no se pinta")
	}
	h.tecla("ctrl+d")
	h.veSiContiene("CONTEXTO")
	// Desplegado, el pie vuelve con la firma del harness: la ruta y la versión
	// no son un dato de ESTADO, así que no se quedan al plegar.
	h.veSiContiene(Nombre)
	// Plegar y desplegar no pierde lo escrito.
	h.veSiContiene("texto a salvo")
}
