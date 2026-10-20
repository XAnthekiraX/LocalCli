package chat

import (
	"fmt"
	"strings"

	"localcli/internal/tui/render"
)

// Hilo compone el hilo completo para el ancho dado: los bloques del historial,
// las líneas del sistema sueltas y, al final, el turno en vivo. Una fila en
// blanco separa una pieza de la siguiente.
func (m *Model) Hilo(ancho int) string {
	interno := ancho - franja - 2
	if interno < 1 {
		interno = 1
	}
	var lineas []string
	var mapa []int
	m.botones = nil
	// agregar cuelga un bloque del hilo, con una fila en blanco de separación
	// antes de cada uno salvo el primero, y anota a qué pieza pertenece cada línea
	// (−1 para la separación y el turno en vivo). Los botones de copia del bloque
	// se traducen de coordenadas del contenido a coordenadas del hilo: se les suma
	// la línea donde empieza el contenido (tras la celda de aire de arriba) y la
	// columna del prefijo del hablante.
	agregar := func(pieza int, bloque string, hablante Hablante, botones []BotonCopia) {
		if bloque == "" {
			return
		}
		if len(lineas) > 0 {
			lineas = append(lineas, "")
			mapa = append(mapa, -1)
		}
		base := len(lineas)
		for _, l := range strings.Split(bloque, "\n") {
			lineas = append(lineas, l)
			mapa = append(mapa, pieza)
		}
		desplazamiento := 1
		if hablante == Agente {
			desplazamiento = franja + 1
		}
		for _, b := range botones {
			b.Linea += base + 1
			b.Col += desplazamiento
			m.botones = append(m.botones, b)
		}
	}
	for i, pieza := range m.piezas {
		switch p := pieza.(type) {
		case *Segmento:
			contenido, botones := contenidoSegmento(*p, m.MostrarRazonamiento, interno)
			agregar(i, bloque(contenido, p.Hablante, ancho), p.Hablante, botones)
		case *Linea:
			agregar(i, lineaSuelta(*p, ancho), "", nil)
		}
	}
	bloqueEnVivo, botonesEnVivo := m.bloqueVivo(ancho, interno)
	agregar(-1, bloqueEnVivo, Agente, botonesEnVivo)
	m.lineasPieza = mapa
	return strings.Join(lineas, "\n")
}

// Vista pinta la ventana del historial para el alto disponible y deja lista la
// posición del scroll. Si el historial cabe entero se ve todo y se sigue el
// final. Si no cabe y nadie ha subido, la ventana se pega al final para que lo
// último dicho siga a la vista; si se subió, se respeta la posición, acotada a lo
// que hay. Anclar en el final mientras el modelo genera evita que la vista salte
// con cada token. Cuando queda historial fuera se avisa con una línea discreta
// («↑ N líneas arriba» / «↓ N líneas abajo»), que ocupa una fila de la ventana.
func (m *Model) Vista(ancho, alto int) string {
	texto := m.Hilo(ancho)
	m.hiloLineas = nil
	if texto != "" {
		m.hiloLineas = strings.Split(texto, "\n")
	}
	m.altoUltimo = alto
	if alto <= 0 || texto == "" {
		m.offset, m.maxOffset, m.scrolleado = 0, 0, false
		return texto
	}
	lineas := strings.Split(texto, "\n")
	if len(lineas) <= alto {
		m.offset, m.maxOffset, m.scrolleado = 0, 0, false
		return texto
	}
	cuerpo := alto - 2
	if cuerpo < 1 {
		cuerpo = 1
	}
	m.maxOffset = len(lineas) - cuerpo
	if m.scrolleado || m.arrastrando {
		m.offset = acotar(m.offset, 0, m.maxOffset)
	} else {
		m.offset = m.maxOffset
	}
	salida := make([]string, 0, cuerpo+2)
	salida = append(salida, m.avisoArriba())
	salida = append(salida, lineas[m.offset:m.offset+cuerpo]...)
	salida = append(salida, m.avisoAbajo())
	return strings.Join(salida, "\n")
}

// Subir asciende el historial n líneas y deja de seguir el final: lo que se está
// leyendo no debe moverse cuando llegue un token nuevo. Durante un arrastre no
// toca el seguimiento, que lo gobierna el propio arrastre.
func (m *Model) Subir(n int) {
	if n <= 0 {
		return
	}
	if !m.arrastrando {
		m.scrolleado = true
	}
	m.offset = acotar(m.offset-n, 0, m.maxOffset)
}

