package tui

import (
	"strings"
	"testing"

	"localcli/internal/tui/render"
)

// appConSesion deja la vista principal con una sesión activa, sin puerto: los
// eventos son valores y no tocan la base ni la red.
func appConSesion() *App {
	app := Nueva(nil)
	app.ancho, app.alto = 100, 24
	app.vista = VistaPrincipal
	app.sesion = &Sesion{ID: "s1", Nombre: "primera", Estado: EstadoInactiva}
	return app
}

func eventoDe(sesion, tipo string, datos map[string]string) Evento {
	return Evento{Sesion: sesion, Tipo: tipo, Datos: datos}
}

// puertoEventos entrega un canal real: sirve para comprobar que arrancar la
// vista deja la escucha del bus en marcha.
type puertoEventos struct {
	puertoFalso
	canal  chan Evento
	cerrar func()
}

func (p *puertoEventos) Suscribir() (<-chan Evento, func()) { return p.canal, p.cerrar }

// Init debe arrancar la escucha del bus: sin eso, la vista nunca recibe el primer
// evento y el turno no se pinta (T-F010-06).
func TestArrancarDejaLaEscuchaDelBusEnMarcha(t *testing.T) {
	canal := make(chan Evento)
	p := &puertoEventos{canal: canal, cerrar: func() {}}
	app := Nueva(p)

	if app.Init() == nil {
		t.Fatal("Init debería devolver la escucha del bus, no nil")
	}
	if app.eventos != (<-chan Evento)(canal) {
		t.Fatal("Init debería suscribirse al canal del puerto")
	}
}

// La vista pinta una sola sesión: lo que llega de otra no entra al chat.
func TestLosEventosDeOtraSesionNoEntranAlChat(t *testing.T) {
	app := appConSesion()

	app.aplicarEvento(eventoDe("s2", EventoToken, map[string]string{"texto": "ajeno"}))
	if !app.chat.Vacio() {
		t.Fatal("un token de otra sesión no debería pintarse")
	}

	app.aplicarEvento(eventoDe("s1", EventoToken, map[string]string{"texto": "mío"}))
	if app.chat.Vacio() {
		t.Fatal("un token de la sesión activa debería pintarse")
	}
	if hilo := render.SinANSI(app.chat.Hilo(80)); !strings.Contains(hilo, "mío") {
		t.Fatalf("el token no llegó al hilo:\n%s", hilo)
	}
}

// `notificacion` es la excepción: se ve aunque sea de otra sesión, porque su
// ausencia detendría el trabajo sin que el usuario lo supiera.
// Ahora se muestra debajo de la caja de entrada, no en el chat.
func TestLaNotificacionDeOtraSesionSiSeVe(t *testing.T) {
	app := appConSesion()

	app.aplicarEvento(eventoDe("s2", EventoNotificacion, map[string]string{
		"motivo":    "esperando aprobación",
		"siguiente": "revisa el panel",
	}))

	if !strings.Contains(app.notificacionBajoInput, "esperando aprobación") || !strings.Contains(app.notificacionBajoInput, "revisa el panel") {
		t.Fatalf("la notificación de otra sesión debería verse debajo del input:\n%s", app.notificacionBajoInput)
	}
}

// El título que genera el modelo renombra la sesión activa sin cambiar su id.
func TestElTituloRenombraLaSesionActiva(t *testing.T) {
	app := appConSesion()

	app.aplicarEvento(eventoDe("s1", EventoTituloSesion, map[string]string{"titulo": "arreglar el login"}))
	if app.sesion.Nombre != "arreglar el login" || app.sesion.ID != "s1" {
		t.Fatalf("el título no se aplicó: %+v", app.sesion)
	}
	// El de otra sesión no renombra la activa.
	app.aplicarEvento(eventoDe("s2", EventoTituloSesion, map[string]string{"titulo": "otra cosa"}))
	if app.sesion.Nombre != "arreglar el login" {
		t.Fatalf("otra sesión renombró la activa: %+v", app.sesion)
	}
}

