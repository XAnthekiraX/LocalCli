package task

// Tests del módulo task (T-B004-07). Cubren las verificaciones pedidas en
// 004-task-task.md subtarea por subtarea:
//
//	01 → round-trip JSON conserva los 7 campos
//	02 → parsea el MAIN-TASKS.md real sin perder filas (+ fixture con wiki-links)
//	03 → localiza los archivos NNN-task-*.md de la capa
//	04 → cambiar solo el estado no altera otras celdas
//	05 → bloqueada_por fuera de estado bloqueada devuelve error
//	06 → orden topológico estable sobre fixture
//	07 → este propio fichero: go test ./internal/task/... pasa

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------- T-B004-01: model.go ----------

func TestRoundTripJSONConservaCampos(t *testing.T) {
	e := NuevoElemento("T-B004", CapaBackend, AccionCrear, EstadoEnProgreso,
		[]string{"T-B001"}, nil, []string{"backend/DECISIONS", "specs/SPEC-CICLO-TRABAJO"})
	b, err := e.AJSON()
	if err != nil {
		t.Fatal(err)
	}
	var crudo map[string]any
	if err := json.Unmarshal(b, &crudo); err != nil {
		t.Fatal(err)
	}
	for _, campo := range []string{"id", "capa", "accion", "estado", "depende_de", "documentos"} {
		if _, ok := crudo[campo]; !ok {
			t.Errorf("falta el campo %q en el JSON (%s)", campo, b)
		}
	}
	got, err := FromJSON(b)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != e.ID || got.Capa != e.Capa || got.Accion != e.Accion || got.Estado != e.Estado {
		t.Fatalf("round-trip perdió escalares: %+v vs %+v", got, e)
	}
	if strings.Join(got.DependeDe, ",") != "T-B001" || strings.Join(got.Documentos, ",") != "backend/DECISIONS,specs/SPEC-CICLO-TRABAJO" {
		t.Fatalf("round-trip perdió listas: %+v", got)
	}
	if got.BloqueadaPor != nil {
		t.Errorf("bloqueada_por debe quedar omitido/vacío sin bloqueo: %v", got.BloqueadaPor)
	}
}

func TestFromJSONRechazaVocabularioInvalido(t *testing.T) {
	malo := `{"id":"X","capa":"backend","accion":"destruir","estado":"pendiente","depende_de":[],"documentos":[]}`
	if _, err := FromJSON([]byte(malo)); err == nil {
		t.Fatal("esperaba error por acción inválida")
	}
}

// ---------- T-B004-02: parse.go ----------

func TestParseaMainTasksRealSinPerderFilas(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "ai", "tasks", "backend", "MAIN-TASKS.md"))
	if err != nil {
		t.Fatal(err)
	}
	elems, err := ParseTabla(b, CapaBackend)
	if err != nil {
		t.Fatal(err)
	}
	if len(elems) != 16 {
		t.Fatalf("MAIN-TASKS.md tiene 16 tareas (T-B000..T-B015), parseé %d", len(elems))
	}
	if elems[0].ID != "T-B000" || elems[15].ID != "T-B015" {
		t.Fatalf("orden de aparición roto: %s … %s", elems[0].ID, elems[15].ID)
	}
	// La fila T-B008 declara dos dependencias separadas por coma.
	var b8 Elemento
	for _, e := range elems {
		if e.ID == "T-B008" {
			b8 = e
		}
	}
	if strings.Join(b8.DependeDe, "+") != "T-B002+T-B007" {
		t.Errorf("dependencias de T-B008 mal parseadas: %v", b8.DependeDe)
	}
	if len(b8.Documentos) == 0 || b8.Documentos[0] != "008-task-fileops.md" {
		t.Errorf("documentos de T-B008 mal parseados: %v", b8.Documentos)
	}
	// "—" significa sin dependencias.
	if elems[0].DependeDe != nil {
		t.Errorf("T-B000 no debe tener dependencias, tengo %v", elems[0].DependeDe)
	}
}

