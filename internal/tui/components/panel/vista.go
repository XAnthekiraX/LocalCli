package panel

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"

	"localcli/internal/tui/render"
)

// Render pinta el sidebar de la sesión activa: el título de la conversación
// arriba, los bloques CONTEXTO y TODO, la lista de tareas del agente y el estado
// compacto (capa y cola, aprobaciones y agente). Abajo, pegado al suelo de la
// columna, va el pie de tres filas: el estado de git, la ruta del proyecto y el
// nombre y la versión del harness.
//
// Recibe el alto de la columna porque el pie ocupa sus filas siempre: si la lista
// de tareas no cabe se recorta y se resume, y si la terminal es más baja que el
// pie entero se recorta el panel por arriba. Es de lectura: aquí no hay nada que
// pulsar.
//
// No pinta el fondo: el layout completa cada fila con el del sidebar, para que el
// marco siga siendo una sola superficie (render.MarcoCompleto).
func (m Model) Render(ancho, alto int) string {
	if ancho < 1 {
		ancho = 1
	}
	// El texto del sidebar deja dos columnas de separación a cada lado: se compone
	// sobre el ancho interior y luego se sangra.
	const sangria = 2
	ancho -= 2 * sangria
	if ancho < 1 {
		ancho = 1
	}

	// El pie, de arriba abajo: git, ruta y harness. La ruta se parte en varias
	// filas si no cabe, para no perder ningún tramo.
	pie := []string{filaDeDato("Git", m.textoGit(), ancho)}
	for _, l := range envolverPorRuta("["+rutaBreve(valorODefecto(m.Ruta))+"]", ancho) {
		pie = append(pie, render.EstiloSutil.Render(l))
	}
	pie = append(pie, render.EstiloEtiqueta.Render(truncar(m.firmaHarness(), ancho)))
	// `disponible` son las filas que quedan por encima del pie, descontando el
	// aire de arriba y el de abajo, y es el presupuesto de la lista de tareas. Si
	// no cabe, vale cero: el pie manda. En una columna más baja que el pie no cabe
	// nada de aire.
	aireAbajo := 1
	if alto < len(pie)+2 {
		aireAbajo = 0
	}
	disponible := max(alto-len(pie)-1-aireAbajo, 0)

	// PARTES FIJAS: su altura no depende de los datos, así que se componen
	// enteras y son las que el recorte final puede llegar a comerse.
	cabeza := []string{
		render.EstiloTitulo.Render(truncar(valorODefecto(m.Sesion)+estadoEntreParentesis(m.Estado), ancho)),
		"",
		// CONTEXTO: tokens usados, porcentaje ocupado y, si toca, el aviso de
		// límite.
		render.EstiloSeccion.Render("CONTEXTO"),
		"  " + truncar(m.textoContexto(), ancho-2),
		"  " + truncar(m.textoPorcentajeContexto(), ancho-2),
	}
	if m.ContextoApretado() {
		cabeza = append(cabeza, "  "+render.EstiloAviso.Render("cerca del límite"))
	}
	// TODO: el elemento en curso y cuántos quedan.
	cabeza = append(cabeza,
		"",
		render.EstiloSeccion.Render("▾ TODO"),
		"  "+truncar(valorODefecto(m.ElementoActual)+" · quedan "+strconv.Itoa(m.ElementosRestantes), ancho-2),
		"",
	)

	// ESTADO: el resto de los datos de la sesión, en filas etiqueta + valor. Git y
	// proyecto no están aquí: viven en el pie, que es donde se leen de un vistazo
	// sin tener que llegar hasta arriba.
	estado := []string{
		render.EstiloSeccion.Render("ESTADO"),
		filaDeDato("Capa y cola", m.textoCapa(), ancho),
		filaDeDato("Aprobaciones", strconv.Itoa(m.Aprobaciones)+" esperando decisión", ancho),
		filaDeDato("Agente", valorODefecto(m.Agente), ancho),
	}

	// LISTA DE TAREAS: los pasos del agente, si queda alguno accionable. Es lo
	// único que crece sin límite, así que es lo que cede cuando la columna es
	// baja. La línea en blanco que la separa de ESTADO se reserva siempre, se
	// pinte lista o no, para que el presupuesto de filas sea el mismo.
	lineas := append([]string{}, cabeza...)
	// El presupuesto de filas de la lista es el hueco entre las dos partes fijas
	// más su separador en blanco. Sin altura conocida no hay tope: se pintan todos
	// los pasos.
	tope := len(m.Tareas) + 1
	if disponible > 0 {
		tope = disponible - len(cabeza) - len(estado) - 1
	}
	tareas := m.lineasDeTareas(ancho, tope)
	if len(tareas) > 0 {
		lineas = append(lineas, tareas...)
		lineas = append(lineas, "")
	}
	lineas = append(lineas, estado...)

	// El cuerpo se ajusta al hueco que deja el pie. Si sobran filas, el relleno va
	// después de ESTADO, que es donde estorban menos, para que el contenido quede
	// arriba; si faltan, se recorta por abajo, que es donde está lo que menos
	// pesa.
	if disponible > 0 {
		if len(lineas) > disponible {
			lineas = lineas[:disponible]
		}
		for len(lineas) < disponible {
			lineas = append(lineas, "")
		}
	}
	lineas = append(lineas, pie...)

	// Aire vertical: una fila en blanco arriba del título y —si la columna tiene
	// sitio— otra debajo del pie, para que el texto no toque los bordes.
	lineas = append([]string{""}, lineas...)
	if aireAbajo > 0 {
		lineas = append(lineas, "")
	}

	// Una columna más baja que el pie entero es el único caso en que no hay
	// recorte limpio: se van las primeras filas del panel, porque lo que no puede
	// caerse es la ruta y la firma. Sin geometría no se toca nada.
	if alto > 0 && len(lineas) > alto {
		lineas = lineas[len(lineas)-alto:]
	}
	// Dos columnas de separación a la izquierda; a la derecha las deja libres el
	// relleno del layout.
	sangrado := strings.Repeat(" ", sangria)
	for i, l := range lineas {
		lineas[i] = sangrado + l
	}
	return strings.Join(lineas, "\n")
}

