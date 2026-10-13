package chat

import (
	"fmt"
	"strings"
	"time"

	"localcli/internal/tui/render"
)

// maxMotivo es lo que se conserva del motivo de un fallo en su línea: una sola
// línea y corta, porque esto va a la pantalla.
const maxMotivo = 60

// formatearDuracion mide igual el tiempo de una respuesta y el de la ejecución de
// una herramienta: milisegundos por debajo del segundo, décimas de segundo por
// debajo del minuto y minutos y segundos a partir de ahí. Es una función pura.
func formatearDuracion(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Second:
		return fmt.Sprintf("%d ms", d.Milliseconds())
	case d < time.Minute:
		return fmt.Sprintf("%.1f s", d.Seconds())
	default:
		total := int(d.Seconds())
		return fmt.Sprintf("%d m %d s", total/60, total%60)
	}
}

// sufijoDuracion es el tiempo de una respuesta: entre paréntesis y atenuado,
// porque el paréntesis es lo que marca «esto es el tiempo de una respuesta» y lo
// distingue del tiempo de una herramienta, que va unido con el separador de su
// línea. Sin medición no pinta nada: ni un cero ni un hueco reservado.
func sufijoDuracion(d time.Duration) string {
	if d <= 0 {
		return ""
	}
	return render.EstiloSutil.Render("(" + formatearDuracion(d) + ")")
}

// LineaHerramientaAbierta compone la línea con la que nace una herramienta: su
// verbo de pantalla y su objetivo —«LEER [ruta]»—. El tema es el argumento que el
// catálogo declara; el resto de argumentos no se expone (INTERFACES §1.1).
func LineaHerramientaAbierta(verbo, tema string) string {
	linea := verbo
	if tema != "" {
		linea += " [" + tema + "]"
	}
	return linea
}

// LineaHerramientaCerrada completa esa misma línea: la marca, la medida del
// resultado, si se recortó, el motivo del fallo y cuánto tardó la ejecución. Lo
// que no haya no deja hueco: una medida ausente y una duración sin medir no
// pintan nada. La salida cruda de la herramienta nunca entra aquí (INTERFACES
// §1.1). Es una función pura.
func LineaHerramientaCerrada(base string, ok, truncado bool, medida, motivo string, duracion time.Duration) string {
	marca := "✓ "
	if !ok {
		marca = "✗ "
	}
	linea := marca + base
	if medida != "" {
		linea += " · " + medida
	}
	if truncado {
		linea += " · recortado"
	}
	if !ok && motivo != "" {
		linea += " · " + recortarMotivo(motivo)
	}
	if duracion > 0 {
		linea += " · " + formatearDuracion(duracion)
	}
	return linea
}

// LineaProceso es la línea con la que se anuncia una etapa de un flujo: cada
// etapa que empieza deja una sola línea con su nombre.
func LineaProceso(nombre string) string { return "[Sub Proceso] " + nombre }

// recortarMotivo abrevia el motivo de un fallo para su línea: una sola línea.
func recortarMotivo(motivo string) string {
	motivo = strings.Join(strings.Fields(motivo), " ")
	runas := []rune(motivo)
	if len(runas) <= maxMotivo {
		return motivo
	}
	return string(runas[:maxMotivo]) + "…"
}
