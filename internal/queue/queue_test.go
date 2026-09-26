package queue

// Tests del módulo queue (T-B012-08). Cubren las verificaciones pedidas en
// 012-task-queue.md subtarea por subtarea:
//
//	01 → la cola es una proyección: no hay escritura a SQLite desde queue
//	02 → el fixture de TODO se reconstruye en orden
//	03 → orden por dependencias; un ciclo es error
//	04 → un elemento tras uno bloqueado aparece bloqueado, con motivo
//	05 → siguiente devuelve el primero elegible; vacío si nada puede arrancar
//	06 → completar un elemento actualiza la fila del archivo
//	07 → lanzar un elemento suelto se deniega
//	08 → este propio fichero: go test ./internal/queue/... pasa

import (
	"context"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"localcli/internal/flow"
	"localcli/internal/task"
)

// El consumidor tiene que cumplir la interfaz que espera el motor.
var _ flow.Cola = (*Consumidor)(nil)

// --- helpers ---------------------------------------------------------------

// copiarTODO copia el TODO de un fixture a un directorio temporal y devuelve la
// raíz, para poder escribir sin tocar testdata.
func copiarTODO(t *testing.T, fixture string) string {
	t.Helper()
	raiz := t.TempDir()
	origen := filepath.Join("testdata", fixture, "ai", "tasks")
	destino := filepath.Join(raiz, "ai", "tasks")
	err := filepath.WalkDir(origen, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(origen, p)
		if err != nil {
			return err
		}
		dest := filepath.Join(destino, rel)
		if d.IsDir() {
			return os.MkdirAll(dest, 0o755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(dest, b, 0o644)
	})
	if err != nil {
		t.Fatalf("copiar el fixture %s: %v", fixture, err)
	}
	return raiz
}

// consumidorDe reconstruye la cola de un fixture y devuelve la vista autorizada.
func consumidorDe(t *testing.T, fixture string) *Consumidor {
	t.Helper()
	cola, err := Reconstruir(filepath.Join("testdata", fixture), task.CapaBackend)
	if err != nil {
		t.Fatalf("Reconstruir(%s): %v", fixture, err)
	}
	c, err := cola.Consumir(PermitirAlMotor())
	if err != nil {
		t.Fatalf("Consumir: %v", err)
	}
	return c
}

func ids(elems []task.Elemento) []string {
	out := make([]string, len(elems))
	for i, e := range elems {
		out[i] = e.ID
	}
	return out
}

// --- T-B012-01: proyección, no almacén ------------------------------------

// La cola no guarda estado: no abre la base ni habla con `store`. Sin esta
// comprobación, "la cola se deriva del TODO" se rompe en silencio en cuanto
// alguien cachea el estado en SQLite.
func TestQueueNoEscribeEnSQLite(t *testing.T) {
	entradas, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entradas {
		n := e.Name()
		if e.IsDir() || !strings.HasSuffix(n, ".go") || strings.HasSuffix(n, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), n, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parsear %s: %v", n, err)
		}
		for _, imp := range f.Imports {
			r := strings.Trim(imp.Path.Value, `"`)
			if r == "database/sql" || r == "modernc.org/sqlite" || r == "localcli/internal/store" {
				t.Errorf("internal/queue/%s importa %q: la cola no almacena estado propio", n, r)
			}
		}
	}
}

// --- T-B012-02 y T-B012-03: reconstrucción y orden ------------------------

func TestReconstruirOrdenaPorDependencias(t *testing.T) {
	cola, err := Reconstruir(filepath.Join("testdata", "cadena"), task.CapaBackend)
	if err != nil {
		t.Fatalf("Reconstruir: %v", err)
	}
	quiero := "T-B001,T-B002,T-B003,T-B004,T-B005"
	if got := strings.Join(ids(cola.Elementos()), ","); got != quiero {
		t.Fatalf("orden = %s, quiero %s", got, quiero)
	}
	if cola.Capa() != task.CapaBackend {
		t.Errorf("capa = %s", cola.Capa())
	}
}