func TestParseFixtureConWikiLinksYEstados(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("testdata", "ai", "tasks", "backend", "MAIN-TASKS.md"))
	if err != nil {
		t.Fatal(err)
	}
	elems, err := ParseTabla(b, CapaBackend)
	if err != nil {
		t.Fatal(err)
	}
	if len(elems) != 5 {
		t.Fatalf("fixture: esperaba 5 elementos, tengo %d", len(elems))
	}
	if elems[4].Accion != AccionActualizar || elems[4].Estado != EstadoPendiente {
		t.Errorf("T-B004 fixture: accion=%s estado=%s", elems[4].Accion, elems[4].Estado)
	}
	if strings.Join(elems[4].DependeDe, ",") != "T-B000..T-B003" {
		t.Errorf("rango de dependencias mal conservado: %v", elems[4].DependeDe)
	}
	// Archivo de tarea pequeña: la sección Referencias (sin columna ID) no
	// interrumpe el parseo de la tabla posterior.
	b2, _ := os.ReadFile(filepath.Join("testdata", "ai", "tasks", "backend", "002-task-a.md"))
	elems2, err := ParseTabla(b2, CapaBackend)
	if err != nil {
		t.Fatal(err)
	}
	if len(elems2) != 2 || elems2[1].Estado != EstadoEnProgreso {
		t.Fatalf("002-task-a.md: %+v", elems2)
	}
}

func TestParseFilaMalformadaEsError(t *testing.T) {
	malo := "| ID | Acción | Tarea | Estado |\n|---|---|---|---|\n| T-X001-01 | inventar | x | pendiente |\n"
	if _, err := ParseTabla([]byte(malo), CapaBackend); err == nil {
		t.Fatal("esperaba error por acción inválida, no descarte silencioso")
	}
}

// ---------- T-B004-03: locate.go ----------

func TestArchivosTODOEncuentraPrefijos(t *testing.T) {
	raiz, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}
	archivos, err := ArchivosTODO(raiz, CapaBackend)
	if err != nil {
		t.Fatal(err)
	}
	// MAIN primero + 002 y 003 ordenadas por prefijo.
	nombres := []string{filepath.Base(archivos[0]), filepath.Base(archivos[1]), filepath.Base(archivos[2])}
	if strings.Join(nombres, ",") != "MAIN-TASKS.md,002-task-a.md,003-task-b.md" {
		t.Fatalf("orden/listado incorrecto: %v", nombres)
	}
	ruta, err := ArchivoTareaPara(raiz, CapaBackend, "T-B002-01")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(ruta) != "002-task-a.md" {
		t.Errorf("T-B002-01 debería mapear a 002-task-a.md, obtuve %s", ruta)
	}
	if _, err := ArchivoTareaPara(raiz, CapaBackend, "T-B009-01"); err == nil {
		t.Error("esperaba error para un prefijo inexistente")
	}
}

func TestCapaDesdeRutaYPrefijo(t *testing.T) {
	capa, err := CapaDesdeRuta("proyecto/ai/tasks/backend/004-task-task.md")
	if err != nil || capa != CapaBackend {
		t.Fatalf("capa=%v err=%v", capa, err)
	}
	if _, err := CapaDesdeRuta("src/main.go"); err == nil {
		t.Error("esperaba error fuera de ai/tasks/")
	}
	for id, want := range map[string]string{"T-B004": "004", "T-B004-02": "004", "T-F012-03": "012"} {
		got, ok := PrefijoDeID(id)
		if !ok || got != want {
			t.Errorf("PrefijoDeID(%q)=%q,%v; quería %q", id, got, ok, want)
		}
	}
}

func TestCargarElementosReales(t *testing.T) {
	raiz, _ := filepath.Abs(".." + string(filepath.Separator) + "..")
	elems, err := CargarElementos(raiz, CapaBackend)
	if err != nil {
		t.Fatal(err)
	}
	// 16 del MAIN + todas las pequeñas de los NNN existentes (≥7 de 004).
	if len(elems) < 16+7 {
		t.Fatalf("CargarElementos devolvió pocos elementos: %d", len(elems))
	}
	if err := ValidarConjunto(elems); err != nil {
		t.Fatalf("el TODO real no valida en conjunto: %v", err)
	}
}

