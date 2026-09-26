package session

// Tests del módulo session (T-B013-09). Cubren las verificaciones pedidas en
// 013-task-session.md subtarea por subtarea:
//
//	01 → transiciones inválidas rechazadas
//	02 → borrar la sesión arrastra mensajes, razonamiento, aprobaciones y auditoría
//	03 → dos proyectos temporales no comparten listado
//	04 → el trabajo sigue avanzando mientras la vista está en otra sesión
//	05 → enviar el mensaje invoca flow una vez
//	06 → pausar detiene después del elemento en curso
//	07 → cada disparador produce su notificación de EVENTS.md
//	08 → el consumidor recibe los eventos en orden
//	09 → este propio fichero: go test ./internal/session/... pasa

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"localcli/internal/flow"
	"localcli/internal/store"
	"localcli/internal/task"
)

// --- dobles ---------------------------------------------------------------

// almacenMem es un Almacen en memoria con la misma semántica mínima que `store`.
type almacenMem struct {
	mu         sync.Mutex
	sesiones   map[string]*store.Session
	mensajes   []store.Message
	aprobacion []store.AprobacionPendiente
	siguiente  int
}

func nuevoAlmacenMem() *almacenMem {
	return &almacenMem{sesiones: map[string]*store.Session{}}
}

func (a *almacenMem) Crear(nombre, capa string) (*store.Session, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.siguiente++
	s := &store.Session{
		ID:     "s" + string(rune('0'+a.siguiente)),
		Name:   nombre,
		Layer:  capa,
		Status: store.StatusInactiva,
	}
	a.sesiones[s.ID] = s
	return s, nil
}

func (a *almacenMem) Obtener(id string) (*store.Session, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	s, ok := a.sesiones[id]
	if !ok {
		return nil, store.ErrNoEncontrado
	}
	copia := *s
	return &copia, nil
}

func (a *almacenMem) Listar(estado string) ([]store.Session, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	var out []store.Session
	for _, s := range a.sesiones {
		if estado == "" || s.Status == estado {
			out = append(out, *s)
		}
	}
	return out, nil
}

func (a *almacenMem) CambiarEstado(id, estado string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	s, ok := a.sesiones[id]
	if !ok {
		return store.ErrNoEncontrado
	}
	if !store.ValidarTransicionSesion(s.Status, estado) {
		return store.ErrEstadoIlegal
	}
	s.Status = estado
	return nil
}

func (a *almacenMem) Borrar(id string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, ok := a.sesiones[id]; !ok {
		return store.ErrNoEncontrado
	}
	delete(a.sesiones, id)
	return nil
}

func (a *almacenMem) Historial(sessionID string) ([]store.MensajeConRazonamiento, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	var out []store.MensajeConRazonamiento
	for _, m := range a.mensajes {
		if m.SessionID == sessionID {
			out = append(out, store.MensajeConRazonamiento{Message: m})
		}
	}
	return out, nil
}

func (a *almacenMem) EscribirMensaje(m *store.Message) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.mensajes = append(a.mensajes, *m)
	return nil
}

func (a *almacenMem) CerrarTurno(sessionID, contenido string, in, out int, razonamiento, estadoSesion string) (*store.Message, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	m := store.Message{SessionID: sessionID, Role: "agent", Content: contenido}
	a.mensajes = append(a.mensajes, m)
	if estadoSesion != "" {
		s, ok := a.sesiones[sessionID]
		if !ok {
			return nil, store.ErrNoEncontrado
		}
		if !store.ValidarTransicionSesion(s.Status, estadoSesion) {
			return nil, store.ErrEstadoIlegal
		}
		s.Status = estadoSesion
	}
	return &m, nil
}

func (a *almacenMem) PendientesDeAprobacion() ([]store.AprobacionPendiente, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]store.AprobacionPendiente(nil), a.aprobacion...), nil
}

func (a *almacenMem) estadoDe(t *testing.T, id string) string {
	t.Helper()
	s, err := a.Obtener(id)
	if err != nil {
		t.Fatalf("Obtener(%s): %v", id, err)
	}
	return s.Status
}

