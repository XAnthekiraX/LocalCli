// config.go — preferencias del usuario que se recuerdan entre ejecuciones.
//
// Fuente de verdad: FRONTEND.md §3/§4 ("el mapa de teclas vive en
// ~/.config/localcli/keys.json, fuera del proyecto, porque es preferencia del
// usuario y no contenido del proyecto") y SPEC-OLLAMA-PERFIL (paso 6: "El perfil
// queda guardado y se aplica"). Igual que el mapa de teclas, estas preferencias
// son globales del usuario, no de un proyecto: el último modelo y el último
// agente se recuerdan en ~/.config/localcli/config.json.
//
// Un archivo ausente o corrupto nunca rompe el arranque: se devuelven
// preferencias vacías y el arranque decide con sus valores por defecto.
package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Preferencias son los valores recordados entre ejecuciones.
type Preferencias struct {
	// Modelo es el último modelo elegido; al arrancar se usa si sigue instalado.
	Modelo string `json:"ultimo_modelo,omitempty"`
	// Agente es el último agente activo (`plan` o `build`).
	Agente string `json:"ultimo_agente,omitempty"`
	// HistorialTokens es el presupuesto de tokens del historial de conversación
	// que se le envía al modelo (SPEC-HISTORIAL-CONVERSACION). 0 = el arranque
	// usa LOCALCLI_CONTEXT_LIMIT o su valor por defecto.
	HistorialTokens int `json:"historial_tokens,omitempty"`
}

// RutaConfig resuelve la ruta del archivo de preferencias, en la misma carpeta
// que keys.json (FRONTEND.md §3).
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

// CargarPreferenciasDesde lee las preferencias desde un directorio. El agente
// se normaliza: un valor raro cae en `plan`.
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
	p.Agente = ValidarAgente(p.Agente)
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

// ValidarAgente normaliza el agente recordado: solo `plan` o `build`; cualquier
// otro valor (o vacío) cae en `plan`, el de fábrica.
func ValidarAgente(agente string) string {
	if agente == AgenteBuild {
		return AgenteBuild
	}
	return AgentePlan
}
