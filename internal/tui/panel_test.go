package tui

// Tests de T-F006: el panel de datos plegable. Una prueba, una regla
// (frontend/05-quality/TESTING.md §3). Las filas de los nueve datos, la marca
// de estimación y el aviso de contexto ya tenían pruebas de T-B014 en
// tui_test.go; aquí van las reglas nuevas de esta tarea.
//
//	T-F006-02 → panel cerrado: el chat recupera todo el ancho
//	T-F006-06 → git cambia con cambio_aplicado; cola global; contador por
//	            eventos; lo de otra sesión no entra al panel

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"localcli/internal/session"
)

// --- T-F006-02 / T-F041: visible por defecto y plegado ------------------------

func TestElSidebarEstaAbiertoPorDefectoYPlegarloDevuelveElAncho(t *testing.T) {
	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})

	if !a.Panel.Abierto || !strings.Contains(sinEstilo(a.View()), "CONTEXTO") {
		t.Fatal("el sidebar está visible por defecto")
	}
	if a.Entrada.Ancho != 100-AnchoPanel-1 {
		t.Errorf("con el sidebar abierto la entrada cede su ancho: %d", a.Entrada.Ancho)
	}
	tecla(t, a, tea.KeyCtrlD)
	if a.Panel.Abierto || a.Entrada.Ancho != 100 {
		t.Errorf("plegado, el chat recupera todo el ancho: abierto=%v ancho=%d", a.Panel.Abierto, a.Entrada.Ancho)
	}
	if strings.Contains(sinEstilo(a.View()), "CONTEXTO") {
		t.Error("plegado no se pinta")
	}
}

// --- T-F006-06: los eventos que alimentan el panel ------------------------------

func TestElCambioAplicadoDejaElGitConCambios(t *testing.T) {
	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	a.Panel.GitLimpio = true
	pulsa(t, a, eventoMsg{Evento: Evento{
		Nombre: EventoCambioAplicado,
		Datos:  map[string]string{"archivo": "main.go"},
	}})
	if a.Panel.GitLimpio {
		t.Error("un cambio aplicado deja el árbol con cambios sin confirmar")
	}
}

func TestLaColaEsGlobalYSeVeDesdeCualquierSesión(t *testing.T) {
	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"

	pulsa(t, a, eventoMsg{Evento: Evento{
		Nombre: EventoColaActualizada,
		Datos:  map[string]string{"capa": "frontend", "activa": "T-F006", "pendientes": "4"},
	}})
	if a.Panel.Capa != "frontend" || a.Panel.ElementoActual != "T-F006" || a.Panel.ElementosRestantes != 4 {
		t.Errorf("el resumen de la cola alimenta el panel: %+v", a.Panel)
	}
}

func TestElContadorDeAprobacionesSeActualizaConCadaEvento(t *testing.T) {
	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"

	pulsa(t, a, eventoMsg{Evento: Evento{
		Nombre: EventoPeticionAprobacion,
		Datos:  map[string]string{"aprobacion": "a1", "sesion": "s1", "descripcion": "crear archivo"},
	}})
	if a.Panel.Aprobaciones != 1 {
		t.Fatalf("la petición suma en el contador: %d", a.Panel.Aprobaciones)
	}
	pulsa(t, a, eventoMsg{Evento: Evento{
		Nombre: EventoAprobacionResuelta,
		Datos:  map[string]string{"aprobacion": "a1"},
	}})
	if a.Panel.Aprobaciones != 0 {
		t.Errorf("la resolución resta en el contador: %d", a.Panel.Aprobaciones)
	}
}

func TestElEstadoDeOtraSesiónNoCambiaElPanel(t *testing.T) {
	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	a.Panel.Estado = session.EstadoTrabajando

	pulsa(t, a, eventoMsg{Evento: Evento{
		Nombre: session.EventoEstadoSesion,
		Datos:  map[string]string{"sesion": "s2", "estado": session.EstadoTerminada},
	}})
	if a.Panel.Estado != session.EstadoTrabajando {
		t.Errorf("el panel refleja la sesión activa, no otra: %s", a.Panel.Estado)
	}
}

