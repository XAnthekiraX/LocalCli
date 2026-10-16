package tui

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"localcli/internal/tui/render"
)

// punto es una celda del marco: fila y columna, en base cero.
type punto struct {
	Fila    int
	Columna int
}

// rect es un rectángulo de celdas del marco.
type rect struct {
	Fila    int
	Columna int
	Ancho   int
	Alto    int
}

func (r rect) contiene(p punto) bool {
	return p.Fila >= r.Fila && p.Fila < r.Fila+r.Alto &&
		p.Columna >= r.Columna && p.Columna < r.Columna+r.Ancho
}

// acotar encierra un punto en el rectángulo: es lo que impide que un arrastre
// salga de la zona donde empezó.
func (r rect) acotar(p punto) punto {
	p.Fila = acotarInt(p.Fila, r.Fila, r.Fila+r.Alto-1)
	p.Columna = acotarInt(p.Columna, r.Columna, r.Columna+r.Ancho-1)
	return p
}

func acotarInt(valor, minimo, maximo int) int {
	if valor < minimo {
		return minimo
	}
	if valor > maximo {
		return maximo
	}
	return valor
}

// tramo es el realce resuelto de una fila: las columnas [Desde, Hasta) del marco vigente.
type tramo struct {
	Fila         int
	Desde, Hasta int
}

// seleccion es el arrastre con el botón izquierdo sobre el marco.
//
// El arrastre se limita a la zona donde empezó —el chat, el sidebar o la caja de
// entrada— y queda anclado al texto: guarda una instantánea del marco al iniciar
// (su «ancla») y resuelve el realce buscando ese mismo texto en cada repintado, de
// modo que ni el desplazamiento ni la respuesta generándose token a token lo mueven.
type seleccion struct {
	arrastrando bool
	inicio      punto
	fin         punto

	zona      rect
	tieneZona bool

	ancla []string // líneas del marco al iniciar el arrastre, sin códigos ANSI

	// En el chat la selección va en coordenadas de contenido: líneas del hilo y
	// columnas de la ventana, para poder prolongarse fuera del viewport.
	contenido              bool
	lineaInicio, colInicio int
	lineaFin, colFin       int
}

func (s *seleccion) iniciar(p punto, lineas []string, zona rect, tieneZona bool) {
	s.limpiar()
	s.arrastrando = true
	s.ancla = lineas
	s.tieneZona = tieneZona
	if tieneZona {
		s.zona = zona
		p = zona.acotar(p)
	}
	s.inicio, s.fin = p, p
}

// iniciarContenido ancla un arrastre del chat en una línea del hilo.
func (s *seleccion) iniciarContenido(linea, col int, zona rect) {
	s.limpiar()
	s.arrastrando = true
	s.contenido = true
	s.tieneZona = true
	s.zona = zona
	s.lineaInicio = linea
	s.colInicio = acotarInt(col, zona.Columna, zona.Columna+zona.Ancho-1)
	s.lineaFin, s.colFin = s.lineaInicio, s.colInicio
}

// extenderContenido lleva el extremo del arrastre del chat a una línea del hilo.
func (s *seleccion) extenderContenido(linea, col int) {
	if !s.arrastrando || !s.contenido {
		return
	}
	s.lineaFin = linea
	s.colFin = acotarInt(col, s.zona.Columna, s.zona.Columna+s.zona.Ancho-1)
}

// extender prolonga el arrastre hasta un punto del marco vigente, ya acotado a la
// zona de origen y traducido a las coordenadas del ancla si el marco se repintó.
func (s *seleccion) extender(p punto, vigentes []string) {
	if !s.arrastrando {
		return
	}
	if s.tieneZona {
		p = s.zona.acotar(p)
	}
	s.fin = punto{Fila: filaAncla(s.ancla, vigentes, p.Fila), Columna: p.Columna}
}

func (s *seleccion) limpiar() { *s = seleccion{} }

// extremos devuelve el arrastre ordenado por filas.
func (s seleccion) extremos() (arriba, abajo punto) {
	if s.inicio.Fila <= s.fin.Fila {
		return s.inicio, s.fin
	}
	return s.fin, s.inicio
}

