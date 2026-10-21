package tui

import tea "github.com/charmbracelet/bubbletea"

// tecla traduce una pulsación. Con un modal abierto, sus teclas son solo suyas:
// nada llega al input ni a la vista de abajo, y las que no son acción se pierden
// en el modal, que es un diálogo. Sin modal, el texto y las teclas de edición van
// al componente con el foco; el resto se resuelve como acción con el KeyResolver
// y se despacha. Ningún switch compara ya strings de tecla de acción.
func (a *App) tecla(m tea.KeyMsg) (tea.Model, tea.Cmd) {
	if a.modalAbierto() {
		return a.teclaDeModal(m)
	}
	// El menú del globo, mientras está desplegado, es dueño del teclado: ↑/↓ lo
	// recorren, Enter resuelve la opción y Esc lo cierra; el resto no pasa a la
	// caja ni a la vista de abajo.
	if a.globo.Visible() {
		return a.teclaMenuGlobo(m)
	}
	// `esc` detiene al agente desde el núcleo, sin depender del keymap del
	// usuario: un `keys.json` que ignore `cancel` no puede dejarte sin parada.
	if m.Type == tea.KeyEsc && !a.teclaRes.EsperandoLider() && a.cancelarTurno != nil && a.chat.HayTurno() {
		a.detenerTurno()
		return a, nil
	}
	// Con el panel de aprobaciones enfocado (Ctrl+A), a/d resuelven la línea
	// resaltada y arriba/abajo la mueven; sin foco, la caja sigue escribiendo.
	if a.aprobaciones.Abierto && a.focoAprobaciones {
		return a.teclaAprobaciones(m)
	}
	// Cuando la paleta está visible, interceptar teclas específicas.
	if a.paleta.Visible() {
		switch m.Type {
		case tea.KeyUp:
			a.paleta.Subir()
			return a, nil
		case tea.KeyDown:
			a.paleta.Bajar()
			return a, nil
		case tea.KeyPgUp:
			a.paleta.ScrollPaginaArriba()
			return a, nil
		case tea.KeyPgDown:
			a.paleta.ScrollPaginaAbajo()
			return a, nil
		case tea.KeyTab:
			// Tab autocompleta: sustituye la línea en la vista vigente
			if a.vista == VistaBienvenida {
				completado := a.paleta.CompletarLinea(a.bienvenida.Texto)
				a.bienvenida.Texto = completado
				a.bienvenida.Pos = len([]rune(completado))
			} else {
				completado := a.paleta.CompletarLinea(a.entrada.Texto)
				a.entrada.Texto = completado
				a.entrada.Pos = len([]rune(completado))
			}
			a.paleta.ActualizarDesdeLinea(a.textoVigente())
			return a, nil
		}
		// Enter no se interrumpe aquí: cae al envío normal y ejecuta lo escrito
		// (SPEC-INTERFAZ §Zonas 2: `Tab` autocompleta y `Enter` ejecuta).
	}
	if !a.teclaRes.EsperandoLider() && EsTeclaDeEdicion(m) {
		return a, a.editar(m)
	}
	accion, cmdLider := a.teclaRes.Resolver(m, ContextoInput)
	if accion == AccionNinguna && a.teclaRes.EsperandoLider() {
		// El resolver entró en LIDER: la pulsación no hace nada más y el comando
		// arma el temporizador que produce LiderExpiradoMsg.
		return a, cmdLider
	}
	return a.despachar(accion, m)
}

// teclaAprobaciones resuelve una pulsación con el panel de aprobaciones con el
// foco: arriba/abajo mueven el resaltado y `a`/`d` —los literales del mapa del
// usuario— resuelven la línea resaltada.
func (a *App) teclaAprobaciones(m tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.Type {
	case tea.KeyUp:
		a.aprobaciones.Mover(-1)
		return a, nil
	case tea.KeyDown:
		a.aprobaciones.Mover(1)
		return a, nil
	}
	accion, cmdLider := a.teclaRes.Resolver(m, ContextoAprobaciones)
	if accion == AccionNinguna && a.teclaRes.EsperandoLider() {
		return a, cmdLider
	}
	return a.despachar(accion, m)
}

