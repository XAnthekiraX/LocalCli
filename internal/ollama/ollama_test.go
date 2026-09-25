package ollama

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// --- helpers -------------------------------------------------------------

func nuevoServidorOllama(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewClient(srv.URL)
}

func leerFixture(t *testing.T, nombre string) []byte {
	t.Helper()
	datos, err := os.ReadFile("testdata/" + nombre)
	if err != nil {
		t.Fatalf("fixture %s: %v", nombre, err)
	}
	return datos
}

// --- T-B005-01: cliente básico -------------------------------------------

func TestPostGenerateResponde(t *testing.T) {
	var recibido GenerarRequest
	c := nuevoServidorOllama(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/generate" {
			t.Errorf("ruta inesperada %s", r.URL.Path)
		}
		json.NewDecoder(r.Body).Decode(&recibido)
		w.Header().Set("Content-Type", "application/x-ndjson")
		io.WriteString(w, "{\"response\":\"ok\",\"done\":true}\n")
	})
	events, err := c.Generar(context.Background(), GenerarRequest{Model: "m", Prompt: "p"})
	if err != nil {
		t.Fatal(err)
	}
	var final *RespuestaFinal
	for ev := range events {
		if ev.Tipo == EventoDone {
			final = ev.Done
		}
		if ev.Tipo == EventoError {
			t.Fatalf("evento error: %v", ev.Error)
		}
	}
	if final == nil || final.Texto != "ok" {
		t.Fatalf("respuesta final no recibida: %+v", final)
	}
	if !recibido.Stream {
		t.Error("el cliente debe pedir stream:true siempre")
	}
}

func TestBaseURLPorDefectoYConfigurable(t *testing.T) {
	if got := NewClient("").BaseURL(); got != DirPorDefecto {
		t.Errorf("vacío → %q, quiero el default documentado", got)
	}
	if got := NewClient("http://otro:9999/").BaseURL(); got != "http://otro:9999" {
		t.Errorf("trim barra final: %q", got)
	}
}

// --- T-B005-02: streaming NDJSON fixture → tokens en orden ----------------

func TestStreamFixtureTokensEnOrden(t *testing.T) {
	fixture := leerFixture(t, "stream_simple.ndjson")
	c := nuevoServidorOllama(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write(fixture)
	})
	events, err := c.Generar(context.Background(), GenerarRequest{Model: "m", Prompt: "p"})
	if err != nil {
		t.Fatal(err)
	}
	var tokens []string
	var final *RespuestaFinal
	for ev := range events {
		switch ev.Tipo {
		case EventoToken:
			tokens = append(tokens, ev.Texto)
		case EventoDone:
			final = ev.Done
		case EventoError:
			t.Fatalf("error inesperado: %v", ev.Error)
		}
	}
	want := []string{"Hola", " mundo", "!"}
	if len(tokens) != len(want) {
		t.Fatalf("tokens=%v, quiero %v", tokens, want)
	}
	for i := range want {
		if tokens[i] != want[i] {
			t.Fatalf("token %d = %q, quiero %q", i, tokens[i], want[i])
		}
	}
	if final == nil || final.Texto != "Hola mundo!" {
		t.Fatalf("acumulación final incorrecta: %+v", final)
	}
	if final.TokensEntr != 11 || final.TokensSal != 3 {
		t.Errorf("métricas: %+v", final)
	}
}

func TestProcesarLineaPureza(t *testing.T) {
	var acc acumulador
	evs, err := procesarLinea([]byte(`{"thinking":"pienso","response":"digo","done":false}`), &acc)
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) != 2 || evs[0].Tipo != EventoRazonamiento || evs[1].Tipo != EventoToken {
		t.Fatalf("eventos mal clasificados: %+v", evs)
	}
}

// --- T-B005-03: razonamiento separado -------------------------------------

func TestStreamReasoningSeparaTextoYRazonamiento(t *testing.T) {
	fixture := leerFixture(t, "stream_reasoning.ndjson")
	c := nuevoServidorOllama(t, func(w http.ResponseWriter, r *http.Request) { w.Write(fixture) })
	events, err := c.Generar(context.Background(), GenerarRequest{Model: "r1", Prompt: "p"})
	if err != nil {
		t.Fatal(err)
	}
	var texto, razon strings.Builder
	for ev := range events {
		switch ev.Tipo {
		case EventoToken:
			texto.WriteString(ev.Texto)
		case EventoRazonamiento:
			razon.WriteString(ev.Texto)
		case EventoError:
			t.Fatal(ev.Error)
		}
	}
	if texto.String() != "La respuesta." {
		t.Errorf("texto=%q", texto.String())
	}
	if razon.String() != "Voy a pensar. " {
		t.Errorf("razonamiento=%q", razon.String())
	}
}

