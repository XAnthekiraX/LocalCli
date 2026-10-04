// modelos.go — T-B036-05: lista de modelos de `llama-server`.
//
// Fuente de verdad: INTEGRATIONS §llama.cpp («`/models` del router, con
// `/v1/models` como alternativa»). El router en modo router sirve `/models`; si
// no está, se prueba `/v1/models`.
package llamacpp

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"localcli/internal/llm"
)

// respuestaModelos es la forma OpenAI de la lista.
type respuestaModelos struct {
	Data []struct {
		ID   string `json:"id"`
		Meta *struct {
			NCtxTrain int `json:"n_ctx_train"`
		} `json:"meta"`
	} `json:"data"`
}

// ListarModelos consulta `/models` y, si no existe, `/v1/models`.
func (c *Client) ListarModelos(ctx context.Context) ([]llm.Modelo, error) {
	resp, err := c.get(ctx, "/models")
	if err != nil {
		if esNoEncontrado(err) {
			resp, err = c.get(ctx, "/v1/models")
		}
		if err != nil {
			return nil, err
		}
	}
	defer resp.Body.Close()
	var r respuestaModelos
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, llm.NuevoErrorNoDisponible("respuesta ilegible de la lista de modelos", err.Error())
	}
	out := make([]llm.Modelo, 0, len(r.Data))
	for _, m := range r.Data {
		modelo := llm.Modelo{Nombre: m.ID}
		if m.Meta != nil {
			modelo.ContextLength = m.Meta.NCtxTrain
		}
		out = append(out, modelo)
	}
	return out, nil
}

// esNoEncontrado informa si el error viene de un 404, para probar la ruta de
// reserva `/v1/models`.
func esNoEncontrado(err error) bool {
	var e *llm.ErrorMotor
	if errors.As(err, &e) {
		return strings.Contains(e.Mensaje, "(404)")
	}
	return false
}
