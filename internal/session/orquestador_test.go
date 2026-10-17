package session

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"localcli/internal/agent"
	"localcli/internal/llama"
	"localcli/internal/llm/openai"
	"localcli/internal/models"
	"localcli/internal/store"
	"localcli/internal/tui"
)

// marcaTitulo es el fragmento del prompt de título que distingue esa llamada de
// las demás en el servidor de prueba.
const marcaTitulo = "tulo breve"

// servidorSSE devuelve la respuesta `texto` en un solo fragmento. Una petición
// de título recibe un título distinto, para poder comprobarlo.
func servidorSSE(t *testing.T, texto string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cuerpo, _ := io.ReadAll(r.Body)
		respuesta := texto
		if bytes.Contains(cuerpo, []byte(marcaTitulo)) {
			respuesta = "Titulo generado"
		}
		escribirSSE(w, respuesta)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// escribirSSE emite una respuesta completa (un fragmento de texto, el uso y el
// cierre) por el canal SSE.
func escribirSSE(w http.ResponseWriter, respuesta string) {
	w.Header().Set("Content-Type", "text/event-stream")
	flusher, _ := w.(http.Flusher)
	emite := func(v any) {
		b, _ := json.Marshal(v)
		fmt.Fprintf(w, "data: %s\n\n", b)
		if flusher != nil {
			flusher.Flush()
		}
	}
	emite(map[string]any{"choices": []any{map[string]any{"delta": map[string]any{"content": respuesta}}}})
	emite(map[string]any{"choices": []any{}, "usage": map[string]any{"prompt_tokens": 4, "completion_tokens": 2}})
	fmt.Fprint(w, "data: [DONE]\n\n")
}

// capturador acumula los cuerpos de las peticiones que no son de título.
type capturador struct {
	mu      sync.Mutex
	cuerpos []string
}

func (c *capturador) añadir(body string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cuerpos = append(c.cuerpos, body)
}

func (c *capturador) ultimo() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.cuerpos) == 0 {
		return ""
	}
	return c.cuerpos[len(c.cuerpos)-1]
}

// servidorTitulos responde los turnos con `texto` y las peticiones de título con
// `titulo`, contando cuántas hubo. Con `falla`, rechaza las de título.
func servidorTitulos(t *testing.T, texto, titulo string, titulos *int32, falla bool) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cuerpo, _ := io.ReadAll(r.Body)
		if bytes.Contains(cuerpo, []byte(marcaTitulo)) {
			atomic.AddInt32(titulos, 1)
			if falla {
				http.Error(w, "sin modelo", http.StatusInternalServerError)
				return
			}
			escribirSSE(w, titulo)
			return
		}
		escribirSSE(w, texto)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// servidorCaptura responde siempre `respuesta` y guarda el cuerpo de cada
// petición de turno (no la de título).
func servidorCaptura(t *testing.T, respuesta string, cap *capturador) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cuerpo, _ := io.ReadAll(r.Body)
		if !bytes.Contains(cuerpo, []byte(marcaTitulo)) {
			cap.añadir(string(cuerpo))
		}
		escribirSSE(w, respuesta)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// agentesDePrueba deja un agente `plan` en un directorio temporal.
func agentesDePrueba(t *testing.T) *agent.Catalogo {
	t.Helper()
	base := t.TempDir()
	dir := filepath.Join(base, "plan")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "agent.yaml"), []byte("name: plan\ndescription: d\npermissions:\n  default: deny\n  read: allow\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "prompt.md"), []byte("eres plan"), 0o644); err != nil {
		t.Fatal(err)
	}
	catalogo, avisos := agent.CargarAditivo(base, "")
	if len(avisos) != 0 {
		t.Fatalf("avisos de agentes: %v", avisos)
	}
	return catalogo
}

func drenar(canal <-chan tui.Evento) []tui.Evento {
	var eventos []tui.Evento
	for len(canal) > 0 {
		eventos = append(eventos, <-canal)
	}
	return eventos
}

