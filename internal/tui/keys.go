package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// archivoKeys es la forma del JSON en disco. `keybinds` es la lista de literales
// por acción; además viajan la líder y su timeout (SPEC-KEYBINDS §Configuración).
type archivoKeys struct {
	Lider     string              `json:"leader,omitempty"`
	TimeoutMs int                 `json:"leader_timeout_ms,omitempty"`
	Keybinds  map[string][]string `json:"keybinds,omitempty"`
}

// accionDeNombre resuelve un identificador estable a acción. La lista es abierta:
// añadir una acción añade su nombre en nombreDeAccion.
func accionDeNombre(s string) (Accion, bool) {
	for _, a := range OrdenDeAcciones() {
		if nombreDeAccion(a) == s {
			return a, true
		}
	}
	return AccionNinguna, false
}

// RutaKeys resuelve la ruta del archivo de preferencias fuera del proyecto:
// ~/.config/localcli/keys.json (FRONTEND.md §3).
func RutaKeys() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("tui: no se pudo resolver la carpeta del usuario: %w", err)
	}
	return filepath.Join(home, ".config", "localcli", "keys.json"), nil
}

// CargarKeymap carga el mapa completo desde la ruta del usuario.
func CargarKeymap() (*Keymap, error) {
	ruta, err := RutaKeys()
	if err != nil {
		return nil, err
	}
	return CargarKeymapDesde(filepath.Dir(ruta))
}

// CargarKeymapDesde carga el mapa desde un directorio. Sin archivo no hay error:
// se devuelve el mapa de fábrica. Lo ausente se rellena con los valores de
// fábrica y lo inválido (ID desconocido, literal vacío, duplicado, timeout fuera
// de rango) se rechaza entero, no se adivina.
func CargarKeymapDesde(dir string) (*Keymap, error) {
	fabrica := MapasPorDefecto()
	bruto, err := os.ReadFile(filepath.Join(dir, "keys.json"))
	if os.IsNotExist(err) {
		return NuevoKeymap(LiderPorDefecto, TimeoutPorDefectoMs, fabrica)
	}
	if err != nil {
		return nil, err
	}
	var archivo archivoKeys
	if err := json.Unmarshal(bruto, &archivo); err != nil {
		return nil, err
	}

	porAccion := map[Accion][]string{}
	for accion, literales := range fabrica {
		porAccion[accion] = append([]string{}, literales...)
	}
	// keybinds con listas. Una lista vacía deshabilita la acción; una ausente
	// conserva su valor de fábrica.
	for nombre, literales := range archivo.Keybinds {
		accion, ok := accionDeNombre(nombre)
		if !ok {
			return nil, fmt.Errorf("tui: keys.json nombra una acción desconocida: %s", nombre)
		}
		lista := append([]string{}, literales...)
		if lista == nil {
			lista = []string{}
		}
		porAccion[accion] = lista
	}

	return NuevoKeymap(archivo.Lider, archivo.TimeoutMs, porAccion)
}

// GuardarKeymap guarda el mapa completo (líder, timeout y keybinds) en la ruta
// del usuario.
func GuardarKeymap(km *Keymap) error {
	ruta, err := RutaKeys()
	if err != nil {
		return err
	}
	return GuardarKeymapEn(filepath.Dir(ruta), km)
}

// GuardarKeymapEn guarda el mapa completo en un directorio, creándolo si no
// existe. Lo que se escribe es exactamente lo que se lee (idempotencia).
func GuardarKeymapEn(dir string, km *Keymap) error {
	archivo := archivoKeys{
		Lider:     km.Lider(),
		TimeoutMs: km.TimeoutMs(),
		Keybinds:  map[string][]string{},
	}
	for _, e := range km.Entradas() {
		archivo.Keybinds[nombreDeAccion(e.Accion)] = literalesDe(e)
	}
	return escribirArchivoKeys(dir, archivo)
}

// escribirArchivoKeys serializa y escribe keys.json con los permisos de siempre.
func escribirArchivoKeys(dir string, archivo archivoKeys) error {
	bruto, err := serializarKeys(archivo)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "keys.json"), bruto, 0o644)
}

// serializarKeys compone el JSON con la notación `<leader>` tal cual: el archivo
// es del usuario y se edita a mano, y el escapado de HTML que hace el paquete
// estándar lo dejaría en `\u003cleader\u003e`, que se lee peor sin ganar nada.
func serializarKeys(archivo archivoKeys) ([]byte, error) {
	var b bytes.Buffer
	codificador := json.NewEncoder(&b)
	codificador.SetEscapeHTML(false)
	codificador.SetIndent("", "  ")
	if err := codificador.Encode(archivo); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// Reasignar deja una acción con exactamente estos literales y devuelve el mapa
// resultante, ya validado. Una lista vacía deshabilita la acción. El mapa
// anterior no se toca si el nuevo es inválido.
func (k *Keymap) Reasignar(accion Accion, literales ...string) (*Keymap, error) {
	porAccion := map[Accion][]string{}
	for _, e := range k.entradas {
		porAccion[e.Accion] = literalesDe(e)
	}
	porAccion[accion] = append([]string{}, literales...)
	return NuevoKeymap(k.lider, k.timeoutMs, porAccion)
}