// extremosContenido ordena el arrastre del chat por líneas del hilo.
func (s seleccion) extremosContenido() (arriba, abajo punto) {
	a := punto{Fila: s.lineaInicio, Columna: s.colInicio}
	b := punto{Fila: s.lineaFin, Columna: s.colFin}
	if a.Fila <= b.Fila {
		return a, b
	}
	return b, a
}

// tramo devuelve las columnas seleccionadas de una fila.
func (s seleccion) tramo(fila int) (desde, hasta int, hay bool) {
	arriba, abajo := s.extremos()
	return s.tramoDe(arriba, abajo, fila)
}

// tramoContenido devuelve las columnas seleccionadas de una línea del hilo.
func (s seleccion) tramoContenido(linea int) (desde, hasta int, hay bool) {
	arriba, abajo := s.extremosContenido()
	return s.tramoDe(arriba, abajo, linea)
}

// tramoDe reparte el rectángulo [arriba, abajo] en las columnas que caen en `fila`.
func (s seleccion) tramoDe(arriba, abajo punto, fila int) (desde, hasta int, hay bool) {
	if fila < arriba.Fila || fila > abajo.Fila {
		return 0, 0, false
	}
	switch {
	case arriba.Fila == abajo.Fila:
		return min(arriba.Columna, abajo.Columna), max(arriba.Columna, abajo.Columna) + 1, true
	case fila == arriba.Fila:
		return arriba.Columna, s.bordeDerecho() + 1, true
	case fila == abajo.Fila:
		return s.bordeIzquierdo(), abajo.Columna + 1, true
	default:
		return s.bordeIzquierdo(), s.bordeDerecho() + 1, true
	}
}

// bordeIzquierdo y bordeDerecho acotan el ancho de las filas intermedias del
// arrastre: con zona, hasta su rectángulo; sin ella, el marco entero.
func (s seleccion) bordeIzquierdo() int {
	if s.tieneZona {
		return s.zona.Columna
	}
	return 0
}

func (s seleccion) bordeDerecho() int {
	if s.tieneZona {
		return s.zona.Columna + s.zona.Ancho - 1
	}
	return anchoSinFin
}

const anchoSinFin = 1 << 20

// texto extrae lo elegido del ancla, sin los espacios de la superficie: el padding
// del marco y las celdas de aire no viajan al portapapeles.
func (s seleccion) texto() string {
	var seleccionadas []string
	for fila, linea := range s.ancla {
		desde, hasta, hay := s.tramo(fila)
		if !hay {
			continue
		}
		tramo := render.TramoColumnas(linea, desde, hasta)
		seleccionadas = append(seleccionadas, strings.Trim(tramo, " "))
	}
	return strings.Trim(strings.Join(seleccionadas, "\n"), "\n")
}

// textoContenido extrae lo elegido en el hilo: recorre las líneas de contenido,
// así que la copia incluye las que quedan fuera del viewport.
func (s seleccion) textoContenido(lineas []string) string {
	arriba, abajo := s.extremosContenido()
	var seleccionadas []string
	for l := arriba.Fila; l <= abajo.Fila; l++ {
		if l < 0 || l >= len(lineas) {
			continue
		}
		desde, hasta, hay := s.tramoContenido(l)
		if !hay {
			continue
		}
		seleccionadas = append(seleccionadas, strings.Trim(render.TramoColumnas(lineas[l], desde, hasta), " "))
	}
	return strings.Trim(strings.Join(seleccionadas, "\n"), "\n")
}

// resolver mapea la selección anclada a las filas del marco vigente: el realce
// sigue al texto elegido aunque un repintado haya movido las líneas.
func (s seleccion) resolver(vigentes []string) []tramo {
	arriba, abajo := s.extremos()
	var tramos []tramo
	for fila := arriba.Fila; fila <= abajo.Fila; fila++ {
		desde, hasta, hay := s.tramo(fila)
		if !hay {
			continue
		}
		destino := filaVigente(s.ancla, vigentes, fila)
		if destino < 0 || destino >= len(vigentes) {
			continue
		}
		ancho := render.Ancho(vigentes[destino])
		if desde >= ancho {
			continue
		}
		tramos = append(tramos, tramo{Fila: destino, Desde: desde, Hasta: min(hasta, ancho)})
	}
	return tramos
}