// Una sesión que ya estaba trabajando reanuda su contador al abrirla, y al
// terminar el turno se cierra: el indicador en vivo no queda encendido.
func TestElEstadoReanudaYCierraElTurno(t *testing.T) {
	app := appConSesion()

	app.aplicarEvento(eventoDe("s1", EventoEstadoSesion, map[string]string{"estado": EstadoTrabajando}))
	if !app.chat.HayTurno() || app.sesion.Estado != EstadoTrabajando {
		t.Fatalf("el turno no se reanudó: %v %q", app.chat.HayTurno(), app.sesion.Estado)
	}
	app.aplicarEvento(eventoDe("s1", EventoEstadoSesion, map[string]string{"estado": EstadoTerminada}))
	if app.chat.HayTurno() {
		t.Fatal("el turno debería haberse cerrado")
	}
}

// El consumo del turno va bajo la caja, y el contexto y su límite al panel, con
// el tope del harness marcado cuando lo es.
func TestLosTokensAlimentanLaCajaYElPanel(t *testing.T) {
	app := appConSesion()

	app.aplicarEvento(eventoDe("s1", EventoTokensTurno, map[string]string{
		"contexto":       "1234",
		"limite":         "2000",
		"limite_harness": "true",
		"salida":         "54",
	}))

	if app.entrada.Tokens != 54 {
		t.Fatalf("el consumo del turno = %d, quiero 54", app.entrada.Tokens)
	}
	if app.panel.ContextoTokens != 1234 || app.panel.LimiteTokens != 2000 {
		t.Fatalf("el contexto del panel = %d/%d", app.panel.ContextoTokens, app.panel.LimiteTokens)
	}
	if !app.panel.TokensEstimados || !app.panel.LimiteDelHarness {
		t.Fatal("la estimación y el tope del harness deberían marcarse")
	}
	if pc := app.panel.PorcentajeContexto(); pc != 61 {
		t.Fatalf("el porcentaje = %d%%, quiero 61%%", pc)
	}
}

// La lista de pasos es de la sesión activa y llega entera en un solo evento; la
// de otra sesión no entra al panel.
func TestLaListaDePasosEsDeLaSesionActiva(t *testing.T) {
	app := appConSesion()
	lista := `[{"contenido":"escribir las doradas","estado":"en_progreso"},{"contenido":"revisar el marco","estado":"pendiente"}]`

	app.aplicarEvento(eventoDe("s2", EventoTodoActualizada, map[string]string{"elementos": lista}))
	if len(app.panel.Tareas) != 0 {
		t.Fatal("la lista de otra sesión no debería entrar al panel")
	}

	app.aplicarEvento(eventoDe("s1", EventoTodoActualizada, map[string]string{"elementos": lista}))
	if len(app.panel.Tareas) != 2 || app.panel.Tareas[0].Contenido != "escribir las doradas" {
		t.Fatalf("la lista de la sesión activa no se aplicó: %+v", app.panel.Tareas)
	}
	// Una lista ilegible no vacía el panel: se queda lo que había.
	app.aplicarEvento(eventoDe("s1", EventoTodoActualizada, map[string]string{"elementos": "{roto}"}))
	if len(app.panel.Tareas) != 2 {
		t.Fatalf("una lista ilegible no debería vaciar el panel: %+v", app.panel.Tareas)
	}
}

// El contador de aprobaciones es de todas las sesiones: una que espera decisión
// no deja de contar porque la activa sea otra.
func TestElContadorDeAprobacionesEsDeTodas(t *testing.T) {
	app := appConSesion()

	app.aplicarEvento(eventoDe("s2", EventoPeticionAprobacion, map[string]string{"aprobacion": "a1"}))
	app.aplicarEvento(eventoDe("s3", EventoPeticionAprobacion, map[string]string{"aprobacion": "a2"}))
	if app.panel.Aprobaciones != 2 {
		t.Fatalf("las aprobaciones pendientes = %d, quiero 2", app.panel.Aprobaciones)
	}
	app.aplicarEvento(eventoDe("s2", EventoAprobacionResuelta, map[string]string{"aprobacion": "a1"}))
	if app.panel.Aprobaciones != 1 {
		t.Fatalf("al resolverse una quedan %d, quiero 1", app.panel.Aprobaciones)
	}
	// Una resolución repetida no resta de más.
	app.aplicarEvento(eventoDe("s2", EventoAprobacionResuelta, map[string]string{"aprobacion": "a1"}))
	if app.panel.Aprobaciones != 1 {
		t.Fatalf("una resolución repetida dejó el contador en %d", app.panel.Aprobaciones)
	}
}

