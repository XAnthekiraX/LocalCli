package tui

// Tests de T-F013: el modal de modelos y la línea de modelo de la bienvenida.
// Una prueba, una regla (frontend/05-quality/TESTING.md §3), contra
// SPEC-INTERFAZ §Modal de modelos y §Línea de modelo, DOMAIN §3 y
// SPEC-KEYBINDS §Resolución por contexto:
//
//	T-F013-01 → la bienvenida no lista modelos: solo la línea del modelo en uso
//	T-F013-02 → el modal: cargando, navegar, aplicar, descartar, sin modelos
//	T-F013-03 → la lista se pide al abrir, no al arrancar, y no bloquea
//	T-F013-04 → se abre desde las dos vistas y captura las teclas
//	T-F013-05 → el modelo elegido viaja con la primera petición

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// errSinOllama es el fallo del doble cuando Ollama no responde: el modal lo
// muestra como aviso y no bloquea nada.
var errSinOllama = errors.New("no hay conexión con Ollama")

// modelosDePrueba es la lista que devuelve el doble: dos modelos locales.
func modelosDePrueba() []ModeloLocal {
	return []ModeloLocal{{Nombre: "llama3.2"}, {Nombre: "qwen2.5"}}
}

// --- T-F013-02: el modal como componente ------------------------------------

func TestElModalDeModelosAbrePidiendoLaLista(t *testing.T) {
	mm := &ModelsModal{}
	mm.AbrirModelos("")
	if !mm.Abierto {
		t.Fatal("el modal se abre")
	}
	if mm.Aviso != AvisoCargando {
		t.Errorf("mientras Ollama no responde se ve «cargando…»: %q", mm.Aviso)
	}
	if len(mm.Lineas) != 0 {
		t.Errorf("la lista todavía no ha llegado: %v", mm.Lineas)
	}
}

func TestElModalNavegaResaltandoCadaModelo(t *testing.T) {
	mm := &ModelsModal{}
	mm.AbrirModelos("")
	mm.FijarModelos(modelosDePrueba(), nil)

	if mm.ModeloElegido() != "llama3.2" {
		t.Errorf("arranca resaltando el primero: %q", mm.ModeloElegido())
	}
	mm.Mover(1)
	if mm.ModeloElegido() != "qwen2.5" {
		t.Errorf("↓ mueve el resaltado: %q", mm.ModeloElegido())
	}
	mm.Mover(-1)
	if mm.ModeloElegido() != "llama3.2" {
		t.Errorf("↑ vuelve al anterior: %q", mm.ModeloElegido())
	}
	mm.Mover(-1)
	if mm.ModeloElegido() != "qwen2.5" {
		t.Errorf("el recorrido es circular: %q", mm.ModeloElegido())
	}
}

func TestElModalDeModelosEnfocaElModeloEnUso(t *testing.T) {
	mm := &ModelsModal{}
	mm.AbrirModelos("qwen2.5")
	mm.FijarModelos(modelosDePrueba(), nil)
	if mm.ModeloElegido() != "qwen2.5" {
		t.Errorf("al abrir se resalta el modelo en uso: %q", mm.ModeloElegido())
	}

	// Si el modelo en uso ya no está en la lista, el resaltado cae en el primero.
	mm2 := &ModelsModal{}
	mm2.AbrirModelos("desinstalado")
	mm2.FijarModelos(modelosDePrueba(), nil)
	if mm2.ModeloElegido() != "llama3.2" {
		t.Errorf("sin coincidencia se resalta el primero: %q", mm2.ModeloElegido())
	}
}

func TestElModalMarcaLosModelosSinHerramientas(t *testing.T) {
	mm := &ModelsModal{}
	mm.AbrirModelos("")
	mm.FijarModelos([]ModeloLocal{
		{Nombre: "llama3.2"},
		{Nombre: "sin-tools", SinHerramientas: true},
	}, nil)
	v := sinEstilo(mm.Render(80, 24))
	if !strings.Contains(v, "sin-tools  (sin herramientas)") {
		t.Errorf("el modelo sin herramientas se marca:\n%s", v)
	}
	if strings.Contains(v, "llama3.2  (sin herramientas)") {
		t.Errorf("el que sí puede no se marca:\n%s", v)
	}
}

func TestSinOllamaElModalAvisaYSeCierraSinBloquear(t *testing.T) {
	mm := &ModelsModal{}
	mm.AbrirModelos("")
	mm.FijarModelos(nil, errSinOllama)
	if mm.Aviso != AvisoSinModelos {
		t.Errorf("sin Ollama el modal avisa: %q", mm.Aviso)
	}
	if v := mm.Render(80, 24); !strings.Contains(v, AvisoSinModelos) {
		t.Errorf("el aviso se ve en el modal:\n%s", v)
	}
	// Se puede cerrar igual: el error del puerto no bloquea nada.
	mm.Mover(1)
	if mm.ModeloElegido() != "" {
		t.Errorf("sin lista no hay nada que elegir: %q", mm.ModeloElegido())
	}
	mm.Cerrar()
	if mm.Abierto {
		t.Error("el modal se cierra con el aviso a la vista")
	}
}

