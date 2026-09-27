package tui

// Tests de keys (T-F001-05, completada con la persistencia de T-F010-04 y
// ampliada por T-F012-01): ida y vuelta del mapa, archivo ausente, rechazo de
// duplicados al guardar y de acciones desconocidas al cargar, migración del
// esquema antiguo y idempotencia de serializar.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	atajos := AtajosPorDefecto()
	// El usuario mueve el panel a ctrl+y y asigna pausar a ctrl+u.
	atajos = fijarLiteral(atajos, AccionPanel, "ctrl+y")
	atajos = fijarLiteral(atajos, AccionPausar, "ctrl+u")

	if err := GuardarKeysEn(dir, atajos); err != nil {
		t.Fatalf("guardar: %v", err)
	}
	cargados, err := CargarKeysDesde(dir)
	if err != nil {
		t.Fatalf("cargar: %v", err)
	}
	if accion, ok := AccionDe(cargados, "ctrl+y"); !ok || accion != AccionPanel {
		t.Errorf("ctrl+y debe ser panel tras la ida y vuelta: %d, %v", accion, ok)
	}
	if accion, ok := AccionDe(cargados, "ctrl+u"); !ok || accion != AccionPausar {
		t.Errorf("ctrl+u debe ser pausar tras la ida y vuelta: %d, %v", accion, ok)
	}
	// La reasignación no cambia reglas de permiso, solo la invocación: el
	// resto del mapa queda intacto.
	if accion, ok := AccionDe(cargados, "ctrl+c"); !ok || accion != AccionSalir {
		t.Errorf("el resto del mapa se conserva: %d, %v", accion, ok)
	}
	if accion, ok := ResolverSecuencia(cargados, "<leader>l"); !ok || accion != AccionSelector {
		t.Errorf("la secuencia con líder sobrevive a la ida y vuelta: %d, %v", accion, ok)
	}
}

// ResolverSecuencia busca un literal con prefijo `<leader>` en una lista de
// atajos: `AccionDe` solo reconoce literales exactos ya expandidos.
func ResolverSecuencia(entradas []Atajo, literal string) (Accion, bool) {
	seg := Secuencia{Paso2: strings.TrimPrefix(literal, "<leader>")}
	for _, e := range entradas {
		for _, sec := range e.Secuencias {
			if sec.Paso2 == seg.Paso2 && sec.Paso1 == LíderPorDefecto {
				return e.Accion, true
			}
		}
	}
	return AccionNinguna, false
}

func TestCargarSinArchivoDevuelveLosDeFabrica(t *testing.T) {
	cargados, err := CargarKeysDesde(t.TempDir())
	if err != nil {
		t.Fatalf("sin archivo no hay error: %v", err)
	}
	fabrica := AtajosPorDefecto()
	if len(cargados) != len(fabrica) {
		t.Fatalf("devuelve el mapa de fábrica: %d contra %d", len(cargados), len(fabrica))
	}
}

func TestGuardarRechazaElDuplicadoYNoEscribe(t *testing.T) {
	dir := t.TempDir()
	malo := []Atajo{
		{Secuencias: []Secuencia{{Paso1: "ctrl+d"}}, Accion: AccionPanel},
		{Secuencias: []Secuencia{{Paso1: "ctrl+d"}}, Accion: AccionSalir},
	}
	if err := GuardarKeysEn(dir, malo); err == nil {
		t.Fatal("dos acciones con la misma tecla se rechazan al guardar")
	}
	if _, err := os.Stat(filepath.Join(dir, "keys.json")); !os.IsNotExist(err) {
		t.Error("un mapa inválido no deja nada escrito")
	}
}

