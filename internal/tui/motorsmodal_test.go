package tui

// Tests de T-F044: el modal de motores (`Ctrl+X i`) y su recorrido por el
// `app`. Una prueba, una regla (frontend/05-quality/TESTING.md §3), contra
// SPEC-INTERFAZ §Modales, SPEC-MODELO-MOTOR §Gestión/§Cambiar a mitad,
// SPEC-KEYBINDS §Acción y §Resolución por contexto, DOMAIN §1 `motorsmodal`:
//
//	T-F044-04 → la lista distingue activo de desactivado y el editado
//	            aplicado avisa de que el cambio espera al reinicio
//	T-F044-05 → alta y edición validan nombre, tipo y URL; el borrado con
//	            sesiones en uso pide confirmación y el libre borra al momento
//	T-F044-06 → el modal se abre en las dos vistas y sus teclas no llegan a
//	            la vista de abajo (Ctrl+C sigue saliendo)
//	T-F044-08 → aplicar un motor que no declara el modelo en uso lo vacía
//	T-F044-11 → abrir y aplicar no toca el historial ni el nombre de la
//	            sesión; editar un motor aplicado avisa del reinicio

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"localcli/internal/session"
)

// motoresDePrueba es el registro que devuelve el doble: dos motores activos
// del mismo proyecto, uno por tipo.
func motoresDePrueba() []MotorLocal {
	return []MotorLocal{
		{ID: "m1", Nombre: "Ollama local", Tipo: "ollama", URL: "http://localhost:11434", Activo: true},
		{ID: "m2", Nombre: "llama.cpp local", Tipo: "llamacpp", URL: "http://localhost:8080", Activo: true},
	}
}

// recorre entrega los mensajes que produce un comando y los de los comandos
// que devuelve Update, encadenados como el bucle real de Bubble Tea (con un
// tope para que un latido eterno no gire para siempre).
func recorre(t *testing.T, a *App, cmd tea.Cmd) {
	t.Helper()
	for i := 0; cmd != nil && i < 8; i++ {
		msgs := mensajesDe(cmd)
		cmd = nil
		for _, msg := range msgs {
			_, siguiente := a.Update(msg)
			if siguiente != nil {
				cmd = siguiente
			}
		}
	}
}

// --- T-F044-04: activo, desactivado y aviso de reinicio ---------------------

func TestElModalDistingueActivoDeDesactivadoYAvisaDelReinicio(t *testing.T) {
	mm := &MotorsModal{}
	mm.AbrirMotores("m1")
	mm.FijarMotores([]MotorLocal{
		{ID: "m1", Nombre: "Ollama local", Tipo: "ollama", URL: "http://localhost:11434", Activo: true},
		{ID: "m2", Nombre: "llama.cpp local", Tipo: "llamacpp", URL: "http://localhost:8080", Activo: false},
		{ID: "m3", Nombre: "Ollama remoto", Tipo: "ollama", URL: "https://api.ejemplo.com", Activo: true, PendienteDeReinicio: true},
	}, nil)

	// Cada fila lleva nombre, tipo y estado; el de la sesión se marca con «←».
	lineas := strings.Split(sinEstilo(mm.Render(100, 30)), "\n")
	buscar := func(fragmento string) string {
		t.Helper()
		for _, l := range lineas {
			if strings.Contains(l, fragmento) {
				return l
			}
		}
		t.Fatalf("ninguna fila contiene %q:\n%s", fragmento, strings.Join(lineas, "\n"))
		return ""
	}
	if fila := buscar("Ollama local  [ollama · activo]"); !strings.Contains(fila, "←") {
		t.Errorf("el motor de la sesión se marca en su propia fila: %q", fila)
	}
	filaDesactivada := buscar("llama.cpp local  [llamacpp · desactivado]")
	if strings.Contains(filaDesactivada, "←") {
		t.Errorf("el desactivado no es el de la sesión: %q", filaDesactivada)
	}
	// Solo el editado estando aplicado avisa del reinicio, en SU fila.
	filaPendiente := buscar("(se aplica al reiniciar)")
	if !strings.Contains(filaPendiente, "Ollama remoto") {
		t.Errorf("el aviso de reinicio va en el motor editado: %q", filaPendiente)
	}
	if n := strings.Count(strings.Join(lineas, "\n"), "(se aplica al reiniciar)"); n != 1 {
		t.Errorf("el aviso de reinicio es solo del editado, aparece %d veces", n)
	}

	// El resaltado arranca en el motor de la sesión y ↓ lo mueve.
	if mm.MotorResaltadoID() != "m1" {
		t.Errorf("el foco arranca en el motor de la sesión: %q", mm.MotorResaltadoID())
	}
	mm.Mover(1)
	if mm.MotorResaltadoID() != "m2" {
		t.Errorf("↓ mueve el resaltado: %q", mm.MotorResaltadoID())
	}
}