// La línea de herramienta nace al invocar y se completa con su medida y su
// tiempo; la salida cruda no entra al hilo.
func TestLaLineaDeHerramientaNaceYSeCompleta(t *testing.T) {
	app := appConSesion()

	app.aplicarEvento(eventoDe("s1", EventoHerramientaInvocada, map[string]string{
		"herramienta": "leer", "verbo": "LEER", "tema": "internal/tui/app.go", "salida": "no debería verse",
	}))
	if hilo := render.SinANSI(app.chat.Hilo(80)); !strings.Contains(hilo, "LEER [internal/tui/app.go]") {
		t.Fatalf("la invocación no se pintó:\n%s", hilo)
	}

	app.aplicarEvento(eventoDe("s1", EventoHerramientaResultado, map[string]string{
		"herramienta": "leer", "ok": "true", "medida": "70 líneas", "duracion": "400",
	}))
	hilo := render.SinANSI(app.chat.Hilo(80))
	if !strings.Contains(hilo, "✓ LEER [internal/tui/app.go] · 70 líneas · 400 ms") {
		t.Fatalf("la línea no se completó:\n%s", hilo)
	}
}

// Cada etapa de flujo deja su línea, y un cambio aplicado suma al árbol de git
// sin volver a preguntar al repositorio.
func TestLasEtapasYLosCambiosDejanSuRastro(t *testing.T) {
	app := appConSesion()
	app.panel.GitCambios = 1

	app.aplicarEvento(eventoDe("s1", EventoEtapaIniciada, map[string]string{"nombre": "Diagnosticar"}))
	app.aplicarEvento(eventoDe("s1", EventoCambioAplicado, map[string]string{"archivo": "internal/tui/app.go"}))

	hilo := render.SinANSI(app.chat.Hilo(80))
	if !strings.Contains(hilo, "[Sub Proceso] Diagnosticar") {
		t.Fatalf("la etapa no dejó su línea:\n%s", hilo)
	}
	if !strings.Contains(hilo, "cambio aplicado en internal/tui/app.go") {
		t.Fatalf("el cambio no dejó su aviso:\n%s", hilo)
	}
	if app.panel.GitCambios != 2 {
		t.Fatalf("los cambios sin confirmar = %d, quiero 2", app.panel.GitCambios)
	}
}

// La cola es del proyecto, no de una sesión: se ve desde cualquiera.
func TestLaColaSeVeDesdeCualquierSesion(t *testing.T) {
	app := appConSesion()

	app.aplicarEvento(eventoDe("", EventoColaActualizada, map[string]string{
		"capa": "frontend", "activa": "T-F009", "pendientes": "3",
	}))

	if app.panel.Capa != "frontend" || app.panel.ElementoActual != "T-F009" || app.panel.ElementosRestantes != 3 {
		t.Fatalf("la cola no se aplicó: %+v", app.panel.Datos)
	}
}

// Sin sesión activa no hay a quién atribuir un token: se descarta sin romper
// nada, pero el contador de aprobaciones sigue vivo.
func TestSinSesionActivaLosEventosNoRompenNada(t *testing.T) {
	app := Nueva(nil)
	app.ancho, app.alto = 100, 24

	app.aplicarEvento(eventoDe("s1", EventoToken, map[string]string{"texto": "hola"}))
	if !app.chat.Vacio() {
		t.Fatal("sin sesión activa no debería pintarse nada")
	}
	app.aplicarEvento(eventoDe("s1", EventoPeticionAprobacion, map[string]string{"aprobacion": "a1"}))
	if app.panel.Aprobaciones != 1 {
		t.Fatalf("el contador de aprobaciones = %d, quiero 1", app.panel.Aprobaciones)
	}
	if len(app.chat.Piezas()) != 0 {
		t.Fatalf("el chat no debería tener piezas: %+v", app.chat.Piezas())
	}
}
