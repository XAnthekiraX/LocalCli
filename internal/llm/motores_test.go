// motores_test.go — T-B037-03: el registro de motores sobre motores.json.
//
// Fuente de verdad: ai/docs/specs/SPEC-MODELO-MOTOR (criterios de
// aceptación: «Sin registro... los dos por defecto», «Un registro ilegible no
// impide arrancar») y ai/docs/backend/05-quality/ERRORS.md §3.
package llm

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// En producción los adaptadores se registran solos al cargarse; aquí se
// registran dobles sin red para que el registro pueda construir por tipo.
func TestMain(m *testing.M) {
	for _, tipo := range []string{TipoOllama, TipoLlamaCPP} {
		tipo := tipo
		RegistrarTipo(tipo, func(url string) Motor {
			return &motorFalso{nombre: tipo, url: url}
		})
	}
	os.Exit(m.Run())
}

// TestRegistroIlegibleArrancaConLosDosPorDefecto — un motores.json mal formado
// se avisa con E_REGISTRO_MOTORES_INVALIDO y se sigue con los dos motores por
// defecto: un registro ilegible nunca deja al usuario sin motor.
func TestRegistroIlegibleArrancaConLosDosPorDefecto(t *testing.T) {
	dir := t.TempDir()
	ruta := filepath.Join(dir, "motores.json")
	if err := os.WriteFile(ruta, []byte("{esto no es json"), 0o644); err != nil {
		t.Fatal(err)
	}

	var reg Registro
	avisos := reg.Cargar(ruta)

	if len(avisos) != 1 {
		t.Fatalf("quería un aviso, llegaron %d: %v", len(avisos), avisos)
	}
	if avisos[0].Codigo != CodigoRegistroMotoresInvalido {
		t.Errorf("el aviso lleva el código %q, quería %q", avisos[0].Codigo, CodigoRegistroMotoresInvalido)
	}
	entradas := reg.Todas()
	if len(entradas) != 2 {
		t.Fatalf("quería los dos motores por defecto, hay %d", len(entradas))
	}
	if entradas[0].Tipo != TipoOllama || entradas[1].Tipo != TipoLlamaCPP {
		t.Errorf("los defaults son ollama y llamacpp, llegó %q y %q", entradas[0].Tipo, entradas[1].Tipo)
	}
	for _, e := range entradas {
		if !e.Activo || e.ID == "" || e.URL == "" {
			t.Errorf("el motor por defecto viene activo, con id y con URL: %+v", e)
		}
	}
}

// TestRegistroAusenteArrancaConLosDosPorDefecto — sin archivo se registran
// `ollama` y `llamacpp` con sus direcciones por defecto y sin aviso.
func TestRegistroAusenteArrancaConLosDosPorDefecto(t *testing.T) {
	var reg Registro
	avisos := reg.Cargar(filepath.Join(t.TempDir(), "motores.json"))

	if len(avisos) != 0 {
		t.Errorf("sin archivo no hay aviso, llegaron: %v", avisos)
	}
	if entradas := reg.Todas(); len(entradas) != 2 ||
		entradas[0].URL != URLPorDefectoOllama || entradas[1].URL != URLPorDefectoLlamaCPP {
		t.Errorf("los dos por defecto con sus URLs: %+v", entradas)
	}
}

// TestSeSaltaLaEntradaDeTipoDesconocido — un tipo fuera del catálogo cerrado
// se rechaza al cargar nombrando la entrada, y las demás se cargan; si no
// queda ninguna, se usa el ollama por defecto.
func TestSeSaltaLaEntradaDeTipoDesconocido(t *testing.T) {
	dir := t.TempDir()
	ruta := filepath.Join(dir, "motores.json")
	bruto, _ := json.Marshal(archivoMotores{Motores: []Entrada{
		{ID: "vllm", Nombre: "VLLM de prueba", Tipo: "vllm", URL: "http://localhost:8000", Activo: true},
		{ID: "buena", Nombre: "La buena", Tipo: TipoOllama, URL: "http://localhost:11434", Activo: true},
	}})
	if err := os.WriteFile(ruta, bruto, 0o644); err != nil {
		t.Fatal(err)
	}

	var reg Registro
	avisos := reg.Cargar(ruta)

	if len(avisos) != 1 || avisos[0].Codigo != CodigoMotorTipoDesconocido {
		t.Fatalf("quería un aviso de tipo desconocido, llegó: %v", avisos)
	}
	if entradas := reg.Todas(); len(entradas) != 1 || entradas[0].ID != "buena" {
		t.Errorf("se salta la desconocida y carga la buena: %+v", entradas)
	}
	if _, ok := reg.PorID("vllm"); ok {
		t.Error("la entrada de tipo desconocido no queda registrada")
	}
}

