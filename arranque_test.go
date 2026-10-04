package main

// Test de la elección del modelo en el arranque: el último usado se recuerda y
// prevalece si sigue instalado (SPEC-OLLAMA-PERFIL).

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"localcli/internal/agent"
	"localcli/internal/flow"
	"localcli/internal/llm"
	"localcli/internal/ollama"
	"localcli/internal/tools"
	"localcli/internal/tui"
)

// init registra un motor sin red para los dos tipos del catálogo: los tests del
// arranque construyen adaptadores sin hablar con ningún servidor.
func init() {
	for _, tipo := range []string{llm.TipoOllama, llm.TipoLlamaCPP} {
		llm.RegistrarTipo(tipo, func(string) llm.Motor { return motorDePrueba{} })
	}
}

// TestMotorRecordadoInexistenteNoFallaElArranque — T-B037-11: un `ultimo_motor`
// que ya no está registrado no deja al usuario sin motor: se autodetecta uno
// activo.
func TestMotorRecordadoInexistenteNoFallaElArranque(t *testing.T) {
	motores := &llm.Registro{}
	motores.Cargar(filepath.Join(t.TempDir(), "motores.json"))
	id, avisos := elegirMotorDeArranque(motores, "ya-no-existe")
	if id == "" {
		t.Fatalf("con motores por defecto se debe resolver un motor; avisos: %v", avisos)
	}
	if e, ok := motores.PorID(id); !ok || !e.Activo {
		t.Errorf("el motor resuelto debe estar registrado y activo: %q", id)
	}
}

// TestElParRecordadoSoloSeReusaSiSuMotorEstaRegistrado — T-B037-12: el modelo
// guardado no se reutiliza si su motor ya no es el resuelto (el par no cuadra).
func TestElParRecordadoSoloSeReusaSiSuMotorEstaRegistrado(t *testing.T) {
	prefs := tui.Preferencias{Modelo: "qwen3:8b", Motor: "ya-no-existe"}
	if got := modeloRecordado(prefs, "ollama-local"); got != "" {
		t.Errorf("un par cuyo motor no es el resuelto no se reutiliza: %q", got)
	}
}

func servidorDeTags(t *testing.T, modelos ...string) *ollama.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Errorf("ruta inesperada %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		items := make([]string, 0, len(modelos))
		for _, m := range modelos {
			items = append(items, `{"name":"`+m+`","size":0}`)
		}
		_, _ = w.Write([]byte(`{"models":[` + strings.Join(items, ",") + `]}`))
	}))
	t.Cleanup(srv.Close)
	return ollama.NewClient(srv.URL)
}

func TestElegirModeloPrefiereElUltimoUsado(t *testing.T) {
	c := servidorDeTags(t, "llama3.2", "qwen2.5")
	got, err := elegirModelo(c, "qwen2.5")
	if err != nil {
		t.Fatal(err)
	}
	if got != "qwen2.5" {
		t.Errorf("el último modelo usado prevalece si sigue instalado: %q", got)
	}
}

// TestElModeloRecordadoSoloSeReusaConSuMotor — T-B037-12: el mismo nombre no
// significa lo mismo en motores distintos, así que el modelo guardado solo se
// reutiliza si su motor registrado coincide con el elegido.
func TestElModeloRecordadoSoloSeReusaConSuMotor(t *testing.T) {
	prefs := tui.Preferencias{Modelo: "qwen3:8b", Motor: "ollama-local"}
	if got := modeloRecordado(prefs, "ollama-local"); got != "qwen3:8b" {
		t.Errorf("con su motor debe reutilizarse: %q", got)
	}
	if got := modeloRecordado(prefs, "llamacpp-local"); got != "" {
		t.Errorf("con otro motor debe autodetectarse: %q", got)
	}
}

// proveedorCaido es un doble de `llm.Motor` que no responde: reproduce un
// runtime apagado.
type proveedorCaido struct{}

