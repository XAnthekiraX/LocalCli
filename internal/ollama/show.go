// show.go — capacidades del modelo (GET/POST /api/show).
//
// Fuente de verdad: SPEC-OLLAMA-PERFIL (el modelo lo elige el usuario; la
// herramienta informa y avisa). Ollama declara en /api/show qué sabe hacer cada
// modelo ("completion", "tools", "vision"…). LocalCli no usa function-calling
// nativo —las herramientas viajan como texto en el prompt—, así que esta
// capacidad no bloquea nada: sirve para avisar al usuario de que el modelo que
// eligió probablemente no sabrá pedir herramientas.
package ollama

import (
	"context"
	"encoding/json"
)

// CapacidadHerramientas es el nombre de la capacidad que declara un modelo que
// sabe usar herramientas.
const CapacidadHerramientas = "tools"

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
	for _, c := range capacidades {
		if c == CapacidadHerramientas {
			return true
		}
	}
	return false
}