// resolverContenido resuelve el realce del chat: mapea cada línea de contenido a
// su fila de ventana con `filaVentana`, de modo que solo se realza lo visible.
func (s seleccion) resolverContenido(lineas []string, filaVentana func(int) (int, bool)) []tramo {
	arriba, abajo := s.extremosContenido()
	var tramos []tramo
	for l := arriba.Fila; l <= abajo.Fila; l++ {
		if l < 0 || l >= len(lineas) {
			continue
		}
		destino, ok := filaVentana(l)
		if !ok {
			continue
		}
		desde, hasta, _ := s.tramoContenido(l)
		if ancho := render.Ancho(lineas[l]); desde < ancho {
			tramos = append(tramos, tramo{Fila: destino, Desde: desde, Hasta: min(hasta, ancho)})
		}
	}
	return tramos
}

// filaVigente busca en el marco vigente la fila que contiene el mismo texto que la
// fila `filaAncla` del ancla, prefiriendo la más cercana a su posición original.
func filaVigente(ancla, vigentes []string, filaAncla int) int {
	if filaAncla < 0 || filaAncla >= len(ancla) {
		return -1
	}
	if filaAncla < len(vigentes) && vigentes[filaAncla] == ancla[filaAncla] {
		return filaAncla
	}
	return masCercana(ancla[filaAncla], vigentes, filaAncla)
}

// filaAncla es la vuelta atrás: traduce una fila del marco vigente a la del ancla.
func filaAncla(ancla, vigentes []string, filaVigente int) int {
	if filaVigente < 0 || filaVigente >= len(vigentes) || filaVigente >= len(ancla) {
		return filaVigente
	}
	if ancla[filaVigente] == vigentes[filaVigente] {
		return filaVigente
	}
	if destino := masCercana(vigentes[filaVigente], ancla, filaVigente); destino >= 0 {
		return destino
	}
	return filaVigente
}

// masCercana devuelve la fila de `lineas` con ese texto más próxima a `fila`, o -1.
func masCercana(texto string, lineas []string, fila int) int {
	mejor, distancia := -1, 0
	for i, linea := range lineas {
		if linea != texto {
			continue
		}
		d := absInt(i - fila)
		if mejor == -1 || d < distancia {
			mejor, distancia = i, d
		}
	}
	return mejor
}

func absInt(valor int) int {
	if valor < 0 {
		return -valor
	}
	return valor
}

// lineasRueda es lo que avanza cada muesca de la rueda del ratón en el historial
// del chat.
const lineasRueda = 3