func TestReconstruirCicloEsError(t *testing.T) {
	if _, err := Reconstruir(filepath.Join("testdata", "ciclo"), task.CapaBackend); err == nil {
		t.Fatal("un TODO con ciclo no se puede ordenar: esperaba error")
	}
}

// --- T-B012-04: bloqueos y su motivo --------------------------------------

func TestBloqueadosPropagaElBloqueo(t *testing.T) {
	cola, err := Reconstruir(filepath.Join("testdata", "cadena"), task.CapaBackend)
	if err != nil {
		t.Fatal(err)
	}
	bloqueos := map[string]Bloqueo{}
	for _, b := range Bloqueados(cola.Elementos()) {
		bloqueos[b.ID] = b
	}
	if _, ok := bloqueos["T-B001"]; ok {
		t.Error("T-B001 está completado: no puede estar bloqueado")
	}
	if _, ok := bloqueos["T-B002"]; ok {
		t.Error("T-B002 solo espera a un elemento completado: no puede estar bloqueado")
	}
	b4, ok := bloqueos["T-B004"]
	if !ok {
		t.Fatal("T-B004 espera a T-B002 y debe estar bloqueado")
	}
	if strings.Join(b4.Falta, ",") != "T-B002" || b4.Motivo == "" {
		t.Errorf("bloqueo de T-B004 = %+v; quiero falta T-B002 y motivo", b4)
	}
	if b4.Codigo() != CodigoElementoBloqueado {
		t.Errorf("código = %s, quiero %s", b4.Codigo(), CodigoElementoBloqueado)
	}
	b5, ok := bloqueos["T-B005"]
	if !ok {
		t.Fatal("T-B005 espera a T-B004 y debe estar bloqueado")
	}
	if !strings.Contains(b5.Motivo, "T-B004") || !strings.Contains(b5.Motivo, "bloqueado") {
		t.Errorf("el motivo de T-B005 debe señalar que su dependencia está bloqueada: %q", b5.Motivo)
	}
}

// --- T-B012-05: siguiente elegible ----------------------------------------

func TestSiguienteDevuelveElPrimeroElegible(t *testing.T) {
	c := consumidorDe(t, "cadena")
	elem, err := c.Siguiente(context.Background())
	if err != nil {
		t.Fatalf("Siguiente: %v", err)
	}
	if elem == nil {
		t.Fatal("con T-B002 pendiente y sin dependencias vivas, la cola no puede estar vacía")
	}
	if elem.ID != "T-B002" {
		t.Fatalf("siguiente = %s, quiero T-B002 (el bloqueado T-B004 no frena a los demás)", elem.ID)
	}
	if !strings.Contains(elem.Objetivo, "T-B002") {
		t.Errorf("objetivo = %q, debe nombrar el elemento", elem.Objetivo)
	}
	if v := c.Vista(); v.Elegible != "T-B002" {
		t.Errorf("la vista dice elegible=%q, quiero T-B002", v.Elegible)
	}
}

func TestSiguienteNilCuandoNadaPuedeArrancar(t *testing.T) {
	c := consumidorDe(t, "detenida")
	elem, err := c.Siguiente(context.Background())
	if err != nil {
		t.Fatalf("Siguiente no debe fallar por un bloqueo (ERRORS.md: la cola sigue con otra): %v", err)
	}
	if elem != nil {
		t.Fatalf("nada puede arrancar, Siguiente devolvió %s", elem.ID)
	}
	if !c.Detenida() {
		t.Error("la cola debe informar de que está detenida con trabajo pendiente")
	}
	// Y el bloqueo declarado en el TODO se informa con su motivo.
	if bs := c.Bloqueados(); len(bs) == 0 {
		t.Error("esperaba al menos un bloqueo informado")
	}
}