// Bajar desciende n líneas. Al alcanzar el final vuelve a seguir el final, de
// modo que la vista retoma sola las respuestas nuevas, salvo durante un arrastre.
func (m *Model) Bajar(n int) {
	if n <= 0 {
		return
	}
	m.offset += n
	if m.offset >= m.maxOffset {
		m.offset = m.maxOffset
		if !m.arrastrando {
			m.scrolleado = false
		}
	}
}

// SubirPagina y BajarPagina avanzan una pantalla menos una línea, como el
// paginado de un lector.
func (m *Model) SubirPagina() { m.Subir(m.pagina()) }

// BajarPagina baja una pantalla.
func (m *Model) BajarPagina() { m.Bajar(m.pagina()) }

// IrAlFinal vuelve a pegar la vista al final y a seguir las respuestas nuevas.
func (m *Model) IrAlFinal() {
	m.scrolleado = false
	m.offset = m.maxOffset
}

func (m *Model) pagina() int {
	if m.altoUltimo <= 1 {
		return 1
	}
	return m.altoUltimo - 1
}

// avisoArriba y avisoAbajo dicen cuánto historial queda fuera de la ventana. La
// fila se deja vacía cuando no queda nada: el sitio ya está reservado.
func (m *Model) avisoArriba() string {
	if m.offset <= 0 {
		return ""
	}
	return render.EstiloSutil.Render(fmt.Sprintf("↑ %d líneas arriba", m.offset))
}

func (m *Model) avisoAbajo() string {
	fuera := m.maxOffset - m.offset
	if fuera <= 0 {
		return ""
	}
	return render.EstiloSutil.Render(fmt.Sprintf("↓ %d líneas abajo", fuera))
}

// contenidoSegmento compone las líneas de un segmento: el razonamiento revelado
// arriba, una fila en blanco y la respuesta, con su tiempo al final y atenuado.
// De cada parte sin nada que decir no queda hueco. La respuesta del agente se
// interpreta como Markdown —solo al pintar— y devuelve además los botones de copia
// de sus bloques de código, en coordenadas del contenido (SPEC-VISTA-CODIGO).
func contenidoSegmento(s Segmento, revelar bool, ancho int) ([]string, []BotonCopia) {
	var lineas []string
	if r := renderRazonamiento(s.Razonamiento, revelar, s.SinRazonamiento, ancho); r != "" {
		lineas = append(lineas, strings.Split(r, "\n")...)
	}
	respuesta, botones := respuestaSegmento(s, ancho)
	if len(respuesta) == 0 {
		return lineas, nil
	}
	if len(lineas) > 0 {
		lineas = append(lineas, "")
	}
	sufijo := sufijoDuracion(s.Duracion)
	for i, linea := range respuesta {
		if i == len(respuesta)-1 && sufijo != "" {
			// El tiempo se cuelga de la última línea; del relleno de una superficie
			// de código solo se quita el sobrante, para no comerse el sufijo.
			lineas = append(lineas, sinPaddingFinal(linea)+" "+sufijo)
			continue
		}
		lineas = append(lineas, linea)
	}
	return lineas, botones
}

// respuestaSegmento compone la respuesta de un segmento ya pintada: la del agente
// se interpreta como Markdown y expone los botones de copia de sus bloques; la del
// usuario —y cualquier otra— se envuelve y se pinta llana, sin interpretar nada.
// Es una función pura.
func respuestaSegmento(s Segmento, ancho int) ([]string, []BotonCopia) {
	if s.Hablante == Agente {
		return renderRespuesta(s.Texto, ancho)
	}
	filas := envolver(s.Texto, ancho)
	salida := make([]string, 0, len(filas))
	for _, fila := range filas {
		salida = append(salida, render.EstiloBlanco.Render(fila))
	}
	return salida, nil
}

// sinPaddingFinal quita los espacios de relleno del final de una línea ya pintada,
// conservando su último código ANSI (el reinicio de estilo): así un sufijo pegado
// a la línea no queda tras el relleno y no lo recorta el ancho. Es una función pura.
func sinPaddingFinal(linea string) string {
	if idx := strings.LastIndexByte(linea, 0x1b); idx >= 0 {
		return strings.TrimRight(linea[:idx], " ") + linea[idx:]
	}
	return strings.TrimRight(linea, " ")
}

