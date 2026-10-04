// motorsmodal.go — T-F044: el modal de motores (`Ctrl+X i`).
//
// Fuente de verdad: SPEC-INTERFAZ §Modales (los cuatro modales comparten la
// misma mecánica: uno abierto a la vez, `↑`/`↓` navegan, `Enter` aplica y
// cierra, `Esc` cierra sin cambios) y §Pantalla de bienvenida, DOMAIN §1
// (`motorsmodal`: nombre, tipo y estado; `Ctrl+X a` añade, `Ctrl+X e` edita y
// `Ctrl+X d` elimina, con confirmación si alguna sesión lo usa; avisa de que
// editar uno ya aplicado se aplica al reiniciar) y SPEC-MODELO-MOTOR §Gestión
// de motores.
//
// El modal no decide nada ni guarda el registro (DOMAIN §4): pinta la lista que
// le llega y devuelve la elección al `app`. El registro y su persistencia son de
// `llm`. Aquí solo hay lista, resaltado, un formulario de alta/edición y el
// descarte; la mecánica compartida vive en `Modal` (modelsmodal.go).
package tui

import (
	"fmt"
	"net/url"
	"strings"

	"localcli/internal/session"
)

// MotorsModal es el modal de motores: el registro declarado, el motor de la
// sesión activa y el formulario de alta/edición.
type MotorsModal struct {
	Modal
	// Motores es la lista real, en el mismo orden que Modal.Lineas.
	Motores []MotorLocal
	// Actual es el `id` del motor de la sesión activa: se marca en su fila.
	Actual string
	// Usos dice qué motores usa alguna sesión del proyecto: es lo que hace
	// pedir confirmación antes de borrarlos (SPEC-MODELO-MOTOR §Eliminar).
	Usos map[string]bool
	// CatalogoExtensiones devuelve, por tipo, las extensiones nativas que el
	// registro admite. Lo inyecta el `app` desde el puerto: el modal no lleva
	// copiada la lista, porque qué extensión puede declarar cada tipo lo decide
	// el registro, no la vista (SPEC-MODELO-MOTOR §El catálogo es cerrado). Sin
	// él no hay contra qué validar y solo se comprueba la forma del campo.
	CatalogoExtensiones func(tipo string) []string

	// editando abre el formulario de alta (`editandoIndice == -1`) o de edición
	// del resaltado. Mientras está abierto, sus campos reciben el teclado.
	editando       bool
	editandoIndice int
	editandoID     string
	form           formularioMotor
	errorForm      string
	// activoOriginal guarda el estado con el que abrió la edición, para saber si
	// el usuario lo cambió: solo entonces el `app` lo aplica al momento.
	activoOriginal bool

	// pidiendoBorrar guarda el `id` del motor resaltado pendiente de confirmar.
	pidiendoBorrar string
}

// AvisoSinMotores es lo que se ve cuando el registro no tiene motores o la
// lectura falló. Se muestra y se cierra con Esc, como «sin modelos».
const AvisoSinMotores = "sin motores"

// PieMotores recuerda las teclas propias del modal además de navegar y aplicar
// (SPEC-KEYBINDS §Acción: motor_new, motor_edit y motor_delete). Las tres son
// secuencias con líder: se muestran con la líder expandida.
const PieMotores = "(↑/↓ mover · enter aplicar · ctrl+x a añadir · ctrl+x e editar · ctrl+x d eliminar · esc cerrar)"

// Título del modal, fijo para la lista y el formulario.
const tituloMotores = "MOTORES"

// formularioMotor es el estado del alta/edición: campos de texto —nombre, tipo,
// URL y extensiones declaradas— y el interruptor de estado. `Campo` dice cuál
// tiene el foco. `Extensiones` es texto separado por comas; se normaliza al
// guardar.
type formularioMotor struct {
	Nombre      string
	Tipo        string
	URL         string
	Extensiones string
	Activo      bool
	Campo       int
}

