package task

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

const raizTestdata = "testdata"

// T-B018-02: el parser reconoce las filas y su vocabulario.
func TestParseTablaReconoceFilas(t *testing.T) {
	contenido := []byte("# x\n\n| ID     | Acción | Tarea | Dep    | Estado    |\n" +
		"| ------ | ------ | ----- | ------ | --------- |\n" +
		"| T-B001 | crear  | uno   | —      | pendiente |\n" +
		"| T-B002 | crear  | dos   | T-B001 | completada |\n")
	elems, err := ParseTabla(contenido, CapaBackend)
	if err != nil {
		t.Fatalf("ParseTabla: %v", err)
	}
	if len(elems) != 2 {
		t.Fatalf("elementos = %d, queremos 2", len(elems))
	}
	if elems[0].ID != "T-B001" || len(elems[0].DependeDe) != 0 {
		t.Errorf("T-B001 inesperado: %+v", elems[0])
	}
	if want := []string{"T-B001"}; !reflect.DeepEqual(elems[1].DependeDe, want) {
		t.Errorf("dependencias T-B002 = %v, queremos %v", elems[1].DependeDe, want)
	}
}

// T-B018-02: una fila malformada es un error, no se descarta en silencio.
func TestParseTablaFilaMalformadaEsError(t *testing.T) {
	contenido := []byte("| ID     | Acción | Tarea | Estado    |\n" +
		"| ------ | ------ | ----- | --------- |\n" +
		"| T-B001 | crear  | uno   | inventado |\n")
	if _, err := ParseTabla(contenido, CapaBackend); err == nil {
		t.Fatal("esperaba error por estado fuera del catálogo")
	}
}

// T-B018-01: el round-trip JSON conserva los 7 campos del frontmatter. Se usa
// un elemento bloqueado porque `bloqueada_por` solo viaja con estado bloqueada
// ([[backend/DECISIONS]] [28]).
func TestRoundTripJSON(t *testing.T) {
	e := NuevoElemento("T-B001-01", CapaBackend, AccionCrear, EstadoBloqueada,
		[]string{"T-B001"}, []string{"T-B000"}, []string{"[[backend/BACKEND]]"})
	b, err := e.AJSON()
	if err != nil {
		t.Fatalf("AJSON: %v", err)
	}
	var crudo map[string]json.RawMessage
	if err := json.Unmarshal(b, &crudo); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, campo := range CamposDevueltos {
		if _, ok := crudo[campo]; !ok {
			t.Errorf("falta el campo %q en el JSON", campo)
		}
	}
	vuelta, err := FromJSON(b)
	if err != nil {
		t.Fatalf("FromJSON: %v", err)
	}
	if !reflect.DeepEqual(e, vuelta) {
		t.Errorf("round-trip = %+v, queremos %+v", vuelta, e)
	}
}

// T-B018-04: `bloqueada_por` solo es válido con estado bloqueada.
func TestValidarReglaBloqueo(t *testing.T) {
	sinMotivo := NuevoElemento("T-B001", CapaBackend, AccionCrear, EstadoBloqueada, nil, nil, nil)
	if err := ValidarReglaBloqueo(sinMotivo); err == nil {
		t.Error("estado bloqueada sin bloqueada_por debía fallar")
	}
	malEstado := NuevoElemento("T-B001", CapaBackend, AccionCrear, EstadoPendiente, nil, []string{"T-B002"}, nil)
	if err := ValidarReglaBloqueo(malEstado); err == nil {
		t.Error("bloqueada_por con estado pendiente debía fallar")
	}
	ok := NuevoElemento("T-B001", CapaBackend, AccionCrear, EstadoBloqueada, nil, []string{"T-B002"}, nil)
	if err := ValidarReglaBloqueo(ok); err != nil {
		t.Errorf("bloqueo correcto no debía fallar: %v", err)
	}
}

// T-B018-05: el orden respeta las dependencias y es determinista.
func TestOrdenTopologicoRespetaDependencias(t *testing.T) {
	elems := []Elemento{
		NuevoElemento("T-B002", CapaBackend, AccionCrear, EstadoPendiente, []string{"T-B001"}, nil, nil),
		NuevoElemento("T-B001", CapaBackend, AccionCrear, EstadoCompletada, nil, nil, nil),
	}
	orden, err := OrdenarTopologico(elems)
	if err != nil {
		t.Fatalf("OrdenarTopologico: %v", err)
	}
	if orden[0].ID != "T-B001" || orden[1].ID != "T-B002" {
		t.Errorf("orden = %s,%s; queremos T-B001,T-B002", orden[0].ID, orden[1].ID)
	}
}

