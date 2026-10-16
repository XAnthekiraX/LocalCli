package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"localcli/internal/tui/render"
)

// RutaEstilos resuelve la ruta de la paleta del usuario fuera del proyecto:
// ~/.config/localcli/styles.json (FRONTEND.md §3).
func RutaEstilos() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("tui: no se pudo resolver la carpeta del usuario: %w", err)
	}
	return filepath.Join(home, ".config", "localcli", "styles.json"), nil
}

// CargarEstilos aplica la paleta del usuario a los estilos de la vista.
func CargarEstilos() error {
	ruta, err := RutaEstilos()
	if err != nil {
		return err
	}
	return CargarEstilosDesde(filepath.Dir(ruta))
}

// CargarEstilosDesde aplica la paleta desde un directorio. Sin archivo no hay
// error: se siembra `styles.json` con los valores de fábrica, para que exista y
// sea editable desde el primer arranque, y se usan esos. Un archivo inválido
// —JSON roto, token desconocido o color mal formado— se rechaza entero, no se
// adivina, igual que el mapa de teclas.
func CargarEstilosDesde(dir string) error {
	ruta := filepath.Join(dir, "styles.json")
	bruto, err := os.ReadFile(ruta)
	if os.IsNotExist(err) {
		return sembrarEstilos(dir)
	}
	if err != nil {
		return err
	}
	tokens := map[string]string{}
	if err := json.Unmarshal(bruto, &tokens); err != nil {
		return err
	}
	return render.AplicarTokens(tokens)
}

// sembrarEstilos escribe `styles.json` con la paleta de fábrica.
func sembrarEstilos(dir string) error {
	return escribirEstilos(dir, render.EstilosPorDefecto())
}

// escribirEstilos serializa la paleta y la escribe con los permisos de siempre.
func escribirEstilos(dir string, tokens map[string]string) error {
	bruto, err := serializarEstilos(tokens)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "styles.json"), bruto, 0o644)
}

// serializarEstilos compone el JSON con indentación, sin escapar HTML: el
// archivo es del usuario y se edita a mano.
func serializarEstilos(tokens map[string]string) ([]byte, error) {
	var b bytes.Buffer
	codificador := json.NewEncoder(&b)
	codificador.SetEscapeHTML(false)
	codificador.SetIndent("", "  ")
	if err := codificador.Encode(tokens); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
