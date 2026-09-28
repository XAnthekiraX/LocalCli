package tui

// Tests de la paleta de comandos de flujo (comandos.go). Una prueba, una regla
// (frontend/05-quality/TESTING.md §3):
//
//   - al escribir `/` se despliega la lista encima del input;
//   - el filtro y las flechas la recorren;
//   - Tab autocompleta (`/comando [petición]`) y Enter ejecuta;
//   - un comando no llega al modelo: se responde en el chat con su aviso;
//   - lo que no es comando (aunque empiece por `/`) sigue siendo chat.

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// teclear compone texto tecla a tecla, como el teclado real. Bubble Tea entrega
// el espacio como `KeySpace` CON su runa: sin ella, el textinput no inserta
// nada.
func teclear(t *testing.T, a *App, texto string) {
	t.Helper()
	for _, r := range texto {
		if r == ' ' {
			pulsa(t, a, tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
			continue
		}
		pulsa(t, a, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
}

// mainConSesion deja la interfaz principal con una sesión activa y tamaño
// conocido, lista para escribir en la línea de entrada.
func mainConSesion(t *testing.T) (*App, *puertoStub) {
	t.Helper()
	p := &puertoStub{}
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	return a, p
}

// --- el catálogo ------------------------------------------------------------

func TestComandoFlujoDeReconoceLosSeisComandos(t *testing.T) {
	nombres := []string{"/planificar", "/crear", "/actualizar", "/eliminar", "/resolver", "/ejecutar"}
	if got := len(comandosDeFlujo()); got != len(nombres) {
		t.Fatalf("el catálogo tiene %d comandos, quiero %d", got, len(nombres))
	}
	for _, n := range nombres {
		if _, ok := ComandoFlujoDe(comandosDeFlujo(), n); !ok {
			t.Errorf("%q debe reconocerse como comando de flujo", n)
		}
	}
	// Con objetivo detrás, el comando sigue siendo el primero de la línea.
	if c, ok := ComandoFlujoDe(comandosDeFlujo(), "/resolver arregla el login"); !ok || c.Nombre != "/resolver" {
		t.Errorf("/resolver con objetivo = %q, %v", c.Nombre, ok)
	}
}

func TestComandoFlujoDeRechazaTextoQueNoEsComando(t *testing.T) {
	for _, texto := range []string{"", "hola", "/otro", "/resolverx"} {
		if _, ok := ComandoFlujoDe(comandosDeFlujo(), texto); ok {
			t.Errorf("%q no debe ser un comando de flujo", texto)
		}
	}
}

// El catálogo de la paleta sale del puerto: un flujo propio declarado por el
// proyecto se lista y se reconoce, no solo los seis oficiales.
func TestLaPaletaListaElCatalogoDelPuerto(t *testing.T) {
	p := &puertoStub{comandos: []ComandoFlujo{
		{Nombre: "/resolver", Descripcion: "resolver un problema"},
		{Nombre: "/demo", Descripcion: "un flujo propio"},
	}}
	a := Nuevo(p)
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})

	teclear(t, a, "/")
	v := sinEstilo(a.View())
	if !strings.Contains(v, "/demo") {
		t.Fatalf("la paleta lista el flujo propio del proyecto:\n%s", v)
	}
	if strings.Contains(v, "/planificar") {
		t.Errorf("con catálogo del puerto no se cuela el respaldo oficial:\n%s", v)
	}
	if c, ok := ComandoFlujoDe(a.Comandos, "/demo algo"); !ok || c.Nombre != "/demo" {
		t.Errorf("el flujo propio se reconoce: %q, %v", c.Nombre, ok)
	}
}

// --- desplegar y filtrar ----------------------------------------------------

func TestLaPaletaSeDespliegaAlEscribirBarraYListaLosComandos(t *testing.T) {
	a, _ := mainConSesion(t)
	teclear(t, a, "/")

	if !a.Paleta.Abierto {
		t.Fatal("al escribir `/` se despliega la paleta de comandos")
	}
	v := sinEstilo(a.View())
	for _, c := range comandosDeFlujo() {
		if !strings.Contains(v, c.Nombre) {
			t.Errorf("la paleta debe listar %q:\n%s", c.Nombre, v)
		}
	}
	if c, _ := a.Paleta.Seleccionado(); c.Nombre != "/planificar" {
		t.Errorf("el resaltado arranca en el primer comando: %q", c.Nombre)
	}
}

func TestLaPaletaFiltraPorLoEscrito(t *testing.T) {
	a, _ := mainConSesion(t)
	teclear(t, a, "/res")

	v := sinEstilo(a.View())
	if !strings.Contains(v, "/resolver") {
		t.Fatalf("el filtro debe dejar /resolver:\n%s", v)
	}
	if strings.Contains(v, "/planificar") {
		t.Errorf("el filtro retira lo que no empieza por /res:\n%s", v)
	}
}

func TestLaPaletaSeRetiraAlEscribirLaPeticion(t *testing.T) {
	a, _ := mainConSesion(t)
	teclear(t, a, "/resolver ")
	if a.Paleta.Abierto {
		t.Error("con el espacio empieza la petición y la paleta se retira")
	}
}

func TestLasFlechasRecorrenLaPaleta(t *testing.T) {
	a, _ := mainConSesion(t)
	teclear(t, a, "/")

	pulsa(t, a, tea.KeyMsg{Type: tea.KeyDown})
	if c, _ := a.Paleta.Seleccionado(); c.Nombre != "/crear" {
		t.Errorf("la flecha abajo mueve el resaltado: %q", c.Nombre)
	}
	pulsa(t, a, tea.KeyMsg{Type: tea.KeyUp})
	if c, _ := a.Paleta.Seleccionado(); c.Nombre != "/planificar" {
		t.Errorf("la flecha arriba lo devuelve: %q", c.Nombre)
	}
}

// --- Tab autocompleta -------------------------------------------------------

func TestTabAutocompletaElComandoResaltado(t *testing.T) {
	a, _ := mainConSesion(t)
	teclear(t, a, "/res")
	tecla(t, a, tea.KeyTab)

	if a.Entrada.Texto() != "/resolver " {
		t.Fatalf("tab autocompleta el comando y deja escribir la petición: %q", a.Entrada.Texto())
	}
	if a.Paleta.Abierto {
		t.Error("tras autocompletar, la paleta se retira")
	}
	// El agente no cambió: Tab lo usa la paleta mientras está desplegada.
	if a.Agente != AgentePlan {
		t.Errorf("con la paleta abierta, tab no cambia de agente: %q", a.Agente)
	}
}

// --- Enter ejecuta ----------------------------------------------------------

func TestEnterEjecutaElComandoYLanzaElFlujo(t *testing.T) {
	a, p := mainConSesion(t)
	teclear(t, a, "/resolver")
	ejecuta(t, a, pulsa(t, a, tea.KeyMsg{Type: tea.KeyEnter}))

	if a.Entrada.Texto() != "" {
		t.Errorf("tras ejecutar, la línea queda limpia: %q", a.Entrada.Texto())
	}
	if a.Paleta.Abierto {
		t.Error("la paleta se retira al ejecutar")
	}
	// El comando va al motor por el puerto, que reconoce el flujo; la vista no
	// corre etapas.
	if len(p.enviados) != 1 || p.enviados[0] != "s1|/resolver" {
		t.Fatalf("el comando se arranca por el puerto: %v", p.enviados)
	}
	if !strings.Contains(sinEstilo(a.View()), "/resolver") {
		t.Errorf("se ve el eco del comando:\n%s", sinEstilo(a.View()))
	}
}

func TestEjecutarUnComandoConPeticionConservaLaPeticion(t *testing.T) {
	a, p := mainConSesion(t)
	teclear(t, a, "/resolver el login falla")
	ejecuta(t, a, pulsa(t, a, tea.KeyMsg{Type: tea.KeyEnter}))

	if len(p.enviados) != 1 || p.enviados[0] != "s1|/resolver el login falla" {
		t.Fatalf("la petición viaja con el comando: %v", p.enviados)
	}
	if !strings.Contains(sinEstilo(a.View()), "/resolver el login falla") {
		t.Errorf("el eco conserva la petición:\n%s", sinEstilo(a.View()))
	}
}

func TestUnTextoConBarraQueNoEsComandoSeRespondeComoChat(t *testing.T) {
	a, p := mainConSesion(t)
	teclear(t, a, "/otro")

	if a.Paleta.Abierto {
		t.Error("sin comandos que empiecen por /otro, la paleta no se despliega")
	}
	ejecuta(t, a, pulsa(t, a, tea.KeyMsg{Type: tea.KeyEnter}))
	if len(p.enviados) != 1 || p.enviados[0] != "s1|/otro" {
		t.Fatalf("lo que no es comando se envía como chat: %v", p.enviados)
	}
}

// --- la bienvenida ----------------------------------------------------------

func TestLaBienvenidaDespliegaLaPaletaYEjecutaElComando(t *testing.T) {
	p := &puertoStub{}
	a := Nuevo(p)
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})

	teclear(t, a, "/resolver")
	if !a.Paleta.Abierto {
		t.Fatal("la paleta también se despliega en la bienvenida")
	}
	if !strings.Contains(sinEstilo(a.View()), "/resolver") {
		t.Error("la paleta se ve en la bienvenida")
	}

	ejecuta(t, a, pulsa(t, a, tea.KeyMsg{Type: tea.KeyEnter}))
	if a.Vista != VistaPrincipal {
		t.Error("ejecutar un comando lleva a la interfaz principal")
	}
	// Desde la bienvenida no había sesión: se crea una y el comando se arranca
	// por el puerto, que reconoce el flujo.
	if p.creadas != 1 {
		t.Errorf("el comando desde la bienvenida crea una sesión: %d", p.creadas)
	}
	if len(p.enviados) != 1 || p.enviados[0] != "nueva|/resolver" {
		t.Fatalf("el comando se arranca por el puerto: %v", p.enviados)
	}
}
