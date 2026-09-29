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
	pasadas             [][]ollama.Evento
	llamadas            int
	ultimos             []ollama.Mensaje
	ultimasHerramientas []ollama.Herramienta
	ultimoNumCtx        int
	// ultimoPensar es el `think` de la última pasada: nil quiere decir que el
	// campo no viajó.
	ultimoPensar *bool
}

func (g *generadorGuion) Generar(ctx context.Context, a Agente, modelo string, mensajes []ollama.Mensaje, herramientas []ollama.Herramienta, numCtx int, pensar *bool) (<-chan ollama.Evento, error) {
	g.ultimos = append([]ollama.Mensaje(nil), mensajes...)
	g.ultimasHerramientas = append([]ollama.Herramienta(nil), herramientas...)
	g.ultimoNumCtx = numCtx
	g.ultimoPensar = pensar
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

// sinkGrabador recoge lo que el bucle emite al consumidor, incluidos los avisos.
type sinkGrabador struct {
	partes []string
	avisos []string
}

func (s *sinkGrabador) Token(texto string, esRazon bool) { s.partes = append(s.partes, texto) }
func (s *sinkGrabador) Aviso(texto string)               { s.avisos = append(s.avisos, texto) }

// --- constructores de guiones ------------------------------------------------

// respuesta es una pasada que termina con texto y sin peticiones.
func respuesta(texto string) []ollama.Evento {
	var evs []ollama.Evento
	if texto != "" {
		evs = append(evs, ollama.Evento{Tipo: ollama.EventoToken, Texto: texto})
	}
	evs = append(evs, ollama.Evento{
		Tipo: ollama.EventoDone,
		Done: &ollama.RespuestaFinal{Texto: texto, Done: true},
	})
	return evs
}

// pedido es una pasada que termina pidiendo herramientas, con su texto.
func pedido(texto string, calls ...ollama.ToolCall) []ollama.Evento {
	evs := respuesta(texto)
	evs[len(evs)-1].Done.ToolCalls = calls
	return evs
}

// llamada arma una petición de herramienta con sus argumentos.
func llamada(nombre, args string) ollama.ToolCall {
	var c ollama.ToolCall
	c.Function.Name = nombre
	c.Function.Arguments = []byte(args)
	return c
}

// registroStub arma el registro del catálogo con handlers que devuelven un
// valor fijo. Si `orden` no es nil, apunta el nombre de cada herramienta
// ejecutada en orden.
func registroStub(orden *[]string) *tools.Registro {
	var impls []tools.Herramienta
	for _, nombre := range tools.NombresCatalogo() {
		n := nombre
		h, _ := tools.NuevaHerramienta(n, func(ctx context.Context, args any, c tools.Contexto) (tools.Resultado, error) {
			if orden != nil {
				*orden = append(*orden, n)
			}
			return tools.Resultado{Salida: "resultado de " + n}, nil
		})
		impls = append(impls, h)
	}
	r, err := tools.NuevoRegistro(impls)
	if err != nil {
		panic(err)
	}
	return r
}

// --- tests -------------------------------------------------------------------

// TestElBucleRespondeSinHerramientasEnUnaPasada — si el modelo no pide nada,
// el bucle cierra en la primera pasada con el texto completo.
func TestElBucleRespondeSinHerramientasEnUnaPasada(t *testing.T) {
	g := &generadorGuion{pasadas: [][]ollama.Evento{respuesta("hola mundo")}}
	sink := &sinkGrabador{}
	e := &Ejecutor{Runner: g}
	res, err := e.Ejecutar(context.Background(), Agente{Nombre: "plan"}, "m", "ctx", nil, nil, 0, false, sink)
	if err != nil {
		t.Fatalf("Ejecutar: %v", err)
	}
	if res.Texto != "hola mundo" {
		t.Errorf("texto = %q", res.Texto)
	}
	if g.llamadas != 1 {
		t.Errorf("llamadas al modelo = %d, quiero 1", g.llamadas)
	}
	if strings.Join(sink.partes, "") != "hola mundo" {
		t.Errorf("el sink recibió %v", sink.partes)
	}
}

// El razonamiento del turno se resuelve una vez y viaja en la pasada: es lo que
// hace que el interruptor del pie llegue al modelo.
func TestElTurnoMandaElRazonamientoQueLePiden(t *testing.T) {
	si := true
	g := &generadorGuion{pasadas: [][]ollama.Evento{respuesta("ok")}}
	e := &Ejecutor{Runner: g, Pensar: func(modelo string) *bool { return &si }}
	if _, err := e.Ejecutar(context.Background(), Agente{Nombre: "chat"}, "qwen3:4b", "ctx", nil, nil, 0, false, nil); err != nil {
		t.Fatalf("Ejecutar: %v", err)
	}
	if g.ultimoPensar == nil || !*g.ultimoPensar {
		t.Errorf("la pasada debe llevar el razonamiento encendido: %v", g.ultimoPensar)
	}
}

// Sin función que lo decida no se manda nada: el harness no decide por su cuenta
// que un modelo razone.
func TestSinDecisionDeRazonamientoNoSeMandaNada(t *testing.T) {
	g := &generadorGuion{pasadas: [][]ollama.Evento{respuesta("ok")}}
	e := &Ejecutor{Runner: g}
	if _, err := e.Ejecutar(context.Background(), Agente{Nombre: "chat"}, "llama3.2", "ctx", nil, nil, 0, false, nil); err != nil {
		t.Fatalf("Ejecutar: %v", err)
	}
	if g.ultimoPensar != nil {
		t.Errorf("sin decisión, `think` no viaja: %v", *g.ultimoPensar)
	}
}

// El razonamiento se decide UNA vez por turno: todas las pasadas llevan lo mismo,
// aunque el modelo pida herramientas por el camino.
func TestElRazonamientoNoCambiaEntrePasadas(t *testing.T) {
	no := false
	consultas := 0
	g := &generadorGuion{pasadas: [][]ollama.Evento{
		pedido("voy", llamada("leer_archivo", `{"ruta":"a.md"}`)),
		respuesta("listo"),
	}}
	e := &Ejecutor{
		Runner:     g,
		Despachar:  NuevoDespachador(registroStub(nil)),
		MaxPasadas: 2,
		Pensar: func(string) *bool {
			consultas++
			return &no
		},
	}
	if _, err := e.Ejecutar(context.Background(), Agente{Nombre: "build"}, "qwen3:4b", "ctx", nil, nil, 0, false, nil); err != nil {
		t.Fatalf("Ejecutar: %v", err)
	}
	if consultas != 1 {
		t.Errorf("la decisión se consulta una vez, no %d", consultas)
	}
	if g.ultimoPensar == nil || *g.ultimoPensar {
		t.Errorf("la última pasada lleva el mismo valor: %v", g.ultimoPensar)
	}
}

// El historial de conversación se antepone al contexto del turno.
func TestElBucleAnteponeElHistorialAlContexto(t *testing.T) {
	g := &generadorGuion{pasadas: [][]ollama.Evento{respuesta("ok")}}
	e := &Ejecutor{Runner: g}
	historial := []ollama.Mensaje{
		{Role: "user", Content: "hola"},
		{Role: "assistant", Content: "qué tal"},
	}
	if _, err := e.Ejecutar(context.Background(), Agente{Nombre: "plan"}, "m", "ctx", historial, nil, 0, false, nil); err != nil {
		t.Fatalf("Ejecutar: %v", err)
	}
	if len(g.ultimos) != len(historial)+1 {
		t.Fatalf("mensajes = %d, quiero %d: %+v", len(g.ultimos), len(historial)+1, g.ultimos)
	}
	if g.ultimos[0].Content != "hola" || g.ultimos[1].Role != "assistant" {
		t.Errorf("el historial no se antepone: %+v", g.ultimos)
	}
	if ult := g.ultimos[len(g.ultimos)-1]; ult.Role != "user" || ult.Content != "ctx" {
		t.Errorf("el contexto del turno debe ir al final como usuario: %+v", ult)
	}
}

// Las imágenes del turno viajan solo en el mensaje de usuario del turno actual.
func TestElTurnoLlevaImagenesAlModelo(t *testing.T) {
	g := &generadorGuion{pasadas: [][]ollama.Evento{respuesta("ok")}}
	e := &Ejecutor{Runner: g}
	imgs := []string{"aG9sYQ=="}
	if _, err := e.Ejecutar(context.Background(), Agente{Nombre: "plan"}, "m", "ctx", []ollama.Mensaje{{Role: "user", Content: "hola"}}, imgs, 0, false, nil); err != nil {
		t.Fatalf("Ejecutar: %v", err)
	}
	if len(g.ultimos[0].Images) != 0 {
		t.Errorf("el historial no debe llevar imágenes: %+v", g.ultimos[0])
	}
	ult := g.ultimos[len(g.ultimos)-1]
	if len(ult.Images) != 1 || ult.Images[0] != imgs[0] {
		t.Errorf("el turno actual debe llevar las imágenes: %+v", ult)
	}
}

// TestElBucleEjecutaHerramientaYVuelveAlModelo — el ciclo central: el modelo
// pide una herramienta por el canal nativo, se ejecuta y su resultado vuelve
// como mensaje de rol `tool`.
func TestElBucleEjecutaHerramientaYVuelveAlModelo(t *testing.T) {
	var orden []string
	d := NuevoDespachador(registroStub(&orden))
	g := &generadorGuion{pasadas: [][]ollama.Evento{
		pedido("", llamada("leer_archivo", `{"ruta":"a.md"}`)),
		respuesta("listo"),
	}}
	e := &Ejecutor{Runner: g, Despachar: d, MaxPasadas: 3}
	ag := Agente{Nombre: "plan", Permisos: []Permiso{{Accion: "leer", Efecto: EfectoPermitir}}}

	res, err := e.Ejecutar(context.Background(), ag, "m", "ctx", nil, nil, 0, false, nil)
	if err != nil {
		t.Fatalf("Ejecutar: %v", err)
	}
	if res.Texto != "listo" {
		t.Errorf("texto final = %q", res.Texto)
	}
	if g.llamadas != 2 {
		t.Errorf("llamadas al modelo = %d, quiero 2", g.llamadas)
	}
	if len(orden) != 1 || orden[0] != "leer_archivo" {
		t.Errorf("herramientas ejecutadas = %v, quiero [leer_archivo]", orden)
	}
	// La segunda llamada lleva el resultado como mensaje de rol `tool`.
	var visto bool
	for _, m := range g.ultimos {
		if m.Role == "tool" && strings.Contains(m.Content, "resultado de leer_archivo") {
			visto = true
		}
	}
	if !visto {
		t.Errorf("el resultado debe volver al modelo como mensaje de herramienta: %+v", g.ultimos)
	}
}

// TestElBucleEjecutaEnElOrdenPedido — las herramientas de un turno se ejecutan
// en el orden en que el modelo las pidió, nunca en paralelo.
func TestElBucleEjecutaEnElOrdenPedido(t *testing.T) {
	var orden []string
	d := NuevoDespachador(registroStub(&orden))
	g := &generadorGuion{pasadas: [][]ollama.Evento{
		pedido("", llamada("listar_carpeta", `{"ruta":"."}`), llamada("buscar_archivos", `{"patron":"*.go"}`)),
		respuesta("hecho"),
	}}
	e := &Ejecutor{Runner: g, Despachar: d, MaxPasadas: 3}
	ag := Agente{Nombre: "plan", Permisos: []Permiso{{Accion: "leer", Efecto: EfectoPermitir}}}
	if _, err := e.Ejecutar(context.Background(), ag, "m", "ctx", nil, nil, 0, false, nil); err != nil {
		t.Fatalf("Ejecutar: %v", err)
	}
	if len(orden) != 2 || orden[0] != "listar_carpeta" || orden[1] != "buscar_archivos" {
		t.Errorf("orden = %v, quiero el pedido por el modelo", orden)
	}
}

// TestElBucleUnRechazoNoCortaElTurno — un fallo no corregible viaja al modelo
// como resultado y el turno continúa.
func TestElBucleUnRechazoNoCortaElTurno(t *testing.T) {
	d := NuevoDespachador(registroStub(nil))
	g := &generadorGuion{pasadas: [][]ollama.Evento{
		// `plan` no tiene escritura: la petición es E_TOOL_NOT_ALLOWED.
		pedido("", llamada("crear_archivo", `{"ruta":"a.md","contenido":"x"}`)),
		respuesta("no puedo"),
	}}
	e := &Ejecutor{Runner: g, Despachar: d, MaxPasadas: 3}
	ag := Agente{Nombre: "plan", Permisos: []Permiso{{Accion: "leer", Efecto: EfectoPermitir}}}

	res, err := e.Ejecutar(context.Background(), ag, "m", "ctx", nil, nil, 0, false, nil)
	if err != nil {
		t.Fatalf("un rechazo no debe cortar el turno: %v", err)
	}
	if res.Texto != "no puedo" {
		t.Errorf("texto final = %q", res.Texto)
	}
	var dicho bool
	for _, m := range g.ultimos {
		if m.Role == "tool" && strings.Contains(m.Content, "no tiene la herramienta") {
			dicho = true
		}
	}
	if !dicho {
		t.Errorf("el motivo debe llegar al modelo: %+v", g.ultimos)
	}
}

// TestElBucleCierraConSintesisAlAgotarPasadas — si el modelo pide herramientas
// siempre, al agotar las rondas el bucle hace UNA pasada final SIN herramientas
// para que el turno cierre con texto real, no con el preámbulo que acompañaba a
// la última petición.
func TestElBucleCierraConSintesisAlAgotarPasadas(t *testing.T) {
	d := NuevoDespachador(registroStub(nil))
	g := &generadorGuion{pasadas: [][]ollama.Evento{
		pedido("voy a mirar a", llamada("leer_archivo", `{"ruta":"a.md"}`)),
		pedido("voy a mirar b", llamada("leer_archivo", `{"ruta":"b.md"}`)),
		respuesta("síntesis final"),
	}}
	e := &Ejecutor{Runner: g, Despachar: d, MaxPasadas: 2}
	ag := Agente{Nombre: "plan", Permisos: []Permiso{{Accion: "leer", Efecto: EfectoPermitir}}}

	res, err := e.Ejecutar(context.Background(), ag, "m", "ctx", nil, nil, 0, false, nil)
	if err != nil {
		t.Fatalf("Ejecutar: %v", err)
	}
	if g.llamadas != 3 {
		t.Errorf("llamadas al modelo = %d, quiero 3 (2 con herramientas + 1 de síntesis)", g.llamadas)
	}
	if res.Texto != "síntesis final" {
		t.Errorf("texto = %q, quiero la síntesis, no el preámbulo", res.Texto)
	}
	if len(g.ultimasHerramientas) != 0 {
		t.Errorf("la pasada de síntesis no debe ofrecer herramientas: %+v", g.ultimasHerramientas)
	}
}

// TestElBucleSinHerramientasNoLasOfrece — con `sinHerramientas` el turno no
// presenta definiciones y cierra en la primera pasada con su texto.
func TestElBucleSinHerramientasNoLasOfrece(t *testing.T) {
	d := NuevoDespachador(registroStub(nil))
	g := &generadorGuion{pasadas: [][]ollama.Evento{respuesta("compuesto")}}
	e := &Ejecutor{Runner: g, Despachar: d, MaxPasadas: 3}
	ag := Agente{Nombre: "plan", Permisos: []Permiso{{Accion: "leer", Efecto: EfectoPermitir}}}

	res, err := e.Ejecutar(context.Background(), ag, "m", "ctx", nil, nil, 0, true, nil)
	if err != nil {
		t.Fatalf("Ejecutar: %v", err)
	}
	if res.Texto != "compuesto" || g.llamadas != 1 {
		t.Errorf("texto = %q, llamadas = %d, quiero «compuesto» en 1 pasada", res.Texto, g.llamadas)
	}
	if len(g.ultimasHerramientas) != 0 {
		t.Errorf("no debe ofrecer herramientas: %+v", g.ultimasHerramientas)
	}
}

// TestElBuclePropagaElErrorDelModelo — un fallo del stream se propaga tal cual.
func TestElBuclePropagaElErrorDelModelo(t *testing.T) {
	fallo := errors.New("modelo caído")
	g := &generadorGuion{pasadas: [][]ollama.Evento{{{Tipo: ollama.EventoError, Error: fallo}}}}
	e := &Ejecutor{Runner: g}
	if _, err := e.Ejecutar(context.Background(), Agente{Nombre: "plan"}, "m", "ctx", nil, nil, 0, false, nil); !errors.Is(err, fallo) {
		t.Fatalf("err = %v, quiero %v", err, fallo)
	}
}

// TestAcumulaLosTokensDelTurno — T-B024-13: el conteo de tokens del evento de
// fin se consume en vez de descartarse.
func TestAcumulaLosTokensDelTurno(t *testing.T) {
	g := &generadorGuion{pasadas: [][]ollama.Evento{{
		{Tipo: ollama.EventoToken, Texto: "hola"},
		{Tipo: ollama.EventoDone, Done: &ollama.RespuestaFinal{Texto: "hola", Done: true, TokensEntr: 11, TokensSal: 3}},
	}}}
	e := &Ejecutor{Runner: g}
	res, err := e.Ejecutar(context.Background(), Agente{Nombre: "plan"}, "m", "ctx", nil, nil, 0, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.TokensEntrada != 11 || res.TokensSalida != 3 {
		t.Errorf("tokens = %d/%d, quiero 11/3", res.TokensEntrada, res.TokensSalida)
	}
}

// TestElTestigoSeTomaPorPeticion — T-B024-14: el turno de inferencia se toma
// una vez por pasada, no una vez por ejecución completa: una espera de
// aprobación entre pasadas no retiene el modelo.
func TestElTestigoSeTomaPorPeticion(t *testing.T) {
	d := NuevoDespachador(registroStub(nil))
	g := &generadorGuion{pasadas: [][]ollama.Evento{
		pedido("", llamada("leer_archivo", `{"ruta":"a.md"}`)),
		respuesta("listo"),
	}}
	var tomas int
	e := &Ejecutor{
		Runner:     g,
		Despachar:  d,
		MaxPasadas: 3,
		Testigo: func(ctx context.Context, fn func(context.Context) error) error {
			tomas++
			return fn(ctx)
		},
	}
	ag := Agente{Nombre: "plan", Permisos: []Permiso{{Accion: "leer", Efecto: EfectoPermitir}}}
	if _, err := e.Ejecutar(context.Background(), ag, "m", "ctx", nil, nil, 0, false, nil); err != nil {
		t.Fatal(err)
	}
	if tomas != 2 {
		t.Errorf("el testigo se tomó %d veces, quiero 2 (una por petición al modelo)", tomas)
	}
}

// TestSinHerramientasDegradaAConversacion — T-B024-16: si el modelo no declara
// capacidad de herramientas, no se le presentan y el agente avisa de que va a
// conversar.
func TestSinHerramientasDegradaAConversacion(t *testing.T) {
	g := &generadorGuion{pasadas: [][]ollama.Evento{respuesta("converso")}}
	sink := &sinkGrabador{}
	e := &Ejecutor{
		Runner:            g,
		Despachar:         NuevoDespachador(registroStub(nil)),
		PuedeHerramientas: func(modelo string) bool { return false },
	}
	ag := Agente{Nombre: "plan", Prompt: "p", Permisos: []Permiso{{Accion: "leer", Efecto: EfectoPermitir}}}
	res, err := e.Ejecutar(context.Background(), ag, "m", "ctx", nil, nil, 0, false, sink)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.ultimasHerramientas) != 0 {
		t.Errorf("no deben viajar herramientas: %+v", g.ultimasHerramientas)
	}
	if len(sink.avisos) != 1 || !strings.Contains(sink.avisos[0], "conversar") {
		t.Errorf("avisos = %v, quiero uno diciendo que va a conversar", sink.avisos)
	}
	if res.Aviso == "" {
		t.Error("el resultado debe llevar el aviso")
	}
}