// motorStub permite controlar el desenlace del flujo y su duración.
type motorStub struct {
	mu       sync.Mutex
	llamadas []string
	estado   flow.EstadoFlujo
	err      error
	// bloqueo, si no es nil, mantiene el flujo vivo hasta que se cierre.
	bloqueo chan struct{}
	// iniciado avisa de que el flujo arrancó.
	iniciado chan struct{}

	// ConsumirCola
	cola        flow.Cola
	primerHecho chan struct{} // se cierra al completar el primer elemento
	continuar   chan struct{} // el motor espera esto tras el primer elemento
	marcados    int
}

func nuevoMotor(estado flow.EstadoFlujo) *motorStub {
	return &motorStub{estado: estado, iniciado: make(chan struct{})}
}

func (m *motorStub) EjecutarFlujo(ctx context.Context, f flow.Flujo, objetivo string) (flow.EstadoFlujo, error) {
	m.mu.Lock()
	m.llamadas = append(m.llamadas, objetivo)
	bloqueo := m.bloqueo
	iniciado := m.iniciado
	m.mu.Unlock()

	if iniciado != nil {
		select {
		case <-iniciado:
		default:
			close(iniciado)
		}
	}
	if bloqueo != nil {
		select {
		case <-bloqueo:
		case <-ctx.Done():
			return flow.EstadoDetenido, ctx.Err()
		}
	}
	if m.err != nil {
		return flow.EstadoConError, m.err
	}
	return m.estado, nil
}

func (m *motorStub) ConsumirCola(ctx context.Context, cola flow.Cola, f flow.Flujo) error {
	m.mu.Lock()
	m.cola = cola
	primer, continuar := m.primerHecho, m.continuar
	m.mu.Unlock()
	for {
		elem, err := cola.Siguiente(ctx)
		if err != nil {
			return err
		}
		if elem == nil {
			return nil
		}
		if err := cola.Marcar(elem.ID, task.EstadoCompletada); err != nil {
			return err
		}
		m.mu.Lock()
		m.marcados++
		n := m.marcados
		m.mu.Unlock()
		if n == 1 && primer != nil {
			close(primer)
			if continuar != nil {
				<-continuar
			}
		}
	}
}

func (m *motorStub) llamadasHechas() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.llamadas...)
}

func (m *motorStub) elementosMarcados() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.marcados
}

// colaStub es una cola de elementos fijos para el motor.
type colaStub struct {
	ids      []string
	i        int
	marcados []string
}

func (c *colaStub) Siguiente(ctx context.Context) (*flow.ElementoCola, error) {
	if c.i >= len(c.ids) {
		return nil, nil
	}
	id := c.ids[c.i]
	return &flow.ElementoCola{ID: id, Objetivo: "ejecutar " + id}, nil
}

func (c *colaStub) Marcar(id string, estado task.Estado) error {
	c.marcados = append(c.marcados, id+":"+string(estado))
	c.i++
	return nil
}

// gestorDe arma un gestor con almacén en memoria dentro de una carpeta temporal.
func gestorDe(t *testing.T, motor Motor) (*Gestor, *almacenMem) {
	t.Helper()
	alm := nuevoAlmacenMem()
	alc, err := NuevoAlcance(t.TempDir(), alm)
	if err != nil {
		t.Fatalf("NuevoAlcance: %v", err)
	}
	g, err := NuevoGestor(alc, motor, NuevoBus())
	if err != nil {
		t.Fatalf("NuevoGestor: %v", err)
	}
	return g, alm
}

