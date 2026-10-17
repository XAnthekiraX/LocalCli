package tui

import (
	"strings"

	"localcli/internal/tui/render"
)

// Comando es una entrada de la paleta: lo que se escribe tras la barra. Para un
// flujo lleva el prefijo `flow:` (p. ej. "flow:planificar"); para un comando de
// la interfaz, su nombre sin más (p. ej. "models").
type Comando struct {
	Nombre      string // p.ej. "flow:planificar", "models", "sessions", "redo", "init"
	Alias       string // alias opcional
	Descripcion string // opcional
}

// PaletaModel es el estado de la paleta de comandos de flujo.
type PaletaModel struct {
	// comandos es el catálogo completo recibido.
	comandos []Comando
	// filtro es el texto escrito tras '/' para filtrar.
	filtro string
	// resaltado es el índice del comando resaltado en la lista filtrada.
	resaltado int
	// visible indica si la paleta está desplegada.
	visible bool
	// maxAlto es cuántos comandos muestra la paleta a la vez (máximo
	// elementosVisiblesPaleta); el resto se alcanza con scroll o filtrando.
	maxAlto int
	// offsetScroll es el desplazamiento vertical del viewport cuando hay overflow.
	offsetScroll int
}

// NuevoPaleta crea un modelo de paleta vacío.
func NuevoPaleta() PaletaModel {
	return PaletaModel{
		comandos:     []Comando{},
		filtro:       "",
		resaltado:    0,
		visible:      false,
		maxAlto:      elementosVisiblesPaleta,
		offsetScroll: 0,
	}
}

// EstablecerCatalogo recibe el catálogo completo de comandos.
func (p *PaletaModel) EstablecerCatalogo(cmds []Comando) {
	p.comandos = make([]Comando, len(cmds))
	copy(p.comandos, cmds)
	p.resaltado = 0
	p.offsetScroll = 0
	if len(p.filtrados()) == 0 {
		p.resaltado = 0
	}
}

// Catalogo devuelve el catálogo completo.
func (p PaletaModel) Catalogo() []Comando {
	res := make([]Comando, len(p.comandos))
	copy(res, p.comandos)
	return res
}

// Filtro devuelve el texto de filtro actual.
func (p PaletaModel) Filtro() string {
	return p.filtro
}

// EstablecerFiltro establece el texto de filtro.
func (p *PaletaModel) EstablecerFiltro(f string) {
	p.filtro = f
	p.resaltado = 0
	p.offsetScroll = 0
}

// Resaltado devuelve el índice resaltado.
func (p PaletaModel) Resaltado() int {
	return p.resaltado
}

// EstablecerResaltado establece el índice resaltado (se clampa a la lista filtrada).
func (p *PaletaModel) EstablecerResaltado(i int) {
	lista := p.filtrados()
	if len(lista) == 0 {
		p.resaltado = 0
		p.offsetScroll = 0
		return
	}
	if i < 0 {
		p.resaltado = len(lista) - 1
	} else if i >= len(lista) {
		p.resaltado = 0
	} else {
		p.resaltado = i
	}
	p.ajustarScroll()
}

// Subir mueve el resaltado hacia arriba y ajusta el scroll si es necesario.
func (p *PaletaModel) Subir() {
	p.EstablecerResaltado(p.resaltado - 1)
	p.ajustarScroll()
}

// Bajar mueve el resaltado hacia abajo y ajusta el scroll si es necesario.
func (p *PaletaModel) Bajar() {
	p.EstablecerResaltado(p.resaltado + 1)
	p.ajustarScroll()
}

// Visible devuelve si la paleta está visible.
func (p PaletaModel) Visible() bool {
	return p.visible
}

// Mostrar muestra la paleta.
func (p *PaletaModel) Mostrar() {
	p.visible = true
	if len(p.filtrados()) > 0 && p.resaltado < 0 {
		p.resaltado = 0
	}
}

// Ocultar oculta la paleta y limpia el filtro/resaltado.
func (p *PaletaModel) Ocultar() {
	p.visible = false
	p.filtro = ""
	p.resaltado = 0
	p.offsetScroll = 0
}

// Alternar alterna la visibilidad.
func (p *PaletaModel) Alternar() {
	if p.visible {
		p.Ocultar()
	} else {
		p.Mostrar()
	}
}

// elementosVisiblesPaleta es el número máximo de comandos que la paleta muestra
// a la vez. El resto se alcanza desplazando el viewport (▲/▼, `pgup`/`pgdown` o
// la rueda del ratón) o filtrando por el input.
const elementosVisiblesPaleta = 5

// EstablecerMaxAlto fija cuántos comandos muestra la paleta a la vez: como mucho
// `elementosVisiblesPaleta`, y menos si la terminal no da para tanto.
func (p *PaletaModel) EstablecerMaxAlto(altoTerminal int) {
	if altoTerminal <= 0 {
		p.maxAlto = 0
		return
	}
	calculado := elementosVisiblesPaleta
	if altoTerminal < calculado {
		calculado = altoTerminal
	}
	p.maxAlto = calculado
	p.ajustarScroll()
}

