package session

// ciclos_test.go — T-B019: detección de trabajo ordenado, consumo de la cola y
// lo que deja cada ciclo oficial al cerrar.
//
// Fuente de verdad: [[backend/01-domain/BUSINESS_RULES]] §Detección de trabajo
// ordenado y §Cola, [[specs/SPEC-COLA-TAREAS]] y [[specs/SPEC-RESOLVER]].

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"localcli/internal/flow"
	"localcli/internal/queue"
	"localcli/internal/store"
	"localcli/internal/task"
	"localcli/internal/tui"
)

func TestDetectarTrabajoOrdenado(t *testing.T) {
	casos := []struct {
		nombre   string
		peticion string
		quiere   []string
	}{
		{
			"lista numerada",
			"1. crear la spec\n2. actualizar el código\n3. correr las pruebas",
			[]string{"crear la spec", "actualizar el código", "correr las pruebas"},
		},
		{
			"lista con viñetas",
			"- documentar la capa\n- implementar la tarea",
			[]string{"documentar la capa", "implementar la tarea"},
		},
		{
			"conectores de secuencia",
			"primero documentar, luego implementar",
			[]string{"primero documentar", "implementar"},
		},
		{
			"ordinales de trabajo",
			"ejecutar tarea 1, tarea 2, tarea 3",
			[]string{"tarea 1", "tarea 2", "tarea 3"},
		},
		{
			"capa por capa",
			"documentar capa por capa",
			CapasPorDefecto,
		},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			p, ok := DetectarTrabajoOrdenado(c.peticion, nil)
			if !ok {
				t.Fatalf("%q debería proponer un TODO", c.peticion)
			}
			if p.Peticion != c.peticion {
				t.Errorf("petición = %q, quiero %q", p.Peticion, c.peticion)
			}
			if got := strings.Join(p.Pasos, " | "); got != strings.Join(c.quiere, " | ") {
				t.Fatalf("pasos = %q, quiero %q", got, strings.Join(c.quiere, " | "))
			}
		})
	}
}

// Sin enumeración no hay propuesta: la petición se responde en el chat.
func TestNoProponeSinEnumeracion(t *testing.T) {
	peticiones := []string{
		"¿cómo funciona el nodo de contexto?",
		"arregla el bug del streaming",
		"una tarea suelta",
		"",
	}
	for _, p := range peticiones {
		if prop, ok := DetectarTrabajoOrdenado(p, nil); ok {
			t.Errorf("%q no debería proponer un TODO (propuso %q)", p, prop.Pasos)
		}
	}
}

// "Documentar capa por capa" propone un paso por capa, y las capas que le llegan
// mandan sobre las de por defecto.
func TestCapaPorCapaProponeLasCapasDelProyecto(t *testing.T) {
	p, ok := DetectarTrabajoOrdenado("documentar capa por capa", []string{"backend", "frontend"})
	if !ok {
		t.Fatal("debería proponer un TODO")
	}
	if got := strings.Join(p.Pasos, "|"); got != "backend|frontend" {
		t.Fatalf("pasos = %q, quiero %q", got, "backend|frontend")
	}
}

// La detección propone y nada más: anuncia la propuesta y no toca la cola ni
// crea la lista de pasos. Es la diferencia entre proponer un TODO y arrancarlo.
func TestLaDeteccionSoloPropone(t *testing.T) {
	o := &Orquestador{eventos: make(chan tui.Evento, 4)}

	o.proponerTodo("s1", "documentar capa por capa")

	select {
	case e := <-o.eventos:
		if e.Tipo != tui.EventoPropuestaTodo {
			t.Fatalf("evento = %q, quiero %q", e.Tipo, tui.EventoPropuestaTodo)
		}
		if e.Sesion != "s1" {
			t.Errorf("sesión = %q, quiero %q", e.Sesion, "s1")
		}
		if got := len(strings.Split(e.Datos["pasos"], "\n")); got != len(CapasPorDefecto) {
			t.Errorf("la propuesta lleva %d pasos, quiero %d", got, len(CapasPorDefecto))
		}
	default:
		t.Fatal("debería haber emitido `propuesta_todo`")
	}

	// No hay un segundo evento: nada de arrancar la cola.
	select {
	case e := <-o.eventos:
		t.Fatalf("evento de más: %q", e.Tipo)
	default:
	}

	// Y una petición sin lista no emite nada.
	o.proponerTodo("s1", "hola, ¿qué tal?")
	select {
	case e := <-o.eventos:
		t.Fatalf("no debería emitir nada, emitió %q", e.Tipo)
	default:
	}

	if o.colas != nil {
		t.Error("la detección no debe tocar la cola")
	}
}

// El TODO de prueba: dos elementos encadenados, los dos pendientes.
const mainTareasDePrueba = "| ID     | Acción | Tarea | Dep    | Estado    |\n" +
	"| ------ | ------ | ----- | ------ | --------- |\n" +
	"| T-B001 | crear  | uno   | —      | pendiente |\n" +
	"| T-B002 | crear  | dos   | T-B001 | pendiente |\n"