// --- T-F006-08: el contexto del panel -----------------------------------------

// El CONTEXTO del panel son los tokens del chat que forman el contexto de la
// sesión (mensajes de usuario y de agente); no el consumo del turno. Se alimenta
// al cargar la sesión y al cerrarse cada turno.
func TestElContextoDelPanelSonLosTokensDelChatDeLaSesion(t *testing.T) {
	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, historialMsg{Sesion: "s1", ContextoTokens: 3210, LimiteTokens: 16000})
	if a.Panel.ContextoTokens != 3210 || a.Panel.LimiteTokens != 16000 {
		t.Fatalf("la carga del historial trae los números del contexto: %+v", a.Panel)
	}
	if !a.Panel.TokensEstimados {
		t.Error("el contexto del chat es una estimación y debe marcarse como tal")
	}
	// El cierre de un turno actualiza el contexto y, aparte, el consumo del
	// turno que se pinta bajo la entrada.
	pulsa(t, a, eventoMsg{Evento: Evento{
		Nombre: EventoTokensTurno,
		Datos:  map[string]string{"entrada": "100", "salida": "50", "contexto": "4000", "limite": "16000"},
	}})
	if a.Panel.ContextoTokens != 4000 {
		t.Errorf("el cierre del turno actualiza el contexto del chat: %d", a.Panel.ContextoTokens)
	}
	if a.Panel.Tokens != 50 {
		t.Errorf("la línea bajo la entrada es el consumo del turno: %d", a.Panel.Tokens)
	}
}

// --- T-F006-07: la lista de pasos del agente (SPEC-TOOLS) ---------------------

func TestElPanelPintaLaListaDePasosDelAgente(t *testing.T) {
	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	a.Panel.Abierto = true
	a.Panel.SesionID = "s1"
	pulsa(t, a, eventoMsg{Evento: Evento{
		Nombre: EventoTodoActualizada,
		Datos: map[string]string{
			"sesion":    "s1",
			"elementos": `[{"contenido":"leer el esquema","estado":"completada"},{"contenido":"migrar la tabla","estado":"en_progreso"},{"contenido":"escribir la doc","estado":"pendiente"}]`,
		},
	}})
	v := sinEstilo(a.View())
	for _, esperado := range []string{
		"LISTA DE TAREAS",
		"[✓] leer el esquema",
		"[•] migrar la tabla",
		"[ ] escribir la doc",
	} {
		if !strings.Contains(v, esperado) {
			t.Errorf("el panel no muestra %q:\n%s", esperado, v)
		}
	}
}

func TestLaListaDePasosSeOcultaCuandoTodoEstaHecho(t *testing.T) {
	p := Panel{Tareas: []TareaPanel{{Contenido: "a", Estado: "completada"}, {Contenido: "b", Estado: "cancelada"}}}
	if strings.Contains(sinEstilo(p.Render(30, 30)), "LISTA DE TAREAS") {
		t.Error("un checklist sin nada accionable no se pinta")
	}
	p.Tareas = append(p.Tareas, TareaPanel{Contenido: "c", Estado: "pendiente"})
	if !strings.Contains(sinEstilo(p.Render(30, 30)), "LISTA DE TAREAS") {
		t.Error("con un paso pendiente, la sección se pinta")
	}
}

func TestLaListaDePasosDeOtraSesiónNoEntraAlPanel(t *testing.T) {
	a := Nuevo(&puertoStub{})
	a.Vista = VistaPrincipal
	a.Panel.SesionID = "s1"
	pulsa(t, a, eventoMsg{Evento: Evento{
		Nombre: EventoTodoActualizada,
		Datos:  map[string]string{"sesion": "s2", "elementos": `[{"contenido":"x","estado":"pendiente"}]`},
	}})
	if len(a.Panel.Tareas) != 0 {
		t.Errorf("el panel refleja la sesión activa, no otra: %+v", a.Panel.Tareas)
	}
}

