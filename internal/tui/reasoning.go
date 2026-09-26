// reasoning.go — T-B014-03: el razonamiento en vivo.
//
// Fuente de verdad: SPEC-INTERFAZ §Razonamiento del modelo ("Se muestra en vivo,
// arriba de la respuesta, mientras el modelo genera… Se puede ocultar para leer
// solo la respuesta. Ocultarlo no detiene la generación") y SPEC-PANEL-CONTEXTO
// ("El razonamiento se muestra tal como lo devuelve el modelo, sin editarlo",
// "Si el modelo no expone razonamiento: se indica que no está disponible").
//
// Ocultar no borra: el texto se sigue acumulando igual y solo cambia lo que se
// pinta. Si ocultar parase la acumulación, volver a mostrarlo dejaría un hueco y
// la vista mentiría sobre lo que dijo el modelo.
package tui

import "strings"

// Razonamiento es el bloque de razonamiento de la petición actual.
type Razonamiento struct {
	Visible      bool
	NoDisponible bool
	buffer       strings.Builder
}

// NuevoRazonamiento crea el bloque visible, que es el estado por defecto.
func NuevoRazonamiento() Razonamiento { return Razonamiento{Visible: true} }

// Añadir acumula un fragmento de razonamiento.
func (r *Razonamiento) Añadir(texto string) {
	r.NoDisponible = false
	r.buffer.WriteString(texto)
}

// Texto devuelve lo acumulado.
func (r *Razonamiento) Texto() string { return r.buffer.String() }

// Hay informa si llegó algo de razonamiento en esta petición.
func (r *Razonamiento) Hay() bool { return strings.TrimSpace(r.buffer.String()) != "" }

// Alternar muestra u oculta el bloque, sin tocar lo acumulado.
func (r *Razonamiento) Alternar() { r.Visible = !r.Visible }

// CerrarTurno marca si el modelo no entregó razonamiento y deja el bloque listo
// para la petición siguiente.
func (r *Razonamiento) CerrarTurno() {
	r.NoDisponible = !r.Hay()
	r.buffer.Reset()
}

// Render pinta el bloque. Devuelve "" si está oculto o si no hay nada que decir:
// una cabecera vacía solo estorba. El dibujo vive en styles.go (T-F002); aquí
// solo se le pasa lo acumulado y si se ve.
func (r *Razonamiento) Render(ancho int) string {
	return renderReasoning(r.buffer.String(), r.Visible, r.NoDisponible, ancho)
}