// El mismo TODO con la tarea que aprueba un ciclo de trabajo.
const mainConTareaNueva = mainTareasDePrueba +
	"| T-B003 | crear  | tres  | T-B002 | pendiente |\n"

func escribirTareasDePrueba(t *testing.T, raiz, contenido string) {
	t.Helper()
	dir := filepath.Join(raiz, "ai", "tasks", "backend")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("crear ai/tasks/backend: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, task.NombreMainTasks), []byte(contenido), 0o644); err != nil {
		t.Fatalf("escribir el TODO: %v", err)
	}
}

// leerFlujoOficial carga un flujo de los oficiales, como lo hace la siembra.
func leerFlujoOficial(t *testing.T, nombre string) flow.Flujo {
	t.Helper()
	datos, err := os.ReadFile(filepath.Join("..", "arranque", "datos", "flows", nombre))
	if err != nil {
		t.Fatalf("leer %s: %v", nombre, err)
	}
	f, err := flow.DecodificarFlujo(datos)
	if err != nil {
		t.Fatalf("decodificar %s: %v", nombre, err)
	}
	return f
}

// motivosDeAviso devuelve el motivo de cada notificación de los eventos dados.
func motivosDeAviso(eventos []tui.Evento) []string {
	var motivos []string
	for _, e := range eventos {
		if e.Tipo == tui.EventoNotificacion {
			motivos = append(motivos, e.Datos["motivo"])
		}
	}
	return motivos
}

func contiene(xs []string, sub string) bool {
	for _, x := range xs {
		if strings.Contains(x, sub) {
			return true
		}
	}
	return false
}

// T-B019-03: la cola se consume un elemento por iteración, y cada elemento solo
// queda completado cuando su turno terminó bien.
func TestLaColaConsumeUnElementoPorIteracion(t *testing.T) {
	o, sesion := orquestadorDeFlujo(t)
	escribirTareasDePrueba(t, o.Carpeta(), mainTareasDePrueba)

	cola, err := queue.Reconstruir(o.Carpeta(), task.CapaBackend)
	if err != nil {
		t.Fatalf("derivar la cola: %v", err)
	}
	o.ConCola(cola)
	canal, cerrar := o.Suscribir()
	defer cerrar()

	if err := o.arrancarCola(context.Background(), sesion, flujoDePrueba(), task.CapaBackend); err != nil {
		t.Fatalf("arrancar la cola: %v", err)
	}

	// Los dos elementos quedaron completados en el archivo: la cola no guarda
	// estado propio, el marcado va a la celda Estado de su tarea.
	nueva, err := queue.Reconstruir(o.Carpeta(), task.CapaBackend)
	if err != nil {
		t.Fatalf("releer el TODO: %v", err)
	}
	for _, e := range nueva.Elementos() {
		if e.Estado != task.EstadoCompletada {
			t.Errorf("%s quedó %q, quiero completada", e.ID, e.Estado)
		}
	}

	// La cola se anunció al avanzar y avisó al vaciarse.
	var actualizaciones, avisos int
	for _, e := range drenar(canal) {
		switch e.Tipo {
		case tui.EventoColaActualizada:
			actualizaciones++
		case tui.EventoNotificacion:
			avisos++
		}
	}
	if actualizaciones < 2 {
		t.Errorf("la cola debería anunciarse en cada iteración, se anunció %d veces", actualizaciones)
	}
	if avisos == 0 {
		t.Error("al vaciarse la cola debería avisar")
	}
}

// Un TODO sin pendientes no arranca nada y lo avisa, en vez de quedarse en
// silencio.
func TestLaColaVaciaAvisaYTermina(t *testing.T) {
	o, sesion := orquestadorDeFlujo(t)
	const soloCompletado = "| ID     | Acción | Tarea | Dep | Estado     |\n" +
		"| ------ | ------ | ----- | --- | ---------- |\n" +
		"| T-B001 | crear  | uno   | —   | completada |\n"
	escribirTareasDePrueba(t, o.Carpeta(), soloCompletado)
	cola, err := queue.Reconstruir(o.Carpeta(), task.CapaBackend)
	if err != nil {
		t.Fatalf("derivar la cola: %v", err)
	}
	o.ConCola(cola)
	canal, cerrar := o.Suscribir()
	defer cerrar()

	if err := o.arrancarCola(context.Background(), sesion, flujoDePrueba(), task.CapaBackend); err != nil {
		t.Fatalf("arrancar la cola: %v", err)
	}

	avisos := motivosDeAviso(drenar(canal))
	if len(avisos) == 0 {
		t.Fatal("una cola sin pendientes debería avisar de que se vació")
	}
	if !contiene(avisos, "se vació") {
		t.Errorf("el aviso debería decir que la cola se vació: %q", avisos)
	}
}