// MaxAlto devuelve el máximo de elementos visibles configurado.
func (p PaletaModel) MaxAlto() int {
	return p.maxAlto
}

// ajustarScroll asegura que el offsetScroll sea válido para la lista filtrada actual.
func (p *PaletaModel) ajustarScroll() {
	lista := p.filtrados()
	if len(lista) <= p.maxAlto {
		p.offsetScroll = 0
		return
	}
	// El resaltado debe estar visible: si está por debajo del viewport, bajar el scroll
	if p.resaltado >= p.offsetScroll+p.maxAlto {
		p.offsetScroll = p.resaltado - p.maxAlto + 1
	}
	// Si está por encima del viewport, subir el scroll
	if p.resaltado < p.offsetScroll {
		p.offsetScroll = p.resaltado
	}
	// Clamp máximo
	maxOffset := len(lista) - p.maxAlto
	if p.offsetScroll > maxOffset {
		p.offsetScroll = maxOffset
	}
	if p.offsetScroll < 0 {
		p.offsetScroll = 0
	}
}

// ScrollArriba desplaza el viewport una fila hacia arriba.
func (p *PaletaModel) ScrollArriba() {
	if p.offsetScroll > 0 {
		p.offsetScroll--
	}
}

// ScrollAbajo desplaza el viewport una fila hacia abajo.
func (p *PaletaModel) ScrollAbajo() {
	lista := p.filtrados()
	maxOffset := len(lista) - p.maxAlto
	if maxOffset < 0 {
		maxOffset = 0
	}
	if p.offsetScroll < maxOffset {
		p.offsetScroll++
	}
}

// ScrollPaginaArriba desplaza el viewport una página hacia arriba.
func (p *PaletaModel) ScrollPaginaArriba() {
	pagina := p.maxAlto
	if pagina < 1 {
		pagina = 1
	}
	p.offsetScroll -= pagina
	if p.offsetScroll < 0 {
		p.offsetScroll = 0
	}
}

// ScrollPaginaAbajo desplaza el viewport una página hacia abajo.
func (p *PaletaModel) ScrollPaginaAbajo() {
	lista := p.filtrados()
	maxOffset := len(lista) - p.maxAlto
	if maxOffset < 0 {
		maxOffset = 0
	}
	pagina := p.maxAlto
	if pagina < 1 {
		pagina = 1
	}
	p.offsetScroll += pagina
	if p.offsetScroll > maxOffset {
		p.offsetScroll = maxOffset
	}
}

// Filtrados devuelve la lista de comandos filtrados por el texto actual.
func (p PaletaModel) Filtrados() []Comando {
	return p.filtrados()
}

func (p PaletaModel) filtrados() []Comando {
	f := strings.ToLower(strings.TrimSpace(p.filtro))
	if f == "" {
		res := make([]Comando, len(p.comandos))
		copy(res, p.comandos)
		return res
	}
	var res []Comando
	for _, c := range p.comandos {
		if strings.Contains(strings.ToLower(c.Nombre), f) ||
			strings.Contains(strings.ToLower(c.Alias), f) ||
			strings.Contains(strings.ToLower(c.Descripcion), f) {
			res = append(res, c)
		}
	}
	return res
}

// ComandoResaltado devuelve el comando resaltado, o el valor cero si no hay ninguno.
func (p PaletaModel) ComandoResaltado() Comando {
	lista := p.filtrados()
	if len(lista) == 0 || p.resaltado < 0 || p.resaltado >= len(lista) {
		return Comando{}
	}
	return lista[p.resaltado]
}

// NombreResaltado devuelve el nombre del comando resaltado.
func (p PaletaModel) NombreResaltado() string {
	c := p.ComandoResaltado()
	if c.Nombre == "" {
		return ""
	}
	return c.Nombre
}

// Autocompletar devuelve el texto con el que debe sustituirse la línea actual
// para dejar "/comando [petición]" (según especificación). Delega en CompletarLinea.
func (p PaletaModel) Autocompletar(linea string) string {
	return p.CompletarLinea(linea)
}

// CompletarLinea genera el texto resultante tras autocompletar con el comando
// resaltado, dejando espacio para la petición tras el comando.
func (p PaletaModel) CompletarLinea(linea string) string {
	c := p.ComandoResaltado()
	if c.Nombre == "" {
		return linea
	}
	return "/" + c.Nombre + " "
}

// ActualizarDesdeLinea actualiza el estado de la paleta según el texto
// escrito: aparece al escribir "/" y se retira al escribir un espacio.
func (p *PaletaModel) ActualizarDesdeLinea(linea string) {
	if strings.TrimSpace(linea) == "" {
		p.Ocultar()
		return
	}
	// Si contiene espacio, la petición empieza → retirar paleta
	if strings.Contains(linea, " ") {
		p.Ocultar()
		return
	}
	// Si empieza con /, mostrar y filtrar por lo que viene después
	if strings.HasPrefix(linea, "/") {
		p.Mostrar()
		p.EstablecerFiltro(strings.TrimPrefix(linea, "/"))
		return
	}
	p.Ocultar()
}