// esperarEstado espera a que la sesión quede en el estado pedido.
func esperarEstado(t *testing.T, g *Gestor, id, quiero string) {
	t.Helper()
	limite := time.Now().Add(3 * time.Second)
	for time.Now().Before(limite) {
		s, err := g.Estado(id)
		if err != nil {
			t.Fatalf("Estado(%s): %v", id, err)
		}
		if s.Estado == quiero {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	s, _ := g.Estado(id)
	t.Fatalf("la sesión %s quedó en %q, quería %q", id, s.Estado, quiero)
}

// --- T-B013-01: modelo y transiciones -------------------------------------

func TestTransicionesInvalidasSeRechazan(t *testing.T) {
	casos := []struct {
		desde, hacia string
		ok           bool
	}{
		{EstadoInactiva, EstadoTrabajando, true},
		{EstadoTrabajando, EstadoEsperandoPermiso, true},
		{EstadoEsperandoPermiso, EstadoTrabajando, true},
		{EstadoEsperandoPermiso, EstadoTerminada, false},
		{EstadoTerminada, EstadoTrabajando, false},
		{EstadoError, EstadoTrabajando, false},
		{EstadoTerminada, EstadoInactiva, true},
	}
	for _, c := range casos {
		s := &Sesion{ID: "s1", Estado: c.desde}
		err := s.PuedePasarA(c.hacia)
		if c.ok && err != nil {
			t.Errorf("%s → %s debería ser válida: %v", c.desde, c.hacia, err)
		}
		if !c.ok && err == nil {
			t.Errorf("%s → %s debería rechazarse", c.desde, c.hacia)
		}
	}
	if err := (&Sesion{ID: "s1", Estado: EstadoInactiva}).PuedePasarA("inventado"); err == nil {
		t.Error("un estado fuera del catálogo debe rechazarse")
	}
}

// --- T-B013-02: persistencia real, con cascada ----------------------------

func TestBorrarSesionArrastraSuContenido(t *testing.T) {
	raiz := proyectoDePrueba(t)
	db, err := store.Open(raiz)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	defer db.Close()

	sesion, err := store.CrearSesion(db, "trabajo", store.LayerBackend)
	if err != nil {
		t.Fatal(err)
	}
	msg := &store.Message{SessionID: sesion.ID, Role: "user", Content: "hola"}
	if err := store.InsertarMensaje(db, msg); err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertRazonamiento(db, msg.ID, "pensando"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.PedirAprobacion(db, sesion.ID, "crear archivo"); err != nil {
		t.Fatal(err)
	}
	if err := store.RegistrarAuditoria(db, &store.ContextAudit{
		SessionID: sesion.ID, Stage: "etapa", Document: "a.md", Decision: store.AuditIncluido, Tokens: 3,
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.RegistrarCambio(db, &store.ChangeHistory{
		SessionID: sesion.ID, Operation: store.OpCrearArchivo, FilePath: "x.go", AfterContent: "x",
	}); err != nil {
		t.Fatal(err)
	}

	if err := store.BorrarSesion(db, sesion.ID); err != nil {
		t.Fatalf("BorrarSesion: %v", err)
	}

	if h, err := store.HistorialSesion(db, sesion.ID); err != nil || len(h) != 0 {
		t.Errorf("el historial debía caer en cascada: %v (%d filas)", err, len(h))
	}
	if a, err := store.ListarAprobacionesDeSesion(db, sesion.ID, ""); err != nil || len(a) != 0 {
		t.Errorf("las aprobaciones debían caer en cascada: %v (%d filas)", err, len(a))
	}
	if a, err := store.AuditoriaDeEtapa(db, sesion.ID, "etapa"); err != nil || len(a) != 0 {
		t.Errorf("la auditoría debía caer en cascada: %v (%d filas)", err, len(a))
	}
	// change_history sobrevive, con session_id a NULL.
	historial, err := store.HistorialDeArchivo(db, "x.go")
	if err != nil || len(historial) != 1 {
		t.Fatalf("change_history debe sobrevivir a la sesión: %v (%d filas)", err, len(historial))
	}
	if historial[0].SessionID != "" {
		t.Errorf("session_id debía quedar a NULL, quedó %q", historial[0].SessionID)
	}
}

// --- T-B013-03: aislamiento por carpeta -----------------------------------

func TestDosCarpetasNoCompartenSesiones(t *testing.T) {
	raizA, raizB := proyectoDePrueba(t), proyectoDePrueba(t)
	dbA, err := store.Open(raizA)
	if err != nil {
		t.Fatal(err)
	}
	defer dbA.Close()
	dbB, err := store.Open(raizB)
	if err != nil {
		t.Fatal(err)
	}
	defer dbB.Close()

	alcA, err := NuevoAlcance(raizA, store.Sesiones{DB: dbA})
	if err != nil {
		t.Fatal(err)
	}
	alcB, err := NuevoAlcance(raizB, store.Sesiones{DB: dbB})
	if err != nil {
		t.Fatal(err)
	}
	rutaA, _ := alcA.RutaBase()
	rutaB, _ := alcB.RutaBase()
	if rutaA == rutaB {
		t.Fatalf("dos carpetas no pueden compartir el archivo de estado: %s", rutaA)
	}
	if MismaCarpeta(raizA, raizB) {
		t.Fatal("las dos carpetas de prueba deben ser distintas")
	}

	motor := nuevoMotor(flow.EstadoTerminado)
	gA, err := NuevoGestor(alcA, motor, NuevoBus())
	if err != nil {
		t.Fatal(err)
	}
	gB, err := NuevoGestor(alcB, motor, NuevoBus())
	if err != nil {
		t.Fatal(err)
	}
	sesA, err := gA.Crear("de la carpeta A", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := gB.Crear("de la carpeta B", ""); err != nil {
		t.Fatal(err)
	}

	listaA, err := gA.Listar()
	if err != nil {
		t.Fatal(err)
	}
	if len(listaA) != 1 || listaA[0].ID != sesA.ID || listaA[0].Nombre != "de la carpeta A" {
		t.Fatalf("la carpeta A ve %+v", listaA)
	}
	// El alcance B no contiene la sesión de A.
	if ok, err := alcB.Contiene(sesA.ID); err != nil || ok {
		t.Errorf("la sesión de A no puede estar en el alcance de B (ok=%v err=%v)", ok, err)
	}
}

// --- T-B013-04: segundo plano --------------------------------------------

func TestElTrabajoSigueAunqueLaVistaEstéEnOtraSesión(t *testing.T) {
	motor := nuevoMotor(flow.EstadoTerminado)
	motor.bloqueo = make(chan struct{})
	g, _ := gestorDe(t, motor)

	a, err := g.Crear("sesión A", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := g.Enviar(context.Background(), a.ID, "trabaja en A"); err != nil {
		t.Fatal(err)
	}
	// La "vista" se va a otra sesión: se crea y se usa la B sin tocar la A.
	b, err := g.Crear("sesión B", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := g.Enviar(context.Background(), b.ID, "trabaja en B"); err != nil {
		t.Fatal(err)
	}
	if !g.EnCurso(a.ID) {
		t.Fatal("cambiar de sesión no debe detener la A")
	}

	close(motor.bloqueo)
	esperarEstado(t, g, a.ID, EstadoTerminada)
	esperarEstado(t, g, b.ID, EstadoTerminada)
	if len(motor.llamadasHechas()) != 2 {
		t.Fatalf("cada sesión arranca su flujo una vez: %v", motor.llamadasHechas())
	}
}

func TestCerrarSesiónEnCursoPideDecisión(t *testing.T) {
	motor := nuevoMotor(flow.EstadoTerminado)
	motor.bloqueo = make(chan struct{})
	g, _ := gestorDe(t, motor)
	s, err := g.Crear("en curso", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := g.Enviar(context.Background(), s.ID, "algo largo"); err != nil {
		t.Fatal(err)
	}
	if err := g.Cerrar(s.ID, false); err == nil {
		t.Fatal("cerrar con trabajo en curso debe preguntar qué hacer con el flujo")
	}
	if err := g.Cerrar(s.ID, true); err != nil {
		t.Fatalf("forzar el cierre: %v", err)
	}
	if _, err := g.Estado(s.ID); !errors.Is(err, store.ErrNoEncontrado) {
		t.Errorf("tras cerrar, la sesión no debe existir: %v", err)
	}
}

// --- T-B013-05: arrancar el flujo ----------------------------------------

func TestEnviarInvocaFlowUnaVez(t *testing.T) {
	motor := nuevoMotor(flow.EstadoTerminado)
	g, alm := gestorDe(t, motor)
	s, err := g.Crear("sesión", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := g.Enviar(context.Background(), s.ID, "  implementa la API  "); err != nil {
		t.Fatal(err)
	}
	esperarEstado(t, g, s.ID, EstadoTerminada)

	llamadas := motor.llamadasHechas()
	if len(llamadas) != 1 {
		t.Fatalf("flow se invoca una vez por mensaje, se invocó %d", len(llamadas))
	}
	if llamadas[0] != "implementa la API" {
		t.Errorf("objetivo = %q, quiero el mensaje del usuario sin espacios", llamadas[0])
	}
	if err := g.Enviar(context.Background(), s.ID, ""); err == nil {
		t.Error("un mensaje vacío no debe arrancar nada")
	}
	h, err := alm.Historial(s.ID)
	if err != nil || len(h) != 2 {
		t.Fatalf("el historial debe tener el mensaje del usuario y el cierre del turno: %v (%d)", err, len(h))
	}
	if h[0].Role != "user" || h[1].Role != "agent" {
		t.Errorf("roles inesperados: %s, %s", h[0].Role, h[1].Role)
	}
}

func TestFalloDejaLaSesiónEnError(t *testing.T) {
	motor := nuevoMotor(flow.EstadoConError)
	motor.err = errors.New("E_STAGE_FAILED: la etapa falló")
	g, _ := gestorDe(t, motor)
	s, _ := g.Crear("sesión", "")
	if err := g.Enviar(context.Background(), s.ID, "algo"); err != nil {
		t.Fatal(err)
	}
	esperarEstado(t, g, s.ID, EstadoError)
}

func TestPausaPorPermisoDejaLaSesiónEsperando(t *testing.T) {
	motor := nuevoMotor(flow.EstadoPausadoPermiso)
	g, _ := gestorDe(t, motor)
	s, _ := g.Crear("sesión", "")
	if err := g.Enviar(context.Background(), s.ID, "algo que pide permiso"); err != nil {
		t.Fatal(err)
	}
	esperarEstado(t, g, s.ID, EstadoEsperandoPermiso)
}

// --- T-B013-06: pausar, reanudar, cancelar --------------------------------

func TestPausarDetieneDespuésDelElementoEnCurso(t *testing.T) {
	motor := nuevoMotor(flow.EstadoTerminado)
	motor.primerHecho = make(chan struct{})
	motor.continuar = make(chan struct{})
	g, _ := gestorDe(t, motor)
	s, _ := g.Crear("cola", "")
	cola := &colaStub{ids: []string{"T-B001", "T-B002"}}

	if err := g.ConsumirCola(context.Background(), s.ID, cola); err != nil {
		t.Fatal(err)
	}
	<-motor.primerHecho // el primer elemento ya está completado

	if err := g.Pausar(s.ID); err != nil {
		t.Fatalf("Pausar: %v", err)
	}
	if !g.Pausada(s.ID) {
		t.Fatal("la sesión debería estar pausada")
	}
	close(motor.continuar)
	g.Esperar(s.ID)

	if motor.elementosMarcados() != 1 {
		t.Errorf("pausar debe detener TRAS el elemento en curso: se marcaron %d", motor.elementosMarcados())
	}
	if len(cola.marcados) != 1 || !strings.HasPrefix(cola.marcados[0], "T-B001:") {
		t.Errorf("marcas inesperadas: %v", cola.marcados)
	}
	if s, _ := g.Estado(s.ID); s.Estado != EstadoTerminada {
		t.Errorf("estado = %s, quiero terminada", s.Estado)
	}

	// Con el trabajo terminado ya no hay nada que reanudar.
	if err := g.Reanudar(s.ID); err == nil {
		t.Error("reanudar una sesión sin trabajo en curso debe avisar")
	}
}

func TestReanudarVuelveADejarPasarElementos(t *testing.T) {
	motor := nuevoMotor(flow.EstadoTerminado)
	motor.primerHecho = make(chan struct{})
	motor.continuar = make(chan struct{})
	g, _ := gestorDe(t, motor)
	s, _ := g.Crear("cola", "")
	cola := &colaStub{ids: []string{"T-B001", "T-B002"}}

	if err := g.ConsumirCola(context.Background(), s.ID, cola); err != nil {
		t.Fatal(err)
	}
	<-motor.primerHecho
	if err := g.Pausar(s.ID); err != nil {
		t.Fatal(err)
	}
	if err := g.Reanudar(s.ID); err != nil {
		t.Fatalf("Reanudar: %v", err)
	}
	if g.Pausada(s.ID) {
		t.Fatal("tras reanudar, la sesión no puede seguir pausada")
	}
	close(motor.continuar)
	g.Esperar(s.ID)

	if motor.elementosMarcados() != 2 {
		t.Errorf("tras reanudar deben ejecutarse los dos elementos, se marcaron %d", motor.elementosMarcados())
	}
}

func TestPausarSinTrabajoAvisa(t *testing.T) {
	g, _ := gestorDe(t, nuevoMotor(flow.EstadoTerminado))
	s, _ := g.Crear("quieta", "")
	if err := g.Pausar(s.ID); err == nil {
		t.Error("pausar una sesión sin trabajo en curso debe avisar")
	}
}

func TestCancelarCortaElTrabajo(t *testing.T) {
	motor := nuevoMotor(flow.EstadoTerminado)
	motor.bloqueo = make(chan struct{})
	g, _ := gestorDe(t, motor)
	s, _ := g.Crear("cancelable", "")
	if err := g.Enviar(context.Background(), s.ID, "algo largo"); err != nil {
		t.Fatal(err)
	}
	<-motor.iniciado
	g.Cancelar(s.ID)
	esperarEstado(t, g, s.ID, EstadoInactiva)
}

// --- T-B013-07: notificaciones -------------------------------------------

func TestNotificaAlTerminar(t *testing.T) {
	g, _ := gestorDe(t, nuevoMotor(flow.EstadoTerminado))
	eventos, baja := g.Bus.Suscribir()
	defer baja()

	s, _ := g.Crear("sesión", "")
	if err := g.Enviar(context.Background(), s.ID, "algo"); err != nil {
		t.Fatal(err)
	}
	esperarEstado(t, g, s.ID, EstadoTerminada)

	recibidos := recoger(eventos)
	if !contieneEvento(recibidos, EventoEstadoSesion, "trabajando") {
		t.Errorf("falta estado_sesion trabajando: %v", recibidos)
	}
	if !contieneEvento(recibidos, EventoEstadoSesion, "terminada") {
		t.Errorf("falta estado_sesion terminada: %v", recibidos)
	}
	if !contieneEvento(recibidos, EventoNotificacion, "terminó el trabajo") {
		t.Errorf("falta la notificación de término: %v", recibidos)
	}
}

func TestNotificaFalloConSuMotivo(t *testing.T) {
	motor := nuevoMotor(flow.EstadoConError)
	motor.err = errors.New("E_STAGE_FAILED: reventó")
	g, _ := gestorDe(t, motor)
	eventos, baja := g.Bus.Suscribir()
	defer baja()

	s, _ := g.Crear("sesión", "")
	_ = g.Enviar(context.Background(), s.ID, "algo")
	esperarEstado(t, g, s.ID, EstadoError)

	if !contieneEvento(recoger(eventos), EventoNotificacion, "con error") {
		t.Error("el fallo debe notificarse")
	}
}

func TestAvisaDeLaEsperaDePermisoUnaVez(t *testing.T) {
	g, alm := gestorDe(t, nuevoMotor(flow.EstadoTerminado))
	s, _ := g.Crear("sesión", "")
	alm.aprobacion = []store.AprobacionPendiente{{
		Approval:    store.Approval{ID: "a1", SessionID: s.ID, Description: "crear archivo"},
		SessionName: "sesión",
	}}
	eventos, baja := g.Bus.Suscribir()
	defer baja()

	n, err := g.AvisarEsperaDePermiso()
	if err != nil || n != 1 {
		t.Fatalf("AvisarEsperaDePermiso = %d, %v", n, err)
	}
	// Llamar de más no duplica el aviso del mismo estado.
	if n, _ := g.AvisarEsperaDePermiso(); n != 1 {
		t.Errorf("la comprobación debe seguir viendo la aprobación: %d", n)
	}
	notificaciones := 0
	for _, e := range recoger(eventos) {
		if e.Nombre == EventoNotificacion {
			notificaciones++
		}
	}
	if notificaciones != 1 {
		t.Errorf("notificaciones = %d, quiero 1 (idempotente por estado)", notificaciones)
	}
}

func TestNotificacionPorEstado(t *testing.T) {
	for _, estado := range []string{EstadoEsperandoPermiso, EstadoTerminada, EstadoError} {
		motivo, siguiente, ok := Notificacion(estado)
		if !ok || motivo == "" || siguiente == "" {
			t.Errorf("%s debe tener motivo y qué hacer a continuación: %q / %q", estado, motivo, siguiente)
		}
	}
	for _, estado := range []string{EstadoInactiva, EstadoTrabajando} {
		if _, _, ok := Notificacion(estado); ok {
			t.Errorf("%s no se notifica", estado)
		}
	}
}

// --- T-B013-08: canales hacia la tui -------------------------------------

func TestLosEventosLleganEnOrden(t *testing.T) {
	bus := NuevoBus()
	ch, baja := bus.Suscribir()
	defer baja()

	for _, n := range []string{"uno", "dos", "tres"} {
		bus.Emitir(Evento{Nombre: n})
	}
	var nombres []string
	for i := 0; i < 3; i++ {
		e := <-ch
		nombres = append(nombres, e.Nombre)
	}
	if strings.Join(nombres, ",") != "uno,dos,tres" {
		t.Fatalf("orden = %v", nombres)
	}
}

func TestUnSuscriptorLentoNoBloquea(t *testing.T) {
	bus := NuevoBus()
	_, baja := bus.Suscribir()
	defer baja()

	hecho := make(chan struct{})
	go func() {
		for i := 0; i < BufferSuscriptor*4; i++ {
			bus.Emitir(Evento{Nombre: "token"})
		}
		close(hecho)
	}()
	select {
	case <-hecho:
	case <-time.After(2 * time.Second):
		t.Fatal("un suscriptor que no lee no puede bloquear al productor")
	}
}

func TestCerrarElBusLiberaALosSuscriptores(t *testing.T) {
	bus := NuevoBus()
	ch, _ := bus.Suscribir()
	bus.Cerrar()
	if _, abierto := <-ch; abierto {
		t.Error("tras cerrar, el canal del suscriptor debe cerrarse")
	}
	bus.Emitir(Evento{Nombre: "tarde"}) // no debe entrar en pánico
}

// --- helpers ---------------------------------------------------------------

func recoger(ch <-chan Evento) []Evento {
	var out []Evento
	for {
		select {
		case e, ok := <-ch:
			if !ok {
				return out
			}
			out = append(out, e)
		default:
			return out
		}
	}
}

func contieneEvento(eventos []Evento, nombre, fragmento string) bool {
	for _, e := range eventos {
		if e.Nombre != nombre {
			continue
		}
		for _, v := range e.Datos {
			if strings.Contains(v, fragmento) {
				return true
			}
		}
	}
	return false
}

// proyectoDePrueba crea una carpeta que `store.Open` reconoce como proyecto.
func proyectoDePrueba(t *testing.T) string {
	t.Helper()
	raiz := t.TempDir()
	docs := filepath.Join(raiz, "ai", "docs")
	if err := os.MkdirAll(docs, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docs, "PROJECT.md"), []byte("# proyecto\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return raiz
}