func TestSeparadorEnTextoMarcadoresInline(t *testing.T) {
	var s SeparadorEnTexto
	var texto, razon strings.Builder
	feed := func(tok string) {
		for _, f := range s.Push(tok) {
			if f.EsRazonamiento {
				razon.WriteString(f.Texto)
			} else {
				texto.WriteString(f.Texto)
			}
		}
	}
	// Tokens que parten el marcador por la mitad:
	feed("Resp: 42. ")
	feed("<think") // retenido: podría ser apertura
	feed(">voy<")  // ahora se resuelve
	feed("/think>sí")
	for _, f := range s.Cerrar() {
		if f.EsRazonamiento {
			razon.WriteString(f.Texto)
		} else {
			texto.WriteString(f.Texto)
		}
	}
	if texto.String() != "Resp: 42. sí" {
		t.Errorf("texto=%q", texto.String())
	}
	if razon.String() != "voy" {
		t.Errorf("razon=%q", razon.String())
	}
}

// --- T-B005-04: listado de modelos -----------------------------------------

func TestListarModelosParseaTags(t *testing.T) {
	fixture := leerFixture(t, "tags.json")
	c := nuevoServidorOllama(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Errorf("ruta %s", r.URL.Path)
		}
		w.Write(fixture)
	})
	modelos, err := c.ListarModelos(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(modelos) != 3 {
		t.Fatalf("quiero 3 modelos, tengo %d", len(modelos))
	}
	if modelos[0].Nombre != "qwen2.5:3b" || modelos[0].TamanoBytes != 1900000000 {
		t.Errorf("modelo 0: %+v", modelos[0])
	}
	if modelos[2].TamanoBytes != 0 {
		t.Errorf("sin tamaño debe quedar 0: %+v", modelos[2])
	}
}

// --- T-B005-05: perfil de hardware ------------------------------------------

func TestPerfilAvisoModeloQueNoCabe(t *testing.T) {
	h := PerfilPorDefecto() // 4 GiB VRAM / 16 GiB RAM
	grande := Modelo{Nombre: "llama3:70b", TamanoBytes: 40_000_000_000}
	aviso := h.AvisoNoCabe(grande)
	if aviso == nil {
		t.Fatal("un modelo de 40 GB debe avisar contra 4 GB de VRAM")
	}
	if !errors.Is(aviso, ErrModeloNoCabe) {
		t.Fatalf("debe ser E_MODEL_TOO_BIG, es %v", aviso)
	}
	// El aviso NO aborta: sigue siendo usable como valor.
	var e *ErrorOllama
	if !errors.As(aviso, &e) || e.Codigo != CodigoModeloNoCabe {
		t.Errorf("código: %+v", e)
	}
	peque := Modelo{Nombre: "qwen2.5:3b", TamanoBytes: 1_900_000_000}
	if h.AvisoNoCabe(peque) != nil {
		t.Error("3B cuantizado cabe en 4 GB; no debe avisar")
	}
	// Tamaño desconocido: informamos pero no decidimos (regla del spec).
	if h.AvisoNoCabe(Modelo{Nombre: "x"}) != nil {
		t.Error("tamaño desconocido no debe producir aviso duro")
	}
}

func TestClasificarModelos(t *testing.T) {
	h := PerfilPorDefecto()
	lista := []Modelo{
		{Nombre: "cabe", TamanoBytes: 1_900_000_000},
		{Nombre: "no-cabe", TamanoBytes: 40_000_000_000},
		{Nombre: "desconocido"},
	}
	caben, noCaben, descon := h.ClasificarModelos(lista)
	if len(caben) != 1 || caben[0].Nombre != "cabe" {
		t.Errorf("caben=%v", caben)
	}
	if len(noCaben) != 1 || noCaben[0].Nombre != "no-cabe" {
		t.Errorf("noCaben=%v", noCaben)
	}
	if len(descon) != 1 || descon[0].Nombre != "desconocido" {
		t.Errorf("desconocidos=%v", descon)
	}
}

func TestDetectarHardwareNoFallaSinProc(t *testing.T) {
	h := DetectarHardware()
	// En Linux real RAMTotalBytes > 0; en cualquier caso no panic y >= 0.
	if h.RAMTotalBytes < 0 {
		t.Errorf("RAM negativa: %+v", h)
	}
}

func TestContextoLimitado(t *testing.T) {
	h := PerfilPorDefecto()
	m := Modelo{TamanoBytes: 2 * GiB}
	got := h.ContextoLimitadoTokens(m, 32768)
	if got <= 0 || got > 32768 {
		t.Errorf("contexto fuera de rango: %d", got)
	}
}

// --- T-B005-06: FIFO serializa y respeta orden ------------------------------