// --- T-F044-05: alta y edición validadas ------------------------------------

func TestAltaYEdicionValidanNombreYURL(t *testing.T) {
	mm := &MotorsModal{}
	mm.AbrirMotores("m1")
	mm.FijarMotores(motoresDePrueba(), nil)

	// Un alta sin nombre no se registra y el error se ve en el formulario.
	mm.AbrirAlta()
	if _, ok := mm.GuardarFormulario(); ok || !mm.FormAbierto() {
		t.Fatal("el alta sin nombre se rechaza y el formulario sigue abierto")
	}
	if v := sinEstilo(mm.Render(80, 24)); !strings.Contains(v, "el nombre no puede quedar vacío") {
		t.Errorf("el error se ve en el formulario:\n%s", v)
	}
	// Un nombre ya registrado tampoco, aunque cambien las mayúsculas.
	mm.form.Nombre = "ollama LOCAL"
	if _, ok := mm.GuardarFormulario(); ok || !strings.Contains(mm.errorForm, "ese nombre") {
		t.Errorf("el nombre repetido se rechaza: %q", mm.errorForm)
	}
	// El tipo entra en el catálogo cerrado.
	mm.form.Nombre = "tercero"
	mm.form.Tipo = "vllm"
	if _, ok := mm.GuardarFormulario(); ok || !strings.Contains(mm.errorForm, "ollama o llamacpp") {
		t.Errorf("un tipo fuera del catálogo se rechaza: %q", mm.errorForm)
	}
	// La URL tiene que ser http/https con host.
	mm.form.Tipo = "ollama"
	mm.form.URL = "localhost:8080"
	if _, ok := mm.GuardarFormulario(); ok || !strings.Contains(mm.errorForm, "http") {
		t.Errorf("una URL sin esquema se rechaza: %q", mm.errorForm)
	}
	mm.form.URL = "ftp://ejemplo.com"
	if _, ok := mm.GuardarFormulario(); ok {
		t.Errorf("una URL que no es http/https se rechaza: %q", mm.errorForm)
	}
	// Correcta: se guarda con el id vacío, que lo genera el registro.
	mm.form.URL = "http://localhost:9000"
	motor, ok := mm.GuardarFormulario()
	if !ok {
		t.Fatalf("un alta válida se guarda: %q", mm.errorForm)
	}
	if motor.ID != "" || motor.Nombre != "tercero" || motor.Tipo != "ollama" || motor.URL != "http://localhost:9000" || !motor.Activo {
		t.Errorf("el motor guardado = %+v", motor)
	}
	if mm.FormAbierto() {
		t.Error("guardar cierra el formulario")
	}

	// Edición: el motor no choca con su propio nombre…
	mm.AbrirEdicion()
	if mm.form.Nombre != "Ollama local" || mm.form.URL != "http://localhost:11434" {
		t.Fatalf("el formulario se abre con el motor resaltado: %+v", mm.form)
	}
	if motor, ok := mm.GuardarFormulario(); !ok || motor.ID != "m1" {
		t.Errorf("editar sin cambiar el nombre propio es válido: %+v, %v", motor, ok)
	}
	// …pero sí con el de otro.
	mm.AbrirEdicion()
	mm.form.Nombre = "llama.cpp local"
	if _, ok := mm.GuardarFormulario(); ok || !strings.Contains(mm.errorForm, "ese nombre") {
		t.Errorf("chocar con otro nombre se rechaza: %q", mm.errorForm)
	}
}

// --- T-F045-04: la lista distingue sin extensiones de con extensiones --------

