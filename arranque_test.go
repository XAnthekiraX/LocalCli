package main

// Test de la elección del modelo en el arranque: el último usado se recuerda y
// prevalece si sigue instalado (SPEC-OLLAMA-PERFIL).

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"localcli/internal/agent"
	"localcli/internal/flow"
	"localcli/internal/ollama"
	"localcli/internal/tools"
)

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
	// La ventana del modelo va cacheada: el turno no lista modelos ni necesita
	// Ollama.
	ad.modelo = "m"
	ad.contextos = map[string]int{"m": 4096}
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
// `.localcli/agents/*.json`; un archivo roto se ignora sin tumbar el arranque, y los
// base siguen disponibles aunque falte su JSON.
func TestAgenteBaseCargaAgentesPropios(t *testing.T) {
	raiz := t.TempDir()
	dir := filepath.Join(raiz, ".localcli", "agents")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	escribir := func(nombre, contenido string) {
		if err := os.WriteFile(filepath.Join(dir, nombre), []byte(contenido), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	escribir("plan.json", `{"nombre":"plan","prompt":"soy plan","permisos":[{"accion":"leer","efecto":"permitir"}]}`)
	escribir("revisor.json", `{"nombre":"revisor","prompt":"soy revisor","permisos":[{"accion":"leer","efecto":"permitir"}]}`)
	escribir("roto.json", `{no es json`)

	agentes := agenteBase(raiz)
	if _, ok := agentes["plan"]; !ok {
		t.Error("plan debe estar")
	}
	if _, ok := agentes["revisor"]; !ok {
		t.Error("el agente propio de .localcli/agents se carga")
	}
	if _, ok := agentes["roto"]; ok {
		t.Error("un JSON de agente roto se ignora")
	}
	// build no tiene archivo: queda el de respaldo, para que los flujos
	// oficiales sigan teniendo a quién referirse.
	if _, ok := agentes["build"]; !ok {
		t.Error("build debe existir aunque falte su JSON")
	}
}
