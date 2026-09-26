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

// El TODO real de TODAS las capas, no solo la de backend. El hueco que dejó
// pasar el bug: `TestCargarElementosReales` solo miraba backend, y la capa
// frontend —que usa la acción `verificar` y tenía 64 filas— no la leía nadie.
// Un vocabulario cerrado que no aguanta una capa entera seRoḱa en silencio.
func TestCargarElementosDeTodasLasCapas(t *testing.T) {
	raiz := raizProyecto(t)

	esperado := map[Capa]int{
		CapaBackend:  16,
		CapaFrontend: 12,
	}
	for capa, want := range esperado {
		elems, err := CargarElementos(raiz, capa)
		if err != nil {
			t.Errorf("capa %s: %v", capa, err)
			continue
		}
		if err := ValidarConjunto(elems); err != nil {
			t.Errorf("capa %s: el TODO real no valida en conjunto: %v", capa, err)
		}
		// Los MAIN-TASKS de cada capa tienen que estar entre los cargados. El
		// ID las delata: un elemento grande es T-B000 (6) y una subtarea es
		// T-B000-01 (9). Contar por longitud era demasiado laxo porque las
		// subtareas de frontend tienen 9, igual que el corte que yo pusiera.
		var delMain int
		for _, e := range elems {
			if len(e.ID) == 6 {
				delMain++
			}
		}
		if delMain != want {
			t.Errorf("capa %s: %d elementos grandes de MAIN-TASKS, queremos %d", capa, delMain, want)
		}
		// Ninguna acción fuera de vocabulario, en ninguna fila.
		for _, e := range elems {
			if !EsAccion(string(e.Accion)) {
				t.Errorf("capa %s: %s tiene la acción %q, fuera de vocabulario", capa, e.ID, e.Accion)
			}
			if !EsEstado(string(e.Estado)) {
				t.Errorf("capa %s: %s tiene el estado %q, fuera de vocabulario", capa, e.ID, e.Estado)
			}
		}
	}
}

// Las dos filas quearchivio el vocabulario: una comprobación no es un cambio, y
// una comprobación tampoco genera diff que revisar.
func TestAccionVerificarNoEsCambio(t *testing.T) {
	if !EsAccion("verificar") {
		t.Fatal("verificar debe ser una acción válida: la usan T-F000-02 y T-F011-05")
	}
	if AccionVerificar.EsCambio() {
		t.Error("verificar no es un cambio: no escribe código ni genera diff")
	}
	for _, a := range []Accion{AccionCrear, AccionActualizar, AccionEliminar} {
		if !a.EsCambio() {
			t.Errorf("%s es una acción de cambio y EsCambio() dijo que no", a)
		}
	}
}