// T-B018-05: un ciclo de dependencias es un error.
func TestOrdenTopologicoCicloEsError(t *testing.T) {
	elems := []Elemento{
		NuevoElemento("T-B001", CapaBackend, AccionCrear, EstadoPendiente, []string{"T-B002"}, nil, nil),
		NuevoElemento("T-B002", CapaBackend, AccionCrear, EstadoPendiente, []string{"T-B001"}, nil, nil),
	}
	if _, err := OrdenarTopologico(elems); err == nil {
		t.Fatal("esperaba error por ciclo")
	}
}

// T-B018-05: un rango de una capa no cuenta para otra.
func TestRangoNoCruzaDeCapa(t *testing.T) {
	if dentroDeRango("T-B002..T-B004", "T-F003") {
		t.Error("un ID de frontend no debe caer en un rango de backend")
	}
	if !dentroDeRango("T-B002..T-B004", "T-B003") {
		t.Error("T-B003 debe caer en T-B002..T-B004")
	}
}

// T-B018-05: solo los elegibles con dependencias cumplidas salen, y un
// elemento empezado se retoma.
func TestElegibles(t *testing.T) {
	elems := []Elemento{
		NuevoElemento("T-B001", CapaBackend, AccionCrear, EstadoCompletada, nil, nil, nil),
		NuevoElemento("T-B002", CapaBackend, AccionCrear, EstadoPendiente, []string{"T-B001"}, nil, nil),
		NuevoElemento("T-B003", CapaBackend, AccionCrear, EstadoEnProgreso, []string{"T-B002"}, nil, nil),
	}
	eleg, err := Elegibles(elems)
	if err != nil {
		t.Fatalf("Elegibles: %v", err)
	}
	if len(eleg) != 1 || eleg[0].ID != "T-B002" {
		t.Errorf("elegibles = %+v, queremos solo T-B002", eleg)
	}
}

// T-B018-06: solo cambia la celda Estado; el resto del archivo queda igual.
func TestActualizarEstadoSoloTocaLaCelda(t *testing.T) {
	original := "# TODO\n\n" +
		"| ID     | Acción | Tarea | Estado    |\n" +
		"| ------ | ------ | ----- | --------- |\n" +
		"| T-B001 | crear  | uno   | pendiente |\n" +
		"\ntexto de cola\n"
	nuevo, err := ActualizarEstadoEnContenido([]byte(original), "T-B001", EstadoEnProgreso)
	if err != nil {
		t.Fatalf("ActualizarEstadoEnContenido: %v", err)
	}
	if strings.Contains(string(nuevo), "pendiente") {
		t.Errorf("el estado viejo sigue presente:\n%s", nuevo)
	}
	if !strings.Contains(string(nuevo), "en_progreso") {
		t.Errorf("el estado nuevo no aparece:\n%s", nuevo)
	}
	// El resto del archivo byte a byte: quitando la fila de estado, todo igual.
	if !strings.Contains(string(nuevo), "texto de cola") || !strings.HasPrefix(string(nuevo), "# TODO\n") {
		t.Errorf("se tocó algo fuera de la celda:\n%s", nuevo)
	}
}

// T-B018-06: una fila que no existe es un error.
func TestActualizarEstadoFilaInexistente(t *testing.T) {
	original := "| ID     | Estado    |\n| ------ | --------- |\n| T-B001 | pendiente |\n"
	if _, err := ActualizarEstadoEnContenido([]byte(original), "T-B099", EstadoCompletada); err == nil {
		t.Fatal("esperaba error por fila inexistente")
	}
}

// T-B018-03: la carga junta el índice y las tareas pequeñas de la capa.
func TestCargarElementos(t *testing.T) {
	elems, err := CargarElementos(raizTestdata, CapaBackend)
	if err != nil {
		t.Fatalf("CargarElementos: %v", err)
	}
	ids := map[string]bool{}
	for _, e := range elems {
		ids[e.ID] = true
	}
	for _, q := range []string{"T-B002", "T-B003", "T-B002-01", "T-B002-02", "T-B003-01"} {
		if !ids[q] {
			t.Errorf("falta %s en la carga", q)
		}
	}
	if err := ValidarConjunto(elems); err != nil {
		t.Errorf("ValidarConjunto: %v", err)
	}
}

// T-B018-06: la escritura atómica exige caer dentro de ai/tasks/.
func TestEscribirArchivoFueraDelTODO(t *testing.T) {
	raiz := t.TempDir()
	if err := EscribirArchivo(raiz, raiz+"/fuera.md", []byte("x"), 0o644); err == nil {
		t.Fatal("esperaba error al escribir fuera de ai/tasks/")
	}
}