// Campos del formulario, en orden.
const (
	campoNombre = iota
	campoTipo
	campoURL
	campoExtensiones
	campoActivo
	numCamposMotor
)

// AbrirMotores muestra el modal vacío y pidiendo la lista. Cada apertura es una
// lectura nueva del registro: lo que se ve es lo que hay ahora, no una lista
// guardada.
func (mm *MotorsModal) AbrirMotores(actual string) {
	mm.Motores = nil
	mm.Actual = actual
	mm.Usos = nil
	mm.editando = false
	mm.pidiendoBorrar = ""
	mm.Modal.Pie = PieMotores
	mm.Modal.Abrir(tituloMotores, nil, AvisoCargando)
}

// FijarMotores rellena el modal con el registro y las sesiones que llegaron. El
// error del puerto no bloquea: queda como aviso «sin motores». Si el modal ya
// no está abierto, la lista se descarta (llegó tarde).
func (mm *MotorsModal) FijarMotores(motores []MotorLocal, sesiones []session.Sesion) {
	if !mm.Abierto {
		return
	}
	if len(motores) == 0 {
		mm.Motores = nil
		mm.Modal.Abrir(tituloMotores, nil, AvisoSinMotores)
		return
	}
	mm.Motores = motores
	mm.Usos = map[string]bool{}
	for _, s := range sesiones {
		if s.MotorID != "" {
			mm.Usos[s.MotorID] = true
		}
	}
	mm.repintar(mm.IndiceDe(mm.Actual))
}

// repintar reconstruye las filas y deja el resaltado donde se le dice. Con la
// lista vacía pinta el aviso y no hay nada que resaltar.
func (mm *MotorsModal) repintar(indice int) {
	if !mm.Abierto {
		return
	}
	if len(mm.Motores) == 0 {
		mm.Modal.Abrir(tituloMotores, nil, AvisoSinMotores)
		return
	}
	lineas := make([]string, 0, len(mm.Motores))
	for _, m := range mm.Motores {
		lineas = append(lineas, lineaDeMotor(m, m.ID == mm.Actual))
	}
	mm.Modal.Abrir(tituloMotores, lineas, "")
	mm.Indice = indice
}

// IndiceDe devuelve la posición de un motor en la lista, o 0 si no está: al
// abrir, el resaltado arranca en el motor de la sesión activa.
func (mm *MotorsModal) IndiceDe(id string) int {
	for i, m := range mm.Motores {
		if m.ID == id {
			return i
		}
	}
	return 0
}

// MotorElegido devuelve el motor resaltado, listo para aplicar. Sin lista —o con
// «sin motores»— no hay nada que elegir.
func (mm *MotorsModal) MotorElegido() (MotorLocal, bool) {
	i, ok := mm.Elegida()
	if !ok {
		return MotorLocal{}, false
	}
	return mm.Motores[i], true
}

// MotorResaltadoID devuelve el `id` resaltado, o "" si no hay lista.
func (mm *MotorsModal) MotorResaltadoID() string {
	m, ok := mm.MotorElegido()
	if !ok {
		return ""
	}
	return m.ID
}

// lineaDeMotor compone la fila: nombre, tipo, estado y las extensiones
// declaradas. Los desactivados se distinguen de los activos, un motor editado
// estando aplicado avisa de que su cambio se aplica al reiniciar, y la lista
// vacía se rotula «solo núcleo común» para distinguirla de una declarada
// (SPEC-MODELO-MOTOR §Desactivar, §Editar, §Núcleo común y extensiones
// nativas). El de la sesión activa se marca con «←».
func lineaDeMotor(m MotorLocal, actual bool) string {
	estado := "activo"
	if !m.Activo {
		estado = "desactivado"
	}
	linea := m.Nombre + "  [" + m.Tipo + " · " + estado + "]"
	if len(m.Extensiones) == 0 {
		linea += "  (solo núcleo común)"
	} else {
		linea += "  (" + strings.Join(m.Extensiones, ", ") + ")"
	}
	if m.PendienteDeReinicio {
		linea += "  (se aplica al reiniciar)"
	}
	if actual {
		linea += " ←"
	}
	return linea
}