func TestLaRespuestaTardíaNoRellenaUnModalYaCerrado(t *testing.T) {
	mm := &ModelsModal{}
	mm.AbrirModelos("")
	mm.Cerrar()
	mm.FijarModelos(modelosDePrueba(), nil)
	if mm.Abierto || len(mm.Modelos) != 0 {
		t.Errorf("una lista que llega tarde se descarta: %v", mm.Modelos)
	}
}

// --- T-F013-01: la línea de modelo de la bienvenida -------------------------

func TestLaBienvenidaMuestraElModeloEnUsoYElAtajoParaCambiarlo(t *testing.T) {
	a := Nuevo(&puertoStub{modelo: "llama3.2"})
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})

	v := sinEstilo(a.View())
	if !strings.Contains(v, "modelo: llama3.2") {
		t.Errorf("la línea de modelo enseña el modelo en uso:\n%s", v)
	}
	if !strings.Contains(v, "Ctrl+X m cambiar") {
		t.Errorf("la línea recuerda el atajo que la cambia:\n%s", v)
	}
}

func TestLaBienvenidaNoListaLosModelos(t *testing.T) {
	a := Nuevo(&puertoStub{modelo: "llama3.2"})
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	// Aunque Ollama responda, los nombres no se listan en la pantalla: solo
	// están en el modal (SPEC-INTERFAZ §Pantalla de bienvenida).
	pulsa(t, a, modelosMsg{Modelos: modelosDePrueba()})

	v := sinEstilo(a.View())
	for _, nombre := range []string{"qwen2.5", "Modelos:", "▸"} {
		if strings.Contains(v, nombre) {
			t.Errorf("la bienvenida no lista modelos (%q):\n%s", nombre, v)
		}
	}
	if !strings.Contains(v, "modelo: llama3.2") {
		t.Errorf("y sigue enseñando el modelo en uso:\n%s", v)
	}
}

func TestEnLaBienvenidaLasFlechasNoEligenModelo(t *testing.T) {
	a := Nuevo(&puertoStub{modelo: "llama3.2"})
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	pulsa(t, a, modelosMsg{Modelos: modelosDePrueba()})

	// Sin modal abierto, ↑/↓ no mueven nada: la elección vive en el modal
	// (SPEC-INTERFAZ §Modal de modelos).
	tecla(t, a, tea.KeyDown)
	tecla(t, a, tea.KeyUp)
	if a.Modelo != "llama3.2" || a.Modelos.Abierto {
		t.Errorf("las flechas de la pantalla no eligen modelo: %q", a.Modelo)
	}
}

// --- T-F013-03: la carga es bajo demanda -------------------------------------

func TestArrancarNoPideLaListaDeModelos(t *testing.T) {
	p := &puertoStub{}
	a := Nuevo(p)
	// La lista se pide al abrir el modal, no en el arranque (SPEC-INTERFAZ §Modal
	// de modelos): arrancar no toca Ollama.
	_ = a.Init()
	if p.peticionesModelos != 0 {
		t.Errorf("el arranque no pregunta por los modelos: %d", p.peticionesModelos)
	}
	if v := sinEstilo(a.View()); strings.Contains(v, "llama3.2") || strings.Contains(v, "qwen2.5") {
		t.Errorf("la bienvenida no espera a la lista:\n%s", v)
	}
}

func TestAbrirElModalEmiteUnaSolaPeticion(t *testing.T) {
	p := &puertoStub{modelos: modelosDePrueba(), modelo: "llama3.2"}
	a := Nuevo(p)
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})

	cmd := abreElModalDeModelos(t, a)
	// El contador sube cuando el comando de carga se ejecuta, no al pulsar: el
	// modal ya está en pantalla y solo falta la lista (SPEC-INTERFAZ §Modal de
	// modelos).
	ejecuta(t, a, cmd)
	if p.peticionesModelos != 1 {
		t.Errorf("abrir el modal pide la lista una vez: %d", p.peticionesModelos)
	}
	if v := sinEstilo(a.View()); strings.Contains(v, AvisoCargando) {
		t.Errorf("con la lista ya, el aviso de carga desaparece:\n%s", v)
	}
	// Volver a abrirlo vuelve a pedirla: lo que se ve es lo que hay ahora.
	tecla(t, a, tea.KeyEsc)
	ejecuta(t, a, abreElModalDeModelos(t, a))
	if p.peticionesModelos != 2 {
		t.Errorf("cada apertura es una petición nueva: %d", p.peticionesModelos)
	}
}

