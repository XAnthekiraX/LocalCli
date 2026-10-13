package welcome

import (
	"strings"

	"localcli/internal/tui/render"
)

// Datos son los datos de la sesión que la bienvenida pinta.
type Datos struct {
	Agente string
	Modelo string
	Motor  string
	Aviso  string
	Pensar bool
}

// Rect es un rectángulo de celdas en coordenadas del marco.
type Rect struct {
	Fila    int
	Columna int
	Ancho   int
	Alto    int
}

// Limites es el rectángulo de la caja de entrada dentro del marco y el del
// interruptor de razonamiento que vive en su línea de estado.
type Limites struct {
	Fila    int
	Columna int
	Ancho   int
	Alto    int
	Pensar  Rect
}

// Vista compone la pantalla: logotipo, caja de entrada, pistas y versión, centrados.
func (m Model) Vista(ancho, alto int, datos Datos) string {
	marco, _ := m.VistaLimites(ancho, alto, datos)
	return marco
}

// VistaLimites compone la pantalla y devuelve, además, el rectángulo que ocupa su
// caja de entrada en coordenadas del marco.
func (m Model) VistaLimites(ancho, alto int, datos Datos) (string, Limites) {
	if ancho <= 0 || alto <= 0 {
		return m.bloqueLlano(datos), Limites{}
	}
	arte := strings.TrimRight(m.arteEfectivo(ancho), "\n")
	anchoCaja := m.anchoCaja(ancho, arte)
	interior := max(anchoCaja-2, 1)
	lineas, filaPensar, colPensar := lineasCaja(m, interior, datos)
	cajaEntrada := caja(m, anchoCaja, lineas)
	partes := []string{
		arte,
		cajaEntrada,
		render.RecortarColumnas(barraPistas(), anchoCaja),
	}
	if datos.Aviso != "" {
		partes = append(partes, render.EstiloSutil.Render(render.RecortarColumnas(datos.Aviso, anchoCaja)))
	}
	marco, fila, columna := render.CentrarConOrigen(strings.Join(partes, "\n\n"), ancho, alto)
	limites := Limites{
		Fila:    fila + cuentaLineas(arte) + 1,
		Columna: columna,
		Ancho:   min(anchoCaja, max(ancho-columna, 0)),
		Alto:    len(lineas) + 2,
	}
	// El interruptor vive en una de las líneas interiores de la caja.
	if filaPensar >= 0 {
		limites.Pensar = Rect{
			Fila:    limites.Fila + 1 + filaPensar,
			Columna: limites.Columna + 1 + colPensar,
			Ancho:   render.Ancho(chipPensar(true)),
			Alto:    1,
		}
	}
	return conVersion(marco, ancho), limites
}

// cuentaLineas cuenta las filas de un bloque de texto.
func cuentaLineas(bloque string) int { return strings.Count(bloque, "\n") + 1 }

// arteEfectivo elige el logotipo según el ancho de la terminal: el compacto en
// terminales estrechas, el completo en cuanto hay sitio para él (SPEC-INTERFAZ
// §Arte canónico del logotipo).
func (m Model) arteEfectivo(ancho int) string {
	if m.compacto != "" && ancho < anchoMinimoLogotipoCompleto {
		return m.compacto
	}
	return m.arte
}

// anchoCaja reparte el ancho de la caja: el aire de los lados y el tope, con el
// logotipo en uso como suelo. Nunca excede la terminal, de modo que el bloque
// siempre cabe por estrecha que sea.
func (m Model) anchoCaja(ancho int, arte string) int {
	anchoCaja := max(min(ancho-4, anchoMaximoCaja), anchoDeArte(arte)+2)
	return max(min(anchoCaja, ancho), 1)
}

// anchoArte es el ancho del logotipo completo: el de su fila más ancha.
func (m Model) anchoArte() int { return anchoDeArte(m.arte) }

// anchoDeArte es el ancho de un arte: el de su fila más ancha.
func anchoDeArte(arte string) int {
	ancho := 0
	for _, linea := range strings.Split(arte, "\n") {
		ancho = max(ancho, render.Ancho(linea))
	}
	return ancho
}

// bloqueLlano es lo que se pinta antes de conocer el tamaño de la terminal.
func (m Model) bloqueLlano(datos Datos) string {
	return strings.Join([]string{
		strings.TrimRight(m.arte, "\n"),
		render.EstiloBlanco.Render(render.Nombre + " · " + render.Version),
		render.EstiloSutil.Render("modelo: ") + render.EstiloBlanco.Render(rotuloModelo(datos)),
		render.EstiloAgente.Render("["+datos.Agente+"] > ") + strings.Join(textoConCursor(m, anchoEntradaLlana), "\n"),
	}, "\n\n")
}

// conVersion firma la esquina inferior derecha.
func conVersion(cuerpo string, ancho int) string {
	lineas := strings.Split(cuerpo, "\n")
	ultima := len(lineas) - 1
	if ultima < 0 || strings.TrimSpace(render.SinANSI(lineas[ultima])) != "" {
		return cuerpo
	}
	lineas[ultima] = render.AlinearDerecha(render.EstiloSutil.Render(render.Version), ancho)
	return strings.Join(lineas, "\n")
}