func raizProyecto(t *testing.T) string {
	t.Helper()
	raiz, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return raiz
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

// ---------- IDs, rangos y orden topológico ----------

// Un rango pertenece a una capa. Comparando solo los dígitos, "T-B002..T-B007"
// se daba por cumplido con T-F003 y la dependencia jamás se detectaba rota.
func TestRangoNoAceptaIDsDeOtraCapa(t *testing.T) {
	if dentroDeRango("T-B002..T-B007", "T-F003") {
		t.Error("T-F003 no está dentro de un rango de backend")
	}
	if !dentroDeRango("T-B002..T-B007", "T-B003") {
		t.Error("T-B003 sí está dentro de T-B002..T-B007")
	}
	if !dentroDeRango("T-F002..T-F007", "T-F003") {
		t.Error("T-F003 sí está dentro de un rango de frontend")
	}
	if dentroDeRango("T-B002..T-B007", "T-B009") {
		t.Error("T-B009 está fuera del rango")
	}
	// Un rango que cruza capas no significa nada.
	if dentroDeRango("T-B002..T-F007", "T-B003") {
		t.Error("un rango no puede cruzar de capa")
	}
	if dentroDeRango("basura", "T-B003") {
		t.Error("una dependencia que no es rango no cae dentro de ningún rango")
	}
}

// La regex de IDs no puede estar suelta: "abc1234-x" devolvía el prefijo "234"
// y cualquier fila con tres dígitos pasaba por una tarea legítima.
func TestPrefijoDeIDSoloAceptaLaFormaReal(t *testing.T) {
	buenos := map[string]string{
		"T-B000":    "000",
		"T-B004-02": "004",
		"T-F011-05": "011",
		"T-D001":    "001",
	}
	for id, want := range buenos {
		got, ok := PrefijoDeID(id)
		if !ok || got != want {
			t.Errorf("PrefijoDeID(%q) = %q,%v queremos %q", id, got, ok, want)
		}
		if !IDValido(id) {
			t.Errorf("%q es un ID válido y IDValido lo rechazó", id)
		}
	}
	malos := []string{
		"abc1234-x", "T-B00", "T-B0000", "T-b000", "T-000", "",
		"T-B000-1", "T-B000-001", "T-B000 ", "TX-B000", "T-B000-01-02",
	}
	for _, id := range malos {
		if _, ok := PrefijoDeID(id); ok {
			t.Errorf("PrefijoDeID(%q) aceptó un ID malformado", id)
		}
		if IDValido(id) {
			t.Errorf("IDValido(%q) es true; no tiene la forma T-B000 / T-B000-01", id)
		}
	}
}

// El orden topológico tiene que respetar un rango igual que una dependencia
// simple. Los rangos se saltaban al construir el grafo, así que el nodo con
// rango quedaba con grado 0 y podía Ahead de lo que lo bloquea.
func TestOrdenTopologicoRespetaLosRangos(t *testing.T) {
	elems := []Elemento{
		// T-B010 depende del rango T-B011..T-B012: numerically behind, así que
		// sólo el rango lo puede colocar detrás.
		NuevoElemento("T-B010", CapaBackend, AccionCrear, EstadoPendiente, []string{"T-B011..T-B012"}, nil, nil),
		NuevoElemento("T-B011", CapaBackend, AccionCrear, EstadoPendiente, nil, nil, nil),
		NuevoElemento("T-B012", CapaBackend, AccionCrear, EstadoPendiente, nil, nil, nil),
	}
	orden, err := OrdenarTopologico(elems)
	if err != nil {
		t.Fatal(err)
	}
	pos := map[string]int{}
	for i, e := range orden {
		pos[e.ID] = i
	}
	if pos["T-B010"] < pos["T-B011"] || pos["T-B010"] < pos["T-B012"] {
		t.Errorf("T-B010 depende de T-B011..T-B012 y no puede ir antes: %v", pos)
	}

	// Un rango sin ningún nodo en el conjunto no cuelga: nadie depende de nada.
	solo, err := OrdenarTopologico([]Elemento{
		NuevoElemento("T-B020", CapaBackend, AccionCrear, EstadoPendiente, []string{"T-B099..T-B100"}, nil, nil),
	})
	if err != nil {
		t.Fatalf("un rango vacío no debe ser error: %v", err)
	}
	if len(solo) != 1 {
		t.Errorf("esperaba 1 elemento, tengo %d", len(solo))
	}
}

// El rango se refiere a las tareas grandes. Si|Schwarz tocó también a las
// subtareas, el estado de una subtarea desbloquearía la tarea que depende del
// rango, al revés de lo que dice la fila.
func TestRangoSoloCuentaTareasGrandes(t *testing.T) {
	elems := []Elemento{
		NuevoElemento("T-B030", CapaBackend, AccionCrear, EstadoPendiente, []string{"T-B031..T-B032"}, nil, nil),
		NuevoElemento("T-B031", CapaBackend, AccionCrear, EstadoPendiente, nil, nil, nil),
		NuevoElemento("T-B031-01", CapaBackend, AccionCrear, EstadoCompletada, nil, nil, nil),
	}
	listas, err := Elegibles(elems)
	if err != nil {
		t.Fatal(err)
	}
	var hayB030, hayB031 bool
	for _, e := range listas {
		if e.ID == "T-B030" {
			hayB030 = true
		}
		if e.ID == "T-B031" {
			hayB031 = true
		}
	}
	if hayB030 {
		t.Error("T-B030 depende de T-B031, que sigue pendiente: no debe ser elegible")
	}
	if !hayB031 {
		t.Error("T-B031 no tiene dependencias: sí debe ser elegible")
	}
}

// ---------- Frontera y atomicidad de la escritura ----------

// proyectoConTasks arma un proyecto con ai/tasks/ vacío y devuelve la raíz.
func proyectoConTasks(t *testing.T) string {
	t.Helper()
	raiz := t.TempDir()
	if err := os.MkdirAll(filepath.Join(raiz, "ai", "tasks", "backend"), 0o755); err != nil {
		t.Fatal(err)
	}
	return raiz
}

// Este módulo escribe el TODO y nada más. Sin la comprobación, un id de tarea
// mal construido —o un bug— позволяía escribir cualquier archivo del repo.
func TestEscribirArchivoRechazaFueraDeTasks(t *testing.T) {
	raiz := proyectoConTasks(t)
	prohibidos := []string{
		filepath.Join(raiz, "AGENTS.md"),
		filepath.Join(raiz, "ai", "docs", "PROJECT.md"),
		filepath.Join(raiz, "internal", "task", "write.go"),
		filepath.Join(raiz, "main.go"),
	}
	for _, ruta := range prohibidos {
		if err := EscribirArchivo(raiz, ruta, []byte("x"), 0o644); err == nil {
			t.Errorf("escribió %s, que está fuera de ai/tasks/", ruta)
		}
	}
	// Y sí escribe dentro.
	dentro := filepath.Join(raiz, "ai", "tasks", "backend", "MAIN-TASKS.md")
	if err := EscribirArchivo(raiz, dentro, []byte("ok"), 0o644); err != nil {
		t.Fatalf("no escribió dentro de ai/tasks/: %v", err)
	}
}

// Un prefijo de texto no es una frontera: `ai/tasks-secreto` empieza por
// `ai/tasks`. Por eso la comparación va por filepath.Rel, no por HasPrefix.
func TestFronteraNoConfundePrefijosSimiles(t *testing.T) {
	raiz := t.TempDir()
	for _, d := range []string{"ai/tasks", "ai/tasks-secreto", "ai/tasks2"} {
		if err := os.MkdirAll(filepath.Join(raiz, filepath.FromSlash(d)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	permitido := filepath.Join(raiz, "ai", "tasks")
	for _, falso := range []string{"ai/tasks-secreto/x.md", "ai/tasks2/x.md"} {
		ruta := filepath.Join(raiz, filepath.FromSlash(falso))
		if err := EscribirArchivo(raiz, ruta, []byte("x"), 0o644); err == nil {
			t.Errorf("%s parece ai/tasks/ pero no lo es: debió rechazarse", falso)
		}
	}
	if !dentroDe(filepath.Join(permitido, "a.md"), permitido) {
		t.Error("un archivo directamente dentro de ai/tasks/ sí está dentro")
	}
	if !dentroDe(permitido, permitido) {
		t.Error("el propio directorio está dentro de sí mismo")
	}
	if dentroDe(raiz, permitido) {
		t.Error("la raíz del proyecto no está dentro de ai/tasks/")
	}
}

// `..` sale de la frontera por la puerta de atrás si no se normaliza antes.
func TestEscribirArchivoRechazaTraversal(t *testing.T) {
	raiz := proyectoConTasks(t)
	ataques := []string{
		filepath.Join(raiz, "ai", "tasks", "..", "docs", "PROJECT.md"),
		filepath.Join(raiz, "ai", "tasks", "backend", "..", "..", "..", "..", "tmp", "x.md"),
	}
	for _, ruta := range ataques {
		if err := EscribirArchivo(raiz, ruta, []byte("x"), 0o644); err == nil {
			t.Errorf("el traversal %s no fue bloqueado", ruta)
		}
	}
}

// Un symlink DENTRO de ai/tasks/ que apunta fuera evade cualquier comparación
// de texto sobre la ruta escrita. Hay que resolver el enlace antes de mirar.
func TestEscribirArchivoRechazaSymlinkQueSaleDeTasks(t *testing.T) {
	raiz := proyectoConTasks(t)
	fuera := filepath.Join(raiz, "secreto.md")
	if err := os.WriteFile(fuera, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	enlace := filepath.Join(raiz, "ai", "tasks", "backend", "atajo.md")
	if err := os.Symlink(fuera, enlace); err != nil {
		t.Skipf("symlinks no disponibles: %v", err)
	}
	if err := EscribirArchivo(raiz, enlace, []byte("hackeado"), 0o644); err == nil {
		t.Fatal("escribió a través del symlink y salió de ai/tasks/")
	}
	b, err := os.ReadFile(fuera)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "original" {
		t.Errorf("el archivo de fuera cambió a %q", b)
	}
}

// La escritura es temp+rename: si algo falla, el destino conserva su
// contenido anterior y completo, y no quedan temporales tirados por ahí.
func TestEscribirArchivoNoDejaBasuraNiRompeElDestino(t *testing.T) {
	raiz := proyectoConTasks(t)
	destino := filepath.Join(raiz, "ai", "tasks", "backend", "MAIN-TASKS.md")
	original := []byte("| ID |\n|----|\n| T-B000 |\n")
	if err := EscribirArchivo(raiz, destino, original, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := EscribirArchivo(raiz, destino, []byte("nuevo"), 0o644); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(destino)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "nuevo" {
		t.Errorf("contenido = %q", b)
	}
	// Ni un temporal sin renombrar.
	entradas, err := os.ReadDir(filepath.Dir(destino))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entradas {
		if e.Name() != "MAIN-TASKS.md" {
			t.Errorf("sobro un archivo tras la escritura: %q", e.Name())
		}
	}
	// Los permisos declarados se respetan.
	info, err := os.Stat(destino)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Errorf("permisos = %v, queremos 0644", info.Mode().Perm())
	}
	_ = original
}

// El nombre del directorio distingue mayúsculas de minúsculas solo si el
// sistema de archivos lo hace. La comparación tiene que ser insensible para no
// rechazar escrituras legítimas por un `AI/Tasks`.
func TestFronteraNoDependeDeLasMinusculas(t *testing.T) {
	raiz := t.TempDir()
	if err := os.MkdirAll(filepath.Join(raiz, "ai", "tasks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !dentroDe(filepath.Join(raiz, "ai", "TASKS", "x.md"), filepath.Join(raiz, "ai", "tasks")) {
		t.Error("AI/TASKS es el mismo directorio: la comparación debe ser insensible")
	}
}

// contieneFila recibía la capa fija CapaBackend, así que un ID de frontend
// nunca se encontraba y CambiarEstadoElemento caía siempre en el NNN-task
// equivocado en lugar de decir que no existe.
func TestContieneFilaRespetaLaCapa(t *testing.T) {
	// Cabecera y valores tomados del MAIN-TASKS.md real de frontend: el parser
	// reconoce las columnas por sus alias, no por posición.
	contenido := []byte("| ID | Acción | Estado | Depende de | Bloqueada por | Documentos |\n" +
		"|----|--------|--------|-------------|--------------|------------|\n" +
		"| T-F001 | crear | pendiente | — | — | — |\n")
	if !contieneFila(contenido, "T-F001", CapaFrontend) {
		t.Error("T-F001 debe encontrarse cuando se busca en la capa frontend")
	}
	if contieneFila(contenido, "T-F001", CapaBackend) {
		t.Error("T-F001 es de frontend: buscarla en backend no debe dar un falso positivo")
	}
}

// CambiarEstadoElemento debe encontrar un ID grande de CUALQUIER capa, no solo
// de backend: MAIN-TASKS.md de frontend existe y tiene filas T-F0xx.
func TestCambiarEstadoElementoEnCapaFrontend(t *testing.T) {
	raiz, _ := filepath.Abs(filepath.Join("..", ".."))
	dir := t.TempDir()
	base := filepath.Join(dir, "ai", "tasks", "frontend")
	if err := os.MkdirAll(base, 0o755); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(raiz, "ai", "tasks", "frontend", "MAIN-TASKS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "MAIN-TASKS.md"), b, 0o644); err != nil {
		t.Fatal(err)
	}
	ruta, err := CambiarEstadoElemento(dir, CapaFrontend, "T-F001", EstadoCompletada)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(ruta) != "MAIN-TASKS.md" {
		t.Errorf("T-F001 debio actualizarse en MAIN-TASKS.md, se hizo en %s", ruta)
	}
	nuevo, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	elems, err := ParseTabla(nuevo, CapaFrontend)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range elems {
		if e.ID == "T-F001" {
			if e.Estado != EstadoCompletada {
				t.Errorf("T-F001 quedo en %s", e.Estado)
			}
			return
		}
	}
	t.Error("T-F001 desaparecio de la tabla")
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