func TestLaRespuestaTardíaRellenaElModalYLaPantallaSigueEscribiéndose(t *testing.T) {
	p := &puertoStub{modelos: modelosDePrueba(), modelo: "llama3.2"}
	a := Nuevo(p)
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	escribe(t, a, "hola")

	// El modal se abre al instante, con el aviso de carga, sin esperar a Ollama.
	cmd := abreElModalDeModelos(t, a)
	if !strings.Contains(sinEstilo(a.View()), AvisoCargando) {
		t.Error("el modal se ve antes de que Ollama responda")
	}
	// Mientras espera, la pantalla se sigue escribiendo: nada está bloqueado.
	tecla(t, a, tea.KeyDown)
	if a.Bienvenida.Texto != "hola" {
		t.Errorf("la escritura de la pantalla no se pierde: %q", a.Bienvenida.Texto)
	}

	// Llega la respuesta y rellena el modal.
	ejecuta(t, a, cmd)
	v := sinEstilo(a.View())
	if !strings.Contains(v, "llama3.2") || !strings.Contains(v, "qwen2.5") {
		t.Errorf("la lista llega al modal:\n%s", v)
	}
}

// --- T-F013-04: la acción enrutada y el contexto del modal --------------------

func TestElModalDeModelosAbreDesdeLasDosVistas(t *testing.T) {
	p := &puertoStub{modelos: modelosDePrueba(), modelo: "llama3.2"}
	a := Nuevo(p)
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})

	// Bienvenida.
	abreElModalDeModelos(t, a)
	if !a.Modelos.Abierto {
		t.Fatal("ctrl+x m abre el modal desde la bienvenida")
	}
	tecla(t, a, tea.KeyEsc)

	// Interfaz principal: es global, se abre igual.
	a.Vista = VistaPrincipal
	abreElModalDeModelos(t, a)
	if !a.Modelos.Abierto {
		t.Error("ctrl+x m abre el modal desde la interfaz principal")
	}
}
func TestConElModalAbiertoLasTeclasSonDelModal(t *testing.T) {
	p := &puertoStub{modelos: modelosDePrueba(), modelo: "llama3.2"}
	a := Nuevo(p)
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	escribe(t, a, "petición")
	cmd := abreElModalDeModelos(t, a)
	ejecuta(t, a, cmd)

	// Enter aplica lo resaltado, no envía la petición.
	tecla(t, a, tea.KeyEnter)
	if len(p.enviados) != 0 {
		t.Errorf("con el modal abierto enter no envía: %v", p.enviados)
	}
	if a.Vista != VistaBienvenida {
		t.Error("con el modal abierto no se cambia de vista")
	}
	if a.Modelos.Abierto {
		t.Error("aplicar cierra el modal")
	}
	if a.Bienvenida.Texto != "petición" {
		t.Errorf("lo escrito sigue a la vista: %q", a.Bienvenida.Texto)
	}
}

func TestConElModalAbiertoLaEscrituraNoLlegaAlInput(t *testing.T) {
	p := &puertoStub{modelos: modelosDePrueba(), modelo: "llama3.2"}
	a := Nuevo(p)
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	cmd := abreElModalDeModelos(t, a)
	ejecuta(t, a, cmd)

	// Escribir con el modal abierto no compone texto: las teclas son suyas
	// (SPEC-INTERFAZ §Modal de modelos).
	escribe(t, a, "hola")
	if a.Bienvenida.Texto != "" {
		t.Errorf("con el modal abierto la línea de entrada no escribe: %q", a.Bienvenida.Texto)
	}
	// Cerrado, la escritura vuelve a funcionar.
	tecla(t, a, tea.KeyEsc)
	escribe(t, a, "hola")
	if a.Bienvenida.Texto != "hola" {
		t.Errorf("al cerrar el modal se escribe otra vez: %q", a.Bienvenida.Texto)
	}
}

// --- T-F013-06: el aviso de modelo sin herramientas --------------------------

