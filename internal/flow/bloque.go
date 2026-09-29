package flow

// bloque.go — el bloque de contexto de un flujo con `BloqueContexto`.
//
// Fuente de verdad: [[specs/SPEC-MOTOR-FLUJOS]] §Bloque de contexto y
// [[specs/SPEC-RESOLVER]]. Un flujo con el flag `bloque_contexto` no descarta el
// resultado de sus etapas: cada una lo entrega, el modelo lo condensa y queda
// guardado en el bloque; la última etapa compone la salida a partir de TODO el
// bloque, sin herramientas. El estado del bloque vive en `store` (una fila por
// etapa y sesión), no en la memoria del motor.

import (
	"context"
	"strings"
)

// EntradaBloque es la aportación de una etapa al bloque de contexto de un flujo:
// qué etapa la produjo, su nombre visible, su posición en la secuencia y el
// texto (ya optimizado) que entregó.
type EntradaBloque struct {
	Etapa     string
	Nombre    string
	Posicion  int
	Contenido string
}

// Bloque guarda y devuelve el bloque de contexto de un flujo. Lo implementa el
// arranque sobre `store`; el motor no persiste por su cuenta. Está acotado a la
// sesión en curso: una ejecución nueva del mismo flujo reemplaza el bloque
// anterior (una aportación por etapa).
type Bloque interface {
	// Limpiar deja el bloque del flujo vacío antes de una ejecución nueva.
	Limpiar(ctx context.Context, flujo string) error
	// Guardar añade o reemplaza la aportación de una etapa.
	Guardar(ctx context.Context, flujo string, e EntradaBloque) error
	// Leer devuelve las aportaciones del flujo, en orden de posición.
	Leer(ctx context.Context, flujo string) ([]EntradaBloque, error)
}

// Optimizador condensa el resultado de una etapa antes de guardarlo en el bloque
// y de encadenarlo a la siguiente. Es una generación corta del modelo local y la
// implementa el arranque. Un fallo no detiene el flujo: el motor cae al resumen
// mecánico de siempre ([[specs/SPEC-MOTOR-FLUJOS]] §Reglas de negocio).
type Optimizador interface {
	Optimizar(ctx context.Context, flujo, etapa, resultado string) (string, error)
}

// PromptOptimizacion arma la petición con la que el modelo condensa el resultado
// de una etapa. Pide lo esencial, sin inventar, y solo el resumen como salida.
func PromptOptimizacion(flujo, etapa, resultado string) string {
	var b strings.Builder
	b.WriteString("Eres el optimizador de resultados de un flujo de trabajo de un harness de desarrollo.\n")
	b.WriteString("Condensa el resultado de la etapa «")
	b.WriteString(etapa)
	b.WriteString("» del flujo «")
	b.WriteString(flujo)
	b.WriteString("» en un resumen breve (unas pocas líneas) que conserve solo lo esencial: hechos, archivos y conclusiones que necesiten las etapas siguientes y la composición final.\n")
	b.WriteString("No inventes datos, archivos ni conclusiones; si el resultado está vacío o incompleto, dilo en una línea.\n")
	b.WriteString("Devuelve únicamente el resumen, sin preámbulos ni títulos.\n\n")
	b.WriteString("## Resultado de la etapa\n")
	b.WriteString(strings.TrimSpace(resultado))
	return b.String()
}