// teclaDeModal resuelve una pulsación con un modal abierto: el contexto es el del
// modal —el de sesiones añade `session_delete` y el de modelos `model_delete`— y
// una tecla sin acción no se entrega a nadie, porque lo de abajo no está en
// pantalla (SPEC-INTERFAZ §Modales).
func (a *App) teclaDeModal(m tea.KeyMsg) (tea.Model, tea.Cmd) {
	if a.modal == ModalDescargar {
		return a.teclaDescarga(m)
	}
	// La pregunta de borrado del modal de modelos se responde con `s`/`n`: no son
	// acciones del mapa, sino la respuesta al diálogo. `esc` la cancela sin
	// cerrar el modal; cualquier otra tecla no contesta.
	if a.modal == ModalModelos && a.modelos.Confirmando() {
		switch {
		case m.Type == tea.KeyEsc:
			a.modelos.CancelarConfirmacion()
			return a, nil
		case m.Type == tea.KeyRunes && len(m.Runes) == 1 && string(m.Runes) == "s":
			return a, a.eliminarModeloResaltado()
		case m.Type == tea.KeyRunes && len(m.Runes) == 1 && string(m.Runes) == "n":
			a.modelos.CancelarConfirmacion()
			return a, nil
		}
	}
	accion, cmdLider := a.teclaRes.Resolver(m, a.contextoModal())
	if accion == AccionNinguna && a.teclaRes.EsperandoLider() {
		// La pulsación entró en la espera de la líder: la tecla no hace nada más
		// y el comando arma el temporizador de la espera.
		return a, cmdLider
	}
	return a.despachar(accion, m)
}

// teclaDescarga atiende el modal de descarga: mientras se escribe la URL las
// teclas de edición van al campo; `enter` lanza la descarga, `Tab` cambia de
// campo (GGUF ↔ mmproj) y `esc` cierra —cancelando antes la descarga en curso—.
// Es un campo de texto, no una lista: no pasa por el resolver de atajos.
func (a *App) teclaDescarga(m tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.Type {
	case tea.KeyEsc:
		if a.descarga.Bajando() && a.puerto != nil {
			a.puerto.CancelarDescarga()
		}
		a.cerrarModales()
		return a, nil
	case tea.KeyEnter:
		return a, a.iniciarDescarga()
	case tea.KeyTab:
		a.descarga.CambiarCampo()
		return a, nil
	}
	a.editarDescarga(m)
	return a, nil
}

// editarDescarga entrega una tecla de edición al campo de URL del modal y
// recalcula el nombre derivado.
func (a *App) editarDescarga(m tea.KeyMsg) {
	switch m.Type {
	case tea.KeyRunes:
		a.descarga.Escribir(string(m.Runes))
	case tea.KeySpace:
		a.descarga.Escribir(" ")
	case tea.KeyBackspace:
		a.descarga.Borrar()
	case tea.KeyDelete:
		a.descarga.BorrarAdelante()
	case tea.KeyLeft:
		a.descarga.Izquierda()
	case tea.KeyRight:
		a.descarga.Derecha()
	case tea.KeyHome, tea.KeyCtrlB:
		a.descarga.Inicio()
	case tea.KeyEnd, tea.KeyCtrlE:
		a.descarga.Fin()
	default:
		return
	}
	if a.puerto != nil {
		a.descarga.FijarNombre(a.puerto.NombreDeURL(a.descarga.URL))
	}
}

// editarBienvenida mueve el cursor y edita la línea de la bienvenida. Son las
// teclas de edición, no acciones del mapa.
func (a *App) editarBienvenida(m tea.KeyMsg) {
	switch m.Type {
	case tea.KeyRunes:
		// Un pegado llega como un único KeyRunes con Paste en alto (bracketed
		// paste): se tokeniza (rutas a `[archivo.ext]`/`[CARPETA N elementos]`,
		// texto a `[PEGADO N líneas]`) y se envía expandido. Lo tecleado letra a
		// letra se escribe tal cual.
		if m.Paste {
			a.bienvenida.InsertarPegado(string(m.Runes))
		} else {
			a.bienvenida.Escribir(string(m.Runes))
		}
	case tea.KeySpace:
		a.bienvenida.Escribir(" ")
	case tea.KeyBackspace:
		a.bienvenida.Borrar()
	case tea.KeyDelete:
		a.bienvenida.BorrarAdelante()
	case tea.KeyLeft:
		a.bienvenida.Izquierda()
	case tea.KeyRight:
		a.bienvenida.Derecha()
	case tea.KeyHome, tea.KeyCtrlB:
		a.bienvenida.Inicio()
	case tea.KeyEnd, tea.KeyCtrlE:
		a.bienvenida.Fin()
	}
	a.paleta.ActualizarDesdeLinea(a.bienvenida.Texto)
}