// ---------- T-B004-04: write.go ----------

func TestCambiarEstadoSoloTocaLaCeldaEstado(t *testing.T) {
	src := filepath.Join("testdata", "ai", "tasks", "backend", "002-task-a.md")
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	nuevo, err := ActualizarEstadoEnContenido(b, "T-B002-01", EstadoCompletada)
	if err != nil {
		t.Fatal(err)
	}
	lineasAntes := strings.Split(string(b), "\n")
	lineasDespues := strings.Split(string(nuevo), "\n")
	if len(lineasAntes) != len(lineasDespues) {
		t.Fatal("el número de líneas cambió")
	}
	diferentes := 0
	for i := range lineasAntes {
		if lineasAntes[i] != lineasDespues[i] {
			diferentes++
			if !strings.Contains(lineasAntes[i], "T-B002-01") {
				t.Errorf("línea inesperadamente modificada:\n  antes: %s\n  ahora: %s", lineasAntes[i], lineasDespues[i])
			}
			// Solo la celda Estado (columna 4) cambia.
			cA := dividirFila(strings.TrimSpace(lineasAntes[i]))
			cB := dividirFila(strings.TrimSpace(lineasDespues[i]))
			if len(cA) != len(cB) {
				t.Fatal("cambió el número de celdas")
			}
			for j := range cA {
				if cA[j] != cB[j] && j != 3 {
					t.Errorf("celda %d alterada: %q → %q", j, cA[j], cB[j])
				}
			}
		}
	}
	if diferentes != 1 {
		t.Errorf("esperaba exactamente 1 línea distinta, tuve %d", diferentes)
	}
	// Y el resultado vuelve a parsearse con el estado nuevo.
	elems, err := ParseTabla(nuevo, CapaBackend)
	if err != nil {
		t.Fatal(err)
	}
	if elems[0].Estado != EstadoCompletada {
		t.Errorf("tras escribir, el estado leído es %s", elems[0].Estado)
	}
}

