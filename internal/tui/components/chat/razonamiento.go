package chat

import (
	"strings"

	"localcli/internal/tui/render"
)

// AlternarRazonamiento revela u oculta el texto crudo del razonamiento (`Ctrl+R`).
// Es solo pintura: no toca la generación ni el texto acumulado, así que volver a
// revelarlo lo recupera entero (SPEC-INTERFAZ §Razonamiento del modelo).
func (m *Model) AlternarRazonamiento() { m.MostrarRazonamiento = !m.MostrarRazonamiento }

// renderRazonamiento pinta el razonamiento revelado con su propio estilo, para
// que nunca se confunda con la respuesta. No revelado no pinta nada; revelado y
// sin texto, avisa de que el modelo no lo entregó. Es una función pura.
func renderRazonamiento(texto string, revelado, noDisponible bool, ancho int) string {
	if !revelado {
		return ""
	}
	if strings.TrimSpace(texto) == "" {
		if noDisponible {
			return render.EstiloRazonamiento.Render("(el modelo no entregó razonamiento)")
		}
		return ""
	}
	return render.EstiloRazonamiento.Render(strings.Join(envolver(texto, ancho), "\n"))
}
