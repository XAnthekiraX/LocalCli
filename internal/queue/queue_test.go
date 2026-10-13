package queue

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"localcli/internal/task"
)

const mainEjemplo = "| ID     | Acción | Tarea | Dep    | Estado     |\n" +
	"| ------ | ------ | ----- | ------ | ---------- |\n" +
	"| T-B001 | crear  | uno   | —      | completada |\n" +
	"| T-B002 | crear  | dos   | T-B001 | pendiente  |\n" +
	"| T-B003 | crear  | tres  | T-B002 | pendiente  |\n"

// escribirTODO crea un TODO de prueba bajo una raíz temporal.
func escribirTODO(t *testing.T, raiz, contenido string) {
	t.Helper()
	dir := filepath.Join(raiz, "ai", "tasks", "backend")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, task.NombreMainTasks), []byte(contenido), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

// T-B018-09: la cola se reconstruye del TODO en orden de dependencias.
func TestReconstruirYOrdenar(t *testing.T) {
	raiz := t.TempDir()
	escribirTODO(t, raiz, mainEjemplo)

	c, err := Reconstruir(raiz, task.CapaBackend)
	if err != nil {
		t.Fatalf("Reconstruir: %v", err)
	}
	elems := c.Elementos()
	want := []string{"T-B001", "T-B002", "T-B003"}
	for i, id := range want {
		if elems[i].ID != id {
			t.Errorf("orden[%d] = %s, queremos %s", i, elems[i].ID, id)
		}
	}
}

// T-B018-09: re-derivar toma el estado nuevo del archivo.
func TestRederrivarVeCambiosDelArchivo(t *testing.T) {
	raiz := t.TempDir()
	escribirTODO(t, raiz, mainEjemplo)
	c, err := Reconstruir(raiz, task.CapaBackend)
	if err != nil {
		t.Fatalf("Reconstruir: %v", err)
	}
	// Otra ejecución marca T-B001 completada (ya lo está); probamos que tras
	// cambiar el archivo, la re-derivación lo refleja.
	if _, err := task.CambiarEstadoElemento(raiz, task.CapaBackend, "T-B002", task.EstadoCompletada); err != nil {
		t.Fatalf("cambiar estado: %v", err)
	}
	if err := c.Rederrivar(); err != nil {
		t.Fatalf("Rederrivar: %v", err)
	}
	e, _ := c.PorID("T-B002")
	if e.Estado != task.EstadoCompletada {
		t.Errorf("T-B002 = %q, queremos completada", e.Estado)
	}
}

// T-B018-10: un elemento con dependencias sin completar queda bloqueado y lo
// dice; la cola no se detiene por él.
func TestBloqueadosInformaYNoFrena(t *testing.T) {
	elems := []task.Elemento{
		task.NuevoElemento("T-B001", task.CapaBackend, task.AccionCrear, task.EstadoPendiente, nil, nil, nil),
		task.NuevoElemento("T-B002", task.CapaBackend, task.AccionCrear, task.EstadoPendiente, []string{"T-B001"}, nil, nil),
		task.NuevoElemento("T-B003", task.CapaBackend, task.AccionCrear, task.EstadoPendiente, nil, nil, nil),
	}
	bloqueados := Bloqueados(elems)
	if len(bloqueados) != 1 || bloqueados[0].ID != "T-B002" {
		t.Fatalf("bloqueados = %+v, queremos solo T-B002", bloqueados)
	}
	if bloqueados[0].Codigo() != CodigoElementoBloqueado {
		t.Errorf("código = %s, queremos %s", bloqueados[0].Codigo(), CodigoElementoBloqueado)
	}
	// T-B001 y T-B003 siguen elegibles: el bloqueo de T-B002 no detiene la cola.
	eleg, err := task.Elegibles(elems)
	if err != nil {
		t.Fatalf("Elegibles: %v", err)
	}
	if len(eleg) != 2 {
		t.Errorf("elegibles = %d, queremos 2", len(eleg))
	}
}