// TestRegistroSoloTiposDesconocidosCaeAlOllamaPorDefecto — si tras saltar las
// entradas no queda ninguna, se usa el `ollama` por defecto.
func TestRegistroSoloTiposDesconocidosCaeAlOllamaPorDefecto(t *testing.T) {
	dir := t.TempDir()
	ruta := filepath.Join(dir, "motores.json")
	bruto, _ := json.Marshal(archivoMotores{Motores: []Entrada{
		{ID: "otro", Nombre: "Otro runtime", Tipo: "vllm", URL: "http://localhost:8000", Activo: true},
	}})
	if err := os.WriteFile(ruta, bruto, 0o644); err != nil {
		t.Fatal(err)
	}

	var reg Registro
	if avisos := reg.Cargar(ruta); len(avisos) != 1 {
		t.Fatalf("quería el aviso de tipo desconocido, llegó: %v", avisos)
	}
	entradas := reg.Todas()
	if len(entradas) != 1 || entradas[0].Tipo != TipoOllama || !entradas[0].Activo {
		t.Errorf("quería el ollama por defecto, llegó: %+v", entradas)
	}
}

// TestVariasInstanciasDelMismoTipo — el tipo es cerrado, las instancias no:
// dos motores del mismo tipo conviven con nombre y URL propios.
func TestVariasInstanciasDelMismoTipo(t *testing.T) {
	var reg Registro
	reg.ruta = filepath.Join(t.TempDir(), "motores.json")

	if err := reg.Agregar(Entrada{ID: "casa", Nombre: "Ollama casa", Tipo: TipoOllama, URL: "http://localhost:11434", Activo: true}); err != nil {
		t.Fatal(err)
	}
	if err := reg.Agregar(Entrada{ID: "portatil", Nombre: "Ollama portátil", Tipo: TipoOllama, URL: "http://portatil.local:11434", Activo: true}); err != nil {
		t.Fatal(err)
	}

	activos := reg.Activos()
	if len(activos) != 2 || activos[0].URL == activos[1].URL {
		t.Errorf("dos instancias del mismo tipo con URL propias: %+v", activos)
	}
}

// TestAgregarRechazaElTipoFueraDelCatalogo — el catálogo es cerrado: un tipo
// desconocido se rechaza con E_MOTOR_TIPO_DESCONOCIDO y no se registra nada.
func TestAgregarRechazaElTipoFueraDelCatalogo(t *testing.T) {
	var reg Registro
	reg.ruta = filepath.Join(t.TempDir(), "motores.json")

	err := reg.Agregar(Entrada{Nombre: "VLLM", Tipo: "vllm", URL: "http://localhost:8000", Activo: true})
	if err == nil {
		t.Fatal("un tipo fuera del catálogo debe rechazarse")
	}
	if !esErrorMotor(err, CodigoMotorTipoDesconocido) {
		t.Errorf("el error lleva E_MOTOR_TIPO_DESCONOCIDO, llegó: %v", err)
	}
	if len(reg.Todas()) != 0 {
		t.Error("el rechazo no deja entrada a medias")
	}
}