func TestColaVacia(t *testing.T) {
	c := consumidorDe(t, "fin")
	elem, err := c.Siguiente(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if elem != nil {
		t.Fatalf("cola vacía, Siguiente devolvió %s", elem.ID)
	}
	if c.Detenida() {
		t.Error("una cola terminada no está detenida: terminó")
	}
	if v := c.Vista(); !v.Vacía() || v.Completados != 2 {
		t.Errorf("vista = %+v, quiero 2 completados y vacía", v)
	}
}

// Un elemento nuevo en el TODO entra en su posición sin reiniciar nada.
func TestRederrivarVeElElementoNuevo(t *testing.T) {
	raiz := copiarTODO(t, "cadena")
	cola, err := Reconstruir(raiz, task.CapaBackend)
	if err != nil {
		t.Fatal(err)
	}
	c, err := cola.Consumir(PermitirAlMotor())
	if err != nil {
		t.Fatal(err)
	}
	main := filepath.Join(raiz, "ai", "tasks", "backend", "MAIN-TASKS.md")
	f, err := os.OpenFile(main, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("| T-B000 | crear | Nueva | — | pendiente | |\n"); err != nil {
		t.Fatal(err)
	}
	f.Close()

	elem, err := c.Siguiente(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if elem == nil || elem.ID != "T-B000" {
		t.Fatalf("el elemento nuevo debe entrar en su posición: %+v", elem)
	}
}

// --- T-B012-06: sincronizar de vuelta al archivo --------------------------

func TestMarcarActualizaLaFilaDelArchivo(t *testing.T) {
	raiz := copiarTODO(t, "cadena")
	cola, err := Reconstruir(raiz, task.CapaBackend)
	if err != nil {
		t.Fatal(err)
	}
	c, err := cola.Consumir(PermitirAlMotor())
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Marcar("T-B002", task.EstadoEnProgreso); err != nil {
		t.Fatalf("Marcar en_progreso: %v", err)
	}
	if err := c.Marcar("T-B002", task.EstadoCompletada); err != nil {
		t.Fatalf("Marcar completada: %v", err)
	}

	main := filepath.Join(raiz, "ai", "tasks", "backend", "MAIN-TASKS.md")
	b, err := os.ReadFile(main)
	if err != nil {
		t.Fatal(err)
	}
	elems, err := task.ParseTabla(b, task.CapaBackend)
	if err != nil {
		t.Fatal(err)
	}
	var visto bool
	for _, e := range elems {
		if e.ID == "T-B002" {
			visto = true
			if e.Estado != task.EstadoCompletada {
				t.Errorf("el archivo dice %s, quiero completada", e.Estado)
			}
		}
	}
	if !visto {
		t.Fatal("no encontré la fila T-B002")
	}
	// Y la cola sigue desde el archivo: el siguiente ya es T-B003.
	elem, err := c.Siguiente(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if elem == nil || elem.ID != "T-B003" {
		t.Fatalf("siguiente = %+v, quiero T-B003", elem)
	}
}

func TestMarcarRechazaElementoAjenoyBloqueo(t *testing.T) {
	raiz := copiarTODO(t, "cadena")
	cola, err := Reconstruir(raiz, task.CapaBackend)
	if err != nil {
		t.Fatal(err)
	}
	c, _ := cola.Consumir(PermitirAlMotor())
	if err := c.Marcar("T-B099", task.EstadoCompletada); err == nil {
		t.Error("un elemento que no está en la cola no se puede marcar")
	}
	if err := c.Marcar("T-B002", task.EstadoBloqueada); err == nil {
		t.Error("marcar bloqueada exigiría escribir bloqueada_por, que el TODO no declara")
	} else if !strings.Contains(err.Error(), "bloqueada_por") {
		t.Errorf("el error debe explicar por qué no se escribe: %v", err)
	}
	if err := c.Marcar("T-B002", task.Estado("inventado")); err == nil {
		t.Error("un estado fuera del catálogo debe rechazarse")
	}
}

// --- T-B012-07: solo el motor consume la cola -----------------------------

func TestConsumirSinPermisoDelMotorSeDeniega(t *testing.T) {
	cola, err := Reconstruir(filepath.Join("testdata", "cadena"), task.CapaBackend)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cola.Consumir(Permiso{}); err == nil {
		t.Error("sin el permiso del motor la cola no se consume")
	}
	if err := cola.LanzarElemento("T-B002"); err == nil {
		t.Error("no hay modo de tarea suelta: lanzar un elemento debe denegarse")
	} else if !strings.Contains(err.Error(), "motor") {
		t.Errorf("el error debe nombrar la regla del motor: %v", err)
	}
}