// Cada fila muestra las extensiones declaradas de su motor, rotula la lista
// vacía como «solo núcleo común» y trata un motor inactivo como cualquier otro
// (SPEC-MODELO-MOTOR §Núcleo común y extensiones nativas, §Desactivar).
func TestLaListaDistingueSinExtensionesDeConExtensiones(t *testing.T) {
	mm := &MotorsModal{}
	mm.AbrirMotores("m1")
	mm.FijarMotores([]MotorLocal{
		{ID: "m1", Nombre: "Ollama completo", Tipo: "ollama", URL: "http://localhost:11434", Activo: true, Extensiones: []string{"num_ctx", "show", "tags"}},
		{ID: "m2", Nombre: "Núcleo pelado", Tipo: "llamacpp", URL: "http://localhost:8080", Activo: false},
	}, nil)

	lineas := strings.Split(sinEstilo(mm.Render(120, 30)), "\n")
	buscar := func(fragmento string) string {
		t.Helper()
		for _, l := range lineas {
			if strings.Contains(l, fragmento) {
				return l
			}
		}
		t.Fatalf("ninguna fila contiene %q:\n%s", fragmento, strings.Join(lineas, "\n"))
		return ""
	}
	// Las declaradas se ven en SU fila.
	if fila := buscar("Ollama completo"); !strings.Contains(fila, "num_ctx, show, tags") {
		t.Errorf("la fila muestra las extensiones del motor: %q", fila)
	}
	// La vacía se distingue: «solo núcleo común», y solo en la que no declara.
	filaPelada := buscar("Núcleo pelado")
	if !strings.Contains(filaPelada, "(solo núcleo común)") {
		t.Errorf("la fila sin extensiones dice «solo núcleo común»: %q", filaPelada)
	}
	if n := strings.Count(strings.Join(lineas, "\n"), "(solo núcleo común)"); n != 1 {
		t.Errorf("solo el motor sin extensiones lo declara, aparece %d veces", n)
	}
	// Un motor inactivo se ve desactivado como cualquier otro: sin estado de
	// fábrica.
	if !strings.Contains(filaPelada, "[llamacpp · desactivado]") {
		t.Errorf("el motor inactivo se rotula desactivado: %q", filaPelada)
	}
}

// --- T-F045-03: el campo de extensiones -------------------------------------

// `extensiones` se valida contra el catálogo del tipo elegido —no contra una
// lista propia de la vista— y viaja en el motor guardado; sin extensiones es
// «solo núcleo común», no un error (SPEC-MODELO-MOTOR §Núcleo común y
// extensiones nativas, §El catálogo es cerrado).
func TestElAltaValidaLasExtensionesDelTipo(t *testing.T) {
	catalogo := func(tipo string) []string {
		switch tipo {
		case "ollama":
			return []string{"num_ctx", "show", "tags"}
		case "llamacpp":
			return []string{"props"}
		}
		return nil
	}
	mm := &MotorsModal{CatalogoExtensiones: catalogo}
	mm.AbrirMotores("m1")
	mm.FijarMotores(motoresDePrueba(), nil)

	// Alta con extensiones válidas: se guardan normalizadas (sin espacios ni
	// repetidas, conservando el orden).
	mm.AbrirAlta()
	mm.form.Nombre = "ollama remoto"
	mm.form.Tipo = "ollama"
	mm.form.URL = "http://localhost:9000"
	mm.form.Extensiones = " num_ctx , show , num_ctx "
	motor, ok := mm.GuardarFormulario()
	if !ok {
		t.Fatalf("una extensión válida se acepta: %q", mm.errorForm)
	}
	if got := strings.Join(motor.Extensiones, ","); got != "num_ctx,show" {
		t.Errorf("las extensiones se normalizan y deduplican: %q", got)
	}

	// Una extensión de OTRO tipo no vale: lo decide el catálogo del registro.
	mm.AbrirAlta()
	mm.form.Nombre = "ollama remoto"
	mm.form.Tipo = "ollama"
	mm.form.URL = "http://localhost:9000"
	mm.form.Extensiones = "props"
	if _, ok := mm.GuardarFormulario(); ok || !strings.Contains(mm.errorForm, "props") {
		t.Errorf("una extensión de otro tipo se rechaza: %q", mm.errorForm)
	}
	if v := sinEstilo(mm.Render(90, 30)); !strings.Contains(v, "sin extensión declarada") {
		t.Errorf("el formulario explica qué pasa sin declarar ninguna:\n%s", v)
	}

	// Sin extensiones declaradas es «solo núcleo común».
	mm.form.Extensiones = ""
	if motor, ok := mm.GuardarFormulario(); !ok || len(motor.Extensiones) != 0 {
		t.Errorf("sin extensiones = solo núcleo común: %+v, %v", motor, ok)
	}

	// La edición carga las declaradas del motor y valida contra SU tipo.
	mm.AbrirMotores("m2")
	mm.FijarMotores([]MotorLocal{
		{ID: "m2", Nombre: "llama.cpp local", Tipo: "llamacpp", URL: "http://localhost:8080", Activo: true, Extensiones: []string{"props"}},
	}, nil)
	mm.AbrirEdicion()
	if mm.form.Tipo != "llamacpp" || mm.form.Extensiones != "props" {
		t.Fatalf("la edición carga las extensiones del motor: %+v", mm.form)
	}
	mm.form.Extensiones = "num_ctx"
	if _, ok := mm.GuardarFormulario(); ok {
		t.Error("num_ctx no es una extensión de llamacpp")
	}
}

