// props.go — T-B036-05 y T-B036-07: propiedades del servidor (`/props`).
//
// Fuente de verdad: INTEGRATIONS §llama.cpp («`/props` las expone en
// `chat_template_caps` (`supports_tool_calls`, `supports_reasoning_effort`) y
// en `modalities.vision`») y §5 («la ventana se lee de
// `default_generation_settings.n_ctx`»).
//
// Solo se hace GET: el servidor se lee, nunca se muta. No hay `POST /props`.
package openai

import (
	"context"
	"encoding/json"

	"localcli/internal/llm"
)

// propsResp es la parte de `/props` que interesa. Los bloques son punteros para
// distinguir «no viene» de «viene en falso».
type propsResp struct {
	DefaultGenerationSettings struct {
		NCtx int `json:"n_ctx"`
	} `json:"default_generation_settings"`
	NCtx             int `json:"n_ctx"`
	ChatTemplateCaps *struct {
		SupportsToolCalls       bool `json:"supports_tool_calls"`
		SupportsReasoningEffort bool `json:"supports_reasoning_effort"`
	} `json:"chat_template_caps"`
	Modalities *struct {
		Vision bool `json:"vision"`
	} `json:"modalities"`
}

// leerProps consulta GET /props.
func (c *Client) leerProps(ctx context.Context) (*propsResp, error) {
	resp, err := c.get(ctx, "/props")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var p propsResp
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return nil, llm.NuevoErrorNoDisponible("respuesta ilegible de las propiedades del servidor", err.Error())
	}
	return &p, nil
}

// Capacidades consulta `/props` y normaliza a las tres que el harness entiende:
// herramientas, visión y razonamiento. Si el servidor no declara ninguna de las
// dos fichas, se devuelve error: no saberlo no es «no puede».
func (c *Client) Capacidades(ctx context.Context, _ string) ([]string, error) {
	p, err := c.leerProps(ctx)
	if err != nil {
		return nil, err
	}
	if p.ChatTemplateCaps == nil && p.Modalities == nil {
		return nil, llm.NuevoErrorNoDisponible("el servidor no declara las capacidades del modelo", "")
	}
	caps := make([]string, 0, 3)
	if p.ChatTemplateCaps != nil {
		if p.ChatTemplateCaps.SupportsToolCalls {
			caps = append(caps, llm.CapacidadHerramientas)
		}
		if p.ChatTemplateCaps.SupportsReasoningEffort {
			caps = append(caps, llm.CapacidadPensar)
		}
	}
	if p.Modalities != nil && p.Modalities.Vision {
		caps = append(caps, llm.CapacidadVision)
	}
	return caps, nil
}

// VentanaDeContexto lee la ventana que fijó quien arrancó el servidor. Devuelve
// `declarada=false` (se lee, no se declara por petición). Si no se puede leer
// —el router todavía no tiene modelo cargado, por ejemplo— la ventana se trata
// como desconocida (0) sin error, y el turno se recortará al tope.
func (c *Client) VentanaDeContexto(ctx context.Context, _ string) (int, bool, error) {
	p, err := c.leerProps(ctx)
	if err != nil {
		return 0, false, nil
	}
	n := p.DefaultGenerationSettings.NCtx
	if n == 0 {
		n = p.NCtx
	}
	return n, false, nil
}