// --- alta y edición ----------------------------------------------------------

// FormAbierto dice si el modal está en el formulario de alta/edición.
func (mm *MotorsModal) FormAbierto() bool { return mm.editando }

// AbrirAlta abre el formulario de un motor nuevo.
func (mm *MotorsModal) AbrirAlta() {
	mm.editando = true
	mm.editandoIndice = -1
	mm.editandoID = ""
	mm.form = formularioMotor{Tipo: "ollama", Activo: true}
	mm.activoOriginal = true
	mm.errorForm = ""
}

// AbrirEdicion abre el formulario con el motor resaltado. Sin nada resaltado no
// hace nada.
func (mm *MotorsModal) AbrirEdicion() {
	m, ok := mm.MotorElegido()
	if !ok {
		return
	}
	mm.editando = true
	mm.editandoIndice = mm.Indice
	mm.editandoID = m.ID
	mm.form = formularioMotor{
		Nombre:      m.Nombre,
		Tipo:        m.Tipo,
		URL:         m.URL,
		Extensiones: strings.Join(m.Extensiones, ", "),
		Activo:      m.Activo,
	}
	mm.activoOriginal = m.Activo
	mm.errorForm = ""
}

// ActivoCambio dice si el interruptor del formulario difiere del estado con el
// que se abrió la edición. En un alta devuelve falso: el estado entra con el
// registro.
func (mm *MotorsModal) ActivoCambio() bool {
	return mm.editandoID != "" && mm.form.Activo != mm.activoOriginal
}

// CancelarFormulario cierra el formulario sin guardar.
func (mm *MotorsModal) CancelarFormulario() {
	mm.editando = false
	mm.errorForm = ""
}

// FormEscribir inserta texto en el campo con el foco. En el interruptor de
// estado, el espacio lo alterna.
func (mm *MotorsModal) FormEscribir(texto string) {
	switch mm.form.Campo {
	case campoNombre:
		mm.form.Nombre += texto
	case campoTipo:
		mm.form.Tipo += texto
	case campoURL:
		mm.form.URL += texto
	case campoExtensiones:
		mm.form.Extensiones += texto
	case campoActivo:
		mm.form.Activo = !mm.form.Activo
	}
}

// FormBorrar quita el último carácter del campo con el foco.
func (mm *MotorsModal) FormBorrar() {
	switch mm.form.Campo {
	case campoNombre:
		mm.form.Nombre = sinUltimaRuna(mm.form.Nombre)
	case campoTipo:
		mm.form.Tipo = sinUltimaRuna(mm.form.Tipo)
	case campoURL:
		mm.form.URL = sinUltimaRuna(mm.form.URL)
	case campoExtensiones:
		mm.form.Extensiones = sinUltimaRuna(mm.form.Extensiones)
	}
}

// FormMoverCampo cambia el campo con el foco, en ciclo.
func (mm *MotorsModal) FormMoverCampo(delta int) {
	mm.form.Campo = (mm.form.Campo + delta + numCamposMotor) % numCamposMotor
}

// sinUltimaRuna quita el último carácter de un texto (en runas).
func sinUltimaRuna(s string) string {
	r := []rune(s)
	if len(r) == 0 {
		return s
	}
	return string(r[:len(r)-1])
}

