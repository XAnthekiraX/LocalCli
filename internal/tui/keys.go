package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
)

// Accion identifica una acción reasignable del teclado de la TUI.
type Accion string

// Acciones reasignables por defecto, según [[frontend/02-interfaces/INTERFACES]] §4.
const (
	AccionPanel        Accion = "panel"          // Abrir o cerrar el panel de datos.
	AccionSelector     Accion = "selector"       // Abrir el selector de sesiones.
	AccionRazonamiento Accion = "razonamiento"   // Mostrar u ocultar el razonamiento.
	AccionAprobaciones Accion = "aprobaciones"   // Abrir el panel de aprobaciones.
	AccionCancelar     Accion = "cancelar_flujo" // Cancelar el flujo en curso.
	AccionAyuda        Accion = "ayuda"          // Ayuda de atajos.
	AccionSalir        Accion = "salir"          // Salir.
)

// Binding es la descripción serializable de una pulsación: tecla y modificadores.
type Binding struct {
	Ctrl  bool   `json:"ctrl"`
	Alt   bool   `json:"alt"`
	Shift bool   `json:"shift"`
	Key   string `json:"key"` // Rún simple ("?") o nombre ("d", "q", "enter").
}

// String devuelve el binding legible, p.ej. "ctrl+d" o "?".
func (b Binding) String() string {
	s := ""
	if b.Ctrl {
		s += "ctrl+"
	}
	if b.Alt {
		s += "alt+"
	}
	if b.Shift && b.Key != "" && len([]rune(b.Key)) == 1 {
		s += "shift+"
	}
	return s + b.Key
}

// KeyMap es el mapa de teclas reasignable: acción → binding.
type KeyMap map[Accion]Binding

// DefaultKeyMap devuelve los siete atajos por defecto documentados.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		AccionPanel:        {Ctrl: true, Key: "d"},
		AccionSelector:     {Ctrl: true, Key: "s"},
		AccionRazonamiento: {Ctrl: true, Key: "r"},
		AccionAprobaciones: {Ctrl: true, Key: "a"},
		AccionCancelar:     {Ctrl: true, Key: "f"},
		AccionAyuda:        {Key: "?"},
		AccionSalir:        {Ctrl: true, Key: "q"},
	}
}

// Matches informa si un mensaje de teclado coincide con el binding de una acción.
func (km KeyMap) Matches(accion Accion, msg tea.Msg) bool {
	msgKey, ok := msg.(tea.KeyMsg)
	if !ok {
		return false
	}
	b, ok := km[accion]
	if !ok {
		return false
	}
	return matchesBinding(b, msgKey)
}

func matchesBinding(b Binding, k tea.KeyMsg) bool {
	if k.Paste {
		return false
	}
	if k.Alt != b.Alt {
		return false
	}
	name := keyName(k) // p.ej. "ctrl+d", "d", "?", "f1"
	want := lowerASCII(b.Key)
	if b.Shift {
		// Con shift documentado, la pulsación llega como mayúscula/símbolo.
		want = upperASCII(want)
	}
	if b.Ctrl != strings.HasPrefix(name, "ctrl+") {
		return false
	}
	return name == want
}

// keyName normaliza el nombre de la tecla presionada a minúsculas, sin el
// prefijo "alt+" (los ctrl+letra ya llegan como "ctrl+d" desde Key.String()).
func keyName(k tea.KeyMsg) string {
	s := k.String()
	if k.Alt {
		s = s[len("alt+"):] // el prefijo ya se comparó vía k.Alt
	}
	return lowerASCII(s)
}

func lowerASCII(s string) string {
	out := []byte(s)
	for i := range out {
		if out[i] >= 'A' && out[i] <= 'Z' {
			out[i] += 'a' - 'A'
		}
	}
	return string(out)
}

func upperASCII(s string) string {
	out := []byte(s)
	for i := range out {
		if out[i] >= 'a' && out[i] <= 'z' {
			out[i] -= 'a' - 'A'
		}
	}
	return string(out)
}

// FindDuplicate devuelve la otra acción con la que un binding choca, o nil.
func (km KeyMap) FindDuplicate(b Binding) (Accion, bool) {
	for a, existing := range km {
		if sameBinding(existing, b) {
			return a, true
		}
	}
	return "", false
}

func sameBinding(x, y Binding) bool {
	return x.Ctrl == y.Ctrl && x.Alt == y.Alt && x.Shift == y.Shift &&
		lowerASCII(x.Key) == lowerASCII(y.Key)
}

// configPath devuelve la ruta XDG ~/.config/localcli/keys.json.
func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("keys: no se pudo resolver el HOME: %w", err)
	}
	return filepath.Join(home, ".config", "localcli", "keys.json"), nil
}

// LoadKeyMap lee el mapa de keys.json; si el archivo no existe, devuelve los
// valores por defecto sin error. Un duplicado en el archivo invalida la carga.
func LoadKeyMap() (KeyMap, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return DefaultKeyMap(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("keys: no se pudo leer %s: %w", path, err)
	}
	km := DefaultKeyMap()
	if err := json.Unmarshal(data, &km); err != nil {
		return nil, fmt.Errorf("keys: JSON inválido en %s: %w", path, err)
	}
	if a, dup := km.findInternalDuplicate(); dup {
		return nil, fmt.Errorf("keys: atajo duplicado asignado a %q en %s", a, path)
	}
	return km, nil
}

// findInternalDuplicate busca dos acciones distintas con el mismo binding.
func (km KeyMap) findInternalDuplicate() (Accion, bool) {
	seen := map[string]Accion{}
	for a, b := range km {
		k := canonical(b)
		if prev, ok := seen[k]; ok && prev != a {
			return a, true
		}
		seen[k] = a
	}
	return "", false
}

func canonical(b Binding) string {
	return fmt.Sprintf("%t|%t|%t|%s", b.Ctrl, b.Alt, b.Shift, lowerASCII(b.Key))
}

// SaveKeyMap persiste el mapa en ~/.config/localcli/keys.json creando el
// directorio y rechazando asignaciones duplicadas antes de escribir.
func SaveKeyMap(km KeyMap) error {
	if a, dup := km.findInternalDuplicate(); dup {
		return fmt.Errorf("keys: no se puede guardar: atajo duplicado en %q", a)
	}
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("keys: no se pudo crear el directorio: %w", err)
	}
	data, err := json.MarshalIndent(km, "", "  ")
	if err != nil {
		return fmt.Errorf("keys: no se pudo serializar el mapa: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("keys: no se pudo escribir %s: %w", path, err)
	}
	return nil
}