func TestElegirUnModeloSinHerramientasAvisaSinBloquear(t *testing.T) {
	p := &puertoStub{
		modelo: "con-tools",
		modelos: []ModeloLocal{
			{Nombre: "con-tools"},
			{Nombre: "sin-tools", SinHerramientas: true},
		},
	}
	a := Nuevo(p)
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	ejecuta(t, a, abreElModalDeModelos(t, a))

	tecla(t, a, tea.KeyDown)  // sin-tools
	tecla(t, a, tea.KeyEnter) // aplica
	if a.Modelo != "sin-tools" {
		t.Fatalf("modelo elegido: %q", a.Modelo)
	}
	if !strings.Contains(sinEstilo(a.View()), "no puede usar herramientas") {
		t.Errorf("el modelo sin herramientas avisa sin bloquear:\n%s", sinEstilo(a.View()))
	}

	// Elegir uno capaz retira el aviso.
	ejecuta(t, a, abreElModalDeModelos(t, a))
	tecla(t, a, tea.KeyUp) // vuelve a con-tools (el foco arranca en el actual)
	tecla(t, a, tea.KeyEnter)
	if a.Modelo != "con-tools" {
		t.Fatalf("modelo elegido: %q", a.Modelo)
	}
	if strings.Contains(sinEstilo(a.View()), "no puede usar herramientas") {
		t.Errorf("el aviso se retira con un modelo capaz:\n%s", sinEstilo(a.View()))
	}
}

// --- T-F013-05: el modelo viaja con la primera petición ----------------------

func TestEnLaPrincipalElModalTambiénSecuestraLaEntrada(t *testing.T) {
	p := &puertoStub{modelos: modelosDePrueba(), modelo: "llama3.2"}
	a := Nuevo(p)
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	escribe(t, a, "algo")
	ejecuta(t, a, abreElModalDeModelos(t, a))

	// Igual que en la bienvenida: con el modal abierto, escribir no compone y
	// enter no envía (SPEC-KEYBINDS §Resolución por contexto).
	escribe(t, a, "más")
	tecla(t, a, tea.KeyEnter)
	if len(p.enviados) != 0 {
		t.Errorf("con el modal abierto la principal tampoco envía: %v", p.enviados)
	}
	if a.Modelos.Abierto || a.Modelo != "llama3.2" {
		t.Errorf("aplicar el primer modelo lo deja elegido: %q", a.Modelo)
	}
}

func TestElModeloElegidoViajaConLaPrimeraPetición(t *testing.T) {
	p := &puertoStub{modelos: modelosDePrueba(), modelo: "llama3.2"}
	a := Nuevo(p)
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	escribe(t, a, "documentar la capa")

	cmd := abreElModalDeModelos(t, a)
	ejecuta(t, a, cmd)
	tecla(t, a, tea.KeyDown) // qwen2.5
	tecla(t, a, tea.KeyEnter)
	ejecuta(t, a, pulsa(t, a, tea.KeyMsg{Type: tea.KeyEnter}))

	// El modelo elegido llega al motor y la petición sale con el, no con el que
	// detectó el arranque (SPEC-OLLAMA-PERFIL).
	if len(p.fijados) == 0 || p.fijados[len(p.fijados)-1] != "qwen2.5" {
		t.Errorf("el motor recibe el modelo elegido: %v", p.fijados)
	}
	if len(p.enviados) != 1 || p.enviados[0] != "nueva|documentar la capa" {
		t.Fatalf("la primera petición sale una vez: %v", p.enviados)
	}
}

func TestLaEleccionDelModalPrevaleceSobreLaAutodetección(t *testing.T) {
	p := &puertoStub{modelos: modelosDePrueba(), modelo: "detectado"}
	a := Nuevo(p)
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	if a.Modelo != "detectado" {
		t.Fatalf("al arrancar se ve el modelo detectado: %q", a.Modelo)
	}

	cmd := abreElModalDeModelos(t, a)
	ejecuta(t, a, cmd)
	tecla(t, a, tea.KeyEnter)

	if a.Modelo != "llama3.2" {
		t.Errorf("la línea de modelo refleja la elección: %q", a.Modelo)
	}
	if !strings.Contains(sinEstilo(a.View()), "modelo: llama3.2") {
		t.Error("la línea de modelo muestra lo elegido, no la autodetección")
	}
}

// --- el modal se pinta centrado ---------------------------------------------

func TestElModalSePintaCentrado(t *testing.T) {
	a := Nuevo(&puertoStub{modelos: modelosDePrueba(), modelo: "llama3.2"})
	pulsa(t, a, tea.WindowSizeMsg{Width: 60, Height: 20})
	cmd := abreElModalDeModelos(t, a)
	ejecuta(t, a, cmd)

	lineas := strings.Split(strings.TrimRight(sinEstilo(a.View()), "\n"), "\n")
	if len(lineas) < 3 {
		t.Fatalf("el modal pinta cabecera, lista y pie:\n%s", a.View())
	}
	// El bloque está centrado: la primera línea del modal no arranca en la
	// columna 0, sino con el margen que deja la ventana (SPEC-INTERFAZ §Modales:
	// "Tres modales centrados").
	if !strings.HasPrefix(lineas[0], " ") {
		t.Errorf("el modal se centra en la ventana:\n%s", a.View())
	}
}
