package flow

import (
	"context"
	"errors"
	"strings"
	"testing"

	"localcli/internal/task"
	"localcli/internal/tools"
)

// --- stubs -----------------------------------------------------------------

type stubContexto struct{ llamadas int }

func (s *stubContexto) ContextoPara(ctx context.Context, objetivo string) (string, error) {
	s.llamadas++
	return "contexto:" + objetivo, nil
}

type stubAgente struct {
	traza  *[]string
	fallar bool
}

func (s *stubAgente) Ejecutar(ctx context.Context, agente, contexto string) (Resultado, error) {
	if s.traza != nil {
		*s.traza = append(*s.traza, "agente:"+agente)
	}
	if s.fallar {
		return Resultado{}, errors.New("boom")
	}
	if !strings.HasPrefix(contexto, "contexto:") {
		return Resultado{}, errors.New("la etapa no recibió contexto")
	}
	return Resultado{Texto: "ok"}, nil
}

type stubAprobador struct {
	traza   *[]string
	aprobar bool
}

func (s *stubAprobador) Aprobar(ctx context.Context, descripcion string) (bool, error) {
	if s.traza != nil {
		*s.traza = append(*s.traza, "aprobado")
	}
	return s.aprobar, nil
}

type stubEmisor struct{ eventos []Evento }

func (s *stubEmisor) Emitir(e Evento) { s.eventos = append(s.eventos, e) }

func (s *stubEmisor) nombres() []string {
	var out []string
	for _, e := range s.eventos {
		out = append(out, e.Nombre)
	}
	return out
}

// --- T-B010-04: el ciclo de planificación no escribe -----------------------

func TestPlanTerminaSinEscribir(t *testing.T) {
	var traza []string
	m := &Motor{
		Contexto:  &stubContexto{},
		Agente:    &stubAgente{traza: &traza},
		Aprobador: &stubAprobador{traza: &traza, aprobar: true},
	}
	estado, err := m.EjecutarFlujo(context.Background(), FlujoPlanificacion(), "documentar")
	if err != nil {
		t.Fatalf("EjecutarFlujo: %v", err)
	}
	if estado != EstadoTerminado {
		t.Fatalf("estado = %s, quiero terminado", estado)
	}
	for _, paso := range traza {
		if paso == "agente:"+tools.AgenteBuild {
			t.Fatal("el ciclo de planificación no debe invocar a build")
		}
	}
}

// --- T-B010-05: build solo arranca tras aprobar el plan --------------------

func TestBuildSoloTrasAprobacion(t *testing.T) {
	var traza []string
	m := &Motor{
		Contexto:  &stubContexto{},
		Agente:    &stubAgente{traza: &traza},
		Aprobador: &stubAprobador{traza: &traza, aprobar: true},
	}
	if _, err := m.EjecutarFlujo(context.Background(), FlujoTrabajo(task.AccionCrear), "crear x"); err != nil {
		t.Fatalf("EjecutarFlujo: %v", err)
	}
	primerBuild, primeraAprobacion := -1, -1
	for i, paso := range traza {
		if paso == "agente:"+tools.AgenteBuild && primerBuild < 0 {
			primerBuild = i
		}
		if paso == "aprobado" && primeraAprobacion < 0 {
			primeraAprobacion = i
		}
	}
	if primerBuild < 0 {
		t.Fatal("el ciclo de trabajo debe llegar a build")
	}
	if primeraAprobacion < 0 || primeraAprobacion > primerBuild {
		t.Fatalf("build arrancó antes de aprobar (build en %d, aprobación en %d)", primerBuild, primeraAprobacion)
	}
}

// --- T-B010-06: el ciclo resolver sigue la secuencia documentada -----------

func TestResolverSecuenciaDocumentada(t *testing.T) {
	var traza []string
	m := &Motor{
		Contexto:  &stubContexto{},
		Agente:    &stubAgente{traza: &traza},
		Aprobador: &stubAprobador{traza: &traza, aprobar: true},
	}
	if _, err := m.EjecutarFlujo(context.Background(), FlujoResolver(), "arreglar x"); err != nil {
		t.Fatalf("EjecutarFlujo: %v", err)
	}
	quiero := []string{
		"agente:" + tools.AgenteBuild, // evidencia
		"agente:" + tools.AgentePlan,  // localizar
		"agente:" + tools.AgentePlan,  // causa raíz
		"agente:" + tools.AgentePlan,  // propuesta
		"agente:" + tools.AgenteBuild, // aplicar
		"agente:" + tools.AgenteBuild, // verificar
	}
	var soloAgentes []string
	for _, p := range traza {
		if strings.HasPrefix(p, "agente:") {
			soloAgentes = append(soloAgentes, p)
		}
	}
	if strings.Join(soloAgentes, ",") != strings.Join(quiero, ",") {
		t.Fatalf("secuencia = %v, quiero %v", soloAgentes, quiero)
	}
}

