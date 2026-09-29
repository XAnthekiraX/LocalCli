// show.go — capacidades del modelo (GET/POST /api/show).
//
// Fuente de verdad: SPEC-OLLAMA-PERFIL (el modelo lo elige el usuario; la
// herramienta informa y avisa). Ollama declara en /api/show qué sabe hacer cada
// modelo ("completion", "tools", "vision"…). LocalCli no usa function-calling
// nativo —las herramientas viajan como texto en el prompt—, así que esta
// capacidad no bloquea nada: sirve para avisar al usuario. Con "vision" pasa lo
// mismo: saber si el modelo interpreta imágenes solo alimenta el aviso; las
// imágenes se adjuntan igual y Ollama decide si las aprovecha.
package ollama

import (
	"context"
	"encoding/json"
)

// CapacidadHerramientas es el nombre de la capacidad que declara un modelo que
// sabe usar herramientas.
const CapacidadHerramientas = "tools"

// CapacidadVision es el nombre de la capacidad que declara un modelo multimodal
// (sabe interpretar imágenes). El harness la usa solo para avisar: las imágenes
// se adjuntan igual, y es Ollama quien decide si el modelo las aprovecha.
const CapacidadVision = "vision"

// CapacidadPensar es el nombre de la capacidad que declara un modelo que razona
// antes de responder (qwen3, deepseek-r1, gpt-oss…). Ollama deja el razonamiento
// ENCENDIDO por defecto en esos modelos, así que el harness manda `think`
// explícito cuando el usuario lo decide. Solo se le manda a quien declara esta
// capacidad: a un modelo que no razona, `think` le pide algo que no entiende.
const CapacidadPensar = "thinking"

// FichaModelo es la parte de la respuesta de /api/show que nos interesa:
// `capabilities` (Ollama moderno) y `details.families`, que es el respaldo
// cuando el servidor no declara capacidades.
type FichaModelo struct {
	Capacidades []string `json:"capabilities"`
	Detalles    struct {
		Familia  string   `json:"family"`
		Familias []string `json:"families"`
	} `json:"details"`
}

// familiasConVision son las arquitecturas que Ollama reporta para modelos
// multimodales. Se usan solo como respaldo: cuando la ficha trae `capabilities`,
// esa es la fuente de verdad.
var familiasConVision = map[string]bool{
	"clip":      true,
	"mllama":    true,
	"llava":     true,
	"qwen2vl":   true,
	"moondream": true,
	"minicpmv":  true,
	"pixtral":   true,
}

// Capacidades consulta POST /api/show y devuelve las capacidades declaradas por
// el modelo. Un fallo de conexión llega como *ErrorOllama. Si la ficha no trae
// `capabilities` se deduce la visión de las familias del modelo; si tampoco hay
// familias, se devuelve error: no saberlo NO es lo mismo que «no puede», y quien
// avisa necesita distinguirlos.
func (c *Client) Capacidades(ctx context.Context, nombre string) ([]string, error) {
	payload := struct {
		Model string `json:"model"`
	}{Model: nombre}
	resp, err := c.post(ctx, "/api/show", payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var ficha FichaModelo
	if err := json.NewDecoder(resp.Body).Decode(&ficha); err != nil {
		return nil, &ErrorOllama{
			Codigo:   CodigoOllamaNoDisponible,
			Mensaje:  "respuesta ilegible de la ficha del modelo",
			Detalle:  err.Error(),
			sentinel: ErrOllamaNoDisponible,
		}
	}
	if len(ficha.Capacidades) > 0 {
		return ficha.Capacidades, nil
	}
	familias := ficha.Detalles.Familias
	if len(familias) == 0 && ficha.Detalles.Familia != "" {
		familias = []string{ficha.Detalles.Familia}
	}
	if len(familias) == 0 {
		return nil, &ErrorOllama{
			Codigo:   CodigoOllamaNoDisponible,
			Mensaje:  "la ficha del modelo no declara capacidades",
			sentinel: ErrOllamaNoDisponible,
		}
	}
	caps := make([]string, 0, 1)
	for _, f := range familias {
		if familiasConVision[f] {
			caps = append(caps, CapacidadVision)
			break
		}
	}
	return caps, nil
}

// PuedeUsarHerramientas dice si el modelo declara la capacidad "tools". Un
// modelo sin ficha clara (lista vacía) no la declara: quien decide avisar es el
// llamante, que ante un error prefiere no alarmar.
func PuedeUsarHerramientas(capacidades []string) bool {
	return tieneCapacidad(capacidades, CapacidadHerramientas)
}

// PuedeVer dice si el modelo declara la capacidad "vision". Igual que con las
// herramientas, una ficha vacía no la declara y el aviso lo decide el llamante.
func PuedeVer(capacidades []string) bool {
	return tieneCapacidad(capacidades, CapacidadVision)
}

// PuedePensar dice si el modelo razona antes de responder. Un modelo que no la
// declara no admite `think`: mandárselo sería pedirle algo que no entiende.
func PuedePensar(capacidades []string) bool {
	return tieneCapacidad(capacidades, CapacidadPensar)
}

func tieneCapacidad(capacidades []string, buscada string) bool {
	for _, c := range capacidades {
		if c == buscada {
			return true
		}
	}
	return false
}
