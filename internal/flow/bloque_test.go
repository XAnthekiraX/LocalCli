package flow

import (
	"context"
	"sort"
	"strings"
	"testing"

	"localcli/internal/tools"
)

// --- stubs del bloque --------------------------------------------------------

type stubOptimizador struct{ llamadas []string }

func (s *stubOptimizador) Optimizar(ctx context.Context, flujo, etapa, resultado string) (string, error) {
	s.llamadas = append(s.llamadas, etapa)
	return "opt(" + resultado + ")", nil
}

type stubBloque struct {
	entradas  map[string]EntradaBloque
	limpiezas int
}

func (s *stubBloque) Limpiar(ctx context.Context, flujo string) error {
	s.limpiezas++
	s.entradas = map[string]EntradaBloque{}
	return nil
}

func (s *stubBloque) Guardar(ctx context.Context, flujo string, e EntradaBloque) error {
	if s.entradas == nil {
		s.entradas = map[string]EntradaBloque{}
	}
	s.entradas[e.Etapa] = e
	return nil
}

func (s *stubBloque) Leer(ctx context.Context, flujo string) ([]EntradaBloque, error) {
	out := make([]EntradaBloque, 0, len(s.entradas))
	for _, e := range s.entradas {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Posicion < out[j].Posicion })
	return out, nil
}

// flujoDePruebaBloque es un flujo mínimo con bloque: dos fases y una composición.
func flujoDePruebaBloque() Flujo {
	return Flujo{
		Nombre:         "prueba",
		Comando:        "/prueba",
		BloqueContexto: true,
		Etapas: []Etapa{
			{ID: "a", Nombre: "Fase A", Agente: tools.AgentePlan, Instruccion: "haz A"},
			{ID: "b", Nombre: "Fase B", Agente: tools.AgentePlan, Instruccion: "haz B"},
			{ID: "final", Nombre: "Componer", Agente: tools.AgentePlan, Instruccion: "compón"},
		},
	}
}

// TestFlujoConBloqueOptimizaYGuardaCadaFaseYComponeSinHerramientas — cada fase
// intermedia deja su aportación optimizada en el bloque; la última recibe el
// bloque entero y corre sin herramientas.
func TestFlujoConBloqueOptimizaYGuardaCadaFaseYComponeSinHerramientas(t *testing.T) {
	bloque := &stubBloque{}
	opt := &stubOptimizador{}
	ag := &stubAgente{}
	m := &Motor{Contexto: &stubContexto{}, Agente: ag, Bloque: bloque, Optimizador: opt}

	if _, err := m.EjecutarFlujo(context.Background(), flujoDePruebaBloque(), "objetivo"); err != nil {
		t.Fatalf("EjecutarFlujo: %v", err)
	}

	if bloque.limpiezas != 1 {
		t.Errorf("limpiezas = %d, quiero 1 (el bloque no hereda la ejecución anterior)", bloque.limpiezas)
	}
	if len(bloque.entradas) != 2 {
		t.Fatalf("aportaciones = %d, quiero 2 (solo las fases intermedias)", len(bloque.entradas))
	}
	if bloque.entradas["a"].Contenido != "opt(ok)" || bloque.entradas["b"].Contenido != "opt(ok)" {
		t.Errorf("las aportaciones deben ser el resultado optimizado: %+v", bloque.entradas)
	}
	if len(opt.llamadas) != 2 || opt.llamadas[0] != "Fase A" || opt.llamadas[1] != "Fase B" {
		t.Errorf("optimización por fase = %v, quiero [Fase A, Fase B]", opt.llamadas)
	}

	// La composición es la última petición: sin herramientas y con el bloque.
	if len(ag.peticiones) != 3 {
		t.Fatalf("peticiones = %d, quiero 3", len(ag.peticiones))
	}
	if ag.peticiones[0].SinHerramientas || ag.peticiones[1].SinHerramientas {
		t.Errorf("las fases intermedias no son composición: %+v", ag.peticiones)
	}
	final := ag.peticiones[2]
	if !final.SinHerramientas {
		t.Errorf("la composición debe correr sin herramientas")
	}
	if final.Silenciosa {
		t.Errorf("la composición es la entrega: debe ser visible")
	}
	if !strings.Contains(final.Contexto, "## Bloque de contexto") {
		t.Errorf("la composición debe recibir el bloque de contexto: %q", final.Contexto)
	}
	if !strings.Contains(final.Contexto, "opt(ok)") {
		t.Errorf("el bloque debe llevar las aportaciones optimizadas: %q", final.Contexto)
	}
}

// TestFlujoConBloqueSinOptimizadorSigueGuardando — sin optimizador conectado el
// flujo no se detiene: guarda el resumen mecánico y compone igual.
func TestFlujoConBloqueSinOptimizadorSigueGuardando(t *testing.T) {
	bloque := &stubBloque{}
	ag := &stubAgente{}
	m := &Motor{Contexto: &stubContexto{}, Agente: ag, Bloque: bloque}

	if _, err := m.EjecutarFlujo(context.Background(), flujoDePruebaBloque(), "objetivo"); err != nil {
		t.Fatalf("EjecutarFlujo: %v", err)
	}
	if len(bloque.entradas) != 2 {
		t.Fatalf("aportaciones = %d, quiero 2", len(bloque.entradas))
	}
	if bloque.entradas["a"].Contenido == "" {
		t.Errorf("sin optimizador debe caer al resumen mecánico, no quedar vacío")
	}
	if ag.peticiones[2].Contexto == "" || !strings.Contains(ag.peticiones[2].Contexto, "## Bloque de contexto") {
		t.Errorf("la composición debe recibir el bloque aunque no haya optimizador")
	}
}