func (proveedorCaido) Nombre() string  { return "caido" }
func (proveedorCaido) BaseURL() string { return "" }
func (proveedorCaido) Chat(context.Context, llm.Peticion) (<-chan llm.Evento, error) {
	return nil, errors.New("no responde")
}
func (proveedorCaido) ListarModelos(context.Context) ([]llm.Modelo, error) {
	return nil, errors.New("no responde")
}
func (proveedorCaido) Capacidades(context.Context, string) ([]string, error) {
	return nil, errors.New("no responde")
}
func (proveedorCaido) VentanaDeContexto(context.Context, string) (int, bool, error) {
	return 0, false, errors.New("no responde")
}

// motorDePrueba es un motor sin red para los tests del arranque: declara una
// ventana de modelo fija y no habla con ningún servidor.
type motorDePrueba struct{}

func (motorDePrueba) Nombre() string  { return llm.TipoOllama }
func (motorDePrueba) BaseURL() string { return "http://localhost:0" }
func (motorDePrueba) Chat(context.Context, llm.Peticion) (<-chan llm.Evento, error) {
	return nil, errors.New("no usado")
}
func (motorDePrueba) ListarModelos(context.Context) ([]llm.Modelo, error) {
	return []llm.Modelo{{Nombre: "m", ContextLength: 4096}}, nil
}
func (motorDePrueba) Capacidades(context.Context, string) ([]string, error) { return nil, nil }
func (motorDePrueba) VentanaDeContexto(context.Context, string) (int, bool, error) {
	return 0, false, nil
}

// TestUnProveedorQueNoRespondeNoImpideArrancar — un motor apagado avisa y la
// interfaz sigue viva: no responde, y elegirModelo devuelve error en vez de
// tumbar el arranque.
func TestUnProveedorQueNoRespondeNoImpideArrancar(t *testing.T) {
	if motorResponde(proveedorCaido{}) {
		t.Error("un motor caído no debe considerarse vivo")
	}
	if _, err := elegirModelo(proveedorCaido{}, ""); err == nil {
		t.Error("sin motor, elegirModelo debe devolver error para que el arranque siga sin modelo")
	}
}

func TestElegirModeloIgnoraUnPreferidoDesinstalado(t *testing.T) {
	c := servidorDeTags(t, "llama3.2")
	got, err := elegirModelo(c, "desinstalado")
	if err != nil {
		t.Fatal(err)
	}
	if got != "llama3.2" {
		t.Errorf("si el preferido no está, se autodetecta: %q", got)
	}
}

// generadorQueSoloPideHerramientas reproduce al modelo que no entiende que se
// le está pidiendo prosa: en cada pasada pide una herramienta y nunca escribe
// texto. Es el doble que hace visible el turno mudo.
type generadorQueSoloPideHerramientas struct{}

func (generadorQueSoloPideHerramientas) Generar(ctx context.Context, a agent.Agente, modelo string, mensajes []ollama.Mensaje, herramientas []ollama.Herramienta, numCtx int, pensar *bool) (<-chan ollama.Evento, error) {
	var c ollama.ToolCall
	c.Function.Name = "leer_archivo"
	c.Function.Arguments = []byte(`{"ruta":"a.md"}`)
	ch := make(chan ollama.Evento, 1)
	ch <- ollama.Evento{Tipo: ollama.EventoDone, Done: &ollama.RespuestaFinal{Done: true, ToolCalls: []ollama.ToolCall{c}}}
	close(ch)
	return ch, nil
}

// despachadorDePrueba enruta las herramientas del catálogo con un handler fijo:
// el turno ejecuta lo que pide el doble sin tocar disco ni terminal.
func despachadorDePrueba(t *testing.T) *agent.Despachador {
	t.Helper()
	var impls []tools.Herramienta
	for _, nombre := range tools.NombresCatalogo() {
		h, ok := tools.NuevaHerramienta(nombre, func(ctx context.Context, args any, c tools.Contexto) (tools.Resultado, error) {
			return tools.Resultado{Salida: "ok"}, nil
		})
		if !ok {
			t.Fatalf("herramienta %q fuera del catálogo", nombre)
		}
		impls = append(impls, h)
	}
	r, err := tools.NuevoRegistro(impls)
	if err != nil {
		t.Fatal(err)
	}
	return agent.NuevoDespachador(r)
}

