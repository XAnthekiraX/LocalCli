// modelsmodal.go — T-F013: el modal de modelos (`Ctrl+X m`).
//
// Fuente de verdad: SPEC-INTERFAZ §Modales ("Tres modales centrados comparten el
// mismo comportamiento: uno abierto a la vez, sus teclas capturan el teclado
// (↑/↓ navegan, Enter aplica y cierra), Esc cierra sin cambios y Ctrl+C sigue
// saliendo") y §Pantalla de bienvenida (línea de modelo y modal: "la lista se
// pide a Ollama al abrir el modal (no en el arranque); si Ollama no responde, el
// modal muestra el aviso «sin modelos» y se puede cerrar con Esc sin bloquear
// nada"), DOMAIN §1 (`modals`: los tres modales con la misma mecánica) y
// SPEC-KEYBINDS §Resolución por contexto (con un modal abierto, sus teclas no
// llegan a la vista de abajo).
//
// El modal no decide nada (DOMAIN §4): entrega la elección al `app`, que la
// entrega al motor. Aquí solo hay lista, resaltado y descarte. La mecánica
// común vive en `Modal`, que heredarán los modales de sesiones y de atajos
// (T-F014).
package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Modal es la mecánica común de los modales de la TUI (DOMAIN §1 `modals`): una
// lista con un elemento resaltado, un aviso y un descarte. No sabe de modelos ni
// de sesiones: solo de "hay una lista abierta y el usuario la está recorriendo".
type Modal struct {
	// Titulo es la cabecera del modal («MODELOS», «SESIONES», «ATAJOS»).
	Titulo string
	// Abierto decide si el modal existe: solo uno a la vez (SPEC-INTERFAZ §Modales).
	Abierto bool
	// Lineas es el texto de cada elemento, ya listo para pintar.
	Lineas []string
	// Indice es el elemento resaltado; 0 mientras la lista llega vacía.
	Indice int
	// Aviso sustituye a la lista cuando la carga falla o aún no ha llegado
	// («cargando…», «sin modelos»). No bloquea: se pinta y se puede cerrar.
	Aviso string
	// Pie es la línea de teclas del pie. Vacío usa PieModal; un modal de solo
	// lectura lo cambia, porque no hay nada que aplicar.
	Pie string
}

// PieModal es el pie de los modales con algo que elegir: navegar, aplicar y
// cerrar (SPEC-INTERFAZ §Modales).
const PieModal = "(↑/↓ mover · enter aplicar · esc cerrar)"

// Abrir muestra el modal con la lista dada. Un modal sin lista sigue siendo un
// modal: se abre con su aviso («cargando…») y se cierra con Esc.
func (m *Modal) Abrir(titulo string, lineas []string, aviso string) {
	m.Titulo = titulo
	m.Lineas = lineas
	m.Aviso = aviso
	m.Indice = 0
	m.Abierto = true
}

// Cerrar lo esconde sin devolver nada: descartar no cambia nada (SPEC-INTERFAZ
// §Modales: "Esc cierra sin cambios").
func (m *Modal) Cerrar() { m.Abierto = false }

// Mover cambia el resaltado sin salirse de la lista. Con la lista vacía o el
// aviso en pantalla no hay a dónde moverse: no hace nada.
func (m *Modal) Mover(delta int) {
	if len(m.Lineas) == 0 {
		return
	}
	m.Indice = (m.Indice + delta + len(m.Lineas)) % len(m.Lineas)
}

// Elegida devuelve el índice resaltado y si hay algo que elegir.
func (m *Modal) Elegida() (int, bool) {
	if len(m.Lineas) == 0 || m.Indice < 0 || m.Indice >= len(m.Lineas) {
		return 0, false
	}
	return m.Indice, true
}

// Render pinta el modal centrado en la ventana: cabecera, lista con el
// resaltado y el pie con las teclas. Con aviso, el aviso sustituye a la lista:
// «cargando…» mientras llega y «sin modelos» si Ollama no responde
// (SPEC-INTERFAZ §Modal de modelos).
//
// Las filas se rellenan a un ancho común: el centrado alinea cada línea por su
// cuenta, así que sin rellenar la tabla saldría escalonada.
func (m *Modal) Render(ancho, alto int) string {
	if !m.Abierto {
		return ""
	}
	filas := make([]string, 0, len(m.Lineas))
	for i, linea := range m.Lineas {
		if i == m.Indice {
			filas = append(filas, estiloUsuario.Render("› "+linea))
			continue
		}
		filas = append(filas, "  "+linea)
	}
	anchoFilas := 0
	for _, f := range filas {
		if w := lipgloss.Width(f); w > anchoFilas {
			anchoFilas = w
		}
	}

	var b strings.Builder
	b.WriteString(estiloTitulo.Render(m.Titulo) + "\n\n")
	if m.Aviso != "" {
		b.WriteString(estiloAviso.Render(m.Aviso) + "\n")
	}
	for _, f := range filas {
		b.WriteString(f + strings.Repeat(" ", anchoFilas-lipgloss.Width(f)) + "\n")
	}
	b.WriteString(estiloSistema.Render(m.pie()))
	return centrar(b.String(), ancho, alto)
}

// pie devuelve el pie del modal: el de fábrica, o el que el modal haya puesto
// —el de atajos, que es de solo lectura—.
func (m *Modal) pie() string {
	if m.Pie != "" {
		return m.Pie
	}
	return PieModal
}

