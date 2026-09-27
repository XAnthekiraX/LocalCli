// welcome.go — T-F003 y T-F013: la pantalla de bienvenida.
//
// Fuente de verdad: frontend/01-domain/DOMAIN.md §3 (reglas de la bienvenida:
// logotipo, nombre con versión, línea con el modelo en uso y una línea de
// entrada, con el bloque centrado; "Se pinta sin esperar a Ollama ni a la
// base"), SPEC-INTERFAZ §Pantalla de bienvenida ("Solo hay logotipo, nombre con
// su versión, una línea que muestra el modelo en uso y una línea de entrada.
// Los modelos NO se listan en la bienvenida: se consultan y eligen desde un
// modal") y §Arte canónico del logotipo ("se pinta tal cual", "no se
// reescala"; el bloque completo sí se centra) e INTERFACES §2 ("Crear o retomar
// sesión: al enviar desde la bienvenida… y luego va el mensaje").
//
// Es un modelo con el estado mínimo —lo escrito— y su propio teclado: en la
// bienvenida no existen los atajos de la interfaz principal, y eso se cumple
// aquí y no con un `if` repartido por app.go. Los modelos ya no viven aquí:
// se eligen en el modal de modelos (`Ctrl+X m`, modelsmodal.go) y de la
// pantalla solo se ve la línea con el modelo en uso.
package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Bienvenida es el modelo de la primera pantalla: lo escrito, que será la
// primera petición, y el foco de su única línea de entrada. No guarda la lista
// de modelos: esa vive en el modal (SPEC-INTERFAZ §Pantalla de bienvenida).
type Bienvenida struct {
	Texto string
	Foco  bool
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

// teclaBienvenida resuelve una pulsación de la bienvenida. T-F012-06: también
// pasa por el KeyResolver. Las acciones propias de la interfaz principal
// —panel, aprobaciones, modal de atajos— no están en este mapa: "los atajos de la interfaz
// principal no existen en la bienvenida" (SPEC-INTERFAZ), y eso se cumple
// filtrando el mapa, no con un `if` suelto. Cualquier otra tecla se escribe.
//
// Con el modal de modelos abierto el contexto es el del modal: sus teclas
// (↑/↓/enter/esc) son suyas y la escritura de la bienvenida no las recibe
// (SPEC-INTERFAZ §Modal de modelos: "Mientras el modal está abierto, las teclas
// son solo del modal"). El atajo de abrirlo, en cambio, es global: se puede
// abrir y cerrar desde aquí.
func (a *App) teclaBienvenida(m tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Ctrl+C es la salida de emergencia de Bubble Tea y sigue valiendo aquí
	// aunque el mapa la reasigne (DOMAIN §3: "la salida es Ctrl+C").
	if m.Type == tea.KeyCtrlC {
		return a, tea.Quit
	}

	ctx := ContextoInput
	if a.modalAbierto() {
		ctx = ContextoModal
	}
	accion, cmd := a.TeclaRes.Resolver(m, ctx)
	if accion != AccionNinguna && !accionesDeBienvenida()[accion] {
		// Atajo de la interfaz principal (panel, aprobaciones, atajos…): en la
		// bienvenida no existe, así que se ignora y el carácter puede escribirse.
		accion = AccionNinguna
	}
	if accion == AccionNinguna && a.TeclaRes.EsperandoLeader() {
		return a, cmd // espera de líder activa: el indicador se pinta igual
	}
	switch accion {
	case AccionSalir:
		return a, tea.Quit
	case AccionModalModelos:
		return a, a.abrirModalModelos()
	case AccionSelector:
		// `Ctrl+X l` abre el modal de sesiones también desde la bienvenida: al
		// elegir una, la vista pasa a la principal con esa sesión
		// (SPEC-INTERFAZ §Pantalla de bienvenida).
		if a.Sesiones.Abierto {
			a.Sesiones.Cerrar()
			return a, nil
		}
		return a, a.abrirModalSesiones()
	case AccionEliminarSesion:
		// `Ctrl+D` borra la resaltada; solo con el modal de sesiones abierto.
		if !a.Sesiones.Abierto {
			return a, nil
		}
		return a, a.eliminarSesion()
	case AccionCiclarAgente:
		// Tab alterna `plan` ↔ `build` también en la bienvenida: el indicador
		// se ve junto a su línea de entrada y cambia al instante
		// (SPEC-INTERFAZ §Zonas 2). Con un modal abierto la acción ni siquiera
		// llega —el resolver la filtra por contexto—, así que no cicla.
		a.ciclarAgente()
		return a, nil
	case AccionCerrarSelector:
		// Esc cierra el modal abierto (dismiss). Sin modal no hace nada visible.
		a.cerrarModales()
		return a, nil
	case AccionSubir:
		// Con un modal abierto la flecha es suya; sin él la línea de entrada la
		// recibe (DOMAIN §3: "sin modal abierto no hay navegación: las flechas
		// escriben/historial según su componente").
		switch {
		case a.Modelos.Abierto:
			a.Modelos.Mover(-1)
		case a.Sesiones.Abierto:
			a.Sesiones.Mover(-1)
		}
		return a, nil
	case AccionBajar:
		switch {
		case a.Modelos.Abierto:
			a.Modelos.Mover(1)
		case a.Sesiones.Abierto:
			a.Sesiones.Mover(1)
		}
		return a, nil
	case AccionEnviar:
		switch {
		case a.Modelos.Abierto:
			return a, a.aplicarModelo()
		case a.Sesiones.Abierto:
			// Elegir una sesión desde la bienvenida lleva a la vista principal
			// con el historial de esa sesión (INTERFACES §2).
			if _, ok := a.Sesiones.SesionElegida(); !ok {
				return a, nil
			}
			a.Vista = VistaPrincipal
			return a, a.elegirSesion()
		}
		return a, a.enviarDesdeBienvenida()
	}
	// Con un modal abierto, todo lo demás se lo queda el modal: lo escrito en
	// la línea de entrada espera (SPEC-INTERFAZ §Modal de modelos).
	if a.modalAbierto() {
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
	// El modelo elegido en el modal viaja con la primera petición (DOMAIN §2):
	// se entrega al motor justo antes de enviar, para que la petición salga con
	// el modelo que el usuario eligió y no con el de la autodetección.
	a.fijarModeloEnPuerto()
	a.Chat.AñadirUsuario(texto)
	a.Vista = VistaPrincipal
	return a.enviarCmd(a.Panel.SesionID, texto)
}

// fijarModeloEnPuerto entrega al motor el modelo en uso. Lo elige el usuario
// desde el modal y prevalece sobre la autodetección del arranque
// (SPEC-OLLAMA-PERFIL: "El modelo lo elige el usuario"), así que se vuelve a
// entregar al enviar: si el modal se abrió y se eligió algo, la primera
// petición sale con ese modelo aunque el arranque hubiera detectado otro.
func (a *App) fijarModeloEnPuerto() {
	if a.Modelo != "" {
		a.Puerto.FijarModelo(a.Modelo)
	}
}

// viewBienvenida compone la pantalla completa: el bloque «logotipo + nombre con
// versión + línea de modelo + línea de entrada» se centra en la ventana. El
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
	b.WriteString(a.lineaDeModelo())
	b.WriteString("\n\n")
	// El indicador del agente precede a la línea de entrada, igual que en la
	// interfaz principal: el render canónico de SPEC-INTERFAZ §Pantalla de
	// bienvenida es `[plan] > En qué te ayudo hoy: █╚` (T-F015-02).
	b.WriteString(IndicadorAgente(a.Agente) + "En qué te ayudo hoy: " + a.Bienvenida.Texto)
	if a.Bienvenida.Foco {
		b.WriteString("▌")
	}
	return centrar(b.String(), a.Ancho, a.Alto)
}

// lineaDeModelo pinta el modelo en uso y el recordatorio del atajo que lo
// cambia (SPEC-INTERFAZ §Línea de modelo: "se muestra solo el modelo en uso —el
// detectado en el arranque o el último elegido en el modal— y el recordatorio
// del atajo para cambiarlo"). Sin modelo detectado —Ollama no estaba al
// arrancar— se muestra el hueco con el mismo recordatorio: la pantalla se
// muestra igual y el modal es el camino para elegir uno.
func (a *App) lineaDeModelo() string {
	nombre := a.Modelo
	if nombre == "" {
		nombre = "—"
	}
	return estiloEtiqueta.Render("modelo: "+nombre) + "  (Ctrl+X m cambiar)"
}
