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

// En /api/chat el razonamiento viene ANIDADO en message.thinking (el harness
// usa ese endpoint). Sin mirarlo, el razonamiento se perdía y la vista decía
// que el modelo no lo entregó.
func TestProcesarLineaRazonamientoAnidadoDeChat(t *testing.T) {
	var acc acumulador
	linea := []byte(`{"message":{"role":"assistant","content":"digo","thinking":"pienso"},"done":false}`)
	evs, err := procesarLinea(linea, &acc)
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) != 2 || evs[0].Tipo != EventoRazonamiento || evs[0].Texto != "pienso" {
		t.Fatalf("razonamiento anidado mal clasificado: %+v", evs)
	}
	if evs[1].Tipo != EventoToken || evs[1].Texto != "digo" {
		t.Fatalf("texto anidado mal clasificado: %+v", evs)
	}
	if acc.razonamiento == nil || string(acc.razonamiento) != "pienso" {
		t.Errorf("razonamiento no acumulado: %q", acc.razonamiento)
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

// El mismo camino que usa el harness: /api/chat, con el razonamiento anidado en
// message.thinking. Fija que el endpoint real separa texto y razonamiento.
func TestStreamChatSeparaTextoYRazonamiento(t *testing.T) {
	fixture := leerFixture(t, "stream_chat.ndjson")
	c := nuevoServidorOllama(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Errorf("ruta inesperada %s", r.URL.Path)
		}
		w.Write(fixture)
	})
	events, err := c.Chat(context.Background(), GenerarRequest{
		Model:    "qwen3",
		Messages: []Mensaje{{Role: "user", Content: "p"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var texto, razon strings.Builder
	var final *RespuestaFinal
	for ev := range events {
		switch ev.Tipo {
		case EventoToken:
			texto.WriteString(ev.Texto)
		case EventoRazonamiento:
			razon.WriteString(ev.Texto)
		case EventoDone:
			final = ev.Done
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
	if final == nil || final.Razonamien != "Voy a pensar. " {
		t.Errorf("razonamiento final incorrecto: %+v", final)
	}
}

// Las imágenes de un turno multimodal viajan en `message.images` de /api/chat,
// que es el campo que espera Ollama para los modelos con visión.
func TestChatEnviaImagenes(t *testing.T) {
	var recibido GenerarRequest
	c := nuevoServidorOllama(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Errorf("ruta inesperada %s", r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&recibido)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{\"model\":\"llava\",\"done\":true}\n"))
	})
	imgs := []string{"aG9sYQ==", "bXVuZG8="}
	events, err := c.Chat(context.Background(), GenerarRequest{
		Model:    "llava",
		Messages: []Mensaje{{Role: "user", Content: "¿qué hay aquí?", Images: imgs}},
	})
	if err != nil {
		t.Fatal(err)
	}
	for range events {
	}
	if len(recibido.Messages) != 1 {
		t.Fatalf("mensajes = %d, quiero 1", len(recibido.Messages))
	}
	got := recibido.Messages[0].Images
	if len(got) != len(imgs) || got[0] != imgs[0] || got[1] != imgs[1] {
		t.Errorf("images = %v, quiero %v", got, imgs)
	}
}

// Sin imágenes el campo no debe viajar: `images` es `omitempty` para no mandar
// un `null` que algunos modelos interpretan como «hay imágenes vacías».
func TestChatOmiteImagenesSinAdjuntos(t *testing.T) {
	var crudo []byte
	c := nuevoServidorOllama(t, func(w http.ResponseWriter, r *http.Request) {
		crudo, _ = io.ReadAll(r.Body)
		_, _ = w.Write([]byte("{\"model\":\"m\",\"done\":true}\n"))
	})
	events, err := c.Chat(context.Background(), GenerarRequest{
		Model:    "m",
		Messages: []Mensaje{{Role: "user", Content: "hola"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	for range events {
	}
	if strings.Contains(string(crudo), "images") {
		t.Errorf("sin imágenes no debe viajar el campo images: %s", crudo)
	}
}

// --- T-B005-03: separador de razonamiento -------------------------------

// Un marcador vacío hace que strings.Index devuelva siempre 0 y que Push
// entre en ciclo infinito: este test fija la constante para que el fallo
// reaparezca como fallo de test y no como cuelgue de `go test ./...`.
func TestMarcadoresRazonamientoNoVacias(t *testing.T) {
	for nombre, m := range map[string]string{"apertura": apertura, "cierre": cierre} {
		if m == "" {
			t.Fatalf("%s está vacía: SeparadorEnTexto.Push no terminaría", nombre)
		}
	}
	if !strings.HasPrefix(apertura, "<") || !strings.HasPrefix(cierre, "<") {
		t.Errorf("marcadores inesperados: %q / %q", apertura, cierre)
	}
	if len(cierre) <= len(apertura) {
		t.Errorf("el cierre (%q) debe ser más largo que la apertura (%q)", cierre, apertura)
	}
	// Sin bloques anidados, el cierre no puede empezar antes que la apertura.
	if strings.Contains(cierre, apertura) {
		t.Errorf("el cierre %q contiene la apertura %q: anidaría bloques", cierre, apertura)
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

// TestAcumulaToolCalls — T-B024-11: las peticiones de herramienta llegan en el
// mensaje del asistente y se acumulan durante el streaming, sin esperar a ellas
// para los tokens de texto.
func TestAcumulaToolCalls(t *testing.T) {
	var acc acumulador
	primera := []byte(`{"message":{"role":"assistant","content":"miro","tool_calls":[{"function":{"name":"leer_archivo","arguments":{"ruta":"a.md"}}}]},"done":false}`)
	evs, err := procesarLinea(primera, &acc)
	if err != nil {
		t.Fatal(err)
	}
	// El texto se emite ya; las herramientas viajan con la señal de fin.
	var texto bool
	for _, e := range evs {
		if e.Tipo == EventoToken && e.Texto == "miro" {
			texto = true
		}
	}
	if !texto {
		t.Fatalf("el texto debe emitirse sin esperar a las herramientas: %+v", evs)
	}

	segunda := []byte(`{"message":{"role":"assistant","tool_calls":[{"function":{"name":"listar_carpeta","arguments":{"ruta":"."}}}]},"done":true,"eval_count":4}`)
	evs, err = procesarLinea(segunda, &acc)
	if err != nil {
		t.Fatal(err)
	}
	var final *RespuestaFinal
	for _, e := range evs {
		if e.Tipo == EventoDone {
			final = e.Done
		}
	}
	if final == nil || len(final.ToolCalls) != 2 {
		t.Fatalf("tool_calls = %+v, quiero las dos acumuladas", final)
	}
	if final.ToolCalls[0].Nombre() != "leer_archivo" || final.ToolCalls[1].Nombre() != "listar_carpeta" {
		t.Errorf("orden de tool_calls = %+v", final.ToolCalls)
	}
	// Los argumentos se normalizan a JSON crudo, tanto si vienen como objeto
	// como si vienen como cadena.
	if got := string(final.ToolCalls[0].ArgumentosJSON()); got != `{"ruta":"a.md"}` {
		t.Errorf("argumentos = %s", got)
	}

	var cadena acumulador
	_, err = procesarLinea([]byte(`{"message":{"tool_calls":[{"function":{"name":"x","arguments":"{\"a\":1}"}}]},"done":true}`), &cadena)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(cadena.final.ToolCalls[0].ArgumentosJSON()); got != `{"a":1}` {
		t.Errorf("argumentos en cadena = %s", got)
	}
}

// TestChatSerializaElCanalDeHerramientas — T-B024-10: la petición lleva un
// objeto por herramienta en el campo `tools`.
func TestChatSerializaElCanalDeHerramientas(t *testing.T) {
	var crudo []byte
	c := nuevoServidorOllama(t, func(w http.ResponseWriter, r *http.Request) {
		crudo, _ = io.ReadAll(r.Body)
		_, _ = w.Write([]byte("{\"model\":\"m\",\"done\":true}\n"))
	})
	events, err := c.Chat(context.Background(), GenerarRequest{
		Model:    "m",
		Messages: []Mensaje{{Role: "user", Content: "hola"}},
		Tools: []Herramienta{{Type: "function", Function: Definicion{
			Name:        "leer_archivo",
			Description: "Lee un archivo.",
			Parameters:  map[string]any{"type": "object"},
		}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	for range events {
	}
	var cuerpo struct {
		Tools []struct {
			Type     string `json:"type"`
			Function struct {
				Name       string `json:"name"`
				Parameters any    `json:"parameters"`
			} `json:"function"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(crudo, &cuerpo); err != nil {
		t.Fatalf("la petición no es JSON: %v (%s)", err, crudo)
	}
	if len(cuerpo.Tools) != 1 {
		t.Fatalf("tools = %d, quiero un objeto por herramienta: %s", len(cuerpo.Tools), crudo)
	}
	if cuerpo.Tools[0].Type != "function" || cuerpo.Tools[0].Function.Name != "leer_archivo" {
		t.Errorf("definición mal serializada: %+v", cuerpo.Tools[0])
	}
}

// Sin herramientas el campo no viaja: no se manda un `null` que el modelo
// pueda interpretar como un catálogo vacío.
func TestChatOmiteToolsSinHerramientas(t *testing.T) {
	var crudo []byte
	c := nuevoServidorOllama(t, func(w http.ResponseWriter, r *http.Request) {
		crudo, _ = io.ReadAll(r.Body)
		_, _ = w.Write([]byte("{\"model\":\"m\",\"done\":true}\n"))
	})
	events, err := c.Chat(context.Background(), GenerarRequest{Model: "m", Messages: []Mensaje{{Role: "user", Content: "hola"}}})
	if err != nil {
		t.Fatal(err)
	}
	for range events {
	}
	if strings.Contains(string(crudo), "\"tools\"") {
		t.Errorf("sin herramientas no debe viajar el campo tools: %s", crudo)
	}
}

// --- Ventana de contexto (num_ctx) -----------------------------------------

// TestVentanaDeModelo — la ventana es el menor entre lo que declara el modelo y
// el tope; sin dato del modelo, manda el tope.
func TestVentanaDeModelo(t *testing.T) {
	if got := VentanaDeModelo(Modelo{ContextLength: 262144}, TopeVentanaPorDefecto); got != TopeVentanaPorDefecto {
		t.Errorf("modelo grande → %d, quiero el tope %d", got, TopeVentanaPorDefecto)
	}
	if got := VentanaDeModelo(Modelo{ContextLength: 8192}, TopeVentanaPorDefecto); got != 8192 {
		t.Errorf("modelo pequeño → %d, quiero 8192", got)
	}
	if got := VentanaDeModelo(Modelo{}, TopeVentanaPorDefecto); got != TopeVentanaPorDefecto {
		t.Errorf("sin dato → %d, quiero el tope", got)
	}
	if got := VentanaDeModelo(Modelo{ContextLength: 262144}, 4096); got != 4096 {
		t.Errorf("tope explícito → %d, quiero 4096", got)
	}
}

// TestChatLlevaNumCtx — T: la ventana viaja en `options.num_ctx` de la petición,
// no como campo suelto en la raíz. Sin ella Ollama corta los turnos con
// herramientas.
func TestChatLlevaNumCtx(t *testing.T) {
	var crudo []byte
	c := nuevoServidorOllama(t, func(w http.ResponseWriter, r *http.Request) {
		crudo, _ = io.ReadAll(r.Body)
		_, _ = w.Write([]byte("{\"model\":\"m\",\"done\":true}\n"))
	})
	events, err := c.Chat(context.Background(), GenerarRequest{
		Model:    "m",
		Messages: []Mensaje{{Role: "user", Content: "hola"}},
		NumCtx:   16384,
	})
	if err != nil {
		t.Fatal(err)
	}
	for range events {
	}
	var cuerpo struct {
		Options map[string]any `json:"options"`
		Suelto  any            `json:"num_ctx"`
	}
	if err := json.Unmarshal(crudo, &cuerpo); err != nil {
		t.Fatalf("petición no JSON: %v (%s)", err, crudo)
	}
	if got, _ := cuerpo.Options["num_ctx"].(float64); int(got) != 16384 {
		t.Errorf("options.num_ctx = %v, quiero 16384: %s", cuerpo.Options["num_ctx"], crudo)
	}
	if cuerpo.Suelto != nil {
		t.Errorf("num_ctx no debe viajar suelto en la raíz: %s", crudo)
	}
}

// TestChatOmiteNumCtxSinVentana — sin ventana no se inventa un `options`: se
// deja que el servidor use su valor (comportamiento anterior intacto).
func TestChatOmiteNumCtxSinVentana(t *testing.T) {
	var crudo []byte
	c := nuevoServidorOllama(t, func(w http.ResponseWriter, r *http.Request) {
		crudo, _ = io.ReadAll(r.Body)
		_, _ = w.Write([]byte("{\"model\":\"m\",\"done\":true}\n"))
	})
	events, err := c.Chat(context.Background(), GenerarRequest{Model: "m", Messages: []Mensaje{{Role: "user", Content: "hola"}}})
	if err != nil {
		t.Fatal(err)
	}
	for range events {
	}
	if strings.Contains(string(crudo), "num_ctx") || strings.Contains(string(crudo), "\"options\"") {
		t.Errorf("sin ventana no debe viajar options/num_ctx: %s", crudo)
	}
}

// TestElErrorIncluyeElMotivoDeOllama — un 500 trae el motivo real en el cuerpo
// (`{"error": …}`); antes se descartaba y la pantalla mostraba un «(500)» pelado.
func TestElErrorIncluyeElMotivoDeOllama(t *testing.T) {
	c := nuevoServidorOllama(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, `{"error":"no user query found in messages"}`)
	})
	_, err := c.Chat(context.Background(), GenerarRequest{Model: "m", Messages: []Mensaje{{Role: "user", Content: "hola"}}})
	if err == nil {
		t.Fatal("quiero error del 500")
	}
	if !strings.Contains(err.Error(), "no user query found in messages") {
		t.Errorf("el error debe incluir el motivo de Ollama: %v", err)
	}
	var e *ErrorOllama
	if !errors.As(err, &e) || e.Codigo != CodigoOllamaNoDisponible {
		t.Errorf("código: %+v", e)
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
	if modelos[0].ContextLength != 32768 {
		t.Errorf("context_length no parseado: %+v", modelos[0])
	}
	if modelos[2].TamanoBytes != 0 || modelos[2].ContextLength != 0 {
		t.Errorf("sin tamaño/contexto deben quedar 0: %+v", modelos[2])
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