func TestCargarRechazaAccionesDesconocidas(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "keys.json"), []byte(`{"atajos":{"inventada":"ctrl+z"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := CargarKeysDesde(dir); err == nil {
		t.Error("una acción desconocida invalida el mapa entero, no se adivina")
	}
}

// Un keys.json del esquema antiguo produce el mapa nuevo equivalente: cada
// nombre viejo se traduce a su ID y lo que el archivo no menciona conserva el
// valor de fábrica (T-F012-01).
func TestUnKeysJsonAntiguoProduceElMapaNuevoEquivalente(t *testing.T) {
	dir := t.TempDir()
	antiguo := `{"atajos":{"panel":"ctrl+y","selector":"ctrl+u","cerrar selector":"ctrl+g","ayuda":"ctrl+k"}}`
	if err := os.WriteFile(filepath.Join(dir, "keys.json"), []byte(antiguo), 0o644); err != nil {
		t.Fatal(err)
	}
	km, err := CargarKeymapDesde(dir)
	if err != nil {
		t.Fatalf("un archivo antiguo carga: %v", err)
	}
	for literal, accion := range map[string]Accion{
		"ctrl+y": AccionPanel,
		"ctrl+u": AccionSelector,
		"ctrl+g": AccionCerrarSelector,
		"ctrl+k": AccionAyuda,
	} {
		if obtenido, ok := AccionDe(km.Entradas(), literal); !ok || obtenido != accion {
			t.Errorf("%s conserva su acción: %d, %v", literal, obtenido, ok)
		}
	}
	// Lo ausente se rellena con los valores de fábrica: ninguna acción queda
	// sin atajo por un archivo incompleto.
	if obtenido, ok := AccionDe(km.Entradas(), "ctrl+c"); !ok || obtenido != AccionSalir {
		t.Errorf("lo ausente conserva el atajo de fábrica: %d, %v", obtenido, ok)
	}
}

// Serializar, cargar y volver a serializar escribe lo mismo: el archivo escrito
// es exactamente el que se lee (T-F012-01).
func TestSerializarYDeserializarEsIdempotente(t *testing.T) {
	dir := t.TempDir()
	km, err := NuevoKeymap("ctrl+g", 1500, MapasPorDefecto())
	if err != nil {
		t.Fatal(err)
	}
	if err := GuardarKeymapEn(dir, km); err != nil {
		t.Fatal(err)
	}
	primero, err := os.ReadFile(filepath.Join(dir, "keys.json"))
	if err != nil {
		t.Fatal(err)
	}
	cargado, err := CargarKeymapDesde(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := GuardarKeymapEn(dir, cargado); err != nil {
		t.Fatal(err)
	}
	segundo, err := os.ReadFile(filepath.Join(dir, "keys.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(primero) != string(segundo) {
		t.Errorf("guardar dos veces el mismo mapa escribe lo mismo:\n%s\n---\n%s", primero, segundo)
	}
	if cargado.Lider() != "ctrl+g" {
		t.Errorf("la líder configurada sobrevive: %q", cargado.Lider())
	}
	if cargado.TimeoutMs() != 1500*time.Millisecond {
		t.Errorf("el timeout configurado sobrevive: %v", cargado.TimeoutMs())
	}
	// Cambiar la líder reexpande las secuencias: `<leader>l` pasa a ser ctrl+g l.
	r := NuevoKeyResolver(cargado)
	if accion, _ := r.Resolver(tea.KeyMsg{Type: tea.KeyCtrlG}, ContextoVista); accion != AccionNinguna {
		t.Fatalf("la nueva líder abre la secuencia: %d", accion)
	}
	if accion, _ := r.Resolver(teclaConNombre("l"), ContextoVista); accion != AccionSelector {
		t.Errorf("la secuencia se resuelve con la líder reexpandidida: %d", accion)
	}
}

// El archivo escrito usa el esquema nuevo: clave estable por acción, lista de
// literales y el prefijo `<leader>` sin expandir (T-F012-01).
func TestElArchivoGuardadoUsaElEsquemaNuevo(t *testing.T) {
	dir := t.TempDir()
	if err := GuardarKeymapEn(dir, KeymapPorDefecto()); err != nil {
		t.Fatal(err)
	}
	bruto, err := os.ReadFile(filepath.Join(dir, "keys.json"))
	if err != nil {
		t.Fatal(err)
	}
	var escrito map[string]any
	if err := json.Unmarshal(bruto, &escrito); err != nil {
		t.Fatal(err)
	}
	if escrito["leader"] != LíderPorDefecto {
		t.Errorf("guarda la líder: %v", escrito["leader"])
	}
	if escrito["leader_timeout_ms"] != float64(TimeoutPorDefectoMs) {
		t.Errorf("guarda el timeout: %v", escrito["leader_timeout_ms"])
	}
	keybinds, ok := escrito["keybinds"].(map[string]any)
	if !ok {
		t.Fatalf("guarda los bindings en `keybinds`: %v", escrito)
	}
	if _, hay := escrito["atajos"]; hay {
		t.Error("el esquema antiguo no se sigue escribiendo")
	}
	for _, clave := range []string{"app_exit", "model_picker", "session_picker", "command_palette", "dismiss", "agent_cycle"} {
		if _, ok := keybinds[clave]; !ok {
			t.Errorf("falta el ID estable %q: %v", clave, keybinds)
		}
	}
	modelos, ok := keybinds["model_picker"].([]any)
	if !ok || len(modelos) != 1 || modelos[0] != "<leader>m" {
		t.Errorf("la secuencia se guarda con su prefijo, no expandida: %v", keybinds["model_picker"])
	}
	pausar, ok := keybinds["pause"].([]any)
	if !ok || len(pausar) != 0 {
		t.Errorf("una acción deshabilitada se guarda con lista vacía: %v", keybinds["pause"])
	}
}
