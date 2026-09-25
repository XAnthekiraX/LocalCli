// models.go — T-B005-04: listado de modelos (equivalente a `ollama list`).
//
// Fuente de verdad: SPEC-OLLAMA-PERFIL §Flujo principal pasos 1-3 ("se
// conecta a Ollama, lee los modelos disponibles, muestra cuáles caben").
// Endpoint: GET /api/tags → {"models":[{"name":...,"size":bytes,...}]}.
package ollama

import (
	"context"
	"encoding/json"
)

// Modelo es una entrada de `ollama list`. SizeBytes puede ser 0 si el
// servidor no lo reporta; en ese caso CabenNo lo sabe y trata como
// desconocido (el perfil informa, no decide).
type Modelo struct {
	Nombre      string
	Familia     string
	Parametros  string
	TamanoBytes int64
}

// respuestaTags es la forma cruda de /api/tags.
type respuestaTags struct {
	Modelos []struct {
		Name       string `json:"name"`
		Model      string `json:"model"`
		Size       int64  `json:"size"`
		Parameters struct {
			Size          string `json:"size"`
			Family        string `json:"family"`
			NumParameters string `json:"num_parameters"`
		} `json:"details"`
	} `json:"models"`
}

// ListarModelos consulta /api/tags y devuelve los modelos en el orden del
// servidor. Fallos de conexión → *ErrorOllama E_OLLAMA_UNAVAILABLE.
func (c *Client) ListarModelos(ctx context.Context) ([]Modelo, error) {
	resp, err := c.get(ctx, "/api/tags")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var r respuestaTags
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, &ErrorOllama{
			Codigo:   CodigoOllamaNoDisponible,
			Mensaje:  "respuesta ilegible de la lista de modelos",
			Detalle:  err.Error(),
			sentinel: ErrOllamaNoDisponible,
		}
	}
	out := make([]Modelo, 0, len(r.Modelos))
	for _, m := range r.Modelos {
		nombre := m.Name
		if nombre == "" {
			nombre = m.Model
		}
		out = append(out, Modelo{
			Nombre:      nombre,
			Familia:     m.Parameters.Family,
			Parametros:  m.Parameters.NumParameters,
			TamanoBytes: m.Size,
		})
	}
	return out, nil
}