// TestEditarUnMotorAplicadoNoCambiaHastaReiniciar — editar un motor que ya
// tiene adaptador vivo no cambia el comportamiento: el adaptador sigue con la
// configuración con la que se conectó, se marca `pendienteDeReinicio` y el
// cambio se aplica al volver a cargar el registro (DECISIONS).
func TestEditarUnMotorAplicadoNoCambiaHastaReiniciar(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "motores.json")
	var reg Registro
	reg.Cargar(ruta) // sin archivo: los dos por defecto

	// Se aplica el motor: su adaptador vivo queda construido con la URL actual.
	enUso, ok := reg.Adaptador("ollama-local")
	if !ok {
		t.Fatal("el motor activo debe tener adaptador")
	}
	if enUso.BaseURL() != URLPorDefectoOllama {
		t.Fatalf("el adaptador vive con la URL configurada: %q", enUso.BaseURL())
	}

	const nuevaURL = "http://otra-maquina.local:11434"
	if err := reg.Editar(Entrada{ID: "ollama-local", Nombre: "Ollama local", Tipo: TipoOllama, URL: nuevaURL}); err != nil {
		t.Fatal(err)
	}

	// El adaptador en uso NO cambió y el motor queda pendiente de reinicio.
	mismo, _ := reg.Adaptador("ollama-local")
	if mismo.BaseURL() != URLPorDefectoOllama {
		t.Errorf("editar no toca el adaptador vivo: sigue en %q", mismo.BaseURL())
	}
	if !reg.PendienteDeReinicio("ollama-local") {
		t.Error("un motor editado estando aplicado queda pendiente de reinicio")
	}

	// Reinicio: registro nuevo desde el mismo archivo → ya con la URL nueva.
	var trasReinicio Registro
	trasReinicio.Cargar(ruta)
	nuevo, ok := trasReinicio.Adaptador("ollama-local")
	if !ok || nuevo.BaseURL() != nuevaURL {
		t.Errorf("al reiniciar se aplica el cambio: %v, %v", nuevo, ok)
	}
	if trasReinicio.PendienteDeReinicio("ollama-local") {
		t.Error("tras reiniciar ya no hay nada pendiente")
	}
}

// TestDesactivarDestruyeElAdaptadorYReactivarLoReconstruye — desactivar,
// reactivar y eliminar se aplican al momento: el adaptador vivo se destruye o
// se reconstruye, y un motor desactivado no tiene adaptador que usar.
func TestDesactivarDestruyeElAdaptadorYReactivarLoReconstruye(t *testing.T) {
	var reg Registro
	reg.ruta = filepath.Join(t.TempDir(), "motores.json")
	if err := reg.Agregar(Entrada{ID: "m", Nombre: "Motor", Tipo: TipoOllama, URL: URLPorDefectoOllama, Activo: true}); err != nil {
		t.Fatal(err)
	}
	if _, ok := reg.Adaptador("m"); !ok {
		t.Fatal("agregar activo deja su adaptador listo")
	}

	if err := reg.Desactivar("m"); err != nil {
		t.Fatal(err)
	}
	if _, ok := reg.Adaptador("m"); ok {
		t.Error("un motor desactivado no tiene adaptador que usar")
	}

	if err := reg.Reactivar("m"); err != nil {
		t.Fatal(err)
	}
	if _, ok := reg.Adaptador("m"); !ok {
		t.Error("reactivar reconstruye el adaptador al momento")
	}

	if err := reg.Eliminar("m"); err != nil {
		t.Fatal(err)
	}
	if _, ok := reg.Adaptador("m"); ok {
		t.Error("eliminar destruye su adaptador")
	}
}

// TestEditarConservaElIdYElEstado — editar cambia nombre, tipo y URL; el id
// (identidad de las sesiones) y el estado activo no cambian.
func TestEditarConservaElIdYElEstado(t *testing.T) {
	var reg Registro
	reg.ruta = filepath.Join(t.TempDir(), "motores.json")

	if err := reg.Agregar(Entrada{ID: "fijo", Nombre: "Antes", Tipo: TipoOllama, URL: "http://localhost:11434", Activo: true}); err != nil {
		t.Fatal(err)
	}
	if err := reg.Editar(Entrada{ID: "fijo", Nombre: "Después", Tipo: TipoLlamaCPP, URL: "http://localhost:8080", Activo: false}); err != nil {
		t.Fatal(err)
	}

	e, ok := reg.PorID("fijo")
	if !ok {
		t.Fatal("el motor sigue registrado tras editar")
	}
	if e.Nombre != "Después" || e.Tipo != TipoLlamaCPP || e.URL != "http://localhost:8080" {
		t.Errorf("los campos editables cambian: %+v", e)
	}
	if !e.Activo {
		t.Error("editar no cambia el estado: activar es Desactivar/Reactivar")
	}
}

