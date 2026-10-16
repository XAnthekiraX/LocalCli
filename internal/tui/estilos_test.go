package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"localcli/internal/tui/render"
)

// escribirEstilosArchivo deja un styles.json con el contenido dado en un
// directorio temporal.
func escribirEstilosArchivo(t *testing.T, contenido string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "styles.json"), []byte(contenido), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// restaurarEstilos devuelve los estilos de fábrica al terminar la prueba: la
// paleta es estado global del paquete de render.
func restaurarEstilos(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		if err := render.AplicarTokens(render.EstilosPorDefecto()); err != nil {
			t.Fatalf("no se pudieron restaurar los estilos: %v", err)
		}
	})
}

// Sin archivo no hay error: se siembra styles.json con la paleta de fábrica.
func TestEstilosSinArchivoSeSiembra(t *testing.T) {
	restaurarEstilos(t)
	dir := t.TempDir()
	if err := CargarEstilosDesde(dir); err != nil {
		t.Fatalf("un archivo ausente no es un error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "styles.json")); err != nil {
		t.Fatalf("styles.json debería sembrarse en el primer arranque: %v", err)
	}
	if render.ColorAgente != lipgloss.Color(render.EstilosPorDefecto()["color_agente"]) {
		t.Fatalf("sin archivo debería usarse la paleta de fábrica")
	}
}

// Un token válido del usuario se aplica sobre la paleta de fábrica.
func TestEstilosDelUsuarioSeAplican(t *testing.T) {
	restaurarEstilos(t)
	dir := escribirEstilosArchivo(t, `{"color_agente": "#000000", "fondo_codigo": "#010203"}`)
	if err := CargarEstilosDesde(dir); err != nil {
		t.Fatalf("una paleta válida debería aplicarse: %v", err)
	}
	if render.ColorAgente != lipgloss.Color("#000000") {
		t.Fatalf("color_agente = %q, quiero #000000", render.ColorAgente)
	}
	if render.FondoCodigo != lipgloss.Color("#010203") {
		t.Fatalf("fondo_codigo = %q, quiero #010203", render.FondoCodigo)
	}
	// Lo que la paleta no nombra conserva su valor de fábrica.
	if render.ColorUsuario != lipgloss.Color(render.EstilosPorDefecto()["color_usuario"]) {
		t.Fatalf("color_usuario debería conservar el valor de fábrica")
	}
}

// Un token desconocido o un color mal formado se rechazan enteros y no tocan los
// estilos vigentes.
func TestEstilosInvalidosSeRechazan(t *testing.T) {
	restaurarEstilos(t)
	casos := map[string]string{
		"token desconocido": `{"color_inventado": "#000000"}`,
		"color mal formado": `{"color_agente": "verde"}`,
		"JSON roto":         `{no es json}`,
	}
	for nombre, contenido := range casos {
		t.Run(nombre, func(t *testing.T) {
			dir := escribirEstilosArchivo(t, contenido)
			if err := CargarEstilosDesde(dir); err == nil {
				t.Fatalf("debería rechazarse: %s", nombre)
			}
			if render.ColorAgente != lipgloss.Color(render.EstilosPorDefecto()["color_agente"]) {
				t.Fatalf("un archivo inválido no debería cambiar los estilos vigentes")
			}
		})
	}
}
