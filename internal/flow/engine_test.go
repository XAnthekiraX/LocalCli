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

type stubContexto struct {
	llamadas int
	etapas   []string
}

func (s *stubContexto) ContextoPara(ctx context.Context, etapa, objetivo string) (string, error) {
	s.llamadas++
	s.etapas = append(s.etapas, etapa)
	return "contexto:" + objetivo, nil
}

type stubAgente struct {
	traza      *[]string
	fallar     bool
	historial  []Mensaje
	contexto   string
	imagenes   []string
	peticiones []PeticionEtapa
}

func (s *stubAgente) Ejecutar(ctx context.Context, p PeticionEtapa) (Resultado, error) {
	s.historial = p.Historial
	s.contexto = p.Contexto
	s.imagenes = p.Imagenes
	s.peticiones = append(s.peticiones, p)
	if s.traza != nil {
		*s.traza = append(*s.traza, "agente:"+p.Agente)
	}
	if s.fallar {
		return Resultado{}, errors.New("boom")
	}
	if !strings.Contains(p.Contexto, "contexto:") {
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

// stubRegistro recoge las líneas de sub-proceso que el motor manda guardar.
type stubRegistro struct{ procesos []string }

func (s *stubRegistro) ProcesoEtapa(nombre string, fallida bool) {
	linea := nombre
	if fallida {
		linea += ":falló"
	}
	s.procesos = append(s.procesos, linea)
}

// --- el hilo del chat: el sub-proceso de cada etapa ------------------------

// El motor guarda una línea de sub-proceso por etapa: es lo que hace que el
// historial recuerde los pasos de un flujo al volver a la sesión.
func TestElRegistroGuardaElSubProcesoDeCadaEtapa(t *testing.T) {
	reg := &stubRegistro{}
	f := FlujoPlanificacion()
	m := &Motor{
		Contexto:  &stubContexto{},
		Agente:    &stubAgente{},
		Aprobador: &stubAprobador{aprobar: true},
		Registro:  reg,
	}
	if _, err := m.EjecutarFlujo(context.Background(), f, "objetivo"); err != nil {
		t.Fatalf("EjecutarFlujo: %v", err)
	}
	if len(reg.procesos) != len(f.Etapas) {
		t.Fatalf("una línea de sub-proceso por etapa: %v", reg.procesos)
	}
}

// Una etapa que falla deja su línea marcada como fallida.
func TestElRegistroMarcaLaEtapaQueFalla(t *testing.T) {
	reg := &stubRegistro{}
	m := &Motor{Contexto: &stubContexto{}, Agente: &stubAgente{fallar: true}, Registro: reg}
	if _, err := m.EjecutarFlujo(context.Background(), FlujoResolver(), "objetivo"); !errors.Is(err, ErrEtapaFallida) {
		t.Fatalf("err = %v, quiero E_STAGE_FAILED", err)
	}
	ultimo := reg.procesos[len(reg.procesos)-1]
	if !strings.HasSuffix(ultimo, ":falló") {
		t.Errorf("la etapa que falla se marca: %v", reg.procesos)
	}
}

// Sin registro conectado el motor no falla: el sub-proceso solo se ve en vivo.
func TestSinRegistroElFlujoNoSeRompe(t *testing.T) {
	m := &Motor{Contexto: &stubContexto{}, Agente: &stubAgente{}, Aprobador: &stubAprobador{aprobar: true}}
	if _, err := m.EjecutarFlujo(context.Background(), FlujoPlanificacion(), "objetivo"); err != nil {
		t.Fatalf("EjecutarFlujo: %v", err)
	}
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
		"agente:" + tools.AgentePlan, // recibir_tarea
		"agente:" + tools.AgentePlan, // entender_problema
		"agente:" + tools.AgentePlan, // buscar_contexto
		"agente:" + tools.AgentePlan, // investigar
		"agente:" + tools.AgentePlan, // diagnosticar
		"agente:" + tools.AgentePlan, // archivos_afectados
		"agente:" + tools.AgentePlan, // diseno
		"agente:" + tools.AgentePlan, // plan_ejecucion
		"agente:" + tools.AgentePlan, // entregar_plan
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

// TestElBriefDelFlujoLlegaAlAgente — las reglas del flujo y la instrucción de
// la etapa viajan con el contexto que recibe el agente: es cómo un flujo
// declara sus reglas (por ejemplo, cómo descubrir la documentación) sin código.
func TestElBriefDelFlujoLlegaAlAgente(t *testing.T) {
	ag := &stubAgente{}
	m := &Motor{Contexto: &stubContexto{}, Agente: ag, Aprobador: &stubAprobador{aprobar: true}}
	if _, err := m.EjecutarFlujo(context.Background(), FlujoResolver(), "arreglar x"); err != nil {
		t.Fatalf("EjecutarFlujo: %v", err)
	}
	if !strings.Contains(ag.contexto, "Reglas del flujo:") {
		t.Errorf("el contexto no lleva las reglas del flujo: %q", ag.contexto)
	}
	if !strings.Contains(ag.contexto, ReglasResolver[0]) {
		t.Errorf("el contexto no lleva la primera regla del flujo: %q", ag.contexto)
	}
	// La última etapa de resolver es `entregar_plan`, con `plan`.
	if !strings.Contains(ag.contexto, "No implementes nada") {
		t.Errorf("el contexto no lleva la instrucción de la etapa: %q", ag.contexto)
	}
}

// --- Encadenado de pasos y etapas silenciosas ------------------------------

// TestCadaEtapaRecibeElResumenDeLaAnterior — el resultado de una etapa se pasa a
// la siguiente como un bloque corto: es el encadenado de SPEC-MOTOR-FLUJOS ("el
// resultado pasa a la etapa siguiente"), sin arrastrar la salida completa.
func TestCadaEtapaRecibeElResumenDeLaAnterior(t *testing.T) {
	ag := &stubAgente{}
	m := &Motor{Contexto: &stubContexto{}, Agente: ag, Aprobador: &stubAprobador{aprobar: true}}
	if _, err := m.EjecutarFlujo(context.Background(), FlujoResolver(), "objetivo"); err != nil {
		t.Fatalf("EjecutarFlujo: %v", err)
	}
	if len(ag.peticiones) < 2 {
		t.Fatalf("peticiones = %d, quiero al menos 2", len(ag.peticiones))
	}
	if strings.Contains(ag.peticiones[0].Contexto, "Resultados de los pasos anteriores") {
		t.Errorf("la primera etapa no debe llevar pasos anteriores: %q", ag.peticiones[0].Contexto)
	}
	seg := ag.peticiones[1].Contexto
	if !strings.Contains(seg, "Resultados de los pasos anteriores") {
		t.Fatalf("la segunda etapa no recibió lo resuelto por la anterior: %q", seg)
	}
	if !strings.Contains(seg, "### Recibir la tarea") {
		t.Errorf("el bloque acumulado no lleva el nombre de la etapa anterior: %q", seg)
	}
}

// TestEtapasIntermediasSilenciosas — una etapa intermedia sin aprobación corre
// en silencio (no se muestra en el chat ni se persiste); la última se muestra.
func TestEtapasIntermediasSilenciosas(t *testing.T) {
	ag := &stubAgente{}
	m := &Motor{Contexto: &stubContexto{}, Agente: ag, Aprobador: &stubAprobador{aprobar: true}}
	if _, err := m.EjecutarFlujo(context.Background(), FlujoResolver(), "objetivo"); err != nil {
		t.Fatalf("EjecutarFlujo: %v", err)
	}
	ult := len(ag.peticiones) - 1
	for i, p := range ag.peticiones {
		if i == ult {
			if p.Silenciosa {
				t.Errorf("la última etapa no puede ser silenciosa")
			}
			continue
		}
		if !p.Silenciosa {
			t.Errorf("la etapa intermedia %q debería ser silenciosa", p.Etapa)
		}
	}
	if ag.peticiones[0].Etapa != "Recibir la tarea" {
		t.Errorf("la etapa no lleva su nombre: %q", ag.peticiones[0].Etapa)
	}
}

// TestEtapaConAprobacionNoEsSilenciosa — el usuario debe ver lo que aprueba: una
// etapa intermedia con aprobación se muestra aunque no sea la última.
func TestEtapaConAprobacionNoEsSilenciosa(t *testing.T) {
	ag := &stubAgente{}
	m := &Motor{Contexto: &stubContexto{}, Agente: ag, Aprobador: &stubAprobador{aprobar: true}}
	if _, err := m.EjecutarFlujo(context.Background(), FlujoTrabajo(task.AccionCrear), "objetivo"); err != nil {
		t.Fatalf("EjecutarFlujo: %v", err)
	}
	visto := false
	for _, p := range ag.peticiones {
		if p.Etapa == "Presentar el plan" { // etapa con Aprobacion: true
			visto = true
			if p.Silenciosa {
				t.Errorf("una etapa con aprobación no debe ser silenciosa")
			}
		}
	}
	if !visto {
		t.Fatal("no se corrió la etapa con aprobación")
	}
}

// --- T-B010-07: eventos por decisión --------------------------------------

// TestCadaEtapaPideContextoConSuEtapa — la etapa viaja con la petición: es lo
// que permite que context_audit distinga una etapa de otra (una fila por
// documento y etapa). Sin ella, todas las etapas comparten la terna y la
// segunda revienta contra el índice único de la base.
func TestCadaEtapaPideContextoConSuEtapa(t *testing.T) {
	ctx := &stubContexto{}
	m := &Motor{
		Contexto:  ctx,
		Agente:    &stubAgente{},
		Aprobador: &stubAprobador{aprobar: true},
	}
	if _, err := m.EjecutarFlujo(context.Background(), FlujoResolver(), "objetivo"); err != nil {
		t.Fatalf("EjecutarFlujo: %v", err)
	}
	if len(ctx.etapas) < 2 {
		t.Fatalf("peticiones de contexto = %d, quiero al menos 2", len(ctx.etapas))
	}
	visto := map[string]bool{}
	for i, e := range ctx.etapas {
		if e == "" {
			t.Fatalf("la petición %d no lleva etapa", i)
		}
		if visto[e] {
			t.Errorf("la etapa %q pidió contexto dos veces con la misma etiqueta", e)
		}
		visto[e] = true
	}
}

// TestConversarPideContextoComoChat — el chat no es una etapa de flujo: pide
// su contexto bajo EtapaChat para que su auditoría no se mezcle con las etapas.
func TestConversarPideContextoComoChat(t *testing.T) {
	ctx := &stubContexto{}
	m := &Motor{Contexto: ctx, Agente: &stubAgente{}}
	if _, err := m.Conversar(context.Background(), tools.AgenteBuild, "hola", nil, nil); err != nil {
		t.Fatalf("Conversar: %v", err)
	}
	if len(ctx.etapas) != 1 || ctx.etapas[0] != EtapaChat {
		t.Fatalf("etapas = %v, quiero [%s]", ctx.etapas, EtapaChat)
	}
}

// El motor transporta el historial tal cual hasta el agente: no lo interpreta.
func TestConversarPasaElHistorialAlAgente(t *testing.T) {
	ag := &stubAgente{}
	m := &Motor{Contexto: &stubContexto{}, Agente: ag}
	historial := []Mensaje{
		{Rol: RolUsuario, Texto: "hola"},
		{Rol: RolAsistente, Texto: "qué tal"},
	}
	if _, err := m.Conversar(context.Background(), tools.AgenteBuild, "sigue", historial, nil); err != nil {
		t.Fatalf("Conversar: %v", err)
	}
	if len(ag.historial) != 2 || ag.historial[0].Texto != "hola" || ag.historial[1].Rol != RolAsistente {
		t.Fatalf("el historial no llegó al agente: %+v", ag.historial)
	}
}

// Las imágenes del turno llegan al agente tal cual; el motor no las interpreta.
func TestConversarReenviaLasImagenesAlAgente(t *testing.T) {
	ag := &stubAgente{}
	m := &Motor{Contexto: &stubContexto{}, Agente: ag}
	imgs := []string{"aG9sYQ==", "bXVuZG8="}
	if _, err := m.Conversar(context.Background(), tools.AgenteBuild, "mira esto", nil, imgs); err != nil {
		t.Fatalf("Conversar: %v", err)
	}
	if len(ag.imagenes) != 2 || ag.imagenes[1] != imgs[1] {
		t.Fatalf("las imágenes no llegaron al agente: %+v", ag.imagenes)
	}
}

// Un flujo no lleva imágenes: sus etapas corren sin ellas (solo el chat las usa).
func TestEjecutarFlujoNoPasaImagenes(t *testing.T) {
	ag := &stubAgente{}
	m := &Motor{Contexto: &stubContexto{}, Agente: ag, Aprobador: &stubAprobador{aprobar: true}}
	if _, err := m.EjecutarFlujo(context.Background(), FlujoPlanificacion(), "objetivo"); err != nil {
		t.Fatalf("EjecutarFlujo: %v", err)
	}
	if len(ag.imagenes) != 0 {
		t.Errorf("una etapa de flujo no debe llevar imágenes: %+v", ag.imagenes)
	}
}

// Un flujo no lleva historial de conversación: sus etapas corren sin él.
func TestEjecutarFlujoNoPasaHistorial(t *testing.T) {
	ag := &stubAgente{}
	m := &Motor{Contexto: &stubContexto{}, Agente: ag, Aprobador: &stubAprobador{aprobar: true}}
	if _, err := m.EjecutarFlujo(context.Background(), FlujoPlanificacion(), "objetivo"); err != nil {
		t.Fatalf("EjecutarFlujo: %v", err)
	}
	if len(ag.historial) != 0 {
		t.Errorf("una etapa de flujo no debe llevar historial: %+v", ag.historial)
	}
}

func TestEventosPorEtapa(t *testing.T) {
	emisor := &stubEmisor{}
	m := &Motor{
		Contexto:  &stubContexto{},
		Agente:    &stubAgente{},
		Aprobador: &stubAprobador{aprobar: true},
		Eventos:   emisor,
	}
	// El ciclo de trabajo tiene etapas con aprobación, que son las que emiten
	// pausa y reanudación; resolver ya no aprueba nada (solo diagnostica).
	if _, err := m.EjecutarFlujo(context.Background(), FlujoTrabajo(task.AccionCrear), "objetivo"); err != nil {
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
	// Payload: etapa_iniciada lleva el nombre para la línea de progreso.
	for _, e := range emisor.eventos {
		if e.Nombre == EventoEtapaIniciada {
			if e.Datos["nombre"] == "" {
				t.Errorf("etapa_iniciada no lleva nombre: %v", e.Datos)
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
