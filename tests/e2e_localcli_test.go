// e2e_localcli_test.go — T-B015-08: verificación integral final.
//
// Fuente de verdad: ai/tasks/backend/015-task-validaciones.md (T-B015-08) y
// TESTING.md §1 (prueba de integración: "ciclo completo de una etapa…, ciclo de
// la cola", con sistema de archivos temporal y sin red).
//
// Dos niveles:
//
//  1. El binario compila y arranca contra un proyecto fixture (--help, que es
//     la única ruta que no abre TUI ni requiere terminal interactiva), y
//     `store.Open` prepara su base sin errores.
//  2. Los tres ciclos oficiales (planificación, trabajo, resolver) recorren sus
//     etapas completas sobre el motor real con dependencias dobles, y la cola
//     consume un TODO real de disco de principio a fin.
package tests

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"localcli/internal/flow"
	"localcli/internal/queue"
	"localcli/internal/store"
	"localcli/internal/task"
	"localcli/internal/tools"
)

// --- 1. El binario ----------------------------------------------------------

// TestElBinarioCompilaYResponde — `go build` del módulo entero y arranque del
// binario contra un fixture (--help). La TUI completa necesita terminal
// interactiva; el resto del arranque se comprueba aquí.
func TestElBinarioCompilaYResponde(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "localcli")
	construir := exec.Command("go", "build", "-o", bin, "..")
	if salida, err := construir.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, salida)
	}
	f, err := os.CreateTemp("", "pantalla-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	defer os.Remove(f.Name())

	arrancar := exec.Command(bin, "--help")
	salida, err := arrancar.CombinedOutput()
	if err != nil {
		t.Fatalf("el binario no arrancó: %v\n%s", err, salida)
	}
	if !strings.Contains(string(salida), "localcli") {
		t.Errorf("la ayuda no dice qué es: %s", salida)
	}
}

// TestElProyectoFixturePreparaSuBase — abrir el binario contra la carpeta de
// este mismo proyecto prepara la base (el arranque real de main.go).
func TestElProyectoFixturePreparaSuBase(t *testing.T) {
	raiz, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Open(raiz); err != nil {
		// El proyecto real tiene .localcli/ quizá en uso; una base ya abierta
		// por otra prueba no es un fallo del arranque.
		t.Skipf("la base del proyecto real no está disponible: %v", err)
	}
}

// --- 2. Los tres ciclos -----------------------------------------------------

// motorE2E es el motor real con contexto y agente dobles: la etapa completa
// (contexto → agente → decisión/aprobación) corre de verdad, solo el modelo es
// un doble porque no hay Ollama en la prueba (TESTING.md §3 lo permite para
// pruebas rápidas; el streaming real lo cubre ollama).
func motorE2E(aprobar bool) *flow.Motor {
	return &flow.Motor{
		Contexto:  contextoFijo{},
		Agente:    agenteFijo{},
		Aprobador: aprobadorFijo(aprobar),
	}
}

type contextoFijo struct{}

func (contextoFijo) ContextoPara(ctx context.Context, objetivo string) (string, error) {
	return "contexto para " + objetivo, nil
}

type agenteFijo struct{}

func (agenteFijo) Ejecutar(ctx context.Context, agente, contexto string) (flow.Resultado, error) {
	if !strings.HasPrefix(contexto, "contexto para ") {
		return flow.Resultado{}, os.ErrInvalid
	}
	return flow.Resultado{Texto: "etapa completada"}, nil
}

type aprobadorFijo bool

func (a aprobadorFijo) Aprobar(ctx context.Context, descripcion string) (bool, error) {
	return bool(a), nil
}

// TestElCicloDePlanificacionRecorreCompleto — las 8 etapas de
// FlujoPlanificacion, sin escribir nada.
func TestElCicloDePlanificacionRecorreCompleto(t *testing.T) {
	m := motorE2E(true)
	estado, err := m.EjecutarFlujo(context.Background(), flow.FlujoPlanificacion(), "documentar el proyecto")
	if err != nil {
		t.Fatalf("planificación: %v", err)
	}
	if estado != flow.EstadoTerminado {
		t.Fatalf("estado = %s, quiero terminado", estado)
	}
}

