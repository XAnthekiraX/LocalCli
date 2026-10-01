// show.go — T-B005 / T-B036-04: capacidades del modelo (GET/POST /api/show).
//
// Fuente de verdad: SPEC-MODELO-PROVEEDOR (las capacidades se normalizan a las
// tres que el harness entiende) y INTEGRATIONS §Ollama (Ollama declara en
// `/api/show` `capabilities`: `completion`, `tools`, `vision`, `thinking`).
// Esta capacidad alimenta el aviso al usuario; no bloquea nada.
package ollama

import (
	"context"
	"encoding/json"

	"localcli/internal/llm"
)

// FichaModelo es la parte de la respuesta de /api/show que nos interesa:
// `capabilities` (Ollama moderno) y `details.families`, el respaldo cuando el
// servidor no declara capacidades.
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
// el modelo, normalizadas a las del harness. Las capacidades de Ollama ya se
// llaman igual que las neutras (`tools`, `vision`, `thinking`). Un fallo de
// conexión llega como `*llm.ErrorProveedor`. Si la ficha no trae `capabilities`
// se deduce la visión de las familias; si tampoco hay familias, se devuelve
// error: no saberlo NO es lo mismo que «no puede».
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
		return nil, llm.NuevoErrorNoDisponible("respuesta ilegible de la ficha del modelo", err.Error())
	}
	if len(ficha.Capacidades) > 0 {
		return ficha.Capacidades, nil
	}
	familias := ficha.Detalles.Familias
	if len(familias) == 0 && ficha.Detalles.Familia != "" {
		familias = []string{ficha.Detalles.Familia}
	}
	if len(familias) == 0 {
		return nil, llm.NuevoErrorNoDisponible("la ficha del modelo no declara capacidades", "")
	}
	caps := make([]string, 0, 1)
	for _, f := range familias {
		if familiasConVision[f] {
			caps = append(caps, llm.CapacidadVision)
			break
		}
	}
	return caps, nil
}
