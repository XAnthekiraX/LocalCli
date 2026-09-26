// integration_errores_test.go — T-B015-05: los errores que emite el motor son
// los de ERRORS.md, ni más ni menos.
//
// Fuente de verdad: ai/docs/backend/05-quality/ERRORS.md §3 (el catálogo de
// códigos) y §5 (qué código toca a cada operación). La prueba recorre los
// códigos que el backend puede emitir hoy y comprueba dos cosas: que el error
// lleva el código en el texto (el mensaje llega a la pantalla) y que el código
// es el documentado para esa operación. Ninguno inventado: si un módulo empieza
// a emitir "E_ALGO" que no está en el catálogo, esta prueba lo delata.
package tests

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"localcli/internal/agent"
	"localcli/internal/exec"
	"localcli/internal/fileops"
	"localcli/internal/flow"
	"localcli/internal/ollama"
	"localcli/internal/queue"
	"localcli/internal/store"
	"localcli/internal/tools"
)

// TestLosCodigosDelCatalogoSonLosDocumentados — los códigos que exportan los
// módulos coinciden letra a letra con ERRORS.md §3. Si alguien renombra una
// constante, esta prueba lo ve.
func TestLosCodigosDelCatalogoSonLosDocumentados(t *testing.T) {
	catalogo := map[string]string{
		// tools
		"tools.unknown":    tools.CodigoHerramientaDesconocida,
		"tools.notallowed": tools.CodigoHerramientaNoPermitida,
		"tools.badargs":    tools.CodigoArgumentosInvalidos,
		// fileops
		"fileops.outside":  fileops.CodigoRutaFuera,
		"fileops.exists":   fileops.CodigoRutaExiste,
		"fileops.approval": fileops.CodigoNecesitaAprobacion,
		"fileops.declined": fileops.CodigoAprobacionDeclinada,
		"fileops.confirm":  fileops.CodigoNecesitaConfirmacion,
		"fileops.badargs":  fileops.CodigoArgumentosInvalidos,
		// exec
		"exec.whitelist": exec.CodigoComandoNoEnBlanco,
		"exec.timeout":   exec.CodigoComandoAgotado,
		"exec.declined":  exec.CodigoAprobacionDeclinada,
		"exec.badargs":   exec.CodigoArgumentosInvalidos,
		// flow
		"flow.stagefailed": flow.CodigoEtapaFallida,
		"flow.cancelled":   flow.CodigoFlujoCancelado,
		"flow.blocked":     flow.CodigoElementoBloqueado,
		// queue
		"queue.blocked": queue.CodigoElementoBloqueado,
		// ollama
		"ollama.unavailable": ollama.CodigoOllamaNoDisponible,
		"ollama.toobig":      ollama.CodigoModeloNoCabe,
	}
	documentados := map[string]bool{
		"E_TOOL_UNKNOWN": true, "E_TOOL_NOT_ALLOWED": true, "E_BAD_ARGS": true,
		"E_PATH_OUTSIDE": true, "E_PATH_EXISTS": true, "E_NEEDS_CONFIRM": true,
		"E_NEEDS_APPROVAL": true, "E_APPROVAL_DECLINED": true,
		"E_CMD_NOT_WHITELISTED": true, "E_CMD_TIMEOUT": true, "E_CMD_OUTPUT_TRUNCATED": true,
		"E_DOC_NOT_FOUND": true, "E_DOC_PARSE": true, "E_CONTEXT_TOO_BIG": true,
		"E_ELEMENTO_BLOQUEADO": true, "E_STAGE_FAILED": true, "E_FLOW_CANCELLED": true,
		"E_NOT_A_PROJECT":  true,
		"E_DB_UNAVAILABLE": true, "E_DB_SCHEMA_OUTDATED": true, "E_DB_CONSTRAINT": true,
		"E_DB_FOREIGN_KEY": true, "E_DB_CONFLICT": true, "E_DB_LOCKED": true,
		"E_OLLAMA_UNAVAILABLE": true, "E_MODEL_TOO_BIG": true, "E_NO_LANDLOCK": true,
	}
	for qué, codigo := range catalogo {
		if !strings.HasPrefix(codigo, "E_") {
			t.Errorf("%s: el código %q no tiene forma de código interno", qué, codigo)
		}
		if !documentados[codigo] {
			t.Errorf("%s: el código %s no está en ERRORS.md §3", qué, codigo)
		}
	}
}