// T-B018-11: se entrega el siguiente elegible y un elemento empezado se retoma.
func TestSiguienteYMarcar(t *testing.T) {
	raiz := t.TempDir()
	escribirTODO(t, raiz, mainEjemplo)
	c, err := Reconstruir(raiz, task.CapaBackend)
	if err != nil {
		t.Fatalf("Reconstruir: %v", err)
	}
	consumidor, err := c.Consumir(PermitirAlOrquestador())
	if err != nil {
		t.Fatalf("Consumir: %v", err)
	}

	primero, err := consumidor.Siguiente(context.Background())
	if err != nil {
		t.Fatalf("Siguiente: %v", err)
	}
	if primero == nil || primero.ID != "T-B002" {
		t.Fatalf("siguiente = %+v, queremos T-B002", primero)
	}

	if err := consumidor.Marcar("T-B002", task.EstadoCompletada); err != nil {
		t.Fatalf("Marcar: %v", err)
	}
	siguiente, err := consumidor.Siguiente(context.Background())
	if err != nil {
		t.Fatalf("Siguiente 2: %v", err)
	}
	if siguiente == nil || siguiente.ID != "T-B003" {
		t.Fatalf("siguiente = %+v, queremos T-B003", siguiente)
	}
	// Reanudar no repite: T-B002 quedó completada en el archivo.
	b, err := os.ReadFile(filepath.Join(raiz, "ai", "tasks", "backend", task.NombreMainTasks))
	if err != nil {
		t.Fatalf("leer: %v", err)
	}
	elems, err := task.ParseTabla(b, task.CapaBackend)
	if err != nil {
		t.Fatalf("reparsear: %v", err)
	}
	for _, e := range elems {
		if e.ID == "T-B002" && e.Estado != task.EstadoCompletada {
			t.Errorf("el archivo no refleja T-B002 completada: %q", e.Estado)
		}
	}
}

// T-B018-11: la cola detenida es la que no tiene elegible pero sí pendientes.
func TestDetenida(t *testing.T) {
	// T-B001 declarado bloqueado y T-B002 dependiente de él: nada es elegible.
	detenida := &Cola{capa: task.CapaBackend, elementos: []task.Elemento{
		task.NuevoElemento("T-B001", task.CapaBackend, task.AccionCrear, task.EstadoBloqueada, nil, []string{"T-B002"}, nil),
		task.NuevoElemento("T-B002", task.CapaBackend, task.AccionCrear, task.EstadoPendiente, []string{"T-B001"}, nil, nil),
	}}
	consumidor, _ := detenida.Consumir(PermitirAlOrquestador())
	if !consumidor.Detenida() {
		t.Error("una cola sin elegible y con pendientes está detenida")
	}

	// Con un elemento sin dependencias, la cola avanza.
	avanza := &Cola{capa: task.CapaBackend, elementos: []task.Elemento{
		task.NuevoElemento("T-B001", task.CapaBackend, task.AccionCrear, task.EstadoPendiente, nil, nil, nil),
	}}
	c2, _ := avanza.Consumir(PermitirAlOrquestador())
	if c2.Detenida() {
		t.Error("una cola con un elegible no está detenida")
	}
}

// T-B018-13: sin el permiso del orquestador no se consume la cola.
func TestConsumoDenegadoSinPermiso(t *testing.T) {
	c := &Cola{}
	if _, err := c.Consumir(Permiso{}); err == nil {
		t.Fatal("esperaba error al consumir sin permiso")
	}
	if _, err := c.Consumir(PermitirAlOrquestador()); err != nil {
		t.Fatalf("con permiso no debía fallar: %v", err)
	}
	if err := c.LanzarElemento("T-B001"); err == nil {
		t.Fatal("lanzar una tarea suelta debía estar denegado")
	}
}

// T-B018-12: marcar bloqueada no se escribe (falta la columna en la tabla).
func TestMarcarBloqueadaNoEscribible(t *testing.T) {
	raiz := t.TempDir()
	escribirTODO(t, raiz, mainEjemplo)
	c, err := Reconstruir(raiz, task.CapaBackend)
	if err != nil {
		t.Fatalf("Reconstruir: %v", err)
	}
	consumidor, _ := c.Consumir(PermitirAlOrquestador())
	if err := consumidor.Marcar("T-B002", task.EstadoBloqueada); err == nil {
		t.Fatal("marcar bloqueada debía fallar")
	}
	// Un ID que no está en la cola también se rechaza.
	if err := consumidor.Marcar("T-B099", task.EstadoCompletada); err == nil {
		t.Fatal("un ID fuera de la cola debía rechazarse")
	}
}