// ModelsModal es el modal de modelos: la lista que reporta Ollama, cargada al
// abrirlo, y el modelo que el usuario elige (SPEC-INTERFAZ §Modal de modelos).
type ModelsModal struct {
	Modal
	// Modelos es la lista real, en el mismo orden que Modal.Lineas: la elegida
	// se traduce con el índice resaltado.
	Modelos []ModeloLocal
	// Actual es el nombre del modelo en uso: al rellenar la lista, el resaltado
	// arranca en él para no obligar a recorrerla desde el principio.
	Actual string
}

// AvisoCargando es lo que se ve mientras Ollama responde: el modal se abre al
// instante y se rellena cuando llega la lista, sin bloquear la pantalla.
const AvisoCargando = "cargando…"

// AvisoSinModelos es lo que se ve cuando Ollama no responde o no tiene modelos.
// Se muestra y se puede cerrar con Esc: nada se bloquea (SPEC-INTERFAZ,
// criterio: "Sin Ollama disponible… al abrir el modal aparece el aviso «sin
// modelos» y se puede escribir y enviar sin él").
const AvisoSinModelos = "sin modelos"

// AbrirModelos muestra el modal vacío y pidiendo la lista. Cada apertura es una
// petición: el modal no guarda la lista entre aperturas para que lo que se ve
// sea siempre lo que hay ahora en Ollama. `actual` es el modelo en uso: el
// resaltado caerá en él cuando llegue la lista.
func (mm *ModelsModal) AbrirModelos(actual string) {
	mm.Modelos = nil
	mm.Actual = actual
	mm.Modal.Abrir("MODELOS", nil, AvisoCargando)
}

// FijarModelos rellena el modal con la lista que llegó. El error del puerto no
// bloquea ni propaga: queda como aviso «sin modelos» (SPEC-INTERFAZ §Modal de
// modelos). Si el modal ya no está abierto, la lista se descarta: llegó tarde.
// Al rellenar, el resaltado arranca en el modelo en uso (o en el primero si no
// está en la lista), igual que el modal de sesiones enfoca la sesión activa.
func (mm *ModelsModal) FijarModelos(modelos []ModeloLocal, err error) {
	if !mm.Abierto {
		return
	}
	if err != nil || len(modelos) == 0 {
		mm.Modelos = nil
		mm.Modal.Abrir(mm.Titulo, nil, AvisoSinModelos)
		return
	}
	mm.Modelos = modelos
	lineas := make([]string, 0, len(modelos))
	for _, m := range modelos {
		lineas = append(lineas, lineaDeModelo(m))
	}
	mm.Modal.Abrir(mm.Titulo, lineas, "")
	mm.Indice = mm.IndiceDe(mm.Actual)
}

// IndiceDe devuelve la posición de un modelo en la lista, o 0 si no está: al
// abrir, el resaltado arranca en el modelo en uso.
func (mm *ModelsModal) IndiceDe(nombre string) int {
	for i, m := range mm.Modelos {
		if m.Nombre == nombre {
			return i
		}
	}
	return 0
}

// lineaDeModelo compone la fila del modelo rotulada `motor / modelo` —para que
// el mismo nombre no se confunda entre motores (SPEC-MODELO-MOTOR §Modelos)—
// más la marca de las capacidades que no declara, para saberlo antes de elegirlo
// (SPEC-OLLAMA-PERFIL: el usuario cambia de modelo si el suyo no sirve). Una
// ficha sin dato se marca con `?` junto al nombre; el que no puede sí lo declara
// con texto: `?` es duda y el texto es dato (SPEC-MODELO-MOTOR §Capacidades).
func lineaDeModelo(m ModeloLocal) string {
	fila := m.Nombre
	if m.Motor != "" {
		fila = m.Motor + " / " + m.Nombre
	}
	marcas := ""
	if m.CapacidadesSinDato {
		marcas += "  [?]"
	}
	if m.SinHerramientas {
		marcas += "  (sin herramientas)"
	}
	if m.SinVision {
		marcas += "  (sin visión)"
	}
	return fila + marcas
}

// ModeloTrasCambiarDeMotor decide qué modelo sigue vigente al aplicar un motor
// distinto: el mismo nombre si el motor nuevo lo declara, y vacío si no —no se
// arrastra un nombre que el motor nuevo no tiene (SPEC-MODELO-MOTOR §Cambiar a
// mitad de conversación)—. Es una función pura: no toca el puerto.
func ModeloTrasCambiarDeMotor(actual string, modelos []ModeloLocal) string {
	if actual == "" {
		return ""
	}
	for _, m := range modelos {
		if m.Nombre == actual {
			return actual
		}
	}
	return ""
}

// ModeloElegidoLocal devuelve el modelo resaltado con todos sus datos (nombre
// y si le faltan herramientas). Sin lista no hay nada que elegir.
func (mm *ModelsModal) ModeloElegidoLocal() (ModeloLocal, bool) {
	i, ok := mm.Elegida()
	if !ok {
		return ModeloLocal{}, false
	}
	return mm.Modelos[i], true
}

// ModeloElegido devuelve el nombre del modelo resaltado, listo para viajar al
// motor. Vacío si el modal está vacío o sin lista todavía.
func (mm *ModelsModal) ModeloElegido() string {
	m, ok := mm.ModeloElegidoLocal()
	if !ok {
		return ""
	}
	return m.Nombre
}
