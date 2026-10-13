package tui

import (
	"os"
	"path/filepath"
	"testing"
)

// escribirKeys deja un keys.json con el contenido dado en un directorio temporal.
func escribirKeys(t *testing.T, contenido string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "keys.json"), []byte(contenido), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// Sin archivo no hay error: se arranca con el mapa de fábrica.
func TestSinArchivoSeUsaElMapaDeFabrica(t *testing.T) {
	km, err := CargarKeymapDesde(t.TempDir())
	if err != nil {
		t.Fatalf("un archivo ausente no es un error: %v", err)
	}
	if km.Lider() != LiderPorDefecto || km.TimeoutMs() != TimeoutPorDefectoMs {
		t.Fatalf("la líder y el timeout de fábrica = %q %d", km.Lider(), km.TimeoutMs())
	}
	if accion, ok := BuscarAccion(km.Entradas(), "ctrl+d"); !ok || accion != AccionPanel {
		t.Fatalf("`ctrl+d` debería ser el panel en la vista: %v %v", accion, ok)
	}
}

// Un literal repetido en el mismo contexto se rechaza al cargar: el mapa es la
// fuente del teclado y nacer ambiguo no es opción.
func TestUnLiteralRepetidoEnElMismoContextoSeRechaza(t *testing.T) {
	duplicado := escribirKeys(t, `{"keybinds": {"up": ["ctrl+j"], "down": ["ctrl+j"]}}`)
	if _, err := CargarKeymapDesde(duplicado); err == nil {
		t.Fatal("el duplicado en el mismo contexto debería rechazarse")
	}

	// El mismo literal en dos contextos distintos sí vale: `ctrl+d` pliega el
	// panel en la vista y borra la sesión resaltada en el modal.
	compartido := escribirKeys(t, `{"keybinds": {"panel_toggle": ["ctrl+d"], "session_delete": ["ctrl+d"]}}`)
	km, err := CargarKeymapDesde(compartido)
	if err != nil {
		t.Fatalf("el mismo literal en ámbitos distintos debería valer: %v", err)
	}
	if accion, _ := BuscarAccion(km.Entradas(), "ctrl+d"); accion != AccionPanel {
		t.Fatalf("`ctrl+d` debería seguir siendo el panel: %v", accion)
	}

	// Una acción global pisa cualquier ámbito, así que compartir su literal se
	// rechaza aunque el otro sea de un contexto concreto.
	global := escribirKeys(t, `{"keybinds": {"app_exit": ["ctrl+c"], "send": ["ctrl+c"]}}`)
	if _, err := CargarKeymapDesde(global); err == nil {
		t.Fatal("el literal de una acción global no puede compartirse")
	}
}

// Lo inválido se rechaza entero y con mensaje: ID desconocido y timeout fuera de
// rango no se adivinan.
func TestLoInvalidoSeRechaza(t *testing.T) {
	desconocida := escribirKeys(t, `{"keybinds": {"inventada": ["ctrl+j"]}}`)
	if _, err := CargarKeymapDesde(desconocida); err == nil {
		t.Fatal("una acción desconocida debería rechazarse")
	}
	timeout := escribirKeys(t, `{"leader_timeout_ms": 50}`)
	if _, err := CargarKeymapDesde(timeout); err == nil {
		t.Fatal("un timeout fuera de rango debería rechazarse")
	}
	roto := escribirKeys(t, `{no es json}`)
	if _, err := CargarKeymapDesde(roto); err == nil {
		t.Fatal("un archivo ilegible debería rechazarse")
	}
}

// Reasignar surte efecto en el mapa devuelto y no toca el anterior si el nuevo es
// inválido.
func TestReasignarNoTocaElMapaAnterior(t *testing.T) {
	km := KeymapPorDefecto()
	nuevo, err := km.Reasignar(AccionChatSubir, "ctrl+k")
	if err != nil {
		t.Fatalf("una reasignación válida no debería fallar: %v", err)
	}
	if accion, _ := BuscarAccion(nuevo.Entradas(), "ctrl+k"); accion != AccionChatSubir {
		t.Fatalf("la reasignación no surtió efecto: %v", accion)
	}
	if _, ok := BuscarAccion(km.Entradas(), "ctrl+k"); ok {
		t.Fatal("el mapa anterior no debería haber cambiado")
	}
	if _, err := km.Reasignar(AccionChatSubir, "ctrl+p"); err == nil {
		t.Fatal("un literal ya usado en el mismo contexto debería rechazarse")
	}
	// Una lista vacía deshabilita la acción.
	sinAtajos, err := km.Reasignar(AccionAyuda)
	if err != nil {
		t.Fatalf("deshabilitar una acción no debería fallar: %v", err)
	}
	if _, ok := BuscarAccion(sinAtajos.Entradas(), "ctrl+p"); ok {
		t.Fatal("la acción deshabilitada no debería responder a su tecla")
	}
}

// Guardar y volver a cargar devuelve el mismo mapa: la reasignación queda
// persistida sin reiniciar, y el archivo se escribe con lo que se lee (formato
// `<leader>` incluido).
func TestGuardarYRecargarPersisteElCambio(t *testing.T) {
	dir := t.TempDir()
	km := KeymapPorDefecto()
	nuevo, err := km.Reasignar(AccionPanel, "<leader>s")
	if err != nil {
		t.Fatal(err)
	}
	if err := GuardarKeymapEn(dir, nuevo); err != nil {
		t.Fatalf("no se pudo guardar el mapa: %v", err)
	}

	cargado, err := CargarKeymapDesde(dir)
	if err != nil {
		t.Fatalf("no se pudo volver a cargar el mapa: %v", err)
	}
	if cargado.Lider() != nuevo.Lider() || cargado.TimeoutMs() != nuevo.TimeoutMs() {
		t.Fatalf("líder y timeout no sobrevivieron: %q %d", cargado.Lider(), cargado.TimeoutMs())
	}
	secuencia := Secuencia{Paso1: cargado.Lider(), Paso2: "s"}
	if accion, _ := BuscarAccion(cargado.Entradas(), secuencia.Describir()); accion != AccionPanel {
		t.Fatalf("la reasignación no sobrevivió al archivo: %v", accion)
	}
	// El archivo guarda el marcador `<leader>`, no la tecla ya expandida: cambiar
	// de líder en el archivo no rompe los atajos.
	bruto, err := os.ReadFile(filepath.Join(dir, "keys.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !contieneTexto(string(bruto), "<leader>s") {
		t.Fatalf("el archivo debería guardar el marcador de líder:\n%s", bruto)
	}
}

func contieneTexto(texto, busca string) bool {
	for i := 0; i+len(busca) <= len(texto); i++ {
		if texto[i:i+len(busca)] == busca {
			return true
		}
	}
	return false
}