// --- T-B010-07: eventos por decisión --------------------------------------

func TestEventosPorEtapa(t *testing.T) {
	emisor := &stubEmisor{}
	m := &Motor{
		Contexto:  &stubContexto{},
		Agente:    &stubAgente{},
		Aprobador: &stubAprobador{aprobar: true},
		Eventos:   emisor,
	}
	if _, err := m.EjecutarFlujo(context.Background(), FlujoResolver(), "objetivo"); err != nil {
		t.Fatalf("EjecutarFlujo: %v", err)
	}
	nombres := emisor.nombres()
	for _, esperado := range []string{EventoEtapaIniciada, EventoEtapaTerminada, EventoFlujoPausado, EventoFlujoReanudado} {
		if !contieneStr(nombres, esperado) {
			t.Errorf("falta el evento %s en %v", esperado, nombres)
		}
	}
	// Payload: etapa_terminada lleva el resumen.
	for _, e := range emisor.eventos {
		if e.Nombre == EventoEtapaTerminada {
			if e.Datos["etapa"] == "" || e.Datos["resumen"] != "ok" {
				t.Errorf("payload de etapa_terminada = %v", e.Datos)
			}
			break
		}
	}
}

// TestEtapaFallidaEmiteEventoYDetiene — una etapa que falla detiene el flujo y
// no encadena la siguiente.
func TestEtapaFallidaEmiteEventoYDetiene(t *testing.T) {
	emisor := &stubEmisor{}
	m := &Motor{
		Contexto: &stubContexto{},
		Agente:   &stubAgente{fallar: true},
		Eventos:  emisor,
	}
	_, err := m.EjecutarFlujo(context.Background(), FlujoResolver(), "objetivo")
	if !errors.Is(err, ErrEtapaFallida) {
		t.Fatalf("err = %v, quiero E_STAGE_FAILED", err)
	}
	if !contieneStr(emisor.nombres(), EventoEtapaFallida) {
		t.Errorf("falta etapa_fallida en %v", emisor.nombres())
	}
}

// TestCancelacionSePropaga — cancelar detiene el flujo con E_FLOW_CANCELLED.
func TestCancelacionSePropaga(t *testing.T) {
	emisor := &stubEmisor{}
	m := &Motor{Contexto: &stubContexto{}, Agente: &stubAgente{}, Eventos: emisor}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := m.EjecutarFlujo(ctx, FlujoResolver(), "objetivo")
	if !errors.Is(err, ErrFlujoCancelado) {
		t.Fatalf("err = %v, quiero E_FLOW_CANCELLED", err)
	}
	if !contieneStr(emisor.nombres(), EventoFlujoCancelado) {
		t.Errorf("falta flujo_cancelado en %v", emisor.nombres())
	}
}

// --- T-B010-08: un elemento por iteración ---------------------------------

type stubCola struct {
	elementos  []ElementoCola
	i          int
	marcas     []string
	enProgreso string
	violacion  bool
}

func (c *stubCola) Siguiente(ctx context.Context) (*ElementoCola, error) {
	if c.i >= len(c.elementos) {
		return nil, nil
	}
	e := c.elementos[c.i]
	c.i++
	return &e, nil
}

func (c *stubCola) Marcar(id string, estado task.Estado) error {
	if estado == task.EstadoEnProgreso {
		if c.enProgreso != "" {
			c.violacion = true // dos en progreso a la vez
		}
		c.enProgreso = id
	}
	if estado == task.EstadoCompletada {
		c.enProgreso = ""
	}
	c.marcas = append(c.marcas, id+":"+string(estado))
	return nil
}

func TestConsumirColaUnElementoPorIteracion(t *testing.T) {
	cola := &stubCola{elementos: []ElementoCola{
		{ID: "T-B001", Objetivo: "uno"},
		{ID: "T-B002", Objetivo: "dos"},
		{ID: "T-B003", Objetivo: "tres"},
	}}
	m := &Motor{
		Contexto:  &stubContexto{},
		Agente:    &stubAgente{},
		Aprobador: &stubAprobador{aprobar: true},
	}
	if err := m.ConsumirCola(context.Background(), cola, FlujoTrabajo(task.AccionCrear)); err != nil {
		t.Fatalf("ConsumirCola: %v", err)
	}
	if cola.violacion {
		t.Error("nunca debe haber dos elementos en progreso a la vez")
	}
	// Cada elemento pasa por en_progreso y luego completada, en orden.
	quiero := []string{
		"T-B001:en_progreso", "T-B001:completada",
		"T-B002:en_progreso", "T-B002:completada",
		"T-B003:en_progreso", "T-B003:completada",
	}
	if strings.Join(cola.marcas, ",") != strings.Join(quiero, ",") {
		t.Fatalf("marcas = %v, quiero %v", cola.marcas, quiero)
	}
}

func contieneStr(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