// GuardarFormulario valida y devuelve el motor listo para persistir. El `id`
// vacío en un alta lo genera el registro. Un error de validación queda en
// `errorForm` y bloquea el guardado: el nombre no puede repetirse y la URL tiene
// que ser http/https con host (SPEC-MODELO-MOTOR §Agregar).
func (mm *MotorsModal) GuardarFormulario() (MotorLocal, bool) {
	nombre := strings.TrimSpace(mm.form.Nombre)
	tipo := strings.ToLower(strings.TrimSpace(mm.form.Tipo))
	dir := strings.TrimSpace(mm.form.URL)

	if nombre == "" {
		mm.errorForm = "el nombre no puede quedar vacío"
		return MotorLocal{}, false
	}
	for _, m := range mm.Motores {
		if m.ID != mm.editandoID && strings.EqualFold(m.Nombre, nombre) {
			mm.errorForm = "ya hay un motor con ese nombre"
			return MotorLocal{}, false
		}
	}
	if tipo != "ollama" && tipo != "llamacpp" {
		mm.errorForm = "el tipo tiene que ser ollama o llamacpp"
		return MotorLocal{}, false
	}
	if !urlMotorValida(dir) {
		mm.errorForm = "la URL tiene que ser http:// o https:// con host"
		return MotorLocal{}, false
	}
	extensiones, errExt := mm.extensionesValidas(tipo, mm.form.Extensiones)
	if errExt != "" {
		mm.errorForm = errExt
		return MotorLocal{}, false
	}

	motor := MotorLocal{
		ID:          mm.editandoID,
		Nombre:      nombre,
		Tipo:        tipo,
		URL:         dir,
		Extensiones: extensiones,
		Activo:      mm.form.Activo,
	}
	mm.editando = false
	mm.errorForm = ""
	return motor, true
}

// separarExtensiones normaliza el campo de texto: comas, espacios sobrantes y
// sin repetidas, conservando el orden en que se escribieron. Vacío = solo núcleo
// común.
func separarExtensiones(texto string) []string {
	var out []string
	vistas := map[string]bool{}
	for _, parte := range strings.Split(texto, ",") {
		ext := strings.TrimSpace(parte)
		if ext == "" || vistas[ext] {
			continue
		}
		vistas[ext] = true
		out = append(out, ext)
	}
	return out
}

// extensionesValidas comprueba las extensiones declaradas contra el catálogo
// del tipo elegido y devuelve las normalizadas o el error de validación. Sin
// catálogo inyectado —un modal construido suelto— solo se normaliza: la lista
// de válidas la decide el registro, no la vista (SPEC-MODELO-MOTOR §El catálogo
// es cerrado).
func (mm *MotorsModal) extensionesValidas(tipo, texto string) ([]string, string) {
	extensiones := separarExtensiones(texto)
	if mm.CatalogoExtensiones == nil {
		return extensiones, ""
	}
	validas := mm.CatalogoExtensiones(tipo)
	for _, ext := range extensiones {
		if !contieneCadena(validas, ext) {
			return nil, "la extensión «" + ext + "» no la admite el tipo " + tipo
		}
	}
	return extensiones, ""
}

// contieneCadena dice si una lista contiene un valor, comparando tal cual.
func contieneCadena(lista []string, valor string) bool {
	for _, v := range lista {
		if v == valor {
			return true
		}
	}
	return false
}

