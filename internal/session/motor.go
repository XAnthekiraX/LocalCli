package session

import (
	"context"
	"errors"
	"fmt"
	"time"

	"localcli/internal/agent"
	"localcli/internal/llm/openai"
)

// motorOpenAI adapta el cliente OpenAI al puerto de motor del agente.
type motorOpenAI struct {
	cliente         *openai.Cliente
	modelo          func() string
	sinRazonamiento func() bool
}

// maxReintentosServidor es el máximo de reintentos para errores transitorios del servidor.
const maxReintentosServidor = 2

// baseBackoff es el tiempo base de espera entre reintentos.
const baseBackoff = 500 * time.Millisecond

// Ronda ejecuta una ronda contra el modelo, traduciendo del agente al protocolo.
// Reintenta automáticamente en errores transitorios del servidor (5xx, 429).
func (m motorOpenAI) Ronda(ctx context.Context, p agent.Peticion, alDelta func(agent.Delta) error) (agent.Respuesta, error) {
	opciones := openai.Opciones{}
	if m.sinRazonamiento != nil && m.sinRazonamiento() {
		opciones.SinRazonamiento = true
	}

	var ultimaErr error
	for intento := 0; intento <= maxReintentosServidor; intento++ {
		respuesta, err := m.cliente.Completar(ctx, m.modelo(), mensajesOpenAI(p.Mensajes), herramientasOpenAI(p.Herramientas), opciones, func(d openai.Delta) error {
			return alDelta(agent.Delta{Texto: d.Texto, Razonamiento: d.Razonamiento})
		})
		if err == nil {
			return agent.Respuesta{
				Texto:    respuesta.Texto,
				Llamadas: llamadasAgent(respuesta.Llamadas),
				Uso:      agent.Uso{Entrada: respuesta.Uso.Prompt, Salida: respuesta.Uso.Completion},
			}, nil
		}

		ultimaErr = err

		// Si el contexto fue cancelado, no reintentar.
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return agent.Respuesta{}, err
		}

		// Solo reintentar errores transitorios del servidor.
		if !openai.EsErrorTransitorio(err) {
			return agent.Respuesta{}, err
		}

		// Esperar antes de reintentar (backoff exponencial simple).
		backoff := baseBackoff * time.Duration(1<<intento)
		select {
		case <-ctx.Done():
			return agent.Respuesta{}, ctx.Err()
		case <-time.After(backoff):
		}
	}

	return agent.Respuesta{}, fmt.Errorf("motor: agotados %d reintentos tras error transitorio: %w", maxReintentosServidor, ultimaErr)
}

func mensajesOpenAI(mensajes []agent.Mensaje) []openai.Mensaje {
	salida := make([]openai.Mensaje, len(mensajes))
	for i, m := range mensajes {
		salida[i] = openai.Mensaje{
			Rol:       m.Rol,
			Contenido: m.Contenido,
			Imagenes:  imagenesOpenAI(m.Imagenes),
			Llamadas:  llamadasOpenAI(m.Llamadas),
			IDLlamada: m.IDLlamada,
		}
	}
	return salida
}

func imagenesOpenAI(imagenes []agent.Imagen) []openai.Imagen {
	if len(imagenes) == 0 {
		return nil
	}
	salida := make([]openai.Imagen, len(imagenes))
	for i, im := range imagenes {
		salida[i] = openai.Imagen{MIME: im.MIME, Datos: im.Datos}
	}
	return salida
}

func herramientasOpenAI(herramientas []agent.Herramienta) []openai.Herramienta {
	if len(herramientas) == 0 {
		return nil
	}
	salida := make([]openai.Herramienta, len(herramientas))
	for i, h := range herramientas {
		salida[i] = openai.Herramienta{
			Tipo:    "function",
			Funcion: openai.Definicion{Nombre: h.Nombre, Descripcion: h.Descripcion, Parametros: h.Esquema},
		}
	}
	return salida
}

func llamadasOpenAI(llamadas []agent.Llamada) []openai.Llamada {
	if len(llamadas) == 0 {
		return nil
	}
	salida := make([]openai.Llamada, len(llamadas))
	for i, l := range llamadas {
		salida[i] = openai.Llamada{
			ID:      l.ID,
			Tipo:    "function",
			Funcion: openai.Funcion{Nombre: l.Nombre, Argumentos: l.Argumentos},
		}
	}
	return salida
}

func llamadasAgent(llamadas []openai.Llamada) []agent.Llamada {
	if len(llamadas) == 0 {
		return nil
	}
	salida := make([]agent.Llamada, len(llamadas))
	for i, l := range llamadas {
		salida[i] = agent.Llamada{ID: l.ID, Nombre: l.Funcion.Nombre, Argumentos: l.Funcion.Argumentos}
	}
	return salida
}
