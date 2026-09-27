package agent

// model.go — T-B006-01: el tipo `Agente` y su contrato JSON.
//
// Fuente de verdad: ai/docs/specs/SPEC-AGENTE-BASE.md §Dónde se definen
// (campos fijos: `nombre`, `descripcion`, `prompt`, `permisos` y `skills`) y
// ai/docs/backend/DECISIONS.md ("JSON de agente con cinco campos; sin herencia
// entre agentes").
//
// El agente declara PERMISOS por acción (`leer`/`editar`/`ejecutar`/`internet`)
// con efecto `permitir`/`denegar`. El catálogo efectivo de herramientas NO se
// declara: se DERIVA de los permisos contra el catálogo cerrado de `tools`.
// Así no hay dos listas que puedan contradecirse: `plan` no permite `editar`,
// y por eso no tiene ninguna herramienta de escritura. La garantía vive en los
// datos, no en el código.
//
// Los campos son exactamente cinco y no hay herencia: qué puede hacer un
// agente se lee en un solo archivo. Por eso la decodificación es estricta y
// rechaza cualquier campo que no sea uno de los cinco, `hereda_de` incluido
// (y también el viejo `herramientas`: ahora la fuente es `permisos`).

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
)

// Efectos de un permiso. `permitir` habilita una acción; `denegar` la deja
// fuera (y es el valor por defecto de toda acción ausente).
const (
	EfectoPermitir = "permitir"
	EfectoDenegar  = "denegar"
)

// Permiso declara qué puede hacer un agente con una acción del catálogo.
type Permiso struct {
	Accion string `json:"accion"`
	Efecto string `json:"efecto"`
}

// CamposDelAgente son los cinco nombres de campo admitidos. Es la lista
// cerrada del contrato; sirve para documentar y para los tests.
var CamposDelAgente = []string{"nombre", "descripcion", "prompt", "permisos", "skills"}

// Agente es una definición cargada desde `ai/agents/*.json`. El prompt es su
// identidad; `Permisos` es su política y `Herramientas` el catálogo efectivo
// que de él se deriva en la carga.
type Agente struct {
	Nombre      string    `json:"nombre"`
	Descripcion string    `json:"descripcion"`
	Prompt      string    `json:"prompt"`
	Permisos    []Permiso `json:"permisos"`
	Skills      []string  `json:"skills"`
	// Herramientas es el catálogo EFECTIVO, derivado de Permisos. No viaja en
	// el JSON: lo rellena la carga. Es la lista que `tools` comprueba y la que
	// sostiene la garantía de que `plan` no escribe.
	Herramientas []string `json:"-"`
}

// SoloConversacion informa si el agente no tiene ninguna herramienta efectiva.
// Un agente así no tiene ruta hacia `tools`: solo conversa (SPEC-AGENTE-BASE).
func (a Agente) SoloConversacion() bool { return len(a.Herramientas) == 0 }

// DecodificarAgente convierte el JSON de un agente en un Agente, rechazando
// campos desconocidos (un `hereda_de` no pasa), comprobando que los campos
// obligatorios están y derivando el catálogo efectivo de los permisos.
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
	herramientas, err := HerramientasDe(a.Permisos)
	if err != nil {
		return Agente{}, err
	}
	a.Herramientas = herramientas
	return a, nil
}

// validar comprueba lo que el JSON debe traer. `permisos` y `skills` pueden ir
// vacíos, pero no ausentes de significado: `nombre` y `prompt` no.
func (a Agente) validar() error {
	if strings.TrimSpace(a.Nombre) == "" {
		return errAgente("falta el campo obligatorio `nombre`")
	}
	if strings.TrimSpace(a.Prompt) == "" {
		return errAgente("el agente " + a.Nombre + " no tiene `prompt`")
	}
	return nil
}
