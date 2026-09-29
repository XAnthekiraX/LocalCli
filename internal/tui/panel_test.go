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
