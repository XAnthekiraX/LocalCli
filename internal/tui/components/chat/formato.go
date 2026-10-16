package chat

import (
	"time"

	"localcli/internal/linea"
	"localcli/internal/tui/render"
)

// formatearDuracion mide igual el tiempo de una respuesta y el de la ejecución de
// una herramienta: la composición vive en `linea`, compartida con quien guarda
// las líneas.
func formatearDuracion(d time.Duration) string { return linea.Duracion(d) }

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
func LineaHerramientaAbierta(verbo, tema string) string { return linea.HerramientaAbierta(verbo, tema) }

// LineaHerramientaCerrada completa esa misma línea: la marca, la medida del
// resultado, si se recortó, el motivo del fallo y cuánto tardó la ejecución. Lo
// que no haya no deja hueco: una medida ausente y una duración sin medir no
// pintan nada. La salida cruda de la herramienta nunca entra aquí (INTERFACES
// §1.1). Es una función pura.
func LineaHerramientaCerrada(base string, ok, truncado bool, medida, motivo string, duracion time.Duration) string {
	return linea.HerramientaCerrada(base, ok, truncado, medida, motivo, duracion)
}

// LineaProceso es la línea con la que se anuncia una etapa de un flujo: cada
// etapa que empieza deja una sola línea con su nombre.
func LineaProceso(nombre string) string { return linea.Proceso(nombre) }
