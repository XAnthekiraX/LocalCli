package tui

import (
	"os"
	"strings"
	"sync"

	"localcli/internal/tui/render"
)

// Aprobacion es una línea del panel de aprobaciones: una decisión que espera
// respuesta de una sesión cualquiera. El identificador es el de la aprobación,
// la sesión es la que la pide y la acción es lo que propone (qué archivo o qué
// comando). Obsoleta marca una línea cuya sesión terminó mientras esperaba: se
// sigue viendo, pero ya no se puede decidir.
type Aprobacion struct {
	ID       string
	Sesion   string
	Accion   string
	Obsoleta bool
}

// Decision es la opción con la que se resuelve una línea del panel.
type Decision int

const (
	// DecisionAprobar aprueba la propuesta; DecisionDeclinar la rechaza.
	DecisionAprobar Decision = iota + 1
	DecisionDeclinar
)

// Las dos opciones cierran la línea y su texto es único para el clic. Se localizan
// por el sufijo completo `aprobar | declinar`, no por la palabra suelta: una
// acción puede contener la misma palabra («aprobar la etapa…») y desplazaría la
// zona del clic.
const (
	opcionAprobar     = "aprobar"
	opcionDeclinar    = "declinar"
	separadorOpciones = " | "
	opcionesLinea     = opcionAprobar + separadorOpciones + opcionDeclinar
)

// Aprobaciones es el panel de decisiones pendientes de todas las sesiones. No es
// el panel de datos: aquel solo informa, este son decisiones que esperan
// respuesta (SPEC-INTERFAZ-ATAJOS), y por eso vive en su propio estado. Se
// muestra solo —para que la decisión se vea— y se cierra en cuanto no queda
// ninguna línea que decidir.
type Aprobaciones struct {
	// Abierto dice si el panel se pinta. Es «hay algo que decidir».
	Abierto bool
	// Items son las líneas, en el orden en que llegaron los eventos.
	Items []Aprobacion
	// Indice es la línea resaltada.
	Indice int
	// casa es la carpeta del usuario, que las líneas abrevian a `~`.
	casa string
}

// NuevaAprobaciones deja el panel vacío y resuelve la carpeta del usuario, que es
// lo único que las líneas necesitan para abreviar las rutas a `~`.
func NuevaAprobaciones() Aprobaciones {
	return Aprobaciones{casa: casaDelUsuario()}
}

// casaDelUsuario se lee la primera vez que hace falta y ya no cambia: la carpeta
// del usuario no se mueve mientras la sesión vive. Si el sistema no la dice, se
// queda vacía y cada ruta se pinta tal cual.
var casaDelUsuario = sync.OnceValue(func() string {
	casa, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return casa
})

// Anotar añade una aprobación pendiente al panel (`peticion_aprobacion`). Un
// identificador repetido actualiza la línea en vez de duplicarla; sin
// identificador no hay nada que decidir y se descarta. Una línea nueva reabre el
// panel: lo que espera decisión se muestra.
func (m *Aprobaciones) Anotar(a Aprobacion) {
	if a.ID == "" {
		return
	}
	for i := range m.Items {
		if m.Items[i].ID == a.ID {
			m.Items[i] = a
			m.Abierto = true
			return
		}
	}
	m.Items = append(m.Items, a)
	m.Abierto = true
}

// Retirar saca la línea resuelta (`aprobacion_resuelta`). Sin nada que decidir el
// panel se cierra solo: no queda hueco. Una resolución de una línea que ya no
// está no hace nada.
func (m *Aprobaciones) Retirar(id string) {
	for i, it := range m.Items {
		if it.ID != id {
			continue
		}
		m.Items = append(m.Items[:i], m.Items[i+1:]...)
		m.ajustar()
		return
	}
}

// MarcarObsoleta deja las líneas de una sesión que terminó visibles pero sin
// decisión posible: la propuesta ya no puede aplicarse (SPEC-INTERFAZ-ATAJOS
// §Flujos alternativos).
func (m *Aprobaciones) MarcarObsoleta(sesionID string) {
	for i := range m.Items {
		if m.Items[i].Sesion == sesionID {
			m.Items[i].Obsoleta = true
		}
	}
}

// Mover desplaza el resaltado por la lista de pendientes sin salirse de ella.
func (m *Aprobaciones) Mover(delta int) {
	if len(m.Items) == 0 {
		return
	}
	m.Indice += delta
	m.ajustar()
}

// Seleccionada devuelve la línea resaltada, si queda alguna que decidir.
func (m *Aprobaciones) Seleccionada() (Aprobacion, bool) {
	if m.Indice < 0 || m.Indice >= len(m.Items) {
		return Aprobacion{}, false
	}
	return m.Items[m.Indice], true
}

// Pendientes cuenta las líneas que siguen esperando decisión: una obsoleta ya no
// espera nada, así que no cuenta.
func (m *Aprobaciones) Pendientes() int {
	n := 0
	for _, it := range m.Items {
		if !it.Obsoleta {
			n++
		}
	}
	return n
}

// Lineas devuelve cada aprobación en el formato documentado, con la seleccionada
// marcada y ajustada al ancho disponible: la acción cede ante las opciones —una
// decisión que no se ve no se puede tomar—. Es texto plano, sin estilo: sirve de
// referencia al ratón.
func (m *Aprobaciones) Lineas(ancho int) []string {
	lineas := make([]string, 0, len(m.Items))
	for i, it := range m.Items {
		lineas = append(lineas, filaAprobacion(it, m.casa, marcaDe(i == m.Indice), ancho))
	}
	return lineas
}