func (a *App) raton(m tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch {
	case m.Action == tea.MouseActionPress && m.Button == tea.MouseButtonLeft:
		p := punto{Fila: m.Y, Columna: m.X}
		a.clicGlobo = clicGlobo{}
		// El menú del globo, si está abierto, atiende primero: un clic sobre una
		// de sus filas resuelve la opción; fuera de él se cierra y el clic sigue
		// su curso normal.
		if a.globo.Visible() {
			if accion, ok := a.clicEnMenuGlobo(p); ok {
				return a, a.ejecutarMenuGlobo(accion)
			}
			a.globo.Cerrar()
		}
		if a.decidirAprobacionEn(p) {
			return a, nil
		}
		if a.sobreInterruptorPensar(p) {
			a.alternarPensar()
			return a, nil
		}
		// El botón de copiar de un bloque de código atiende antes que la selección:
		// copiar no debe arrastrar el ratón ni abrir el menú del globo.
		if cmd, ok := a.copiarBotonEn(p); ok {
			return a, cmd
		}
		a.prepararClicGlobo(p)
		a.empezarSeleccion(p)
	case m.Action == tea.MouseActionMotion && a.sel.arrastrando:
		actual := punto{Fila: m.Y, Columna: m.X}
		// Cualquier movimiento convierte el clic en un arrastre: el candidato a
		// menú se descarta y la selección sigue su curso.
		if a.clicGlobo.activo && actual != a.clicGlobo.inicio {
			a.clicGlobo.activo = false
		}
		if a.sel.contenido {
			a.moverSeleccionChat(actual)
		} else {
			a.sel.extender(actual, a.lineasMarco())
		}
	case m.Action == tea.MouseActionRelease && m.Button == tea.MouseButtonLeft:
		// Un clic sin arrastre sobre el globo de un mensaje del usuario abre su
		// menú en vez de copiar la selección.
		if a.clicGlobo.activo {
			id := a.clicGlobo.mensajeID
			contenido := a.clicGlobo.contenido
			a.clicGlobo = clicGlobo{}
			a.chat.ReanudarSeguimiento(a.scrollAntes)
			a.sel.limpiar()
			a.globo.Abrir(id, contenido)
			return a, nil
		}
		if a.sel.arrastrando {
			return a, a.terminarSeleccion()
		}
		a.clicGlobo = clicGlobo{}
	case m.Button == tea.MouseButtonWheelUp:
		if a.paleta.Visible() && a.ratonSobrePaleta(punto{Fila: m.Y, Columna: m.X}) {
			a.paleta.ScrollArriba()
			return a, nil
		}
		if a.vista == VistaPrincipal {
			a.chat.Subir(lineasRueda)
		}
	case m.Button == tea.MouseButtonWheelDown:
		if a.paleta.Visible() && a.ratonSobrePaleta(punto{Fila: m.Y, Columna: m.X}) {
			a.paleta.ScrollAbajo()
			return a, nil
		}
		if a.vista == VistaPrincipal {
			a.chat.Bajar(lineasRueda)
		}
	}
	return a, nil
}

// ratonSobrePaleta verifica si el ratón está sobre la zona de la paleta de comandos.
func (a *App) ratonSobrePaleta(p punto) bool {
	fila, col, ancho, alto := a.rectPaleta()
	if alto <= 0 {
		return false
	}
	return p.Fila >= fila && p.Fila < fila+alto &&
		p.Columna >= col && p.Columna < col+ancho
}

// decidirAprobacionEn resuelve un clic sobre «aprobar» o «declinar» del panel de
// aprobaciones. Devuelve false si el clic no cae dentro del panel o no resuelve
// ninguna línea: entonces el clic sigue su curso normal (selección).
func (a *App) decidirAprobacionEn(p punto) bool {
	zona := a.zonaAprobaciones
	if zona.Alto <= 0 || !zona.contiene(p) {
		return false
	}
	item, decision, ok := a.aprobaciones.Clic(p.Fila-zona.Fila, p.Columna, zona.Ancho)
	if !ok {
		return false
	}
	a.decidirItem(item, decision == DecisionAprobar)
	return true
}

// empezarSeleccion ancla el arrastre en la zona pulsada. En el chat va en
// coordenadas de contenido y suspende el seguimiento del final al soltar.
func (a *App) empezarSeleccion(p punto) {
	zona, tiene := a.zonaSeleccion(p)
	if tiene && a.vista == VistaPrincipal && zona == a.zonaChat {
		if linea, ok := a.chat.ContenidoBajoVentana(p.Fila - zona.Fila); ok {
			a.scrollAntes = a.chat.SuspenderSeguimiento()
			a.sel.iniciarContenido(linea, p.Columna, zona)
			return
		}
	}
	a.sel.iniciar(p, a.lineasMarco(), zona, tiene)
}

// margenAutoScroll son las filas del borde del chat que, al arrastrar sobre
// ellas, desplazan el historial y prolongan la selección por lo que entra.
const margenAutoScroll = 2

// moverSeleccionChat prolonga el arrastre del chat; contra un borde superior o
// inferior desplaza el historial una línea y abarca la línea que entra.
func (a *App) moverSeleccionChat(p punto) {
	zona := a.zonaChat
	if zona.Alto <= 0 {
		return
	}
	fila := acotarInt(p.Fila-zona.Fila, 0, zona.Alto-1)
	switch {
	case fila < margenAutoScroll:
		a.chat.Subir(1)
	case fila >= zona.Alto-margenAutoScroll:
		a.chat.Bajar(1)
	}
	if linea, ok := a.chat.ContenidoBajoVentana(fila); ok {
		a.sel.extenderContenido(linea, p.Columna)
	}
}

