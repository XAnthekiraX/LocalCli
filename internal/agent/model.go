package agent

// model.go — T-B034-01 y T-B034-02: el tipo `Agente` y su contrato YAML.
//
// Fuente de verdad: ai/docs/specs/SPEC-AGENTE-BASE.md §Dónde se definen
// (`agent.yaml` con `name`, `description` y `permissions`, más `prompt.md` con
// las instrucciones) y ai/docs/backend/01-domain/BUSINESS_RULES.md §Agentes.
//
// El agente es una CARPETA con dos archivos que viajan por caminos distintos:
// `agent.yaml` gobierna el harness —qué herramientas existen para este agente—
// y `prompt.md` gobierna al modelo. El prompt NO viaja en el YAML: se lee de su
// propio archivo.
//
// `permissions` es un mapa de permiso a efecto con solo tres permisos (`read`,
// `write`, `edit`). El catálogo efectivo de herramientas NO se declara: se
// DERIVA de los permisos contra el catálogo cerrado de `tools`. `default` solo
// admite `deny` y, ausente, equivale a `deny`: lo que no se nombre no se
// concede. Por eso `plan` no escribe: sus permisos no conceden `write` ni
// `edit`, y la garantía vive en los datos, no en el código.
//
// La decodificación es estricta: rechaza cualquier campo que no sea `name`,
// `description` o `permissions`, incluidos `prompt`, `herramientas`, `skills`,
// `mode` y `hereda_de`.

import (
	"bytes"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

// Efectos de un permiso en el `agent.yaml`.
const (
	EfectoAllow = "allow"
	EfectoDeny  = "deny"
)

// ClaveDefault es la clave opcional de `permissions`. Su único valor admitido
// es `deny`; ausente equivale a `deny`.
const ClaveDefault = "default"

// CamposDelAgente son los nombres de campo admitidos en `agent.yaml`. Es la
// lista cerrada del contrato; sirve para documentar y para los tests.
var CamposDelAgente = []string{"name", "description", "permissions"}

// Agente es una definición cargada desde una carpeta de `.localcli/agents/`.
// `Permissions` es su política; `Prompt` es su identidad (leída de `prompt.md`)
// y `Herramientas` el catálogo efectivo que de ella se deriva en la carga.
type Agente struct {
	Nombre      string            `yaml:"name"`
	Descripcion string            `yaml:"description"`
	Permissions map[string]string `yaml:"permissions,omitempty"`
	// Prompt es la identidad del agente. NO se decodifica del `agent.yaml`:
	// lo rellena la carga leyendo `prompt.md` entero.
	Prompt string `yaml:"-"`
	// Herramientas es el catálogo EFECTIVO, derivado de Permissions. No viaja
	// en el YAML: lo rellena la carga. Es la lista que `tools` comprueba y la
	// que sostiene la garantía de que `plan` no escribe.
	Herramientas []string `yaml:"-"`
}

// SoloConversacion informa si el agente no tiene ninguna herramienta efectiva.
// Un agente así no tiene ruta hacia `tools`: solo conversa (SPEC-AGENTE-BASE).
func (a Agente) SoloConversacion() bool { return len(a.Herramientas) == 0 }

// DecodificarAgente convierte el YAML de un `agent.yaml` en un Agente,
// rechazando campos desconocidos (un `hereda_de`, un `prompt`, un
// `herramientas:` o un `skills:` no pasan) y derivando el catálogo efectivo de
// los permisos. NO exige `prompt.md`: eso lo hace la carga.
func DecodificarAgente(datos []byte) (Agente, error) {
	var a Agente
	dec := yaml.NewDecoder(bytes.NewReader(datos))
	dec.KnownFields(true)
	if err := dec.Decode(&a); err != nil {
		return Agente{}, err
	}
	// Un segundo documento tampoco es un contrato válido.
	var sobra yaml.Node
	if err := dec.Decode(&sobra); err != io.EOF {
		return Agente{}, errAgente("el archivo lleva contenido de más después del objeto")
	}
	if err := a.validar(); err != nil {
		return Agente{}, err
	}
	herramientas, err := HerramientasDe(a.Permissions)
	if err != nil {
		return Agente{}, err
	}
	a.Herramientas = herramientas
	return a, nil
}

// validar comprueba lo que el YAML debe traer: `name` no puede faltar. El
// prompt no se comprueba aquí —no está en el YAML—: lo hace la carga.
func (a Agente) validar() error {
	if strings.TrimSpace(a.Nombre) == "" {
		return errAgente("falta el campo obligatorio `name`")
	}
	return nil
}