// TestElCodigoViajaEnElMensaje — ERRORS.md §1: "Un error tiene un código
// interno y un mensaje para la persona". El código va dentro del texto porque
// el mensaje es lo que llega a la pantalla.
func TestElCodigoViajaEnElMensaje(t *testing.T) {
	proyecto, db := proyectoTemp(t)
	ops := &fileops.Ops{Proyecto: proyecto, Historial: store.Historial{DB: db}}

	// Sin aprobador → E_NEEDS_APPROVAL en el mensaje.
	_, err := ops.CrearArchivo(context.Background(), "x.txt", "x")
	if err == nil || !strings.Contains(err.Error(), "E_NEEDS_APPROVAL") {
		t.Errorf("el mensaje debe llevar el código: %v", err)
	}
	// Ruta fuera → E_PATH_OUTSIDE en el mensaje.
	_, err = ops.CrearArchivo(context.Background(), "../x.txt", "x")
	if err == nil || !strings.Contains(err.Error(), "E_PATH_OUTSIDE") {
		t.Errorf("el mensaje debe llevar el código: %v", err)
	}
	// Herramienta inventada → E_TOOL_UNKNOWN.
	if _, existe := tools.NuevaPeticion("inventada"); existe {
		t.Fatal("una herramienta fuera del catálogo no puede tener contrato")
	}
	if permErr := tools.ComprobarPermiso(nil, "inventada"); permErr == nil || !strings.Contains(permErr.Error(), "E_TOOL_UNKNOWN") {
		t.Errorf("el mensaje debe llevar el código: %v", permErr)
	}
	// Comando fuera de la lista blanca → E_CMD_NOT_WHITELISTED.
	e := &exec.Ejecutor{Proyecto: proyecto}
	_, err = e.Ejecutar(context.Background(), "no-existe-en-la-lista --opción", "")
	if err == nil || !strings.Contains(err.Error(), "E_CMD_NOT_WHITELISTED") {
		t.Errorf("el mensaje debe llevar el código: %v", err)
	}
}

// TestUnComandoQueFallaNoEsUnE — ERRORS.md §5: "Un comando que sale con error
// no es un E_: es un resultado con la salida de error, y el agente sigue".
// `go test ./no-existe` está en la lista blanca y falla porque el paquete no
// existe: el resultado lleva su salida de error y la herramienta no falla.
func TestUnComandoQueFallaNoEsUnE(t *testing.T) {
	proyecto, _ := proyectoTemp(t)
	e := &exec.Ejecutor{Proyecto: proyecto, Limite: 2 * time.Minute}
	res, err := e.Ejecutar(context.Background(), "go test ./paquete-que-no-existe", "")
	if err != nil {
		t.Fatalf("un comando que falla no debe ser un error de la herramienta: %v", err)
	}
	if res.Codigo == 0 {
		t.Skip("go run devolvió 0; el entorno no permite este caso")
	}
	if strings.Contains(res.Error, "E_") {
		t.Errorf("la salida de error del comando no lleva códigos internos: %q", res.Error)
	}
}

// TestUnAgenteSinHerramientasNoPuedePedirNada — el agente de solo conversación
// (agent) no pide herramientas: cualquier petición suya es E_TOOL_NOT_ALLOWED.
func TestUnAgenteSinHerramientasNoPuedePedirNada(t *testing.T) {
	err := tools.ComprobarPermiso(nil, "leer_archivo")
	if err == nil || !strings.Contains(err.Error(), "E_TOOL_NOT_ALLOWED") {
		t.Errorf("err = %v, quiero E_TOOL_NOT_ALLOWED", err)
	}
}