// --- El pie del panel: git, ruta y firma del harness ---------------------------

func TestElPieDelPanelMuestraLaCarpetaDelProyecto(t *testing.T) {
	a := Nuevo(&puertoStub{carpeta: "/home/dev/mi-api"})
	a.Vista = VistaPrincipal
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})

	if a.Panel.Ruta != "/home/dev/mi-api" {
		t.Fatalf("el panel recibe la carpeta del arranque: %q", a.Panel.Ruta)
	}
	if v := sinEstilo(a.View()); !strings.Contains(v, "[/home/dev/mi-api]") {
		t.Errorf("el pie del panel no muestra la carpeta del proyecto:\n%s", v)
	}
}

func TestElPieSonGitRutaYFirmaEnEseOrden(t *testing.T) {
	p := Panel{Ruta: "/home/dev/mi-api", GitRama: "main", GitLimpio: true, Proyecto: Nombre, Version: Version}
	lineas := strings.Split(sinEstilo(p.Render(AnchoPanel, 30)), "\n")
	pie := lineas[len(lineas)-3:]

	esperado := []string{"Git main · sin cambios", "[/home/dev/mi-api]", Nombre + " · " + Version}
	for i, e := range esperado {
		if strings.TrimRight(pie[i], " ") != e {
			t.Errorf("fila %d del pie = %q, se esperaba %q", i, pie[i], e)
		}
	}
	for _, fila := range lineas[:len(lineas)-3] {
		if strings.HasPrefix(fila, "Git ") {
			t.Errorf("la fila de git se repite en ESTADO: %q", fila)
		}
	}
}

func TestLaFirmaDelHarnessNoRepiteLaEtiquetaProyecto(t *testing.T) {
	// En el pie la firma va suelta, como la ruta entre corchetes: el nombre y la
	// versión se dicen solos y una etiqueta «Proyecto» ahí solo estorba.
	p := Panel{Ruta: "/x", Proyecto: Nombre, Version: Version}
	v := sinEstilo(p.Render(AnchoPanel, 30))
	if strings.Contains(v, "Proyecto") {
		t.Errorf("el pie no rotula la firma del harness:\n%s", v)
	}
	if !strings.Contains(v, Nombre+" · "+Version) {
		t.Errorf("el pie muestra el nombre y la versión del harness:\n%s", v)
	}
}

func TestElPieDiceSinIniciarCuandoElProyectoNoTieneGit(t *testing.T) {
	a := Nuevo(&puertoStub{carpeta: "/home/dev/mi-api"}) // sin git: la rama llega vacía
	a.Vista = VistaPrincipal
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})

	v := sinEstilo(a.View())
	if !strings.Contains(v, "Git sin iniciar") {
		t.Errorf("sin repositorio se dice «sin iniciar», no un hueco:\n%s", v)
	}
	// Y no se disfraza de árbol limpio: sin repo no hay nada que ver de limpio.
	if strings.Contains(v, "sin iniciar · sin cambios") {
		t.Errorf("«sin iniciar» es un dato entero, no la rama vacía:\n%s", v)
	}
}

func TestLaFilaDeGitDistingueElArbolSucio(t *testing.T) {
	casos := []struct {
		panel Panel
		fila  string
	}{
		{Panel{GitRama: "main", GitLimpio: true}, "Git main · sin cambios"},
		{Panel{GitRama: "main", GitLimpio: false}, "Git main · con cambios sin confirmar"},
		{Panel{}, "Git sin iniciar"},
	}
	for _, c := range casos {
		if fila := sinEstilo(filaDeDato("Git", c.panel.textoGit(), AnchoPanel)); strings.TrimRight(fila, " ") != c.fila {
			t.Errorf("fila de git = %q, se esperaba %q", fila, c.fila)
		}
	}
}