// View pinta la paleta desplegada encima de la caja de entrada: cada fila es una
// superficie del color de la paleta, rellenada al ancho de la caja. Si no está
// visible o no hay comandos, devuelve cadena vacía. Respeta maxAlto y muestra
// indicadores de scroll ▲/▼ cuando hay overflow.
func (p PaletaModel) View(ancho int) string {
	if !p.visible {
		return ""
	}
	lista := p.filtrados()
	if len(lista) == 0 || ancho < 1 {
		return ""
	}

	// Si no hay maxAlto configurado o la lista cabe entera, pintar todo sin scroll
	if p.maxAlto <= 0 || len(lista) <= p.maxAlto {
		return p.viewSinScroll(ancho, lista)
	}

	return p.viewConScroll(ancho, lista)
}

// viewSinScroll pinta la lista completa sin scroll (lista cabe en maxAlto o no hay maxAlto).
func (p PaletaModel) viewSinScroll(ancho int, lista []Comando) string {
	lineas := make([]string, 0, len(lista))
	for i, c := range lista {
		texto := "/" + c.Nombre
		if c.Alias != "" && c.Alias != c.Nombre {
			texto = "/" + c.Nombre + " (" + c.Alias + ")"
		}
		texto = recortar(" "+texto, ancho)
		if i == p.resaltado {
			lineas = append(lineas,
				render.PintarFondo(render.EstiloPaletaResaltada.Render(texto), ancho, render.FondoPaletaResaltada))
			continue
		}
		lineas = append(lineas,
			render.PintarFondo(render.EstiloPaleta.Render(texto), ancho, render.FondoPaleta))
	}
	return strings.Join(lineas, "\n")
}

// viewConScroll pinta la ventana visible de la lista con indicadores ▲/▼.
func (p PaletaModel) viewConScroll(ancho int, lista []Comando) string {
	inicio := p.offsetScroll
	fin := inicio + p.maxAlto
	if fin > len(lista) {
		fin = len(lista)
	}

	lineas := make([]string, 0, p.maxAlto)

	// Indicador de scroll arriba (▲) si hay contenido antes del viewport
	if inicio > 0 {
		indicador := " ▲"
		lineas = append(lineas,
			render.PintarFondo(render.EstiloPaleta.Render(indicador), ancho, render.FondoPaleta))
	}

	// Filas visibles
	for i := inicio; i < fin; i++ {
		c := lista[i]
		texto := "/" + c.Nombre
		if c.Alias != "" && c.Alias != c.Nombre {
			texto = "/" + c.Nombre + " (" + c.Alias + ")"
		}
		texto = recortar(" "+texto, ancho)
		if i == p.resaltado {
			lineas = append(lineas,
				render.PintarFondo(render.EstiloPaletaResaltada.Render(texto), ancho, render.FondoPaletaResaltada))
			continue
		}
		lineas = append(lineas,
			render.PintarFondo(render.EstiloPaleta.Render(texto), ancho, render.FondoPaleta))
	}

	// Indicador de scroll abajo (▼) si hay contenido después del viewport
	if fin < len(lista) {
		indicador := " ▼"
		lineas = append(lineas,
			render.PintarFondo(render.EstiloPaleta.Render(indicador), ancho, render.FondoPaleta))
	}

	return strings.Join(lineas, "\n")
}

func recortar(s string, ancho int) string {
	if ancho <= 0 || len([]rune(s)) <= ancho {
		return s
	}
	runes := []rune(s)
	return string(runes[:ancho])
}

// ComandosDeLaEntrada arma el catálogo de la paleta: los flujos que sirve el
// orquestador, con el prefijo `flow:`, y los comandos de la interfaz. Los flujos
// van primero, que son los que más se invocan. Sin puerto (o sin flujos) la
// paleta solo ofrece los comandos locales.
func ComandosDeLaEntrada(p Puerto) []Comando {
	var cmds []Comando
	if p != nil {
		for _, f := range p.Flujos() {
			if f.Comando == "" {
				continue
			}
			cmds = append(cmds, Comando{Nombre: "flow:" + f.Comando})
		}
	}
	return append(cmds,
		Comando{Nombre: "models", Descripcion: "abre el modal de modelos"},
		Comando{Nombre: "descargar", Descripcion: "descarga un modelo desde una URL"},
		Comando{Nombre: "sessions", Descripcion: "abre el modal de sesiones"},
		Comando{Nombre: "undo", Descripcion: "deshace el último turno"},
		Comando{Nombre: "redo", Descripcion: "rehace el turno deshecho"},
		Comando{Nombre: "retry", Descripcion: "repite la última petición"},
		Comando{Nombre: "compactar", Descripcion: "resume el historial de la sesión activa"},
		Comando{Nombre: "init", Descripcion: "inicializa el proyecto (pendiente)"},
	)
}