// ejecutorDeTurnoMudo deja el ejecutor de turno del arranque con un modelo que
// solo pide herramientas, sobre una base y una sesión reales.
func ejecutorDeTurnoMudo(t *testing.T) (*ejecutorPorTurno, *Adaptador, string) {
	t.Helper()
	ad, sesion := adaptadorConSesion(t)
	// Un registro con un motor de prueba: la ventana se calcula sin red.
	motores := &llm.Registro{}
	motores.Cargar(filepath.Join(t.TempDir(), "motores.json"))
	ad.registro = motores
	ad.motorDefectoID = "ollama-local"
	ad.colas = llm.NewColasInferencia()
	ad.modelo = "m"
	e := &ejecutorPorTurno{
		ad: ad,
		ejecutor: &agent.Ejecutor{
			Runner:     generadorQueSoloPideHerramientas{},
			Despachar:  despachadorDePrueba(t),
			MaxPasadas: 1,
		},
		agente: map[string]agent.Agente{"plan": {Nombre: "plan"}},
	}
	return e, ad, sesion
}

// TestElChatNoGuardaRespuestaVacia — un turno de chat que no entrega texto no se
// persiste con el marcador «(respuesta vacía)»: el error se propaga para que la
// sesión quede en `error` y el motivo se vea (SPEC-AGENTE-BASE §El ciclo de un turno).
func TestElChatNoGuardaRespuestaVacia(t *testing.T) {
	e, ad, sesion := ejecutorDeTurnoMudo(t)
	ctx := tools.ConSesion(context.Background(), sesion)

	if _, err := e.Ejecutar(ctx, flow.PeticionEtapa{Agente: "plan", Contexto: "hola"}); !errors.Is(err, agent.ErrSinRespuesta) {
		t.Fatalf("err = %v, quiero E_NO_RESPONSE", err)
	}
	hilo, err := ad.chat.Hilo(sesion)
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range hilo {
		if strings.Contains(l.Content, "(respuesta vacía)") {
			t.Errorf("el chat no guarda un turno mudo como respuesta: %+v", l)
		}
	}
}

// agenteContador cuenta las veces que el motor pide una etapa.
type agenteContador struct {
	inner    flow.Agente
	llamadas int
}

func (a *agenteContador) Ejecutar(ctx context.Context, p flow.PeticionEtapa) (flow.Resultado, error) {
	a.llamadas++
	return a.inner.Ejecutar(ctx, p)
}

// contextoFijo es el contexto de una etapa: lo que el motor le entrega al agente.
type contextoFijo struct{}

func (contextoFijo) ContextoPara(ctx context.Context, etapa, objetivo string) (string, error) {
	return "contexto de " + etapa, nil
}

// TestUnaEtapaMudaSeReintentaYfallaComoAntes — la semántica de una etapa de flujo
// no cambia: el turno sin respuesta vuelve como texto vacío, el motor la reintenta
// una vez y, si sigue muda, el flujo se detiene con E_STAGE_FAILED.
func TestUnaEtapaMudaSeReintentaYfallaComoAntes(t *testing.T) {
	e, _, sesion := ejecutorDeTurnoMudo(t)
	contador := &agenteContador{inner: e}
	motor := &flow.Motor{Contexto: contextoFijo{}, Agente: contador}
	flujo := flow.Flujo{
		Nombre:   "prueba",
		Pregunta: "qué entrega",
		Etapas: []flow.Etapa{{
			ID: "e1", Nombre: "Entender", Agente: "plan", Pregunta: "¿qué pasa?", Entrega: true,
		}},
	}
	ctx := tools.ConSesion(context.Background(), sesion)

	_, err := motor.EjecutarFlujo(ctx, flujo, "objetivo")
	if !errors.Is(err, flow.ErrEtapaFallida) {
		t.Fatalf("err = %v, quiero E_STAGE_FAILED (la etapa se dio por fallida tras el reintento)", err)
	}
	if contador.llamadas != 2 {
		t.Errorf("llamadas = %d, quiero 2: la etapa muda se reintenta una sola vez", contador.llamadas)
	}
}