// lineaSuelta pinta una línea del sistema: va suelta, sin bloque ni banda, en el
// mismo tono atenuado con el que se lee el tiempo de una respuesta. Se recorta al
// ancho para no romper el marco. En una línea de herramienta, el objetivo —la ruta
// entre corchetes— se pinta en el color de ruta. Si la línea lleva un bloque
// visual, este se pinta debajo, separado por una fila en blanco: el bloque es
// adicional, no sustituye a la línea (INTERFACES §1.1). Un bloque de diferencia se
// pinta con el color de cada marca; el resto, atenuado.
func lineaSuelta(l Linea, ancho int) string {
	if strings.TrimSpace(l.Texto) == "" && strings.TrimSpace(l.Bloque) == "" {
		return ""
	}
	var partes []string
	if strings.TrimSpace(l.Texto) != "" {
		texto := render.RecortarColumnas("  "+l.Texto, ancho)
		if l.Tipo == LineaHerramienta {
			partes = append(partes, pintarLineaHerramienta(texto))
		} else {
			partes = append(partes, render.EstiloSutil.Render(texto))
		}
	}
	if strings.TrimSpace(l.Bloque) != "" {
		if len(partes) > 0 {
			partes = append(partes, "")
		}
		if l.BloqueTipo == BloqueDiferencia {
			partes = append(partes, pintarDiferencia(l.Bloque, ancho)...)
		} else {
			for _, bl := range strings.Split(l.Bloque, "\n") {
				partes = append(partes, render.EstiloSutil.Render("  "+bl))
			}
		}
	}
	return strings.Join(partes, "\n")
}

// BloqueDiferencia es el tipo de bloque que pinta el antes y el después de un
// cambio de archivo. Coincide con `chat_evento.bloque_tipo` (ENUMS §Tipo de
// bloque visual).
const BloqueDiferencia = "diferencia"

// pintarDiferencia pinta un diff unificado: las líneas añadidas en verde, las
// quitadas en rojo y los tramos y el contexto atenuados. Las líneas más anchas
// que el hilo se envuelven sin perder texto y la continuación queda alineada
// bajo el contenido; la marca del diff se conserva al principio de cada línea.
func pintarDiferencia(bloque string, ancho int) []string {
	disponible := ancho - 3
	if disponible < 1 {
		disponible = 1
	}
	var filas []string
	for _, linea := range strings.Split(bloque, "\n") {
		estilo := render.EstiloSutil
		switch {
		case strings.HasPrefix(linea, "+"):
			estilo = render.EstiloDiferenciaMas
		case strings.HasPrefix(linea, "-"):
			estilo = render.EstiloDiferenciaMenos
		}
		for i, trozo := range render.EnvolverTexto(linea, disponible) {
			sangria := "  "
			if i > 0 {
				// La continuación deja libre la columna de la marca.
				sangria = "   "
			}
			filas = append(filas, estilo.Render(sangria+trozo))
		}
	}
	return filas
}

// pintarLineaHerramienta pinta una línea de herramienta con su ruta destacada: el
// tramo entre corchetes —el objetivo, «LEER [ruta]» o «LEER [ruta:10-50]»— va en el
// color de ruta y el resto en el tono atenuado. Sin el corchete de cierre —una
// línea recortada— se pinta toda atenuada. Es una función pura.
func pintarLineaHerramienta(texto string) string {
	inicio := strings.IndexByte(texto, '[')
	fin := strings.IndexByte(texto, ']')
	if inicio < 0 || fin < inicio {
		return render.EstiloSutil.Render(texto)
	}
	return render.EstiloSutil.Render(texto[:inicio]) +
		render.EstiloRuta.Render(texto[inicio:fin+1]) +
		render.EstiloSutil.Render(texto[fin+1:])
}

// bloqueVivo es el turno que se está generando: el razonamiento revelado arriba
// y la respuesta. Sin nada que decir no deja bloque. Devuelve también los botones
// de copia de los bloques de código ya cerrados dentro del turno en vivo.
func (m *Model) bloqueVivo(ancho, interno int) (string, []BotonCopia) {
	respuesta := strings.Join(m.respuesta, "")
	razonamiento := strings.Join(m.razonamiento, "")
	if respuesta == "" && razonamiento == "" {
		return "", nil
	}
	enVivo := Segmento{Hablante: Agente, Texto: respuesta, Razonamiento: razonamiento}
	contenido, botones := contenidoSegmento(enVivo, m.MostrarRazonamiento, interno)
	return bloque(contenido, Agente, ancho), botones
}

// acotar encierra un valor entre dos límites.
func acotar(valor, minimo, maximo int) int {
	if valor < minimo {
		return minimo
	}
	if valor > maximo {
		return maximo
	}
	return valor
}
