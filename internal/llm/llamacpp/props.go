package llamacpp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"localcli/internal/llm"
)

// Props es lo que el harness lee de `GET /props` del servidor.
type Props struct {
	// Ventana es `default_generation_settings.n_ctx`: la que fijó quien arrancó
	// el servidor con `-c`. 0 si no se puede leer.
	Ventana int
	// Vision es el estado que declara `modalities.vision`; Desconocida si el
	// servidor no expone ese dato.
	Vision llm.Capacidad
}

// Leer consulta `/props` en la dirección base del servidor.
func Leer(ctx context.Context, baseURL string) (Props, error) {
	pctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	destino := strings.TrimRight(baseURL, "/") + "/props"
	req, err := http.NewRequestWithContext(pctx, http.MethodGet, destino, nil)
	if err != nil {
		return Props{}, errorf("no se pudo preparar la consulta: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Props{}, errorf("no se pudo conectar con el servidor en %s: %w", baseURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Props{}, errorf("el servidor respondió %s: %s", resp.Status, trozo(resp.Body))
	}
	return Decodificar(resp.Body)
}

// VentanaDe lee el `n_ctx` con el que sirve el servidor; 0 si no se puede leer.
// Es la ventana que el servidor acepta de verdad, sin importar con qué se arrancó.
func VentanaDe(ctx context.Context, baseURL string) int {
	p, err := Leer(ctx, baseURL)
	if err != nil || p.Ventana <= 0 {
		return 0
	}
	return p.Ventana
}

// Decodificar lee el cuerpo de `/props`. Un `/props` sin la ventana no es un
// error: la ventana queda desconocida, no cero.
func Decodificar(r io.Reader) (Props, error) {
	var bruto struct {
		DefaultGenerationSettings struct {
			NCtx int `json:"n_ctx"`
		} `json:"default_generation_settings"`
		Modalities *struct {
			Vision bool `json:"vision"`
		} `json:"modalities"`
	}
	if err := json.NewDecoder(r).Decode(&bruto); err != nil {
		return Props{}, errorf("las props no se pudieron leer: %w", err)
	}
	p := Props{Ventana: bruto.DefaultGenerationSettings.NCtx, Vision: llm.Desconocida}
	if bruto.Modalities != nil {
		p.Vision = llm.DeBooleano(bruto.Modalities.Vision)
	}
	return p, nil
}

func errorf(format string, args ...any) error {
	return fmt.Errorf("llm/llamacpp: "+format, args...)
}

func trozo(r io.Reader) string {
	const tope = 300
	datos, _ := io.ReadAll(io.LimitReader(r, tope))
	return strings.TrimSpace(string(datos))
}