func TestLaRutaDelPieSeAbreviaConTilde(t *testing.T) {
	home := rutaDelUsuario()
	if home == "" {
		t.Skip("el sistema no declara la carpeta del usuario")
	}
	casos := []struct{ ruta, breve string }{
		{home, "~"},
		{home + "/dev/mi-api", "~/dev/mi-api"},
		{"/otro/sitio/de/fuera", "/otro/sitio/de/fuera"},
		{"", ""},
	}
	for _, c := range casos {
		if got := rutaBreve(c.ruta); got != c.breve {
			t.Errorf("rutaBreve(%q) = %q, se esperaba %q", c.ruta, got, c.breve)
		}
	}
}

func TestLaRutaSeRecortaPorLaIzquierda(t *testing.T) {
	casos := []struct {
		texto, esperado string
		ancho           int
	}{
		{"/home/dev/mi-api", "/home/dev/mi-api", 38},
		{"/home/dev/mi-api", "…i-api", 6},
		{"/home/dev/mi-api", "…", 1},
		{"/home/dev/mi-api", "", 0},
	}
	for _, c := range casos {
		if got := truncarPorLaIzquierda(c.texto, c.ancho); got != c.esperado {
			t.Errorf("truncarPorLaIzquierda(%q, %d) = %q, se esperaba %q", c.texto, c.ancho, got, c.esperado)
		}
	}
}

func TestElPieNoDesapareceConMuchasTareas(t *testing.T) {
	p := Panel{Sesion: "api de pedidos", Ruta: "/home/dev/mi-api", Proyecto: Nombre, Version: Version}
	for i := range 20 {
		p.Tareas = append(p.Tareas, TareaPanel{Contenido: fmt.Sprintf("paso %d", i), Estado: "pendiente"})
	}
	lineas := strings.Split(sinEstilo(p.Render(AnchoPanel, 24)), "\n")

	if len(lineas) != 24 {
		t.Fatalf("el panel ocupa las 24 filas de la columna, ni una más ni una menos: %d", len(lineas))
	}
	// El pie son tres filas y se quedan abajo enteras: con una lista que no cabe
	// lo que cede es la lista, nunca el pie.
	pie := strings.Join(lineas[len(lineas)-3:], "\n")
	for _, esperado := range []string{"Git", "[/home/dev/mi-api]", Nombre + " · " + Version} {
		if !strings.Contains(pie, esperado) {
			t.Errorf("el pie conserva %q:\n%s", esperado, strings.Join(lineas, "\n"))
		}
	}
	if !strings.Contains(strings.Join(lineas, "\n"), "más") {
		t.Error("los pasos que no caben se resumen en lugar de empujar el pie fuera de la pantalla")
	}
}

func TestElPieNoDesapareceConUnaTerminalMasBajaQueElPanel(t *testing.T) {
	// Menos filas que el pie: no cabe nada, pero el pie entero se sigue viendo
	// porque es lo último que se compone y lo único que no se recorta.
	p := Panel{Ruta: "/home/dev/mi-api", GitRama: "main", GitLimpio: true, Proyecto: Nombre, Version: Version}
	lineas := strings.Split(sinEstilo(p.Render(AnchoPanel, 3)), "\n")

	if len(lineas) != 3 {
		t.Fatalf("con alto 3 solo se pinta el pie: %d filas", len(lineas))
	}
	if !strings.Contains(lineas[0], "main") || !strings.Contains(lineas[1], "/home/dev/mi-api") {
		t.Errorf("el pie sale entero:\n%s", strings.Join(lineas, "\n"))
	}
}

func TestElPanelSinAlturaConocidaNoRecortaLaRuta(t *testing.T) {
	p := Panel{Sesion: "api de pedidos", Ruta: "/home/dev/mi-api"}
	p.Tareas = append(p.Tareas, TareaPanel{Contenido: "migrar la tabla", Estado: "en_progreso"})

	if v := sinEstilo(p.Render(AnchoPanel, 0)); !strings.Contains(v, "[/home/dev/mi-api]") || !strings.Contains(v, "migrar la tabla") {
		t.Errorf("sin geometría no se recorta nada:\n%s", v)
	}
}
