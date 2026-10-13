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
	var bloques []string
	for _, pieza := range m.piezas {
		switch p := pieza.(type) {
		case *Segmento:
			if b := bloque(contenidoSegmento(*p, m.MostrarRazonamiento, interno), p.Hablante, ancho); b != "" {
				bloques = append(bloques, b)
			}
		case *Linea:
			if l := lineaSuelta(*p, ancho); l != "" {
				bloques = append(bloques, l)
			}
		}
	}
	if vivo := m.bloqueVivo(ancho, interno); vivo != "" {
		bloques = append(bloques, vivo)
	}
	return strings.Join(bloques, "\n\n")
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
// De cada parte sin nada que decir no queda hueco.
func contenidoSegmento(s Segmento, revelar bool, ancho int) []string {
	var lineas []string
	if r := renderRazonamiento(s.Razonamiento, revelar, s.SinRazonamiento, ancho); r != "" {
		lineas = append(lineas, strings.Split(r, "\n")...)
	}
	respuesta := envolver(s.Texto, ancho)
	if len(respuesta) == 0 {
		return lineas
	}
	if len(lineas) > 0 {
		lineas = append(lineas, "")
	}
	for i, linea := range respuesta {
		pintada := render.EstiloBlanco.Render(linea)
		if i == len(respuesta)-1 {
			if sufijo := sufijoDuracion(s.Duracion); sufijo != "" {
				pintada += " " + sufijo
			}
		}
		lineas = append(lineas, pintada)
	}
	return lineas
}

// lineaSuelta pinta una línea del sistema: va suelta, sin bloque ni banda, en el
// mismo tono atenuado con el que se lee el tiempo de una respuesta. Se recorta al
// ancho para no romper el marco.
func lineaSuelta(l Linea, ancho int) string {
	if strings.TrimSpace(l.Texto) == "" {
		return ""
	}
	return render.EstiloSutil.Render(render.RecortarColumnas("  "+l.Texto, ancho))
}

// bloqueVivo es el turno que se está generando: el razonamiento revelado arriba
// y la respuesta. Sin nada que decir no deja bloque.
func (m *Model) bloqueVivo(ancho, interno int) string {
	respuesta := strings.Join(m.respuesta, "")
	razonamiento := strings.Join(m.razonamiento, "")
	if respuesta == "" && razonamiento == "" {
		return ""
	}
	enVivo := Segmento{Hablante: Agente, Texto: respuesta, Razonamiento: razonamiento}
	return bloque(contenidoSegmento(enVivo, m.MostrarRazonamiento, interno), Agente, ancho)
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
