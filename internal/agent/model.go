package agent

// model.go — T-B006-01: el tipo `Agente` con los cinco campos exactos.
//
// Fuente de verdad: ai/docs/specs/SPEC-AGENTE-BASE.md §Dónde se definen
// ("campos fijos: nombre, descripcion, prompt, herramientas y skills") y
// ai/docs/backend/DECISIONS.md ("JSON de agente con cinco campos: nombre,
// descripcion, prompt, herramientas, skills; sin herencia entre agentes").
//
// Los campos son exactamente cinco y no hay herencia: qué puede hacer un
// agente se lee en un solo archivo, sin seguir cadenas. Por eso la
// decodificación es estricta y rechaza cualquier campo que no sea uno de los
// cinco, `hereda_de` incluido: un campo desconocido haría que el archivo
// dijera algo que el motor no entiende, y eso es peor que fallar.

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
)

// CamposDelAgente son los cinco nombres de campo admitidos. Es la lista
// cerrada del contrato; sirve para documentar y para los tests.
var CamposDelAgente = []string{"nombre", "descripcion", "prompt", "herramientas", "skills"}

// Agente es una definición cargada desde `ai/agents/*.json`. El prompt es la
// identidad del agente; `Herramientas` es su catálogo y la garantía de
// permisos: `plan` no lista ninguna de escritura y por eso no puede escribir.
type Agente struct {
	Nombre       string   `json:"nombre"`
	Descripcion  string   `json:"descripcion"`
	Prompt       string   `json:"prompt"`
	Herramientas []string `json:"herramientas"`
	Skills       []string `json:"skills"`
}

// SoloConversacion informa si el agente no declara ninguna herramienta. Un
// agente así no tiene ruta hacia `tools`: solo conversa (SPEC-AGENTE-BASE).
func (a Agente) SoloConversacion() bool { return len(a.Herramientas) == 0 }

// DecodificarAgente convierte el JSON de un agente en un Agente, rechazando
// campos desconocidos (un `hereda_de` no pasa) y comprobando que los campos
// obligatorios están.
func DecodificarAgente(datos []byte) (Agente, error) {
	var a Agente
	dec := json.NewDecoder(bytes.NewReader(datos))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&a); err != nil {
		return Agente{}, err
	}
	// Datos sobrantes tras el objeto también son un archivo mal formado.
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return Agente{}, errAgente("el archivo lleva contenido de más después del objeto")
	}
	if err := a.validar(); err != nil {
		return Agente{}, err
	}
	return a, nil
}

// validar comprueba lo que el JSON debe traer. `herramientas` y `skills`
// pueden ir vacíos, pero no ausentes de significado: `nombre` y `prompt` no.
func (a Agente) validar() error {
	if strings.TrimSpace(a.Nombre) == "" {
		return errAgente("falta el campo obligatorio `nombre`")
	}
	if strings.TrimSpace(a.Prompt) == "" {
		return errAgente("el agente " + a.Nombre + " no tiene `prompt`")
	}
	return nil
}