func TestCambiarEstadoElementoEscribeEnDisco(t *testing.T) {
	dir := t.TempDir()
	// Copiamos el fixture al temporal para no tocar testdata.
	base := filepath.Join(dir, "ai", "tasks", "backend")
	if err := os.MkdirAll(base, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"MAIN-TASKS.md", "002-task-a.md", "003-task-b.md"} {
		b, _ := os.ReadFile(filepath.Join("testdata", "ai", "tasks", "backend", n))
		if err := os.WriteFile(filepath.Join(base, n), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ruta, err := CambiarEstadoElemento(dir, CapaBackend, "T-B002", EstadoCompletada)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(ruta) != "MAIN-TASKS.md" {
		t.Errorf("T-B002 (tarea grande) debía actualizarse en MAIN, se hizo en %s", ruta)
	}
	ruta2, err := CambiarEstadoElemento(dir, CapaBackend, "T-B002-02", EstadoCompletada)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(ruta2) != "002-task-a.md" {
		t.Errorf("T-B002-02 debía actualizarse en 002-task-a.md, se hizo en %s", ruta2)
	}
	if _, err := CambiarEstadoElemento(dir, CapaBackend, "T-B099", EstadoPendiente); err == nil {
		t.Error("esperaba error para un id inexistente")
	}
}

// ---------- T-B004-05: validate.go ----------

func TestBloqueadaPorSoloConEstadoBloqueada(t *testing.T) {
	malo := NuevoElemento("T-B001", CapaBackend, AccionCrear, EstadoPendiente,
		nil, []string{"T-B000"}, nil)
	if err := ValidarElemento(malo); err == nil {
		t.Fatal("violación no detectada: bloqueada_por con estado pendiente")
	} else if !strings.Contains(err.Error(), "bloqueada_por") {
		t.Fatalf("mensaje de error inesperado: %v", err)
	}
	ok := NuevoElemento("T-B001", CapaBackend, AccionCrear, EstadoBloqueada,
		nil, []string{"T-B000"}, nil)
	if err := ValidarElemento(ok); err != nil {
		t.Fatalf("elemento bloqueado válido rechazado: %v", err)
	}
	sinMotivo := NuevoElemento("T-B001", CapaBackend, AccionCrear, EstadoBloqueada,
		nil, nil, nil)
	if err := ValidarElemento(sinMotivo); err == nil {
		t.Fatal("bloqueada sin bloqueada_por debe ser error")
	}
}

func TestValidarConjuntoDetectaDuplicadosYDependenciasRotas(t *testing.T) {
	elems := []Elemento{
		NuevoElemento("T-B001", CapaBackend, AccionCrear, EstadoPendiente, []string{"T-B999"}, nil, nil),
	}
	if err := ValidarConjunto(elems); err == nil || !strings.Contains(err.Error(), "T-B999") {
		t.Fatalf("esperaba error de dependencia inexistente, obtuve %v", err)
	}
	dup := []Elemento{
		NuevoElemento("T-B001", CapaBackend, AccionCrear, EstadoPendiente, nil, nil, nil),
		NuevoElemento("T-B001", CapaBackend, AccionCrear, EstadoPendiente, nil, nil, nil),
	}
	if err := ValidarConjunto(dup); err == nil {
		t.Fatal("esperaba error de id duplicado")
	}
}

// ---------- T-B004-06: deps.go ----------

func TestOrdenTopologicoEstableSobreFixture(t *testing.T) {
	b, _ := os.ReadFile(filepath.Join("testdata", "ai", "tasks", "backend", "MAIN-TASKS.md"))
	elems, err := ParseTabla(b, CapaBackend)
	if err != nil {
		t.Fatal(err)
	}
	orden, err := OrdenarTopologico(elems)
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, e := range orden {
		ids = append(ids, e.ID)
	}
	want := "T-B000,T-B001,T-B002,T-B003,T-B004"
	if strings.Join(ids, ",") != want {
		t.Fatalf("orden topológico estable:\n que­ría %s\n tuve  %s", want, strings.Join(ids, ","))
	}
	// Estabilidad: dos pasadas dan el mismo orden.
	orden2, _ := OrdenarTopologico(elems)
	for i := range orden {
		if orden[i].ID != orden2[i].ID {
			t.Fatal("orden no determinista")
		}
	}
}

func TestElegiblesRespetaCompletadas(t *testing.T) {
	elems := []Elemento{
		NuevoElemento("T-B000", CapaBackend, AccionCrear, EstadoCompletada, nil, nil, nil),
		NuevoElemento("T-B001", CapaBackend, AccionCrear, EstadoCompletada, []string{"T-B000"}, nil, nil),
		NuevoElemento("T-B002", CapaBackend, AccionCrear, EstadoPendiente, []string{"T-B001"}, nil, nil),
		NuevoElemento("T-B003", CapaBackend, AccionCrear, EstadoPendiente, []string{"T-B002"}, nil, nil),
	}
	el, err := Elegibles(elems)
	if err != nil {
		t.Fatal(err)
	}
	if len(el) != 1 || el[0].ID != "T-B002" {
		t.Fatalf("elegibles = %v; quería solo T-B002", el)
	}
}

func TestCicloDetectado(t *testing.T) {
	elems := []Elemento{
		NuevoElemento("T-B001", CapaBackend, AccionCrear, EstadoPendiente, []string{"T-B002"}, nil, nil),
		NuevoElemento("T-B002", CapaBackend, AccionCrear, EstadoPendiente, []string{"T-B001"}, nil, nil),
	}
	if _, err := OrdenarTopologico(elems); err == nil || !strings.Contains(err.Error(), "ciclo") {
		t.Fatalf("esperaba error de ciclo, obtuve %v", err)
	}
}