func TestTurnoPersisteYEmite(t *testing.T) {
	srv := servidorSSE(t, "hola")
	db, limpiar, err := store.CrearDBTemporal()
	if err != nil {
		t.Fatal(err)
	}
	defer limpiar()
	cwd := t.TempDir()
	projectID, err := store.ResolucionProyecto(db, cwd)
	if err != nil {
		t.Fatal(err)
	}

	o := Nuevo(db, projectID, cwd, agentesDePrueba(t), openai.NuevoCliente(srv.URL), "m", 8192, nil, nil)
	canal, cerrar := o.Suscribir()
	defer cerrar()

	sesion, err := o.Crear()
	if err != nil {
		t.Fatal(err)
	}
	if err := o.Enviar(context.Background(), sesion.ID, "plan", "saluda"); err != nil {
		t.Fatalf("enviar: %v", err)
	}

	eventos := drenar(canal)
	var textos []string
	estados := map[string]bool{}
	titulo := ""
	idMensaje := ""
	for _, e := range eventos {
		switch e.Tipo {
		case tui.EventoToken:
			textos = append(textos, e.Datos["texto"])
		case tui.EventoEstadoSesion:
			estados[e.Datos["estado"]] = true
		case tui.EventoTituloSesion:
			titulo = e.Datos["titulo"]
		case tui.EventoMensajePersistido:
			if e.Datos["rol"] == "user" {
				idMensaje = e.Datos["mensaje"]
			}
		}
	}
	if len(textos) != 1 || textos[0] != "hola" {
		t.Fatalf("tokens = %v", textos)
	}
	if !estados[tui.EstadoTrabajando] || !estados[tui.EstadoTerminada] {
		t.Fatalf("estados = %v", estados)
	}
	if titulo != "Titulo generado" {
		t.Fatalf("el título debería generarlo el modelo: %q", titulo)
	}

	mensajes, err := store.ListarMensajes(db, projectID, sesion.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(mensajes) != 2 || mensajes[0].Role != "user" || mensajes[1].Role != "agent" || mensajes[1].Content != "hola" {
		t.Fatalf("mensajes = %+v", mensajes)
	}
	if idMensaje != mensajes[0].ID {
		t.Fatalf("mensaje_persistido = %q, quiero el id guardado %q", idMensaje, mensajes[0].ID)
	}
}

func TestTurnoConFalloAvisa(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close()

	db, limpiar, err := store.CrearDBTemporal()
	if err != nil {
		t.Fatal(err)
	}
	defer limpiar()
	cwd := t.TempDir()
	projectID, _ := store.ResolucionProyecto(db, cwd)

	o := Nuevo(db, projectID, cwd, agentesDePrueba(t), openai.NuevoCliente(url), "m", 0, nil, nil)
	canal, cerrar := o.Suscribir()
	defer cerrar()

	sesion, _ := o.Crear()
	if err := o.Enviar(context.Background(), sesion.ID, "plan", "hola"); err == nil {
		t.Fatal("debería fallar al no haber servidor")
	}

	eventos := drenar(canal)
	var huboError, huboAviso bool
	for _, e := range eventos {
		if e.Tipo == tui.EventoEstadoSesion && e.Datos["estado"] == tui.EstadoError {
			huboError = true
		}
		if e.Tipo == tui.EventoNotificacion {
			huboAviso = true
		}
	}
	if !huboError || !huboAviso {
		t.Fatalf("faltó el error o el aviso: error=%v aviso=%v", huboError, huboAviso)
	}
}

func TestAgenteDesconocidoAvisa(t *testing.T) {
	srv := servidorSSE(t, "hola")
	db, limpiar, _ := store.CrearDBTemporal()
	defer limpiar()
	cwd := t.TempDir()
	projectID, _ := store.ResolucionProyecto(db, cwd)

	o := Nuevo(db, projectID, cwd, agentesDePrueba(t), openai.NuevoCliente(srv.URL), "m", 0, nil, nil)
	canal, cerrar := o.Suscribir()
	defer cerrar()

	sesion, _ := o.Crear()
	if err := o.Enviar(context.Background(), sesion.ID, "noexiste", "hola"); err == nil {
		t.Fatal("un agente desconocido debería fallar")
	}
	if !tieneAviso(drenar(canal)) {
		t.Fatal("debería avisar del agente desconocido")
	}
}

func tieneAviso(eventos []tui.Evento) bool {
	for _, e := range eventos {
		if e.Tipo == tui.EventoNotificacion {
			return true
		}
	}
	return false
}

// catalogoDePrueba deja un catálogo con un modelo usable y otro con el GGUF
// borrado.
func catalogoDePrueba(t *testing.T) *models.Gestor {
	t.Helper()
	dir := t.TempDir()
	dirModels := filepath.Join(dir, "models")
	if err := os.MkdirAll(filepath.Join(dirModels, "usable"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dirModels, "usable", "m.gguf"), []byte("gguf"), 0o644); err != nil {
		t.Fatal(err)
	}
	g := models.NuevoGestor(filepath.Join(dir, "models.json"))
	if err := g.Alta(models.Entrada{ID: "usable", Ruta: filepath.Join(dirModels, "usable", "m.gguf")}); err != nil {
		t.Fatal(err)
	}
	if err := g.Alta(models.Entrada{ID: "roto", Ruta: filepath.Join(dirModels, "roto", "no.gguf")}); err != nil {
		t.Fatal(err)
	}
	return g
}

func TestModelosListaSoloElCatalogoUsable(t *testing.T) {
	db, limpiar, _ := store.CrearDBTemporal()
	defer limpiar()
	cwd := t.TempDir()
	projectID, _ := store.ResolucionProyecto(db, cwd)

	o := Nuevo(db, projectID, cwd, agentesDePrueba(t), openai.NuevoCliente("http://127.0.0.1:1"), "", 0, nil, catalogoDePrueba(t))
	modelos, err := o.Modelos()
	if err != nil {
		t.Fatalf("modelos: %v", err)
	}
	if len(modelos) != 1 || modelos[0].Nombre != "usable" {
		t.Fatalf("esperaba solo el modelo usable: %+v", modelos)
	}
}

func TestElegirModeloAusenteAvisa(t *testing.T) {
	db, limpiar, _ := store.CrearDBTemporal()
	defer limpiar()
	cwd := t.TempDir()
	projectID, _ := store.ResolucionProyecto(db, cwd)

	runtime, err := llama.Nuevo("", "http://127.0.0.1:19999", 4096)
	if err != nil {
		t.Fatal(err)
	}
	o := Nuevo(db, projectID, cwd, agentesDePrueba(t), openai.NuevoCliente("http://127.0.0.1:1"), "", 0, runtime, catalogoDePrueba(t))
	canal, cerrar := o.Suscribir()
	defer cerrar()

	o.FijarModelo("no-existe")
	if !tieneAviso(drenar(canal)) {
		t.Fatal("elegir un modelo fuera del catálogo debería avisar")
	}
}

func TestSesionNaceConNombreProvisional(t *testing.T) {
	db, limpiar, _ := store.CrearDBTemporal()
	defer limpiar()
	cwd := t.TempDir()
	projectID, _ := store.ResolucionProyecto(db, cwd)

	o := Nuevo(db, projectID, cwd, agentesDePrueba(t), openai.NuevoCliente("http://127.0.0.1:1"), "", 0, nil, nil)
	s, err := o.Crear()
	if err != nil {
		t.Fatal(err)
	}
	if s.Nombre != "Nueva sesión" {
		t.Fatalf("nombre provisional = %q", s.Nombre)
	}
}

// TestReenviarIncluyeElHistorial: retomar una sesión no pierde el hilo, el
// segundo turno lleva la conversación anterior.
func TestReenviarIncluyeElHistorial(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("LOCALCLI_CONTEXT_LIMIT", "100000")
	cap := &capturador{}
	srv := servidorCaptura(t, "respuesta", cap)
	db, limpiar, _ := store.CrearDBTemporal()
	defer limpiar()
	cwd := t.TempDir()
	projectID, _ := store.ResolucionProyecto(db, cwd)

	o := Nuevo(db, projectID, cwd, agentesDePrueba(t), openai.NuevoCliente(srv.URL), "m", 100000, nil, nil)
	sesion, _ := o.Crear()
	if err := o.Enviar(context.Background(), sesion.ID, "plan", "primero"); err != nil {
		t.Fatalf("primer turno: %v", err)
	}
	if err := o.Enviar(context.Background(), sesion.ID, "plan", "segundo"); err != nil {
		t.Fatalf("segundo turno: %v", err)
	}

	var peticion struct {
		Mensajes []struct {
			Rol       string `json:"role"`
			Contenido string `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal([]byte(cap.ultimo()), &peticion); err != nil {
		t.Fatalf("no se pudo leer la petición: %v", err)
	}
	var vistos []string
	for _, m := range peticion.Mensajes {
		vistos = append(vistos, m.Rol+":"+m.Contenido)
	}
	esperado := []string{"system:eres plan", "user:primero", "assistant:respuesta", "user:segundo"}
	if strings.Join(vistos, "|") != strings.Join(esperado, "|") {
		t.Fatalf("historial = %v, quiero %v", vistos, esperado)
	}
}

// TestCambiarDeSesionNoInterrumpeElTurno: una sesión sigue trabajando mientras
// el usuario está en otra.
func TestCambiarDeSesionNoInterrumpeElTurno(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		_, _ = io.ReadAll(r.Body)
		escribirSSE(w, "listo")
	}))
	t.Cleanup(srv.Close)
	db, limpiar, _ := store.CrearDBTemporal()
	defer limpiar()
	cwd := t.TempDir()
	projectID, _ := store.ResolucionProyecto(db, cwd)

	o := Nuevo(db, projectID, cwd, agentesDePrueba(t), openai.NuevoCliente(srv.URL), "m", 0, nil, nil)
	trabajando, _ := o.Crear()
	otra, _ := o.Crear()

	listo := make(chan error, 1)
	go func() { listo <- o.Enviar(context.Background(), trabajando.ID, "plan", "saluda") }()

	// «Cambiar de sesión»: listar y consultar la otra mientras la primera trabaja.
	if _, err := o.Listar(); err != nil {
		t.Fatalf("listar: %v", err)
	}
	if otra.ID == "" {
		t.Fatal("la otra sesión debería existir")
	}

	select {
	case err := <-listo:
		if err != nil {
			t.Fatalf("el turno en segundo plano falló: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("el turno en segundo plano no terminó")
	}
}

// TestElRazonamientoSeEmiteYLaRespuestaSeGuarda: el camino real del turno emite
// el razonamiento con su marca y guarda la respuesta final.
func TestElRazonamientoSeEmiteYLaRespuestaSeGuarda(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cuerpo, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		emite := func(v any) {
			b, _ := json.Marshal(v)
			fmt.Fprintf(w, "data: %s\n\n", b)
			if flusher != nil {
				flusher.Flush()
			}
		}
		if bytes.Contains(cuerpo, []byte(marcaTitulo)) {
			emite(map[string]any{"choices": []any{map[string]any{"delta": map[string]any{"content": "Titulo"}}}})
		} else {
			emite(map[string]any{"choices": []any{map[string]any{"delta": map[string]any{"reasoning_content": "pienso"}}}})
			emite(map[string]any{"choices": []any{map[string]any{"delta": map[string]any{"content": "respuesta"}}}})
		}
		emite(map[string]any{"choices": []any{}, "usage": map[string]any{"prompt_tokens": 1, "completion_tokens": 1}})
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(srv.Close)
	db, limpiar, _ := store.CrearDBTemporal()
	defer limpiar()
	cwd := t.TempDir()
	projectID, _ := store.ResolucionProyecto(db, cwd)

	o := Nuevo(db, projectID, cwd, agentesDePrueba(t), openai.NuevoCliente(srv.URL), "m", 100000, nil, nil)
	o.FijarPensar(true)
	canal, cerrar := o.Suscribir()
	defer cerrar()
	sesion, _ := o.Crear()

	if err := o.Enviar(context.Background(), sesion.ID, "plan", "hola"); err != nil {
		t.Fatalf("enviar: %v", err)
	}

	var huboRazonamiento, huboTexto bool
	for _, e := range drenar(canal) {
		if e.Tipo != tui.EventoToken {
			continue
		}
		if e.Datos["razonamiento"] == "true" {
			huboRazonamiento = true
		} else {
			huboTexto = true
		}
	}
	if !huboRazonamiento {
		t.Fatal("el razonamiento debería emitirse con su marca")
	}
	if !huboTexto {
		t.Fatal("la respuesta debería emitirse token a token")
	}

	mensajes, err := store.ListarMensajes(db, projectID, sesion.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(mensajes) != 2 || mensajes[1].Role != "agent" || mensajes[1].Content != "respuesta" {
		t.Fatalf("la respuesta no se guardó: %+v", mensajes)
	}
}

// TestApagarElRazonamientoLoPideAlMotor: apagado (por defecto) el turno pide al
// modelo no razonar; encendido, no lo pide.
func TestApagarElRazonamientoLoPideAlMotor(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("LOCALCLI_CONTEXT_LIMIT", "100000")
	cap := &capturador{}
	srv := servidorCaptura(t, "respuesta", cap)
	db, limpiar, _ := store.CrearDBTemporal()
	defer limpiar()
	cwd := t.TempDir()
	projectID, _ := store.ResolucionProyecto(db, cwd)

	o := Nuevo(db, projectID, cwd, agentesDePrueba(t), openai.NuevoCliente(srv.URL), "m", 100000, nil, nil)
	sesion, _ := o.Crear()

	o.FijarPensar(false)
	if err := o.Enviar(context.Background(), sesion.ID, "plan", "hola"); err != nil {
		t.Fatalf("enviar: %v", err)
	}
	if !strings.Contains(cap.ultimo(), `"enable_thinking":false`) {
		t.Fatalf("apagado debería pedir al modelo no razonar:\n%s", cap.ultimo())
	}

	o.FijarPensar(true)
	if err := o.Enviar(context.Background(), sesion.ID, "plan", "otra"); err != nil {
		t.Fatalf("enviar: %v", err)
	}
	if strings.Contains(cap.ultimo(), "enable_thinking") {
		t.Fatalf("encendido no debería desactivar el razonamiento:\n%s", cap.ultimo())
	}
}

// TestCancelarElTurnoLoDetiene: al cancelar el contexto, Enviar corta el turno y
// la sesión queda inactiva, no con error.
func TestCancelarElTurnoLoDetiene(t *testing.T) {
	db, limpiar, _ := store.CrearDBTemporal()
	defer limpiar()
	cwd := t.TempDir()
	projectID, _ := store.ResolucionProyecto(db, cwd)

	o := Nuevo(db, projectID, cwd, agentesDePrueba(t), openai.NuevoCliente("http://127.0.0.1:1"), "m", 0, nil, nil)
	sesion, _ := o.Crear()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := o.Enviar(ctx, sesion.ID, "plan", "hola"); !errors.Is(err, context.Canceled) {
		t.Fatalf("el turno cancelado debería devolver context.Canceled: %v", err)
	}
	s, err := store.ObtenerSesion(db, projectID, sesion.ID)
	if err != nil {
		t.Fatal(err)
	}
	if s.Status != "inactiva" {
		t.Fatalf("una cancelación no es un error; estado = %q", s.Status)
	}
}

// TestCancelarConservaLaRespuestaParcial: detener el turno a mitad del streaming no
// borra lo que el modelo ya entregó; lo guarda como su respuesta, con su tiempo,
// para que la sesión lo siga mostrando al reabrirse.
func TestCancelarConservaLaRespuestaParcial(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	db, limpiar, _ := store.CrearDBTemporal()
	defer limpiar()
	cwd := t.TempDir()
	projectID, _ := store.ResolucionProyecto(db, cwd)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cuerpo, _ := io.ReadAll(r.Body)
		if bytes.Contains(cuerpo, []byte(marcaTitulo)) {
			escribirSSE(w, "Titulo")
			return
		}
		// El turno entrega un fragmento y deja la conexión abierta: el test detiene
		// el turno con la cancelación del contexto y el servidor se cierra con ella.
		w.Header().Set("Content-Type", "text/event-stream")
		if flusher, ok := w.(http.Flusher); ok {
			fmt.Fprint(w, `data: {"choices":[{"delta":{"content":"parcial"}}]}`+"\n\n")
			flusher.Flush()
		}
		<-r.Context().Done()
	}))
	defer srv.Close()

	o := Nuevo(db, projectID, cwd, agentesDePrueba(t), openai.NuevoCliente(srv.URL), "m", 8192, nil, nil)
	canal, cerrar := o.Suscribir()
	defer cerrar()
	sesion, _ := o.Crear()

	ctx, cancel := context.WithCancel(context.Background())
	hecho := make(chan error, 1)
	go func() { hecho <- o.Enviar(ctx, sesion.ID, "plan", "hola") }()

	esperarToken(t, canal, "parcial")
	cancel()

	if err := <-hecho; !errors.Is(err, context.Canceled) {
		t.Fatalf("el turno detenido debería devolver context.Canceled: %v", err)
	}

	mensajes, err := store.ListarMensajes(db, projectID, sesion.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(mensajes) != 2 || mensajes[1].Role != "agent" || mensajes[1].Content != "parcial" {
		t.Fatalf("lo parcial debería quedar guardado como respuesta: %+v", mensajes)
	}
	if mensajes[1].DurationMs == nil {
		t.Fatal("la respuesta parcial debería guardar su duración")
	}
}

// esperarToken consume eventos hasta ver el fragmento esperado o agotar el plazo.
func esperarToken(t *testing.T, canal <-chan tui.Evento, texto string) {
	t.Helper()
	plazo := time.After(2 * time.Second)
	for {
		select {
		case e := <-canal:
			if e.Tipo == tui.EventoToken && e.Datos["texto"] == texto {
				return
			}
		case <-plazo:
			t.Fatalf("no llegó el fragmento %q", texto)
		}
	}
}

// TestElTituloSePideUnaSolaVez: una sesión ya titulada no vuelve a pedirlo.
func TestElTituloSePideUnaSolaVez(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("LOCALCLI_CONTEXT_LIMIT", "100000")
	var titulos int32
	srv := servidorTitulos(t, "respuesta", "Titulo", &titulos, false)
	db, limpiar, _ := store.CrearDBTemporal()
	defer limpiar()
	cwd := t.TempDir()
	projectID, _ := store.ResolucionProyecto(db, cwd)

	o := Nuevo(db, projectID, cwd, agentesDePrueba(t), openai.NuevoCliente(srv.URL), "m", 100000, nil, nil)
	sesion, _ := o.Crear()
	if err := o.Enviar(context.Background(), sesion.ID, "plan", "primero"); err != nil {
		t.Fatalf("primer turno: %v", err)
	}
	if err := o.Enviar(context.Background(), sesion.ID, "plan", "segundo"); err != nil {
		t.Fatalf("segundo turno: %v", err)
	}
	if n := atomic.LoadInt32(&titulos); n != 1 {
		t.Fatalf("el título se pidió %d veces, quiero 1", n)
	}
	s, err := store.ObtenerSesion(db, projectID, sesion.ID)
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "Titulo" {
		t.Fatalf("nombre = %q, quiero el título del modelo", s.Name)
	}
}

// TestTituloConservadoSiElModeloFalla: si el título no se puede generar, la
// sesión conserva el provisional y el turno no se cae.
func TestTituloConservadoSiElModeloFalla(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("LOCALCLI_CONTEXT_LIMIT", "100000")
	var titulos int32
	srv := servidorTitulos(t, "respuesta", "", &titulos, true)
	db, limpiar, _ := store.CrearDBTemporal()
	defer limpiar()
	cwd := t.TempDir()
	projectID, _ := store.ResolucionProyecto(db, cwd)

	o := Nuevo(db, projectID, cwd, agentesDePrueba(t), openai.NuevoCliente(srv.URL), "m", 100000, nil, nil)
	sesion, _ := o.Crear()
	if err := o.Enviar(context.Background(), sesion.ID, "plan", "primero"); err != nil {
		t.Fatalf("un fallo del título no debe tumbar el turno: %v", err)
	}
	s, err := store.ObtenerSesion(db, projectID, sesion.ID)
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "Nueva sesión" {
		t.Fatalf("nombre = %q, debería conservarse el provisional", s.Name)
	}
}
