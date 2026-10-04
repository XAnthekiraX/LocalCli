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
	"github.com/charmbracelet/lipgloss"
)

// Bienvenida es el modelo de la primera pantalla: lo escrito, que será la
// primera petición, el foco de su única línea de entrada y la posición del
// cursor (en runas). No guarda la lista de modelos: esa vive en el modal
// (SPEC-INTERFAZ §Pantalla de bienvenida).
type Bienvenida struct {
	Texto string
	Foco  bool
	// Pos es la posición del cursor en runas dentro de Texto (0..len). Permite
	// editar en cualquier punto, igual que la línea de la vista principal.
	Pos int
	// adjuntos recuerda las imágenes pegadas o arrastradas: se muestran como
	// [nombre.ext] y se expanden a su ruta al enviar.
	adjuntos adjuntos
}

// NuevaBienvenida deja la pantalla lista con la entrada enfocada: es la única
// línea que existe aquí, así que el foco es el estado por defecto.
func NuevaBienvenida() Bienvenida { return Bienvenida{Foco: true} }

func (b *Bienvenida) clampPos() {
	if b.Pos < 0 {
		b.Pos = 0
	}
	if n := len([]rune(b.Texto)); b.Pos > n {
		b.Pos = n
	}
}

// Escribir inserta lo tecleado en la posición del cursor y lo avanza.
func (b *Bienvenida) Escribir(s string) {
	if !b.Foco {
		return
	}
	b.clampPos()
	r := []rune(b.Texto)
	ins := []rune(s)
	nuevo := make([]rune, 0, len(r)+len(ins))
	nuevo = append(nuevo, r[:b.Pos]...)
	nuevo = append(nuevo, ins...)
	nuevo = append(nuevo, r[b.Pos:]...)
	b.Texto = string(nuevo)
	b.Pos += len(ins)
}

// Borrar quita el carácter anterior al cursor (retroceso).
func (b *Bienvenida) Borrar() {
	if !b.Foco {
		return
	}
	b.clampPos()
	r := []rune(b.Texto)
	if b.Pos == 0 {
		return
	}
	nuevo := append([]rune{}, r[:b.Pos-1]...)
	nuevo = append(nuevo, r[b.Pos:]...)
	b.Texto = string(nuevo)
	b.Pos--
}

// BorrarAdelante quita el carácter en la posición del cursor (suprimir).
func (b *Bienvenida) BorrarAdelante() {
	b.clampPos()
	r := []rune(b.Texto)
	if b.Pos >= len(r) {
		return
	}
	nuevo := append([]rune{}, r[:b.Pos]...)
	nuevo = append(nuevo, r[b.Pos+1:]...)
	b.Texto = string(nuevo)
}

// Izquierda/Derecha/Inicio/Fin mueven el cursor sin tocar el texto.
func (b *Bienvenida) Izquierda() { b.Pos--; b.clampPos() }
func (b *Bienvenida) Derecha()   { b.Pos++; b.clampPos() }
func (b *Bienvenida) Inicio()    { b.Pos = 0 }
func (b *Bienvenida) Fin()       { b.Pos = len([]rune(b.Texto)) }

// Limpiar vacía la línea y devuelve el cursor al principio.
func (b *Bienvenida) Limpiar() {
	b.Texto = ""
	b.Pos = 0
	b.adjuntos.Olvidar()
}

// AnotarPegado convierte en tokens las rutas de imagen de un texto pegado o
// arrastrado y devuelve lo que hay que escribir en la línea.
func (b *Bienvenida) AnotarPegado(texto string) string { return b.adjuntos.Anotar(texto) }

// TextoExpandido devuelve lo escrito con los tokens de imagen expandidos a su
// ruta real: es el texto que se envía y con el que trabaja el resto del harness.
func (b *Bienvenida) TextoExpandido() string { return b.adjuntos.Expandir(b.Texto) }

