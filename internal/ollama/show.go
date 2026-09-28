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

// FichaModelo es la parte de la respuesta de /api/show que nos interesa.
type FichaModelo struct {
	Capacidades []string `json:"capabilities"`
}

// Capacidades consulta POST /api/show y devuelve las capacidades declaradas por
// el modelo. Un fallo de conexión llega como *ErrorOllama; una ficha ilegible
// también se reporta, porque de ella depende el aviso al usuario.
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
	return ficha.Capacidades, nil
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

func tieneCapacidad(capacidades []string, buscada string) bool {
	for _, c := range capacidades {
		if c == buscada {
			return true
		}
	}
	return false
}
