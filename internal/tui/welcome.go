// welcome.go — T-F003: la pantalla de bienvenida.
//
// Fuente de verdad: frontend/01-domain/DOMAIN.md §3 (reglas de la bienvenida:
// logotipo, nombre con versión, selector de modelos y una línea de entrada, con
// el bloque centrado; "Se pinta sin esperar a Ollama ni a la base"),
// SPEC-INTERFAZ §Pantalla de bienvenida y §Arte canónico del logotipo ("se
// pinta tal cual", "no se reescala"; el bloque completo sí se centra) e
// INTERFACES §2 ("Crear o retomar sesión: al enviar desde la bienvenida… y
// luego va el mensaje").
//
// Es un modelo con el estado mínimo —lo escrito y lo elegido en el selector— y
// su propio teclado: en la bienvenida no existen los atajos de la interfaz
// principal, y eso se cumple aquí y no con un `if` repartido por app.go.
package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Bienvenida es el modelo de la primera pantalla: lo escrito, que será la
// primera petición, el foco de su única línea de entrada y el selector de
// modelos (SPEC-INTERFAZ §Reglas: «lo elegido pasa al motor como modelo de la
// sesión», así que la selección vive aquí y no en un componente aparte).
type Bienvenida struct {
	Texto string
	Foco  bool

	// Modelos es la lista que reporta el puerto; llega vacía hasta que arriba
	// `modelosMsg` — la bienvenida se pinta sin esperar a Ollama (DOMAIN §3).
	Modelos []ModeloLocal
	// Elegido es el índice resaltado del selector.
	Elegido int
	// Aviso guarda el error del listado («sin modelos»): se muestra, no bloquea.
	Aviso string
}

// NuevaBienvenida deja la pantalla lista con la entrada enfocada: es la única
// línea que existe aquí, así que el foco es el estado por defecto.
func NuevaBienvenida() Bienvenida { return Bienvenida{Foco: true} }

// Escribir añade lo tecleado a la primera petición.
func (b *Bienvenida) Escribir(s string) {
	if b.Foco {
		b.Texto += s
	}
}

// Borrar quita el último carácter escrito.
func (b *Bienvenida) Borrar() {
	if r := []rune(b.Texto); len(r) > 0 {
		b.Texto = string(r[:len(r)-1])
	}
}

// ModeloElegido devuelve el nombre resaltado en el selector, o vacío si no hay
// lista todavía (sin Ollama, o antes de que llegue `modelosMsg`).
func (b *Bienvenida) ModeloElegido() string {
	if b.Elegido < 0 || b.Elegido >= len(b.Modelos) {
		return ""
	}
	return b.Modelos[b.Elegido].Nombre
}

// MoverModelo desplaza el resaltado del selector con ↑/↓ (SPEC-INTERFAZ: «↑/↓
// mueven la selección»). Sin lista no hay a dónde moverse; el recorrido es
// circular para poder alternar entre modelos sin llegar al borde.
func (b *Bienvenida) MoverModelo(paso int) {
	if len(b.Modelos) == 0 {
		return
	}
	n := (b.Elegido + paso) % len(b.Modelos)
	if n < 0 {
		n += len(b.Modelos)
	}
	b.Elegido = n
}

// fijarModelos deja la lista del selector a punto: conserva la elección anterior
// si el modelo sigue disponible y resalta el primero en caso contrario. El error
// del puerto no se propaga ni bloquea: queda como aviso («sin modelos»,
// SPEC-INTERFAZ §Reglas).
func (b *Bienvenida) fijarModelos(modelos []ModeloLocal, err error) {
	previo := b.ModeloElegido()
	b.Modelos = modelos
	b.Aviso = ""
	if err != nil {
		b.Aviso = "sin modelos"
		b.Modelos = nil
	}
	b.Elegido = 0
	for i, m := range b.Modelos {
		if m.Nombre == previo {
			b.Elegido = i
			break
		}
	}
}

// teclaBienvenida resuelve una pulsación de la bienvenida. T-F012-06: también
// pasa por el KeyResolver, con el mapa recortado a lo que existe aquí (salir,
// enviar, subir/bajar del selector de modelos). Las acciones propias de la
// interfaz principal —panel, aprobaciones, ayuda— no están en este mapa: "los
// atajos de la interfaz principal no existen en la bienvenida" (SPEC-INTERFAZ),
// y eso se cumple filtrando el mapa, no con un `if` suelto. Cualquier otra
// tecla se escribe.
func (a *App) teclaBienvenida(m tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Ctrl+C es la salida de emergencia de Bubble Tea y sigue valiendo aquí
	// aunque el mapa la reasigne (DOMAIN §3: "la salida es Ctrl+C").
	if m.Type == tea.KeyCtrlC {
		return a, tea.Quit
	}

	accion, cmd := a.TeclaRes.Resolver(m, ContextoInput)
	if accion != AccionNinguna && !accionesDeBienvenida()[accion] {
		// Atajo de la interfaz principal (panel, aprobaciones, ayuda…): en la
		// bienvenida no existe, así que se ignora y el carácter puede escribirse.
		accion = AccionNinguna
	}
	if accion == AccionNinguna && a.TeclaRes.EsperandoLeader() {
		return a, cmd // espera de líder activa: el indicador se pinta igual
	}
	switch accion {
	case AccionSalir:
		return a, tea.Quit
	case AccionEnviar:
		return a, a.enviarDesdeBienvenida()
	case AccionSubir:
		a.Bienvenida.MoverModelo(-1)
		a.fijarModeloEnPuerto()
		return a, nil
	case AccionBajar:
		a.Bienvenida.MoverModelo(1)
		a.fijarModeloEnPuerto()
		return a, nil
	}
	switch m.Type {
	case tea.KeyRunes:
		a.Bienvenida.Escribir(string(m.Runes))
	case tea.KeySpace:
		a.Bienvenida.Escribir(" ")
	case tea.KeyBackspace:
		a.Bienvenida.Borrar()
	}
	return a, nil
}