// lineasMarco es el marco vigente en texto plano: la referencia de coordenadas del ratón.
func (a *App) lineasMarco() []string {
	return strings.Split(render.SinANSI(a.marco()), "\n")
}

// zonaSeleccion devuelve el rectángulo de la zona de selección que contiene el
// punto: la caja de entrada, el chat o el sidebar. Un punto fuera de las tres —una
// banda vacía del marco— no acota el arrastre. Un arrastre nunca mezcla zonas: el
// realce se queda en la que lo vio empezar (INTERFACES §Ratón).
func (a *App) zonaSeleccion(p punto) (rect, bool) {
	for _, zona := range []rect{a.zonaEntrada, a.zonaChat, a.zonaPanel} {
		if zona.Alto > 0 && zona.Ancho > 0 && zona.contiene(p) {
			return zona, true
		}
	}
	return rect{}, false
}

// terminarSeleccion copia lo elegido y apaga el realce. En el chat el tramo sale
// del hilo entero —incluidas las líneas fuera del viewport— y reanuda el
// seguimiento que el arrastre había suspendido.
func (a *App) terminarSeleccion() tea.Cmd {
	var texto string
	if a.sel.contenido {
		texto = a.sel.textoContenido(a.chat.LineasContenido())
		a.chat.ReanudarSeguimiento(a.scrollAntes)
	} else {
		texto = a.sel.texto()
	}
	a.sel.limpiar()
	if texto == "" {
		return nil
	}
	return tea.Batch(copiarCmd(texto), a.avisarCopiado())
}

func copiarCmd(texto string) tea.Cmd {
	return func() tea.Msg {
		copiarFunc(texto)
		return nil
	}
}

// copiarFunc es la copia efectiva; variable para poder sustituirla en las pruebas.
var copiarFunc = copiarAlPortapapeles

// copiarAlPortapapeles usa las herramientas del sistema y, si no hay ninguna,
// OSC 52, que copia por la propia terminal.
func copiarAlPortapapeles(texto string) {
	for _, orden := range [][]string{
		{"wl-copy"},
		{"xclip", "-selection", "clipboard"},
		{"xsel", "--clipboard", "--input"},
		{"pbcopy"},
	} {
		if _, err := exec.LookPath(orden[0]); err != nil {
			continue
		}
		cmd := exec.Command(orden[0], orden[1:]...)
		cmd.Stdin = strings.NewReader(texto)
		if err := cmd.Run(); err == nil {
			return
		}
	}
	_, _ = fmt.Fprint(os.Stdout, "\x1b]52;c;"+base64.StdEncoding.EncodeToString([]byte(texto))+"\x07")
}

// duracionCopiado es lo que el aviso [Copiado] permanece antes de apagarse solo.
const duracionCopiado = 1500 * time.Millisecond

type copiadoExpiradoMsg struct{ gen uint64 }

func (a *App) avisarCopiado() tea.Cmd {
	a.copiado = true
	a.copiadoGen++
	gen := a.copiadoGen
	return tea.Tick(duracionCopiado, func(time.Time) tea.Msg { return copiadoExpiradoMsg{gen: gen} })
}

// conIndicadorSuperior escribe una etiqueta discreta en la esquina superior
// derecha del marco: el aviso [Copiado] o el indicador de la tecla líder.
func conIndicadorSuperior(frame, etiqueta string) string {
	if etiqueta == "" {
		return frame
	}
	lineas := strings.Split(frame, "\n")
	if len(lineas) == 0 {
		return frame
	}
	inicio := render.Ancho(lineas[0]) - render.Ancho(etiqueta)
	if inicio < 0 {
		return frame
	}
	lineas[0] = render.RecortarColumnas(lineas[0], inicio) + etiqueta
	return strings.Join(lineas, "\n")
}