// El borrado con sesiones en uso pide confirmación; el libre se elimina al
// momento (SPEC-MODELO-MOTOR §Eliminar; DOMAIN §1: sin uso, sin confirmación).
func TestBorrarPideConfirmacionSoloSiAlgunaSesionUsaElMotor(t *testing.T) {
	mm := &MotorsModal{}
	mm.AbrirMotores("m1")
	mm.FijarMotores(motoresDePrueba(), []session.Sesion{
		{ID: "s1", Nombre: "primera", MotorID: "m1", Estado: session.EstadoInactiva},
	})

	// Resaltado el motor usado por una sesión: hay que confirmar.
	id, enUso := mm.PedirBorrado()
	if id != "m1" || !enUso || !mm.ConfirmandoBorrado() {
		t.Fatalf("m1 está en uso: id=%q enUso=%v confirmando=%v", id, enUso, mm.ConfirmandoBorrado())
	}
	if v := sinEstilo(mm.Render(80, 24)); !strings.Contains(v, "hay sesiones que usan «Ollama local»") {
		t.Errorf("la confirmación dice quién lo usa:\n%s", v)
	}
	mm.BorrarCancelar()
	if mm.ConfirmandoBorrado() {
		t.Error("esc/n descarta la confirmación")
	}

	// Resaltado uno sin uso: ni confirmación ni espera.
	mm.Mover(1)
	id, enUso = mm.PedirBorrado()
	if id != "m2" || enUso {
		t.Errorf("m2 no lo usa ninguna sesión: id=%q enUso=%v", id, enUso)
	}
	mm.BorrarCancelar()
}

// --- T-F044-06: se abre en las dos vistas ------------------------------------

func TestElModalDeMotoresAbreEnLasDosVistas(t *testing.T) {
	p := &puertoStub{motores: motoresDePrueba()}
	a := Nuevo(p)
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})

	// Bienvenida.
	recorre(t, a, abreElModalDeMotores(t, a))
	if !a.Motores.Abierto {
		t.Fatal("ctrl+x i abre el modal desde la bienvenida")
	}
	if v := sinEstilo(a.View()); !strings.Contains(v, "MOTORES") || !strings.Contains(v, "Ollama local") {
		t.Errorf("el registro se ve en la bienvenida:\n%s", v)
	}
	tecla(t, a, tea.KeyEsc)
	if a.Motores.Abierto {
		t.Error("esc cierra el modal")
	}

	// Interfaz principal: es global, se abre igual.
	a.Vista = VistaPrincipal
	recorre(t, a, abreElModalDeMotores(t, a))
	if !a.Motores.Abierto {
		t.Error("ctrl+x i abre el modal desde la interfaz principal")
	}
}

// --- T-F044-11: abrir y aplicar sin tocar el hilo ---------------------------