// fijarModeloEnPuerto entrega al motor el modelo resaltado: lo elegido por el
// usuario prevalece sobre la autodetección del arranque (SPEC-OLLAMA-PERFIL).
// Se llama en cada movimiento del selector, para que el cambio sea inmediato.
func (a *App) fijarModeloEnPuerto() {
	if nombre := a.Bienvenida.ModeloElegido(); nombre != "" {
		a.Puerto.FijarModelo(nombre)
	}
}

// cargarModelos pide la lista del selector al puerto. Es un comando asíncrono:
// la pantalla se pinta antes, sin esperar a Ollama (DOMAIN §3), y la lista
// arriba cuando llega — con error incluido, que se muestra como aviso.
func (a *App) cargarModelos() tea.Cmd {
	return func() tea.Msg {
		modelos, err := a.Puerto.Modelos()
		return modelosMsg{Modelos: modelos, Err: err}
	}
}

// enviarDesdeBienvenida manda lo escrito como primera petición: la sesión
// activa se retoma si existe o se crea si no (INTERFACES §2) y después va el
// mensaje. Son las dos operaciones de este envío, cada una exactamente una vez;
// la vista cambia a la principal sin repetir la petición ni pedir
// confirmación (DOMAIN §3).
func (a *App) enviarDesdeBienvenida() tea.Cmd {
	texto := strings.TrimSpace(a.Bienvenida.Texto)
	if texto == "" {
		return nil
	}
	if a.Panel.SesionID == "" {
		ses, err := a.Puerto.ResolverActiva()
		if err != nil {
			a.Chat.AñadirSistema("no se pudo abrir la sesión: " + err.Error())
			return nil
		}
		a.activar(ses)
	}
	a.Bienvenida.Texto = ""
	// Lo elegido en el selector viaja con la primera petición (DOMAIN §2): se
	// fija en el puerto justo antes de enviar, por si el usuario nunca movió
	// las flechas pero Ollama respondió después del arranque.
	a.fijarModeloEnPuerto()
	a.Chat.AñadirUsuario(texto)
	a.Vista = VistaPrincipal
	return a.enviarCmd(a.Panel.SesionID, texto)
}

// viewBienvenida compone la pantalla completa: el bloque «logotipo + nombre con
// versión + selector de modelos + línea de entrada» se centra en la ventana. El
// arte se pinta tal cual —sin reescalar: cada línea conserva sus 53 columnas
// doradas; lo que cambia es solo su posición dentro de la pantalla
// (SPEC-INTERFAZ §Pantalla de bienvenida, donde el render canónico ya muestra el
// bloque centrado)—. Antes de la primera WindowSizeMsg no hay geometría conocida:
// se pinta pegado al margen, como siempre.
func (a *App) viewBienvenida() string {
	var b strings.Builder
	b.WriteString(strings.TrimRight(LogoCanonico, "\n"))
	b.WriteString("\n\n")
	b.WriteString(estiloMarca.Render(Nombre + " · " + Version))
	b.WriteString("\n\n")
	b.WriteString(a.selectorModelos())
	b.WriteString("\n\n")
	b.WriteString("En qué te ayudo hoy: " + a.Bienvenida.Texto)
	if a.Bienvenida.Foco {
		b.WriteString("▌")
	}

	bloque := b.String()
	if a.Ancho <= 0 || a.Alto <= 0 {
		return bloque + "\n"
	}
	return lipgloss.Place(a.Ancho, a.Alto, lipgloss.Center, lipgloss.Center, bloque)
}

// selectorModelos pinta la fila del selector: los nombres separados por el
// marcador «▸», con el elegido resaltado, y el aviso «sin modelos» cuando Ollama
// no respondió (SPEC-INTERFAZ §Reglas). Vacío antes de la primera respuesta: la
// bienvenida nunca espera a nada externo para mostrarse.
func (a *App) selectorModelos() string {
	bienvenida := &a.Bienvenida
	linea := "Modelos: "
	switch {
	case bienvenida.Aviso != "":
		linea += estiloAviso.Render(bienvenida.Aviso)
	case len(bienvenida.Modelos) == 0:
		return ""
	default:
		nombres := make([]string, 0, len(bienvenida.Modelos))
		for i, m := range bienvenida.Modelos {
			if i == bienvenida.Elegido {
				nombres = append(nombres, estiloEtiqueta.Render(m.Nombre))
			} else {
				nombres = append(nombres, m.Nombre)
			}
		}
		linea += strings.Join(nombres, " ▸ ")
	}
	return linea + "    (↑/↓ elegir)"
}