// TestEliminarBorraTambienSuCaché — eliminar borra la configuración y los
// modelos registrados del motor.
func TestEliminarBorraTambienSuCaché(t *testing.T) {
	var reg Registro
	reg.ruta = filepath.Join(t.TempDir(), "motores.json")

	if err := reg.Agregar(Entrada{ID: "temp", Nombre: "Temporal", Tipo: TipoOllama, URL: "http://localhost:11434", Activo: true}); err != nil {
		t.Fatal(err)
	}
	if err := reg.RefrescarModelos("temp", []string{"qwen3:8b"}); err != nil {
		t.Fatal(err)
	}
	if err := reg.Eliminar("temp"); err != nil {
		t.Fatal(err)
	}

	if _, ok := reg.PorID("temp"); ok {
		t.Error("eliminar quita la entrada")
	}
	if modelos := reg.Modelos("temp"); len(modelos) != 0 {
		t.Errorf("eliminar borra sus modelos registrados: %v", modelos)
	}
}

// TestDesactivarNoBorraYReactivarRecupera — desactivar deja el motor visible
// con sus modelos; reactivar lo devuelve con el catálogo conservado, sin
// volver a agregarlo.
func TestDesactivarNoBorraYReactivarRecupera(t *testing.T) {
	var reg Registro
	reg.ruta = filepath.Join(t.TempDir(), "motores.json")

	if err := reg.Agregar(Entrada{ID: "m", Nombre: "Motor", Tipo: TipoOllama, URL: "http://localhost:11434", Activo: true}); err != nil {
		t.Fatal(err)
	}
	if err := reg.RefrescarModelos("m", []string{"qwen3:8b", "gemma3:12b"}); err != nil {
		t.Fatal(err)
	}
	if err := reg.Desactivar("m"); err != nil {
		t.Fatal(err)
	}

	if len(reg.Activos()) != 0 {
		t.Error("desactivado, no queda entre los activos")
	}
	e, ok := reg.PorID("m")
	if !ok || e.Activo {
		t.Errorf("desactivar no borra: la entrada sigue visible y desactivada (%+v)", e)
	}
	if modelos := reg.Modelos("m"); len(modelos) != 2 {
		t.Errorf("desactivar conserva los modelos registrados: %v", modelos)
	}

	if err := reg.Reactivar("m"); err != nil {
		t.Fatal(err)
	}
	if len(reg.Activos()) != 1 || len(reg.Modelos("m")) != 2 {
		t.Error("reactivar recupera el motor con su catálogo, sin volver a agregarlo")
	}
}

// TestElRegistroSeConservaEnElArchivo — lo que guarda el registro vuelve a
// cargarse igual: id, nombre, tipo, url, activo y la caché de modelos.
func TestElRegistroSeConservaEnElArchivo(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "motores.json")

	var reg Registro
	reg.ruta = ruta
	if err := reg.Agregar(Entrada{ID: "id-estable", Nombre: "Mi motor", Tipo: TipoLlamaCPP, URL: "http://localhost:8081", Activo: true}); err != nil {
		t.Fatal(err)
	}
	if err := reg.RefrescarModelos("id-estable", []string{"qwen3-14b-q4"}); err != nil {
		t.Fatal(err)
	}

	var recargada Registro
	if avisos := recargada.Cargar(ruta); len(avisos) != 0 {
		t.Fatalf("una vuelta limpia no avisa: %v", avisos)
	}
	e, ok := recargada.PorID("id-estable")
	if !ok || e.Nombre != "Mi motor" || e.Tipo != TipoLlamaCPP || e.URL != "http://localhost:8081" || !e.Activo {
		t.Errorf("la entrada sobrevive la vuelta: %+v", e)
	}
	if modelos := recargada.Modelos("id-estable"); len(modelos) != 1 || modelos[0] != "qwen3-14b-q4" {
		t.Errorf("la caché de modelos sobrevive la vuelta: %v", modelos)
	}
}

// esErrorMotor comprueba que `err` es un ErrorMotor con ese código.
func esErrorMotor(err error, codigo string) bool {
	e, ok := err.(*ErrorMotor)
	return ok && e.Codigo == codigo
}