func TestAplicarUnMotorNoTocaElHistorialNiElNombreDeLaSesion(t *testing.T) {
	p := &puertoStub{
		motores: motoresDePrueba(),
		modelos: []ModeloLocal{{Nombre: "qwen3:8b"}, {Nombre: "llama3.2"}},
		modelo:  "qwen3:8b",
		sesiones: []session.Sesion{
			{ID: "s1", Nombre: "primera", MotorID: "m1", Estado: session.EstadoInactiva},
		},
		pares: map[string][2]string{"s1": {"m1", "qwen3:8b"}},
	}
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	a.Panel.Sesion = "primera"
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	// El hilo de la sesión ya está cargado.
	recorre(t, a, pulsa(t, a, historialMsg{
		Sesion: "s1",
		Mensajes: []MensajeHistorial{
			{Rol: "user", Texto: "hola"},
			{Rol: "agent", Texto: "qué tal"},
		},
	}))
	if len(a.Chat.Mensajes()) != 2 {
		t.Fatalf("el hilo se cargó: %d mensajes", len(a.Chat.Mensajes()))
	}

	// El modal marca el motor de la sesión y lo aplica con Enter.
	recorre(t, a, abreElModalDeMotores(t, a))
	if a.Motores.MotorResaltadoID() != "m1" {
		t.Fatalf("el foco arranca en el motor de la sesión: %q", a.Motores.MotorResaltadoID())
	}
	tecla(t, a, tea.KeyDown) // m2
	recorre(t, a, tecla(t, a, tea.KeyEnter))

	if a.Motores.Abierto {
		t.Error("enter aplica y cierra el modal")
	}
	if len(p.motorCambios) != 1 || p.motorCambios[0] != "s1=m2" {
		t.Fatalf("el puerto recibe el motor elegido: %v", p.motorCambios)
	}
	// La identidad de la sesión no cambia…
	if a.Panel.SesionID != "s1" || a.Panel.Sesion != "primera" {
		t.Errorf("la sesión sigue siendo la misma: %q / %q", a.Panel.SesionID, a.Panel.Sesion)
	}
	// …ni su historial…
	if len(a.Chat.Mensajes()) != 2 || a.Chat.Mensajes()[0].Texto != "hola" {
		t.Errorf("el hilo queda intacto: %+v", a.Chat.Mensajes())
	}
	// …y la línea de estado pasa a rotular el motor nuevo.
	if a.MotorEtiqueta != "llama.cpp local" {
		t.Errorf("la etiqueta del motor es la del motor aplicado: %q", a.MotorEtiqueta)
	}
	if p.modeloCambios != nil {
		t.Errorf("aplicar un motor no arrastra el modelo cuando sigue existiendo: %v", p.modeloCambios)
	}
	if a.Modelo != "qwen3:8b" {
		t.Errorf("el modelo que el motor nuevo declara se conserva: %q", a.Modelo)
	}
}

// --- T-F044-08: el modelo que el motor nuevo no declara se vacía ------------

func TestCambiarDeMotorVaciaElModeloSiNoExisteAlli(t *testing.T) {
	// Función pura: el nombre se conserva solo si el motor nuevo lo declara.
	if got := ModeloTrasCambiarDeMotor("qwen3:8b", []ModeloLocal{{Nombre: "llama3.2"}}); got != "" {
		t.Errorf("un modelo ausente en el motor nuevo se vacía: %q", got)
	}
	if got := ModeloTrasCambiarDeMotor("qwen3:8b", []ModeloLocal{{Nombre: "qwen3:8b"}}); got != "qwen3:8b" {
		t.Errorf("un modelo presente se conserva: %q", got)
	}
	if got := ModeloTrasCambiarDeMotor("", []ModeloLocal{{Nombre: "llama3.2"}}); got != "" {
		t.Errorf("sin modelo no hay nada que arrastrar: %q", got)
	}

	// A través del app: al aplicar el motor, el modelo en uso se vacía si el
	// motor nuevo no lo declara (SPEC-MODELO-MOTOR §Cambiar a mitad).
	p := &puertoStub{modelo: "qwen3:8b"}
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	recorre(t, a, pulsa(t, a, motorAplicadoMsg{
		Sesion:  "s1",
		MotorID: "m2",
		Modelos: []ModeloLocal{{Nombre: "llama3.2"}},
	}))
	if a.Modelo != "" {
		t.Errorf("el modelo que el motor nuevo no declara no se arrastra: %q", a.Modelo)
	}
	// El aviso de otra sesión no toca la activa.
	recorre(t, a, pulsa(t, a, motorAplicadoMsg{
		Sesion:  "otra",
		MotorID: "m1",
		Modelos: nil,
	}))
	if a.Modelo != "" {
		t.Errorf("un motor de otra sesión no cambia nada aquí: %q", a.Modelo)
	}
}