func TestFifoDosLlamadasConcurentesDeUnaEnOrden(t *testing.T) {
	cola := NewColaInferencia()
	var mu sync.Mutex
	var concurrentes, maxConcurrentes int
	var orden []string

	lanzar := func(id string, demora time.Duration) func() error {
		return func() error {
			err := cola.Encolar(context.Background(), OpcionesEncolar{IdSesion: id}, func(ctx context.Context) error {
				mu.Lock()
				concurrentes++
				if concurrentes > maxConcurrentes {
					maxConcurrentes = concurrentes
				}
				orden = append(orden, id)
				mu.Unlock()
				time.Sleep(demora)
				mu.Lock()
				concurrentes--
				mu.Unlock()
				return nil
			})
			return err
		}
	}

	// Primera entra y tarda; segunda y tercera llegan después, en orden.
	done := make(chan error, 3)
	go func() { done <- lanzar("A", 80*time.Millisecond)() }()
	time.Sleep(20 * time.Millisecond) // A adquiere primero
	go func() { done <- lanzar("B", 10*time.Millisecond)() }()
	time.Sleep(20 * time.Millisecond) // B en espera antes de que llegue C
	go func() { done <- lanzar("C", 10*time.Millisecond)() }()
	for i := 0; i < 3; i++ {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
	if maxConcurrentes != 1 {
		t.Errorf("máximo de concurrencia dentro de la cola: %d, quiero 1", maxConcurrentes)
	}
	if len(orden) != 3 || orden[0] != "A" || orden[1] != "B" || orden[2] != "C" {
		t.Errorf("orden FIFO roto: %v", orden)
	}
}

// --- T-B005-07: estado "esperando al modelo" desde el punto único -----------

func TestSegundaPeticionReportaEsperaMientrasPrimeraGenera(t *testing.T) {
	cola := NewColaInferencia()
	liberar := make(chan struct{})
	esperandoB := make(chan bool, 4)

	go func() {
		cola.Encolar(context.Background(), OpcionesEncolar{IdSesion: "A"}, func(ctx context.Context) error {
			<-liberar
			return nil
		})
	}()
	// asegurar que A está dentro
	for !cola.Ocupada() {
		time.Sleep(time.Millisecond)
	}

	go func() {
		cola.Encolar(context.Background(), OpcionesEncolar{
			IdSesion:  "B",
			AlEsperar: func(e bool) { esperandoB <- e },
		}, func(ctx context.Context) error { return nil })
	}()

	// B reporta espera=true...
	select {
	case v := <-esperandoB:
		if !v {
			t.Fatal("B debía reportar espera=true primero")
		}
	case <-time.After(time.Second):
		t.Fatal("B no reportó espera mientras A generaba")
	}
	// ...y Esperando() (punto único consultable) lo refleja.
	if !cola.Esperando() {
		t.Error("Esperando() debe ser true con B en cola")
	}

	close(liberar)
	// B termina y reporta fin de espera.
	select {
	case v := <-esperandoB:
		if v {
			t.Fatal("segundo evento de B debía ser espera=false")
		}
	case <-time.After(time.Second):
		t.Fatal("B nunca salió de la espera")
	}
	for cola.Esperando() || cola.Ocupada() {
		time.Sleep(time.Millisecond)
	}
}

func TestFifoCancelacionEnEsperaDevuelveTestigo(t *testing.T) {
	cola := NewColaInferencia()
	liberar := make(chan struct{})
	go cola.Encolar(context.Background(), OpcionesEncolar{}, func(ctx context.Context) error {
		<-liberar
		return nil
	})
	for !cola.Ocupada() {
		time.Sleep(time.Millisecond)
	}
	ctx, cancel := context.WithCancel(context.Background())
	errCancelar := make(chan error, 1)
	go func() {
		errCancelar <- cola.Encolar(ctx, OpcionesEncolar{IdSesion: "moribunda"}, func(ctx context.Context) error {
			t.Error("fn no debe ejecutarse si el ctx murió en espera")
			return nil
		})
	}()
	time.Sleep(20 * time.Millisecond)
	cancel()
	if err := <-errCancelar; !errors.Is(err, context.Canceled) {
		t.Fatalf("quiero context.Canceled, tengo %v", err)
	}
	close(liberar)
	// La cola sigue viva: otra petición pasa con timeout razonable.
	done := make(chan error, 1)
	go func() {
		done <- cola.Encolar(context.Background(), OpcionesEncolar{}, func(ctx context.Context) error { return nil })
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("testigo perdido tras cancelación: deadlock")
	}
}

// --- T-B005-08: errores de conexión tipados ---------------------------------

func TestConnectionRefusedErrorTipado(t *testing.T) {
	// Puerto cerrado garantizado: servidor httptest ya cerrado.
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close()

	c := NewClient(url)
	_, err := c.Generar(context.Background(), GenerarRequest{Model: "m", Prompt: "p"})
	if err == nil {
		t.Fatal("quiero error contra servidor apagado")
	}
	if !errors.Is(err, ErrOllamaNoDisponible) {
		t.Fatalf("debe mapear a E_OLLAMA_UNAVAILABLE, tengo %v", err)
	}
	var e *ErrorOllama
	if !errors.As(err, &e) || e.Codigo != CodigoOllamaNoDisponible {
		t.Fatalf("código: %+v", e)
	}
	// El mensaje explica cómo actuar (spec: instrucción clara).
	if !contains(e.Mensaje, "ollama serve") {
		t.Errorf("mensaje sin instrucción: %q", e.Mensaje)
	}
	// Y Ping da el mismo aviso sin matar nada.
	if err := c.Ping(context.Background()); !errors.Is(err, ErrOllamaNoDisponible) {
		t.Errorf("Ping: %v", err)
	}
}

func contains(s, sub string) bool { return strings.Contains(s, sub) }