// urlMotorValida exige esquema http/https y host: una dirección sin host no
// sirve para conectar con un motor.
func urlMotorValida(dir string) bool {
	if dir == "" {
		return false
	}
	u, err := url.Parse(dir)
	if err != nil {
		return false
	}
	return (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

// --- borrado -----------------------------------------------------------------

// ConfirmandoBorrado dice si hay un borrado pendiente de confirmar.
func (mm *MotorsModal) ConfirmandoBorrado() bool { return mm.pidiendoBorrar != "" }

// PedirBorrado deja pendiente el borrado del resaltado. Devuelve si alguna
// sesión usa ese motor: solo entonces hace falta confirmar (el `app` lo decide).
func (mm *MotorsModal) PedirBorrado() (string, bool) {
	id := mm.MotorResaltadoID()
	if id == "" {
		return "", false
	}
	mm.pidiendoBorrar = id
	return id, mm.Usos[id]
}

// BorrarCancelar descarta el borrado pendiente.
func (mm *MotorsModal) BorrarCancelar() { mm.pidiendoBorrar = "" }

// BorrandoID devuelve el `id` pendiente de borrar.
func (mm *MotorsModal) BorrandoID() string { return mm.pidiendoBorrar }

// NombreDe busca el nombre visible de un motor del registro.
func (mm *MotorsModal) NombreDe(id string) string {
	for _, m := range mm.Motores {
		if m.ID == id {
			return m.Nombre
		}
	}
	return id
}

// --- pintado -----------------------------------------------------------------

// Render pinta el modal: el formulario si está abierto, la confirmación del
// borrado o la lista. Los tres comparten el centrado y el pie.
func (mm *MotorsModal) Render(ancho, alto int) string {
	if !mm.Abierto {
		return ""
	}
	if mm.editando {
		return mm.renderFormulario(ancho, alto)
	}
	if mm.pidiendoBorrar != "" {
		return mm.renderConfirmar(ancho, alto)
	}
	return mm.Modal.Render(ancho, alto)
}

// renderFormulario pinta el alta/edición con sus tres campos y el interruptor de
// estado, resaltando el campo con el foco.
func (mm *MotorsModal) renderFormulario(ancho, alto int) string {
	tituloCampo := []string{"nombre", "tipo", "url", "extensiones", "activo"}
	valores := []string{mm.form.Nombre, mm.form.Tipo, mm.form.URL, mm.form.Extensiones, estadoActivo(mm.form.Activo)}

	var b strings.Builder
	b.WriteString(estiloTitulo.Render("MOTOR") + "\n\n")
	for i := range valores {
		etiqueta := fmt.Sprintf("%-11s", tituloCampo[i])
		valor := valores[i]
		if i == mm.form.Campo {
			b.WriteString(estiloUsuario.Render("› "+etiqueta+" "+valor) + "\n")
		} else {
			b.WriteString(estiloBlanco.Render("  "+etiqueta+" "+valor) + "\n")
		}
	}
	// El campo se explica por sí solo: qué extensiones admite el tipo elegido y
	// qué pasa si no se declara ninguna (SPEC-MODELO-MOTOR §Núcleo común y
	// extensiones nativas).
	tipo := strings.ToLower(strings.TrimSpace(mm.form.Tipo))
	if validas := mm.extensionesDelTipo(tipo); len(validas) > 0 {
		b.WriteString("\n" + estiloSistema.Render("(extensiones válidas para "+tipo+": "+strings.Join(validas, ", ")+")"))
	}
	b.WriteString("\n" + estiloSistema.Render("(sin extensión declarada, lo que no se informa queda desconocido)"))
	if mm.errorForm != "" {
		b.WriteString("\n\n" + estiloAviso.Render(mm.errorForm))
	}
	b.WriteString("\n\n" + estiloSistema.Render("(tab/↑↓ campo · enter guardar · esc cancelar)"))
	return centrar(b.String(), ancho, alto)
}

// extensionesDelTipo devuelve las válidas para el tipo si hay catálogo; sin él,
// ninguna (el modal suelto no lleva la lista).
func (mm *MotorsModal) extensionesDelTipo(tipo string) []string {
	if mm.CatalogoExtensiones == nil {
		return nil
	}
	return mm.CatalogoExtensiones(tipo)
}

// renderConfirmar pinta la confirmación del borrado de un motor en uso.
func (mm *MotorsModal) renderConfirmar(ancho, alto int) string {
	texto := estiloTitulo.Render("ELIMINAR MOTOR") + "\n\n" +
		estiloAviso.Render("hay sesiones que usan «"+mm.NombreDe(mm.pidiendoBorrar)+"».") + "\n\n" +
		"¿seguro que deseas eliminarlo? (s/n)"
	return centrar(texto, ancho, alto)
}

// estadoActivo da la etiqueta legible del interruptor del formulario.
func estadoActivo(activo bool) string {
	if activo {
		return "sí"
	}
	return "no"
}