// lineasDeTareas pinta la lista de pasos de la sesión: `[•]` en curso, `[✓]`
// hecha, `[x]` cancelada y `[ ]` pendiente. Devuelve nil cuando no queda nada
// accionable —sin pasos, o todos completados o cancelados—: un checklist
// terminado solo estorba (SPEC-TOOLS).
//
// `tope` es cuántas filas puede ocupar la lista como mucho, encabezado incluido.
// Lo que no cabe no se pierde: se resume en «N más», porque el pie no puede
// caerse por una lista larga.
func (m Model) lineasDeTareas(ancho, tope int) []string {
	// Un tope de 1 o menos es que solo cabe el pie: ni siquiera el encabezado.
	if !accionable(m.Tareas) || tope < 2 {
		return nil
	}
	// El encabezado se queda con una de las filas del tope. Se pintan todos los
	// pasos, no solo los accionables: el glifo de cada uno dice si está hecho,
	// cancelado o pendiente.
	caben := min(len(m.Tareas), tope-1)
	lineas := make([]string, 0, tope)
	lineas = append(lineas, render.EstiloSeccion.Render("LISTA DE TAREAS"))
	for _, t := range m.Tareas[:caben] {
		lineas = append(lineas, "  "+glifoDeTarea(t.Estado)+" "+truncar(t.Contenido, ancho-4))
	}
	if fuera := len(m.Tareas) - caben; fuera > 0 {
		if len(lineas) < tope {
			lineas = append(lineas, "  … "+strconv.Itoa(fuera)+" más")
		} else {
			// No queda fila para el resumen: le cede la suya al último paso, que
			// también cuenta como no pintado.
			lineas[len(lineas)-1] = "  … " + strconv.Itoa(fuera+1) + " más"
		}
	}
	return lineas
}

// filaDeDato alinea en el sidebar la etiqueta de un dato y su valor.
func filaDeDato(etiqueta, valor string, ancho int) string {
	resto := ancho - len([]rune(etiqueta)) - 1
	if resto < 1 {
		resto = 1
	}
	return render.EstiloEtiqueta.Render(etiqueta) + " " + truncar(valor, resto)
}

// truncar deja el principio del texto y pone «…» cuando no cabe. Mide por
// columnas de pantalla, no por runas: un emoji no desborda la fila. Es una
// función pura.
func truncar(texto string, ancho int) string {
	if ancho <= 0 {
		return ""
	}
	if runewidth.StringWidth(texto) <= ancho {
		return texto
	}
	if ancho == 1 {
		return "…"
	}
	return texto[:corteAncho(texto, ancho-1)] + "…"
}

// envolverPorRuta parte una ruta en filas de `ancho` columnas sin perder ningún
// tramo, cortando tras un separador cuando puede. Es una función pura.
func envolverPorRuta(texto string, ancho int) []string {
	if ancho <= 0 {
		return []string{texto}
	}
	var lineas []string
	resto := texto
	for runewidth.StringWidth(resto) > ancho {
		corte := corteAncho(resto, ancho)
		if corte <= 0 {
			// Ni una runa cabe: se fuerza una para no girar en vacío.
			_, tam := utf8.DecodeRuneInString(resto)
			corte = tam
		}
		// Se corta tras el último separador del tramo que cabe, si no queda
		// demasiado pegado al principio.
		if idx := strings.LastIndexAny(resto[:corte], `/\`); idx+1 >= ancho/2 {
			corte = idx + 1
		}
		lineas = append(lineas, resto[:corte])
		resto = resto[corte:]
	}
	if resto != "" {
		lineas = append(lineas, resto)
	}
	return lineas
}

// corteAncho devuelve el índice de byte donde el texto deja de caber en `ancho`
// columnas.
func corteAncho(texto string, ancho int) int {
	w := 0
	for i, r := range texto {
		rw := runewidth.RuneWidth(r)
		if w+rw > ancho {
			return i
		}
		w += rw
	}
	return len(texto)
}
