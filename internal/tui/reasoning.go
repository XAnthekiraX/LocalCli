// reasoning.go — T-B014-03: el razonamiento en vivo.
//
// Fuente de verdad: SPEC-INTERFAZ §Razonamiento del modelo ("Mientras el modelo
// genera se muestra un indicador en vivo ([⠋ Pensando]); el texto del
// razonamiento se revela con `Ctrl+R` para leerlo. Revelarlo no detiene la
// generación") y SPEC-PANEL-CONTEXTO ("El razonamiento se revela tal como lo
// devuelve el modelo, sin editarlo", "Si el modelo no expone razonamiento: se
// indica que no está disponible").
//
// El texto crudo no se vuelca por defecto: la vista pinta el indicador. Ocultar
// no borra: el texto se sigue acumulando igual y solo cambia lo que se pinta. Si
// ocultar parase la acumulación, volver a mostrarlo dejaría un hueco y la vista
// mentiría sobre lo que dijo el modelo.
package tui

import "strings"

// Razonamiento es el bloque de razonamiento de la petición actual.
type Razonamiento struct {
	// Visible es si el texto crudo del razonamiento está revelado. Por defecto
	// no lo está: en su lugar se pinta el indicador en vivo ([⠋ Pensando]). El
	// indicador no depende de esta bandera.
	Visible      bool
	NoDisponible bool
	buffer       strings.Builder
}

// NuevoRazonamiento crea el bloque con el texto oculto, que es el estado por
// defecto: la vista vuelca el indicador y el texto queda tras `Ctrl+R`.
func NuevoRazonamiento() Razonamiento { return Razonamiento{Visible: false} }

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