// Resaltar pinta los tokens de imagen de un texto ya renderizado.
func (b *Bienvenida) Resaltar(texto string) string { return b.adjuntos.Resaltar(texto) }

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
	// El formulario y la confirmación del modal de motores capturan el teclado
	// aunque se hayan abierto desde la bienvenida.
	if a.Motores.Abierto && a.Motores.FormAbierto() {
		return a.teclaFormMotor(m)
	}
	if a.Motores.Abierto && a.Motores.ConfirmandoBorrado() {
		return a.teclaConfirmarBorrarMotor(m)
	}

	ctx := ContextoInput
	if a.modalAbierto() {
		ctx = a.contextoModal()
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
	case AccionAyuda:
		// `Ctrl+P` abre el modal de atajos también desde la bienvenida: la barra
		// de pistas lo anuncia (`Ctrl+P comandos`) y el modal es global
		// (SPEC-INTERFAZ §Modales y §Pantalla de bienvenida).
		if a.AtajosModal.Abierto {
			a.AtajosModal.Cerrar()
			return a, nil
		}
		return a, a.abrirModalAtajos()
	case AccionModalModelos:
		return a, a.abrirModalModelos()
	case AccionModalMotores:
		// `Ctrl+X i` abre el modal de motores también desde la bienvenida: al
		// elegir uno con `Enter` pasa a ser el de la sesión
		// (SPEC-INTERFAZ §Pantalla de bienvenida).
		if a.Motores.Abierto {
			a.Motores.Cerrar()
			return a, nil
		}
		return a, a.abrirModalMotores()
	case AccionMotorNuevo:
		if a.Motores.Abierto && !a.Motores.FormAbierto() {
			a.Motores.AbrirAlta()
		}
		return a, nil
	case AccionMotorEditar:
		if a.Motores.Abierto && !a.Motores.FormAbierto() {
			a.Motores.AbrirEdicion()
		}
		return a, nil
	case AccionEliminarMotor:
		if !a.Motores.Abierto {
			return a, nil
		}
		return a, a.eliminarMotorResaltado()
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
		// Con la paleta desplegada, Tab autocompleta el comando resaltado y deja
		// la línea lista para escribir la petición detrás.
		if a.Paleta.Abierto {
			if c, ok := a.Paleta.Seleccionado(); ok {
				a.Bienvenida.Texto = c.Nombre + " "
				a.Bienvenida.Pos = len([]rune(a.Bienvenida.Texto))
				a.Bienvenida.adjuntos.Olvidar()
				a.Paleta.Filtrar(a.Bienvenida.Texto)
			}
			return a, nil
		}
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
		// Con la paleta desplegada la flecha la recorre; con un modal abierto es
		// suya; sin nada abierto la línea de entrada la recibe (DOMAIN §3: "sin
		// modal abierto no hay navegación: las flechas escriben/historial según
		// su componente").
		if a.Paleta.Abierto {
			a.Paleta.Mover(-1)
			return a, nil
		}
		switch {
		case a.Motores.Abierto:
			a.Motores.Mover(-1)
		case a.Modelos.Abierto:
			a.Modelos.Mover(-1)
		case a.Sesiones.Abierto:
			a.Sesiones.Mover(-1)
		}
		return a, nil
	case AccionBajar:
		if a.Paleta.Abierto {
			a.Paleta.Mover(1)
			return a, nil
		}
		switch {
		case a.Motores.Abierto:
			a.Motores.Mover(1)
		case a.Modelos.Abierto:
			a.Modelos.Mover(1)
		case a.Sesiones.Abierto:
			a.Sesiones.Mover(1)
		}
		return a, nil
	case AccionEnviar:
		switch {
		case a.Motores.Abierto && a.Motores.FormAbierto():
			return a, a.guardarMotor()
		case a.Motores.Abierto:
			return a, a.aplicarMotor()
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
		texto := string(m.Runes)
		if m.Paste {
			texto = a.Bienvenida.AnotarPegado(texto)
		}
		a.Bienvenida.Escribir(texto)
	case tea.KeySpace:
		a.Bienvenida.Escribir(" ")
	case tea.KeyBackspace:
		a.Bienvenida.Borrar()
	case tea.KeyDelete:
		a.Bienvenida.BorrarAdelante()
	case tea.KeyLeft:
		a.Bienvenida.Izquierda()
	case tea.KeyRight:
		a.Bienvenida.Derecha()
	case tea.KeyHome:
		a.Bienvenida.Inicio()
	case tea.KeyEnd:
		a.Bienvenida.Fin()
	}
	// Lo escrito alimenta la paleta de comandos: al empezar por `/` se despliega
	// y al escribir un espacio (la petición) se retira.
	a.Paleta.Filtrar(a.Bienvenida.Texto)
	return a, nil
}