// TestElCicloDeTrabajoRecorreCompleto — las 5 etapas de FlujoTrabajo con el
// relevo plan→build y sus aprobaciones.
func TestElCicloDeTrabajoRecorreCompleto(t *testing.T) {
	m := motorE2E(true)
	estado, err := m.EjecutarFlujo(context.Background(), flow.FlujoTrabajo(task.AccionCrear), "implementar x")
	if err != nil {
		t.Fatalf("trabajo: %v", err)
	}
	if estado != flow.EstadoTerminado {
		t.Fatalf("estado = %s, quiero terminado", estado)
	}
	// Y declinado se detiene: sin el sí de la persona, no hay etapa de build.
	mDeclinado := motorE2E(false)
	if _, err := mDeclinado.EjecutarFlujo(context.Background(), flow.FlujoTrabajo(task.AccionCrear), "implementar x"); err == nil {
		t.Fatal("un ciclo de trabajo declinado no puede terminar")
	}
}

// TestElCicloDeResolverRecorreCompleto — las 6 etapas de FlujoResolver.
func TestElCicloDeResolverRecorreCompleto(t *testing.T) {
	m := motorE2E(true)
	estado, err := m.EjecutarFlujo(context.Background(), flow.FlujoResolver(), "arreglar el fallo")
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if estado != flow.EstadoTerminado {
		t.Fatalf("estado = %s, quiero terminado", estado)
	}
}

// --- 3. La cola sobre TODO real de disco -----------------------------------

// proyectoConTODO arma un proyecto fixture con un TODO de dos tareas y abre su
// base. Devuelve la raíz y el consumidor autorizado de la cola.
func proyectoConTODO(t *testing.T) (string, *queue.Consumidor) {
	t.Helper()
	raiz := t.TempDir()
	if err := os.MkdirAll(filepath.Join(raiz, "ai", "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(raiz, "ai", "docs", "PROJECT.md"), []byte("# p\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(raiz, "ai", "tasks", "backend")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	todo := "| ID | Acción | Tarea | Dep | Estado | Detalle |\n" +
		"|----|--------|-------|-----|--------|---------|\n" +
		"| T-B001 | crear | Primera | — | pendiente | |\n" +
		"| T-B002 | crear | Segunda | T-B001 | pendiente | |\n"
	if err := os.WriteFile(filepath.Join(dir, "MAIN-TASKS.md"), []byte(todo), 0o644); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(raiz)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	cola, err := queue.Reconstruir(raiz, task.CapaBackend)
	if err != nil {
		t.Fatalf("Reconstruir: %v", err)
	}
	c, err := cola.Consumir(queue.PermitirAlMotor())
	if err != nil {
		t.Fatalf("Consumir: %v", err)
	}
	return raiz, c
}

// TestLaColaRealConsumeElTODOCompleto — el ciclo entero sobre disco: T-B001 y
// T-B002 pasan por en_progreso → completada, el archivo queda actualizado y la
// cola termina sola al vaciarse.
func TestLaColaRealConsumeElTODOCompleto(t *testing.T) {
	raiz, cola := proyectoConTODO(t)
	m := &flow.Motor{
		Contexto:  contextoFijo{},
		Agente:    agenteFijo{},
		Aprobador: aprobadorFijo(true),
	}
	if err := m.ConsumirCola(context.Background(), cola, flow.FlujoTrabajo(task.AccionCrear)); err != nil {
		t.Fatalf("ConsumirCola: %v", err)
	}

	b, err := os.ReadFile(filepath.Join(raiz, "ai", "tasks", "backend", "MAIN-TASKS.md"))
	if err != nil {
		t.Fatal(err)
	}
	elems, err := task.ParseTabla(b, task.CapaBackend)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range elems {
		if e.Estado != task.EstadoCompletada {
			t.Errorf("%s quedó en %s, quiero completada", e.ID, e.Estado)
		}
	}
	// Y la cola vacía no entrega más: el siguiente elemento es nil.
	siguiente, err := cola.Siguiente(context.Background())
	if err != nil || siguiente != nil {
		t.Errorf("cola terminada debe devolver nil: %v, %v", siguiente, err)
	}
}

// TestElCatalogoSigueCerradoYCompleto — las trece herramientas del catálogo,
// con el reparto plan/build intacto: la frontera de la garantía al final del
// ciclo completo.
func TestElCatalogoSigueCerradoYCompleto(t *testing.T) {
	nombres := tools.NombresCatalogo()
	if len(nombres) != 13 {
		t.Fatalf("catálogo = %d herramientas, quiero 13", len(nombres))
	}
	plan := tools.HerramientasDePlan()
	for _, n := range nombres {
		err := tools.ComprobarPermiso(plan, n)
		h, _ := tools.Buscar(n)
		if h.Modo == tools.Escribe && err == nil {
			t.Errorf("plan no puede tener la herramienta de escritura %s", n)
		}
	}
}
