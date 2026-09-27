package agent

import (
	"context"
	"errors"
	"strings"
	"testing"

	"localcli/internal/ollama"
	"localcli/internal/tools"
)

// generadorGuion es un doble de `Generador`: en cada llamada emite el siguiente
// guion de eventos. Evita Ollama y hace determinista el bucle.
type generadorGuion struct {
	pasadas  [][]ollama.Evento
	llamadas int
}

func (g *generadorGuion) Generar(ctx context.Context, a Agente, modelo string, mensajes []ollama.Mensaje) (<-chan ollama.Evento, error) {
	var evs []ollama.Evento
	if g.llamadas < len(g.pasadas) {
		evs = g.pasadas[g.llamadas]
	}
	g.llamadas++
	ch := make(chan ollama.Evento, len(evs))
	for _, e := range evs {
		ch <- e
	}
	close(ch)
	return ch, nil
}

// sinkGrabador recoge lo que el bucle emite al consumidor.
type sinkGrabador struct{ partes []string }

func (s *sinkGrabador) Token(texto string, esRazon bool) { s.partes = append(s.partes, texto) }

// registroStub arma un registro de `tools` con handlers que devuelven un
// valor fijo, capturando la última petición.
func registroStub(capturado *tools.Peticion) *tools.Registro {
	handler := func(ctx context.Context, p tools.Peticion) (any, error) {
		if capturado != nil {
			*capturado = p
		}
		return map[string]string{"ok": "resultado"}, nil
	}
	r, err := tools.NuevoRegistro(tools.Destinos{Archivos: handler, Terminal: handler, Internet: handler})
	if err != nil {
		panic(err)
	}
	return r
}

// bloqueHerramienta arma el texto que el modelo escribiría para pedir una
// herramienta.
func bloqueHerramienta(nombre, args string) string {
	return "```herramienta\n" + nombre + "\n" + args + "\n```"
}

// TestElBucleRespondeSinHerramientasEnUnaPasada — si el modelo no pide nada,
// el bucle cierra en la primera pasada con el texto completo.
func TestElBucleRespondeSinHerramientasEnUnaPasada(t *testing.T) {
	g := &generadorGuion{pasadas: [][]ollama.Evento{{
		{Tipo: ollama.EventoToken, Texto: "hola"},
		{Tipo: ollama.EventoToken, Texto: " mundo"},
	}}}
	sink := &sinkGrabador{}
	e := &Ejecutor{Runner: g}
	res, err := e.Ejecutar(context.Background(), Agente{Nombre: "plan"}, "m", "ctx", sink)
	if err != nil {
		t.Fatalf("Ejecutar: %v", err)
	}
	if res.Texto != "hola mundo" {
		t.Errorf("texto = %q, quiero %q", res.Texto, "hola mundo")
	}
	if g.llamadas != 1 {
		t.Errorf("llamadas al modelo = %d, quiero 1", g.llamadas)
	}
	if strings.Join(sink.partes, "") != "hola mundo" {
		t.Errorf("el sink recibió %v", sink.partes)
	}
}

// TestElBucleEjecutaHerramientaYVuelveAlModelo — el ciclo central: el modelo
// pide una herramienta, se despacha, su resultado se reinyecta y el modelo
// vuelve a responder.
func TestElBucleEjecutaHerramientaYVuelveAlModelo(t *testing.T) {
	var capturado tools.Peticion
	d := NuevoDespachador(registroStub(&capturado))
	g := &generadorGuion{pasadas: [][]ollama.Evento{
		{{Tipo: ollama.EventoToken, Texto: bloqueHerramienta("leer_archivo", `{"ruta":"a.md"}`)}},
		{{Tipo: ollama.EventoToken, Texto: "listo"}},
	}}
	e := &Ejecutor{Runner: g, Despachar: d, MaxPasadas: 3}
	ag := Agente{Nombre: "plan", Herramientas: tools.HerramientasDePlan()}

	res, err := e.Ejecutar(context.Background(), ag, "m", "ctx", nil)
	if err != nil {
		t.Fatalf("Ejecutar: %v", err)
	}
	if res.Texto != "listo" {
		t.Errorf("texto final = %q, quiero %q", res.Texto, "listo")
	}
	if capturado.Herramienta != "leer_archivo" {
		t.Errorf("herramienta despachada = %q, quiero leer_archivo", capturado.Herramienta)
	}
	if g.llamadas != 2 {
		t.Errorf("llamadas al modelo = %d, quiero 2", g.llamadas)
	}
}