// agenteBase carga los agentes base y cualquier agente propio de
// `.localcli/agents/`; cada SUBcarpeta es un agente. Una carpeta rota se ignora
// sin tumbar el arranque, y los base siguen disponibles con un respaldo.
func TestAgenteBaseCargaCarpetas(t *testing.T) {
	raiz := t.TempDir()
	dir := filepath.Join(raiz, ".localcli", "agents")
	escribirAgente := func(nombre, yaml, prompt string) {
		sub := filepath.Join(dir, nombre)
		if err := os.MkdirAll(sub, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(sub, "agent.yaml"), []byte(yaml), 0o644); err != nil {
			t.Fatal(err)
		}
		if prompt != "" {
			if err := os.WriteFile(filepath.Join(sub, "prompt.md"), []byte(prompt), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	escribirAgente("plan", "name: plan\ndescription: d\npermissions:\n  read: allow\n", "soy plan")
	escribirAgente("revisor", "name: revisor\ndescription: d\npermissions:\n  read: allow\n", "soy revisor")
	escribirAgente("roto", "name: roto\ndescription: \"sin cierre\n", "soy roto")

	agentes := agenteBase(raiz)
	if _, ok := agentes["plan"]; !ok {
		t.Error("plan debe estar")
	}
	if _, ok := agentes["revisor"]; !ok {
		t.Error("el agente propio de .localcli/agents se carga")
	}
	if _, ok := agentes["roto"]; ok {
		t.Error("una carpeta de agente rota se ignora")
	}
	// build no tiene carpeta: queda el de respaldo, para que los flujos
	// oficiales sigan teniendo a quién referirse.
	if _, ok := agentes["build"]; !ok {
		t.Error("build debe existir aunque falte su carpeta")
	}
}

// TestModeloActualIncluyeElProveedor — T-F043-03: la línea de modelo recibe el
// nombre del modelo con su proveedor, para que se sepa contra qué runtime se
// habla (SPEC-MODELO-PROVEEDOR). La vista no añade nada: pinta lo que el puerto
// le da.
func TestModeloActualIncluyeElProveedor(t *testing.T) {
	motores := &llm.Registro{}
	motores.Cargar(filepath.Join(t.TempDir(), "motores.json"))
	conMotor := &Adaptador{registro: motores, motorDefectoID: "ollama-local", modelo: "qwen3:8b"}
	if got := conMotor.ModeloActual(); got != "qwen3:8b (Ollama local)" {
		t.Errorf("motor = %q, quiero qwen3:8b (Ollama local)", got)
	}
	// Sin modelo no hay nada que rotular.
	sinModelo := &Adaptador{registro: motores, motorDefectoID: "ollama-local"}
	if got := sinModelo.ModeloActual(); got != "" {
		t.Errorf("sin modelo = %q, quiero vacío", got)
	}
}

// TestUnJsonAntiguoSeAvisaNombrandolo — un `*.json` suelto en
// `.localcli/agents/` es el formato antiguo: produce un aviso que nombra el
// archivo y no se carga, para que el agente no desaparezca en silencio.
func TestUnJsonAntiguoSeAvisaNombrandolo(t *testing.T) {
	raiz := t.TempDir()
	dir := filepath.Join(raiz, ".localcli", "agents")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "viejo.json"), []byte(`{"nombre":"viejo"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	original := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	agentes := agenteBase(raiz)
	_ = w.Close()
	os.Stderr = original
	salida, _ := io.ReadAll(r)
	_ = r.Close()

	if _, ok := agentes["viejo"]; ok {
		t.Error("el formato antiguo no se carga")
	}
	if !strings.Contains(string(salida), "viejo.json") {
		t.Errorf("el aviso debe nombrar el archivo: %q", salida)
	}
}
