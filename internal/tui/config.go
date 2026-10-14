package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Preferencias son los valores recordados entre ejecuciones: el último agente,
// el presupuesto del historial y el razonamiento. El modelo activo no vive
// aquí: lo resuelve el catálogo (`models.json`), que es la autoridad al
// arrancar. Son preferencia del usuario, no contenido del proyecto, y viven en
// ~/.config/localcli/config.json.
type Preferencias struct {
	// Agente es el último agente activo. Se guarda tal cual; la vista lo valida
	// contra su lista de agentes disponibles al arrancar.
	Agente string `json:"ultimo_agente,omitempty"`
	// HistorialTokens es el presupuesto de tokens del historial de conversación
	// que se le entrega al modelo. Sin él se usa `LOCALCLI_CONTEXT_LIMIT` y, en
	// su defecto, un valor por defecto. Ver [[specs/SPEC-HISTORIAL-CONVERSACION]].
	HistorialTokens int `json:"historial_tokens,omitempty"`
	// Pensar es si el chat deja razonar al modelo antes de responder.
	Pensar bool `json:"pensar,omitempty"`
	// Compactacion configura el gestor de contexto y la compactación automática.
	// Sin ella se usan los valores por defecto. Ver
	// [[specs/SPEC-HISTORIAL-CONVERSACION]].
	Compactacion *Compactacion `json:"compactacion,omitempty"`
}

// Compactacion es la configuración del gestor de contexto. Todos los campos son
// opcionales: los ceros se resuelven con los valores por defecto a través de los
// métodos `*Efectivo`. Vive dentro de `config.json`, en
// `~/.config/localcli/` ([[specs/SPEC-HISTORIAL-CONVERSACION]] §Disparadores y configuración).
type Compactacion struct {
	// Auto decide si la compactación se dispara sola al superar el umbral o ante
	// un desbordamiento de ventana. Puntero para distinguir «no dicho» de «false».
	Auto *bool `json:"auto,omitempty"`
	// Umbral es la fracción de la ventana de contexto a partir de la cual se
	// compacta el historial (0 < umbral ≤ 1). Por defecto 0.60.
	Umbral float64 `json:"umbral,omitempty"`
	// ConservarTokens es cuántos tokens recientes se entregan literales tras
	// compactar; el resto se resume. Por defecto 15000.
	ConservarTokens int `json:"conservar_tokens,omitempty"`
	// Modelo es el modelo que produce los resúmenes; vacío usa el modelo activo.
	Modelo string `json:"modelo,omitempty"`
	// Formato elige la plantilla del resumen: "opencode" (estructurado, por
	// defecto), "legacy" (resumen libre) o "custom" (usa `Plantilla`).
	Formato string `json:"formato,omitempty"`
	// Plantilla es el prompt de compactación cuando `Formato` es "custom".
	Plantilla string `json:"plantilla,omitempty"`
}

// Valores por defecto de la compactación ([[specs/SPEC-HISTORIAL-CONVERSACION]]).
const (
	// UmbralCompactacionPorDefecto compacta al 60 % de la ventana.
	UmbralCompactacionPorDefecto = 0.60
	// ConservarTokensCompactacionPorDefecto es el tramo reciente literal.
	ConservarTokensCompactacionPorDefecto = 15000
	// FormatoCompactacionPorDefecto es el resumen estructurado estilo OpenCode.
	FormatoCompactacionPorDefecto = "opencode"
)

// AutoActivo dice si la compactación automática está encendida; sin preferencia,
// sí.
func (c *Compactacion) AutoActivo() bool {
	if c == nil || c.Auto == nil {
		return true
	}
	return *c.Auto
}

// UmbralEfectivo devuelve la fracción de la ventana para compactar, acotada a
// (0, 1]; sin preferencia, el 60 %.
func (c *Compactacion) UmbralEfectivo() float64 {
	if c == nil || c.Umbral <= 0 || c.Umbral > 1 {
		return UmbralCompactacionPorDefecto
	}
	return c.Umbral
}

// ConservarEfectivo devuelve cuántos tokens recientes viajan literales; sin
// preferencia, 15000.
func (c *Compactacion) ConservarEfectivo() int {
	if c == nil || c.ConservarTokens <= 0 {
		return ConservarTokensCompactacionPorDefecto
	}
	return c.ConservarTokens
}

// FormatoEfectivo devuelve el formato del resumen; sin preferencia, "opencode".
func (c *Compactacion) FormatoEfectivo() string {
	if c == nil || c.Formato == "" {
		return FormatoCompactacionPorDefecto
	}
	return c.Formato
}

// ModeloDeCompactacion devuelve el modelo configurado para resumir, o vacío si
// debe usarse el activo.
func (c *Compactacion) ModeloDeCompactacion() string {
	if c == nil {
		return ""
	}
	return c.Modelo
}

// PlantillaDeCompactacion devuelve la plantilla del formato "custom".
func (c *Compactacion) PlantillaDeCompactacion() string {
	if c == nil {
		return ""
	}
	return c.Plantilla
}

// RutaConfig resuelve la ruta del archivo de preferencias, en la misma carpeta
// que keys.json.
func RutaConfig() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("tui: no se pudo resolver la carpeta del usuario: %w", err)
	}
	return filepath.Join(home, ".config", "localcli", "config.json"), nil
}

// CargarPreferencias lee el archivo del usuario. Sin archivo, o con uno
// ilegible, devuelve preferencias vacías: no es un error que deba tumbar el
// arranque.
func CargarPreferencias() (Preferencias, error) {
	ruta, err := RutaConfig()
	if err != nil {
		return Preferencias{}, err
	}
	return CargarPreferenciasDesde(filepath.Dir(ruta))
}

// CargarPreferenciasDesde lee las preferencias desde un directorio. El agente se
// devuelve tal cual: la vista lo normaliza contra su lista al construirse.
func CargarPreferenciasDesde(dir string) (Preferencias, error) {
	bruto, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if os.IsNotExist(err) {
		return Preferencias{}, nil
	}
	if err != nil {
		return Preferencias{}, err
	}
	var p Preferencias
	if err := json.Unmarshal(bruto, &p); err != nil {
		return Preferencias{}, nil
	}
	return p, nil
}

// GuardarPreferencias escribe el archivo del usuario, creando la carpeta si no
// existe.
func GuardarPreferencias(p Preferencias) error {
	ruta, err := RutaConfig()
	if err != nil {
		return err
	}
	return GuardarPreferenciasEn(filepath.Dir(ruta), p)
}

// GuardarPreferenciasEn escribe las preferencias en un directorio. Serializar y
// volver a leer es idempotente.
func GuardarPreferenciasEn(dir string, p Preferencias) error {
	bruto, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "config.json"), bruto, 0o644)
}

// ValidarAgente devuelve `agente` si está entre los disponibles; si no (o si
// viene vacío), el primero de la lista. Sin lista, `plan`.
func ValidarAgente(agente string, disponibles []string) string {
	for _, n := range disponibles {
		if n == agente {
			return agente
		}
	}
	if len(disponibles) > 0 {
		return disponibles[0]
	}
	return AgentePlan
}