// T-B019-05: el flujo de planificación registra sus fases en la lista de pasos, y
// la etapa que las registra es de `plan`: la lista de pasos no toca archivos, así
// que el flujo sigue siendo lanzable por `plan`.
func TestElFlujoDePlanificarRegistraLasFases(t *testing.T) {
	f := leerFlujoOficial(t, "planificar.json")
	var ids []string
	for _, e := range f.Etapas {
		ids = append(ids, e.ID)
		if e.Agente != "plan" {
			t.Errorf("la etapa %s es de %q; el flujo de planificación no debe escribir", e.ID, e.Agente)
		}
	}
	if !contiene(ids, "fases") {
		t.Fatalf("el flujo debería registrar las fases; etapas = %v", ids)
	}
	if !flow.PuedeLanzar(f, []string{"read"}) {
		t.Error("`plan` debería poder lanzar el flujo de planificación")
	}
}

// T-B019-05: y al cerrar deja constancia de esas fases: con fases lo dice, y sin
// ninguna avisa en vez de darlo por bueno.
func TestLaPlanificacionCompruebaSusFases(t *testing.T) {
	o, sesion := orquestadorDeFlujo(t)
	canal, cerrar := o.Suscribir()
	defer cerrar()

	o.cerrarCiclo(sesion, flow.Flujo{Comando: ComandoPlanificar, Nombre: "planificacion"})
	motivos := motivosDeAviso(drenar(canal))
	if !contiene(motivos, "sin dejar fases") {
		t.Fatalf("una planificación sin fases debería avisar: %q", motivos)
	}

	for _, fase := range []string{"Fase 1", "Fase 2"} {
		if _, err := store.CrearTodo(o.db, o.projectID, sesion, fase, "pendiente", "media"); err != nil {
			t.Fatalf("registrar la fase: %v", err)
		}
	}
	o.cerrarCiclo(sesion, flow.Flujo{Comando: ComandoPlanificar, Nombre: "planificacion"})
	motivos = motivosDeAviso(drenar(canal))
	if !contiene(motivos, "2 fases") {
		t.Fatalf("debería decir cuántas fases quedaron: %q", motivos)
	}
}

// T-B019-06: la tarea que aprueba un ciclo de trabajo entra en la cola en su
// posición, sin reiniciar nada.
func TestElCicloDeTrabajoMeteLaTareaEnLaCola(t *testing.T) {
	o, sesion := orquestadorDeFlujo(t)
	escribirTareasDePrueba(t, o.Carpeta(), mainTareasDePrueba)
	cola, err := queue.Reconstruir(o.Carpeta(), task.CapaBackend)
	if err != nil {
		t.Fatalf("derivar la cola: %v", err)
	}
	o.ConCola(cola)
	canal, cerrar := o.Suscribir()
	defer cerrar()
	drenar(canal)

	// El ciclo de trabajo aprueba su tarea y la deja escrita en el TODO.
	escribirTareasDePrueba(t, o.Carpeta(), mainConTareaNueva)
	o.cerrarCiclo(sesion, flow.Flujo{Comando: "/crear", Nombre: "trabajo/crear"})

	c, ok := o.Cola(task.CapaBackend)
	if !ok {
		t.Fatal("la capa backend debería tener cola")
	}
	if _, ok := c.PorID("T-B003"); !ok {
		t.Fatal("la tarea aprobada debería estar en la cola")
	}
	anunciada := false
	for _, e := range drenar(canal) {
		if e.Tipo == tui.EventoColaActualizada && e.Datos["capa"] == "backend" {
			anunciada = true
		}
	}
	if !anunciada {
		t.Error("al entrar la tarea nueva debería anunciarse la cola")
	}
}

// T-B019-07: el resolver entrega el PLAN y no crea tareas ni implementa, así que
// no toca la cola. Y su flujo es de solo lectura: todas sus etapas son `plan`.
func TestElResolverNoTocaLaCola(t *testing.T) {
	o, sesion := orquestadorDeFlujo(t)
	escribirTareasDePrueba(t, o.Carpeta(), mainTareasDePrueba)
	cola, err := queue.Reconstruir(o.Carpeta(), task.CapaBackend)
	if err != nil {
		t.Fatalf("derivar la cola: %v", err)
	}
	o.ConCola(cola)
	canal, cerrar := o.Suscribir()
	defer cerrar()
	drenar(canal)

	o.cerrarCiclo(sesion, flow.Flujo{Comando: ComandoResolver, Nombre: "resolver"})

	for _, e := range drenar(canal) {
		if e.Tipo == tui.EventoColaActualizada {
			t.Error("el resolver no debería tocar la cola")
		}
	}
}

func TestElFlujoDelResolverEsDeSoloLectura(t *testing.T) {
	f := leerFlujoOficial(t, "resolver.json")
	if len(f.Etapas) == 0 {
		t.Fatal("el flujo del resolver no tiene etapas")
	}
	for _, e := range f.Etapas {
		if e.Agente != "plan" {
			t.Errorf("la etapa %s es de %q; el resolver no implementa nada", e.ID, e.Agente)
		}
	}
	if !f.Etapas[len(f.Etapas)-1].Entrega {
		t.Error("el resolver debería cerrar con su etapa de entrega")
	}
	if !flow.PuedeLanzar(f, []string{"read"}) {
		t.Error("`plan` debería poder lanzar el resolver")
	}
}