// TestElBucleUnRechazoNoCortaElTurno — un error de `tools` viaja al modelo como
// resultado y el turno continúa.
func TestElBucleUnRechazoNoCortaElTurno(t *testing.T) {
	d := NuevoDespachador(registroStub(nil))
	g := &generadorGuion{pasadas: [][]ollama.Evento{
		// `plan` no tiene escritura: la petición será E_TOOL_NOT_ALLOWED.
		{{Tipo: ollama.EventoToken, Texto: bloqueHerramienta("crear_archivo", `{"ruta":"a.md","contenido":"x"}`)}},
		{{Tipo: ollama.EventoToken, Texto: "no puedo"}},
	}}
	e := &Ejecutor{Runner: g, Despachar: d, MaxPasadas: 3}
	ag := Agente{Nombre: "plan", Herramientas: tools.HerramientasDePlan()}

	res, err := e.Ejecutar(context.Background(), ag, "m", "ctx", nil)
	if err != nil {
		t.Fatalf("un rechazo no debe cortar el turno: %v", err)
	}
	if res.Texto != "no puedo" {
		t.Errorf("texto final = %q, quiero %q", res.Texto, "no puedo")
	}
}

// TestElBucleSeDetieneEnElTopeDePasadas — si el modelo pide herramientas
// siempre, el turno se cierra al agotar las pasadas.
func TestElBucleSeDetieneEnElTopeDePasadas(t *testing.T) {
	d := NuevoDespachador(registroStub(nil))
	g := &generadorGuion{pasadas: [][]ollama.Evento{
		{{Tipo: ollama.EventoToken, Texto: bloqueHerramienta("leer_archivo", `{"ruta":"a.md"}`)}},
		{{Tipo: ollama.EventoToken, Texto: bloqueHerramienta("leer_archivo", `{"ruta":"b.md"}`)}},
		{{Tipo: ollama.EventoToken, Texto: "no debería llegar"}},
	}}
	e := &Ejecutor{Runner: g, Despachar: d, MaxPasadas: 2}
	ag := Agente{Nombre: "plan", Herramientas: tools.HerramientasDePlan()}

	if _, err := e.Ejecutar(context.Background(), ag, "m", "ctx", nil); err != nil {
		t.Fatalf("Ejecutar: %v", err)
	}
	if g.llamadas != 2 {
		t.Errorf("llamadas al modelo = %d, quiero 2 (el tope)", g.llamadas)
	}
}

// TestElBuclePropagaElErrorDelModelo — un fallo del stream se propaga tal cual.
func TestElBuclePropagaElErrorDelModelo(t *testing.T) {
	fallo := errors.New("modelo caído")
	g := &generadorGuion{pasadas: [][]ollama.Evento{{{Tipo: ollama.EventoError, Error: fallo}}}}
	e := &Ejecutor{Runner: g}
	if _, err := e.Ejecutar(context.Background(), Agente{Nombre: "plan"}, "m", "ctx", nil); !errors.Is(err, fallo) {
		t.Fatalf("err = %v, quiero %v", err, fallo)
	}
}

// TestSolicitudesExtraeBloquesValidos — el parser del contrato textual: solo
// los bloques `herramienta` con nombre del catálogo y JSON válido.
func TestSolicitudesExtraeBloquesValidos(t *testing.T) {
	texto := "antes " + bloqueHerramienta("leer_archivo", `{"ruta":"a.md"}`) +
		" " + bloqueHerramienta("inventada", `{}`) +
		" " + bloqueHerramienta("listar_carpeta", "{roto")
	got := Solicitudes(texto)
	if len(got) != 1 {
		t.Fatalf("solicitudes = %d, quiero 1: %+v", len(got), got)
	}
	if got[0].Nombre != "leer_archivo" {
		t.Errorf("nombre = %q, quiero leer_archivo", got[0].Nombre)
	}
}