// Render pinta el panel: el título y una línea por aprobación, con la resaltada
// destacada y la obsoleta atenuada, cada línea ajustada a `ancho`. Sin nada que
// decidir devuelve vacío —el panel desaparece sin dejar hueco— y su visibilidad
// la resuelve quien lo compone.
func (m *Aprobaciones) Render(ancho int) string {
	if !m.Abierto || len(m.Items) == 0 {
		return ""
	}
	lineas := []string{render.RecortarColumnas(render.EstiloTitulo.Render("APROBACIONES"), ancho)}
	plas := m.Lineas(ancho)
	for i, it := range m.Items {
		linea := plas[i]
		switch {
		case it.Obsoleta:
			linea = render.EstiloSutil.Render(linea)
		case i == m.Indice:
			linea = render.EstiloUsuario.Render(linea)
		}
		lineas = append(lineas, linea)
	}
	lineas = append(lineas, render.RecortarColumnas(render.EstiloSutil.Render("(clic en aprobar/declinar)"), ancho))
	return strings.Join(lineas, "\n")
}

// Clic resuelve qué línea y qué opción caen bajo una celda del panel, contando
// desde su esquina superior izquierda (la fila 0 es el título) y midiendo la
// línea como se dibujó a ese `ancho`. Devuelve ok=false si la celda no cae sobre
// «aprobar» ni «declinar» —un clic sobre la sesión, la acción o una línea
// obsoleta no decide nada—.
func (m *Aprobaciones) Clic(fila, columna, ancho int) (Aprobacion, Decision, bool) {
	indice := fila - 1
	if indice < 0 || indice >= len(m.Items) {
		return Aprobacion{}, 0, false
	}
	it := m.Items[indice]
	if it.Obsoleta {
		return Aprobacion{}, 0, false
	}
	linea := m.Lineas(ancho)[indice]
	// Las opciones cierran la línea: se busca su sufijo, no la palabra suelta,
	// para que una acción que contenga «aprobar» no desplace la zona del clic.
	j := strings.LastIndex(linea, opcionesLinea)
	if j < 0 {
		return Aprobacion{}, 0, false
	}
	desde := render.Ancho(linea[:j])
	if columna >= desde && columna < desde+render.Ancho(opcionAprobar) {
		return it, DecisionAprobar, true
	}
	desde += render.Ancho(opcionAprobar) + render.Ancho(separadorOpciones)
	if columna >= desde && columna < desde+render.Ancho(opcionDeclinar) {
		return it, DecisionDeclinar, true
	}
	return Aprobacion{}, 0, false
}

// ajustar deja el resaltado dentro de los límites y cierra el panel cuando ya no
// queda ninguna línea.
func (m *Aprobaciones) ajustar() {
	if m.Indice >= len(m.Items) {
		m.Indice = len(m.Items) - 1
	}
	if m.Indice < 0 {
		m.Indice = 0
	}
	m.Abierto = len(m.Items) > 0
}

// marcaDe es el marcador de selección de una línea: el resaltado o el hueco que
// lo sustituye.
func marcaDe(seleccionada bool) string {
	if seleccionada {
		return "› "
	}
	return "  "
}

// filaAprobacion compone la línea de una aprobación SIN recortar la marca de
// selección: la sesión por su identificador abreviado, la acción propuesta —con
// la carpeta del usuario abreviada a `~`— y las dos opciones, ajustada a `ancho`.
// Una obsoleta cambia las opciones por su marca. La tercera opción de decisiones
// sigue por definir y no reserva hueco.
func filaAprobacion(it Aprobacion, casa, marca string, ancho int) string {
	opciones := opcionesLinea
	if it.Obsoleta {
		opciones = "obsoleta"
	}
	return componerFilaAprobacion(marca, idBreve(it.Sesion), abreviarCasa(it.Accion, casa), opciones, ancho)
}

// componerFilaAprobacion arma `marca id | acción | opciones` sin pasar de `ancho`.
// Si hay que ceder, cede la acción: las opciones se conservan enteras y, si ni
// con el prefijo más corto caben, es el prefijo el que se recorta. Así una
// decisión nunca desaparece de una terminal estrecha. Es una función pura.
func componerFilaAprobacion(marca, id, accion, opciones string, ancho int) string {
	if ancho <= 0 {
		return ""
	}
	prefijo := marca + id
	sufijo := " | " + opciones
	separador := " | "
	disponible := ancho - render.Ancho(prefijo) - render.Ancho(sufijo)
	if disponible < render.Ancho(separador)+1 {
		if render.Ancho(sufijo) >= ancho {
			return render.RecortarColumnas(sufijo, ancho)
		}
		return render.RecortarColumnas(prefijo, ancho-render.Ancho(sufijo)) + sufijo
	}
	return prefijo + separador + render.RecortarColumnas(accion, disponible-render.Ancho(separador)) + sufijo
}

// idBreve acorta el identificador de la sesión: ocho caracteres bastan para
// distinguirla a la vista y la línea no se va de largo. Es una función pura.
func idBreve(id string) string {
	r := []rune(id)
	if len(r) > 8 {
		return string(r[:8])
	}
	return id
}

// abreviarCasa cambia la carpeta del usuario por `~`, como en el pie del panel:
// la línea se lee de un vistazo. Sin carpeta conocida el texto se deja tal cual.
// Es una función pura.
func abreviarCasa(texto, casa string) string {
	if casa == "" {
		return texto
	}
	return strings.Replace(texto, casa, "~", 1)
}