// --- T-F044-11: editar un motor aplicado avisa del reinicio -----------------

func TestEditarUnMotorAplicadoAvisaQueSeAplicaAlReiniciar(t *testing.T) {
	p := &puertoStub{
		motores:  motoresDePrueba(),
		sesiones: []session.Sesion{{ID: "s1", Nombre: "primera", MotorID: "m1", Estado: session.EstadoInactiva}},
		pares:    map[string][2]string{"s1": {"m1", "qwen3:8b"}},
	}
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	a.Panel.Sesion = "primera"
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	recorre(t, a, a.cmdMotorDeSesion("s1"))

	recorre(t, a, abreElModalDeMotores(t, a))
	if a.Motores.MotorResaltadoID() != "m1" {
		t.Fatalf("el foco arranca en el motor de la sesión: %q", a.Motores.MotorResaltadoID())
	}
	// `<leader>e` abre el formulario con el motor y se le añade sufijo al nombre.
	secuencia(t, a, tea.KeyCtrlX, "e")
	if !a.Motores.FormAbierto() {
		t.Fatal("<leader>e edita el motor resaltado")
	}
	escribe(t, a, " v2")
	recorre(t, a, tecla(t, a, tea.KeyEnter))

	if len(p.edicionesMotor) != 1 || p.edicionesMotor[0] != "m1" {
		t.Fatalf("la edición llega al registro: %v", p.edicionesMotor)
	}
	v := sinEstilo(a.View())
	if !strings.Contains(v, "Ollama local v2") {
		t.Errorf("la fila recargada muestra el nombre nuevo:\n%s", v)
	}
	if !strings.Contains(v, "(se aplica al reiniciar)") {
		t.Errorf("editar un motor aplicado avisa de que el cambio espera al reinicio:\n%s", v)
	}
}

// --- T-F044-06/11: las teclas del modal no filtran a la vista de abajo ------

func TestLasTeclasDelModalDeMotoresNoFiltrANaLaVistaDeAbajo(t *testing.T) {
	p := &puertoStub{motores: motoresDePrueba()}
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	escribe(t, a, "media")
	recorre(t, a, abreElModalDeMotores(t, a))
	if !a.Motores.Abierto {
		t.Fatal("ctrl+x i abre el modal")
	}

	// Escribir con el modal abierto no compone texto: las teclas son suyas.
	escribe(t, a, " más")
	if a.Entrada.Texto() != "media" {
		t.Errorf("con el modal abierto la línea de entrada no escribe: %q", a.Entrada.Texto())
	}
	// Enter aplica lo resaltado, no envía la petición.
	tecla(t, a, tea.KeyEnter)
	if len(p.enviados) != 0 {
		t.Errorf("con el modal abierto enter no envía: %v", p.enviados)
	}
	if a.Vista != VistaPrincipal {
		t.Error("con el modal abierto no se cambia de vista")
	}
	// `<leader>a` es tecla del modal (abre el alta), no una letra de la entrada.
	recorre(t, a, abreElModalDeMotores(t, a))
	secuencia(t, a, tea.KeyCtrlX, "a")
	if !a.Motores.FormAbierto() {
		t.Error("<leader>a abre el alta dentro del modal")
	}
	if a.Entrada.Texto() != "media" {
		t.Errorf("la tecla del modal no llega a la entrada: %q", a.Entrada.Texto())
	}
	// Esc cancela el formulario y, con la lista, cierra el modal.
	tecla(t, a, tea.KeyEsc)
	if a.Motores.FormAbierto() {
		t.Error("esc cancela el formulario")
	}
	tecla(t, a, tea.KeyEsc)
	if a.Motores.Abierto {
		t.Error("esc cierra el modal")
	}
	// Cerrado, la escritura vuelve a funcionar.
	escribe(t, a, " más")
	if a.Entrada.Texto() != "media más" {
		t.Errorf("al cerrar se escribe otra vez: %q", a.Entrada.Texto())
	}

	// Y Ctrl+C sigue saliendo con el modal abierto.
	recorre(t, a, abreElModalDeMotores(t, a))
	cmd := tecla(t, a, tea.KeyCtrlC)
	if cmd == nil {
		t.Fatal("ctrl+c sigue saliendo con el modal abierto")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Errorf("ctrl+c produce la salida: %T", cmd())
	}
}