// TestLosCodigosDeBaseSonLosDocumentados — store traduce los errores del driver
// a los códigos de ERRORS.md §3 antes de que salgan del módulo (§4).
func TestLosCodigosDeBaseSonLosDocumentados(t *testing.T) {
	documentados := map[string]bool{
		"E_DB_UNAVAILABLE": true, "E_DB_SCHEMA_OUTDATED": true, "E_DB_CONSTRAINT": true,
		"E_DB_FOREIGN_KEY": true, "E_DB_CONFLICT": true, "E_DB_LOCKED": true,
		"E_NOT_A_PROJECT": true, "E_STAGE_FAILED": true,
	}
	centinelas := []error{
		store.ErrNoDisponible, store.ErrRestriccion, store.ErrClaveForanea,
		store.ErrConflictivo, store.ErrBloqueado, store.ErrEstadoIlegal,
		store.ErrNotAProject,
	}
	for _, e := range centinelas {
		msg := e.Error()
		codigo := msg
		if i := strings.Index(msg, ":"); i > 0 {
			codigo = msg[:i]
		}
		if !documentados[codigo] {
			t.Errorf("el centinela %q usa el código %s, que no está en ERRORS.md", msg, codigo)
		}
	}
	// E_NOT_A_PROJECT es el arranque en una carpeta que no es proyecto (§5).
	if _, err := store.Open(t.TempDir()); err == nil || !strings.Contains(err.Error(), "E_NOT_A_PROJECT") {
		t.Errorf("abrir en una carpeta sin proyecto = %v, quiero E_NOT_A_PROJECT", err)
	}
}

// TestUnAgenteBaseTieneSuCatalogo — el reparto plan/build que sostiene la
// garantía se mantiene: plan lee, build tiene las trece.
func TestUnAgenteBaseTieneSuCatalogo(t *testing.T) {
	plan := tools.HerramientasDePlan()
	build := tools.HerramientasDeBuild()
	if len(plan) == 0 || len(plan) >= len(build) {
		t.Fatalf("reparto roto: plan=%d build=%d", len(plan), len(build))
	}
	// Y los agentes JSON documentados cargan con ese reparto. La ruta es
	// relativa a este paquete (tests/): el proyecto vive un nivel arriba.
	agentes, err := agent.CargarCarpeta(filepath.Join("..", "ai", "agents"))
	if err != nil {
		t.Fatalf("CargarCarpeta: %v", err)
	}
	if len(agentes) < 2 {
		t.Fatalf("el proyecto debe traer plan y build: hay %d", len(agentes))
	}
	planJSON, okPlan := agent.PorNombre(agentes, "plan")
	buildJSON, okBuild := agent.PorNombre(agentes, "build")
	if !okPlan || !okBuild {
		t.Fatalf("faltan los agentes base: %v", nombresDe(agentes))
	}
	if err := agent.ValidarHerramientas(planJSON); err != nil {
		t.Errorf("el agente plan no valida: %v", err)
	}
	if err := agent.ValidarHerramientas(buildJSON); err != nil {
		t.Errorf("el agente build no valida: %v", err)
	}
	if len(planJSON.Herramientas) != len(plan) {
		t.Errorf("plan declara %d herramientas y el catálogo de lectura tiene %d", len(planJSON.Herramientas), len(plan))
	}
	if len(buildJSON.Herramientas) != len(build) {
		t.Errorf("build declara %d herramientas y el catálogo completo tiene %d", len(buildJSON.Herramientas), len(build))
	}
}

func nombresDe(agentes []agent.Agente) []string {
	var out []string
	for _, a := range agentes {
		out = append(out, a.Nombre)
	}
	return out
}
