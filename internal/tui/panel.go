// panel.go — T-B014-04: el panel de datos.
//
// Fuente de verdad: SPEC-INTERFAZ §Zonas 3 (los nueve datos: sesión, contexto,
// TODO, ruta, git, capa y cola, aprobaciones, agente, proyecto) y §Reglas ("El
// panel de datos es de lectura. Nada se escribe desde él", "El panel refleja los
// datos de la sesión activa, no de otra", "Con el panel cerrado, el número de
// aprobaciones pendientes siempre se ve") y SPEC-PANEL-CONTEXTO §Reglas ("El
// conteo de tokens se muestra siempre. Si es una estimación, aparece marcado",
// "El contexto se acerca al límite: se avisa antes de que la petición falle").
//
// El panel es una estructura de datos y un render, nada más: no calcula nada por
// su cuenta ni consulta nada. Quien le pase un dato, manda.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// UmbralContexto es el porcentaje a partir del cual se avisa de que el contexto
// se acerca a su límite (SPEC-PANEL-CONTEXTO §Flujos alternativos).
const UmbralContexto = 80

// Panel es el estado del panel de datos de la sesión activa.
type Panel struct {
	Abierto bool

	// Sesión
	SesionID string
	Sesion   string
	Estado   string

	// Contexto (SPEC-PANEL-CONTEXTO)
	Tokens          int
	TokensEstimados bool
	LimiteTokens    int

	// TODO / capa y cola
	ElementoActual     string
	ElementosRestantes int
	Capa               string
	TareasGrandes      int

	// Ruta y git
	Ruta      string
	GitRama   string
	GitLimpio bool

	// Aprobaciones y agente
	Aprobaciones int
	Agente       string

	// Proyecto
	Proyecto string
	Version  string
}

// NuevoPanel crea el panel cerrado, con el nombre y la versión del proyecto: son
// los dos datos que no dependen de ninguna sesión.
func NuevoPanel() Panel {
	return Panel{Proyecto: Nombre, Version: Version, GitLimpio: true, Agente: "plan"}
}

// PorcentajeContexto devuelve qué parte del límite está ocupada (0 si no hay
// límite conocido).
func (p Panel) PorcentajeContexto() int {
	if p.LimiteTokens <= 0 {
		return 0
	}
	pc := p.Tokens * 100 / p.LimiteTokens
	if pc > 100 {
		return 100
	}
	return pc
}

// ContextoApRetado dice si toca avisar de que el contexto se está llenando.
func (p Panel) ContextoApretado() bool { return p.PorcentajeContexto() >= UmbralContexto }

// Filas devuelve los nueve datos en el orden de la spec.
func (p Panel) Filas() [][2]string {
	contexto := fmt.Sprintf("%d tokens", p.Tokens)
	if p.TokensEstimados {
		contexto += " (estimado)"
	}
	if p.LimiteTokens > 0 {
		contexto += fmt.Sprintf(" · %d%%", p.PorcentajeContexto())
	}
	todo := p.ElementoActual
	if todo == "" {
		todo = "—"
	}
	todo += fmt.Sprintf(" · quedan %d", p.ElementosRestantes)
	git := p.GitRama
	if git == "" {
		git = "—"
	}
	if p.GitLimpio {
		git += " · sin cambios"
	} else {
		git += " · con cambios sin confirmar"
	}
	capa := p.Capa
	if capa == "" {
		capa = "—"
	}
	capa += fmt.Sprintf(" · %d tareas grandes", p.TareasGrandes)
	return [][2]string{
		{"Sesión", valorODefecto(p.Sesion) + estadoEntreParentesis(p.Estado)},
		{"Contexto", contexto},
		{"TODO", todo},
		{"Ruta", valorODefecto(p.Ruta)},
		{"Git", git},
		{"Capa y cola", capa},
		{"Aprobaciones", fmt.Sprintf("%d esperando decisión", p.Aprobaciones)},
		{"Agente", valorODefecto(p.Agente)},
		{"Proyecto", p.Proyecto + " · " + p.Version},
	}
}

func valorODefecto(v string) string {
	if strings.TrimSpace(v) == "" {
		return "—"
	}
	return v
}

func estadoEntreParentesis(estado string) string {
	if estado == "" {
		return ""
	}
	return " (" + estado + ")"
}

// Render pinta el panel. Es de lectura: aquí no hay nada que pulsar.
func (p Panel) Render(ancho int) string {
	var b strings.Builder
	b.WriteString(estiloTitulo.Render("PANEL") + "\n\n")
	for _, f := range p.Filas() {
		b.WriteString(estiloEtiqueta.Render(f[0]) + "\n")
		b.WriteString("  " + recortar(f[1], ancho-2) + "\n")
	}
	if p.ContextoApretado() {
		b.WriteString("\n" + estiloAviso.Render("el contexto se está acercando a su límite"))
	}
	return strings.TrimRight(b.String(), "\n")
}

// AvisoAprobaciones es la línea que se ve SIEMPRE que haya algo esperando, con
// el panel abierto o cerrado: es la única información fuera del panel, porque su
// ausencia detiene el trabajo en segundo plano (SPEC-INTERFAZ).
func (p Panel) AvisoAprobaciones() string {
	if p.Aprobaciones <= 0 {
		return ""
	}
	n := "aprobaciones"
	if p.Aprobaciones == 1 {
		n = "aprobación"
	}
	return estiloAviso.Render(fmt.Sprintf("%d %s esperando tu decisión", p.Aprobaciones, n))
}

// AnchoPanel es el ancho fijo del panel cuando está abierto.
const AnchoPanel = 34

// estilos de la vista, en un solo sitio para que la pantalla no tenga colores
// sueltos por los archivos.
var (
	estiloUsuario      = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	estiloAgente       = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	estiloSistema      = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	estiloRazonamiento = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true)
	estiloTitulo       = lipgloss.NewStyle().Bold(true)
	estiloEtiqueta     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	estiloAviso        = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)
	estiloMarca        = lipgloss.NewStyle().Bold(true)
)