// enviarDesdeBienvenida manda lo escrito como primera petición: no existe una
// sesión previa, así que se crea una nueva (con su nombre provisional; el título
// lo genera el modelo con esta petición) y después va el mensaje. Son las dos
// operaciones de este envío, cada una exactamente una vez; la vista cambia a la
// principal sin repetir la petición ni pedir confirmación (DOMAIN §3,
// SPEC-SESIONES: una sesión nueva se crea con la primera petición desde la
// bienvenida).
func (a *App) enviarDesdeBienvenida() tea.Cmd {
	texto := strings.TrimSpace(a.Bienvenida.TextoExpandido())
	if texto == "" {
		return nil
	}
	// Un comando de flujo se responde en el chat sin mandarlo al modelo; no
	// necesita crear sesión (SPEC-INTERFAZ §Reglas de negocio).
	if c, ok := a.comandoAplicable(texto); ok {
		return a.ejecutarComando(c, texto)
	}
	if a.Panel.SesionID == "" {
		ses, err := a.Puerto.Crear()
		if err != nil {
			a.Chat.AñadirSistema("no se pudo crear la sesión: " + err.Error())
			return nil
		}
		a.activar(ses)
	}
	a.Bienvenida.Limpiar()
	// El modelo elegido en el modal viaja con la primera petición (DOMAIN §2):
	// se entrega al motor justo antes de enviar, para que la petición salga con
	// el modelo que el usuario eligió y no con el de la autodetección.
	a.fijarModeloEnPuerto()
	a.Chat.AñadirUsuario(texto)
	a.iniciarTurno()
	a.Vista = VistaPrincipal
	imagenes := a.prepararAdjuntos(texto)
	return a.enviarCmd(a.Panel.SesionID, texto, imagenes)
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

// etiquetaBienvenida es el rótulo que precede al texto que se escribe en la
// bienvenida, entre el indicador del agente y la línea de entrada.
const etiquetaBienvenida = "En qué te ayudo hoy: "

// anchoMaxBienvenida acota el ancho de las tarjetas de la bienvenida estilo
// opencode: en terminales muy anchas, una caja de lado a lado se ve desierta.
const anchoMaxBienvenida = 72

// placeholderPeticion es el texto de ejemplo que muestra la caja de la
// bienvenida mientras no se ha escrito nada. Va entre comillas, como en la
// maqueta de la pantalla.
const placeholderPeticion = `"` + placeholderEntrada + `"`

// viewBienvenida compone la pantalla completa. Con geometría conocida se pinta
// el diseño estilo opencode —logotipo sobre el fondo, caja de entrada con su
// línea de agente y modelo, barra de pistas de teclado y la versión abajo a la
// derecha—, sin glifos de adorno. Sin geometría (antes de la primera
// WindowSizeMsg) no hay con qué medir ni centrar los bloques: se pinta el bloque
// llano de siempre, que además nunca recorta el arte.
func (a *App) viewBienvenida() string {
	if a.Ancho <= 0 || a.Alto <= 0 {
		return a.bienvenidaLlana()
	}
	return a.bienvenidaTarjetas()
}

// bienvenidaTarjetas pinta la bienvenida: el logotipo sobre el fondo, la caja de
// entrada (una superficie de fondo, sin glifos de borde) y su barra de pistas.
// Así la selección con el ratón cubre el área de la caja y la copia trae solo el
// texto.
func (a *App) bienvenidaTarjetas() string {
	// El ancho de las cajas se acota, pero nunca por debajo del arte: así el
	// estilo de la bienvenida no cambia al estrechar la terminal. El arte se
	// pinta íntegro (la terminal recorta lo que no quepa, sin reescalarlo).
	anchoCaja := max(min(a.Ancho-4, anchoMaxBienvenida), lipgloss.Width(LogoCanonico)+2)

	// El logotipo va directo sobre el fondo: el espacio que lo rodea es parte de
	// la pantalla, no una caja aparte. `centrar` lo alinea con el resto.
	arte := strings.TrimRight(LogoCanonico, "\n")
	cajaEntrada := cajaEntradaBienvenida(anchoCaja, a.lineasEntradaBienvenida(anchoCaja-2))

	partes := []string{arte}
	// La paleta de comandos se despliega encima de la caja de entrada.
	if pal := a.Paleta.Render(); pal != "" {
		partes = append(partes, pal)
	}
	partes = append(partes, cajaEntrada, a.pistasBienvenida())
	// Sin ningún motor dado de alta la bienvenida no bloquea ni inventa uno: lo
	// dice y ofrece abrir el modal (SPEC-MODELO-MOTOR §Ningún motor activo por
	// defecto). El resto de la pantalla sigue viva y se puede escribir.
	if aviso := a.avisoSinMotor(); aviso != "" {
		partes = append(partes, estiloAviso.Render(aviso))
	}
	// El aviso transitorio (p. ej. el modelo sin herramientas) se ve aquí: la
	// bienvenida es donde primero se elige modelo.
	if a.Aviso != "" {
		partes = append(partes, estiloAviso.Render(a.Aviso))
	}

	cuerpo := centrar(strings.Join(partes, "\n\n"), a.Ancho, a.Alto)
	lineas := strings.Split(cuerpo, "\n")
	// La firma con la versión va abajo a la derecha, como en la maqueta, con el
	// texto en el tono de la caja de entrada (#222436). Solo se pisa la última
	// fila si está vacía: en una terminal diminuta podría ser contenido.
	if n := len(lineas); n > 0 && strings.TrimSpace(sinANSI(lineas[n-1])) == "" {
		version := lipgloss.NewStyle().Foreground(fondoBienvenida).Render(Version)
		lineas[n-1] = lineasALaDerecha(version, a.Ancho)
	}
	return strings.Join(lineas, "\n")
}

// lineasEntradaBienvenida compone el contenido de la caja de entrada: el
// placeholder entre comillas mientras está vacía, o lo escrito con su cursor, y
// debajo la línea de estado con el agente y el modelo en uso.
func (a *App) lineasEntradaBienvenida(anchoInterno int) []string {
	var lineas []string
	if strings.TrimSpace(a.Bienvenida.Texto) == "" {
		lineas = append(lineas, estiloSutil.Render(placeholderPeticion))
	} else {
		envueltas, fila, col := envolverConCursor(a.Bienvenida.Texto, a.Bienvenida.Pos, anchoInterno)
		for i, l := range envueltas {
			if a.Bienvenida.Foco && i == fila {
				rr := []rune(l)
				lineas = append(lineas,
					a.Bienvenida.Resaltar(string(rr[:col]))+"▌"+a.Bienvenida.Resaltar(string(rr[col:])))
			} else {
				lineas = append(lineas, a.Bienvenida.Resaltar(l))
			}
		}
	}
	// Una línea en blanco separa el texto escrito del modelo en uso.
	lineas = append(lineas, "", a.lineaEstadoBienvenida())
	return lineas
}

// lineaEstadoBienvenida pinta la línea de estado de la caja: el agente activo y
// el modelo rotulado con el motor que lo sirve («[plan] • qwen3:8b (llama.cpp)»).
func (a *App) lineaEstadoBienvenida() string {
	return estiloIndicador.Render("["+a.Agente+"]") + estiloBlanco.Render(" • "+a.rotuloModelo())
}

// pistasBienvenida compone la barra de pistas de teclado bajo la caja de
// entrada. Se genera desde el mapa vigente, no se escribe a mano: si el usuario
// reasigna un atajo, la pista cambia con él.
func (a *App) pistasBienvenida() string {
	var partes []string
	for _, p := range []struct {
		accion   Accion
		etiqueta string
	}{
		{AccionCiclarAgente, "agentes"},
		{AccionAyuda, "comandos"},
		{AccionModalModelos, "modelos"},
	} {
		if pista := a.pistaDeAccion(p.accion, p.etiqueta); pista != "" {
			partes = append(partes, pista)
		}
	}
	return strings.Join(partes, "   ")
}

// pistaDeAccion devuelve «Tecla etiqueta» para la acción dada, con la tecla en su
// forma legible; vacío si la acción no tiene atajo en el mapa.
func (a *App) pistaDeAccion(accion Accion, etiqueta string) string {
	for _, at := range a.Atajos {
		if at.Accion != accion || len(at.Secuencias) == 0 {
			continue
		}
		return estiloBlanco.Render(etiquetaDeSecuencia(at.Secuencias[0])) + " " + estiloSutil.Render(etiqueta)
	}
	return ""
}

// etiquetaDeSecuencia da la forma legible de un atajo: «TAB», «Ctrl+P» o
// «Ctrl+X M» (con la tecla líder ya expandida en Paso1).
func etiquetaDeSecuencia(s Secuencia) string {
	if s.Paso2 != "" {
		return etiquetaDeTecla(s.Paso1) + " " + strings.ToUpper(s.Paso2)
	}
	return etiquetaDeTecla(s.Paso1)
}

// etiquetaDeTecla pone en forma legible un literal de una sola tecla:
// «ctrl+p» → «Ctrl+P», «tab» → «TAB».
func etiquetaDeTecla(lit string) string {
	partes := strings.Split(lit, "+")
	for i, p := range partes {
		switch strings.ToLower(p) {
		case "ctrl":
			partes[i] = "Ctrl"
		case "alt":
			partes[i] = "Alt"
		case "shift":
			partes[i] = "Shift"
		case "tab":
			partes[i] = "TAB"
		default:
			partes[i] = strings.ToUpper(p)
		}
	}
	return strings.Join(partes, "+")
}

// bienvenidaLlana es el bloque sin tarjetas: logotipo, nombre con versión, línea
// de modelo y línea de entrada, centrado. Se pinta cuando aún no hay geometría o
// cuando la terminal es demasiado estrecha para enmarcar el arte sin recortarlo.
// El arte se pinta tal cual, sin reescalar: cada línea conserva sus 53 columnas.
func (a *App) bienvenidaLlana() string {
	var b strings.Builder
	b.WriteString(strings.TrimRight(LogoCanonico, "\n"))
	b.WriteString("\n\n")
	b.WriteString(estiloMarca.Render(Nombre + " · " + Version))
	b.WriteString("\n\n")
	b.WriteString(a.lineaDeModelo())
	b.WriteString("\n\n")
	// La paleta de comandos se despliega encima de la línea de entrada.
	if pal := a.Paleta.Render(); pal != "" {
		b.WriteString(pal)
		b.WriteString("\n\n")
	}
	// El indicador del agente precede a la línea de entrada, igual que en la
	// interfaz principal. El texto salta de renglón al desbordar el ancho en vez
	// de recortarse (T-F035): las filas de continuación se sangran al ancho del
	// prefijo y el cursor se pinta en su sitio dentro de la línea envuelta.
	prefijo := IndicadorAgente(a.Agente) + etiquetaBienvenida
	anchoPrefijo := anchoIndicador(a.Agente) + len([]rune(etiquetaBienvenida))
	anchoTexto := a.Ancho - anchoPrefijo
	if a.Ancho <= 0 {
		anchoTexto = anchoEntradaPorDefecto
	}
	if anchoTexto < 1 {
		anchoTexto = 1
	}
	lineas, fila, col := envolverConCursor(a.Bienvenida.Texto, a.Bienvenida.Pos, anchoTexto)
	for i, l := range lineas {
		if i > 0 {
			b.WriteString("\n")
			b.WriteString(strings.Repeat(" ", anchoPrefijo))
		} else {
			b.WriteString(prefijo)
		}
		if a.Bienvenida.Foco && i == fila {
			rr := []rune(l)
			b.WriteString(a.Bienvenida.Resaltar(string(rr[:col])) + "▌" + a.Bienvenida.Resaltar(string(rr[col:])))
		} else {
			b.WriteString(a.Bienvenida.Resaltar(l))
		}
	}
	// Sin ningún motor dado de alta la bienvenida no bloquea ni inventa uno: lo
	// dice y ofrece abrir el modal (SPEC-MODELO-MOTOR §Ningún motor activo por
	// defecto).
	if aviso := a.avisoSinMotor(); aviso != "" {
		b.WriteString("\n\n" + estiloAviso.Render(aviso))
	}
	// El aviso transitorio (p. ej. el modelo sin herramientas) también se ve
	// aquí: la bienvenida es donde primero se elige modelo.
	if a.Aviso != "" {
		b.WriteString("\n\n" + estiloAviso.Render(a.Aviso))
	}
	return centrar(b.String(), a.Ancho, a.Alto)
}

// avisoSinMotor es la línea de la bienvenida cuando no hay ningún motor dado de
// alta: lo dice y ofrece el modal para añadir uno. No bloquea nada ni inventa un
// motor: vacío cuando el arranque sí tiene uno (SPEC-MODELO-MOTOR §Ningún motor
// activo por defecto).
func (a *App) avisoSinMotor() string {
	if a.MotorEtiqueta != "" {
		return ""
	}
	return "no hay ningún motor dado de alta; añade uno con Ctrl+X i"
}

// lineaDeModelo pinta el modelo en uso y el recordatorio del atajo que lo
// cambia (SPEC-INTERFAZ §Línea de modelo: "se muestra solo el modelo en uso —el
// detectado en el arranque o el último elegido en el modal— y el recordatorio
// del atajo para cambiarlo"). Sin modelo detectado —Ollama no estaba al
// arrancar— se muestra el hueco con el mismo recordatorio: la pantalla se
// muestra igual y el modal es el camino para elegir uno.
func (a *App) lineaDeModelo() string {
	return estiloEtiqueta.Render("modelo: "+a.rotuloModelo()) + "  (Ctrl+X m cambiar)"
}
