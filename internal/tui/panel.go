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

	// TODO del agente (SPEC-TOOLS): la lista de pasos de la sesión activa.
	Tareas []TareaPanel

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

// NuevoPanel crea el sidebar visible por defecto, con el nombre y la versión del
// proyecto: son los dos datos que no dependen de ninguna sesión. Se pliega con
// `Ctrl+D` (SPEC-INTERFAZ §Disposición).
func NuevoPanel() Panel {
	return Panel{Abierto: true, Proyecto: Nombre, Version: Version, GitLimpio: true, Agente: "plan"}
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

// textoContexto compone el dato del contexto: los tokens usados, marcados como
// estimados cuando lo son. El porcentaje ocupado va en su propia línea.
func (p Panel) textoContexto() string {
	t := fmt.Sprintf("%d tokens", p.Tokens)
	if p.TokensEstimados {
		t += " (estimado)"
	}
	return t
}

// textoGit compone la rama activa y si el árbol tiene cambios sin confirmar.
func (p Panel) textoGit() string {
	git := valorODefecto(p.GitRama)
	if p.GitLimpio {
		return git + " · sin cambios"
	}
	return git + " · con cambios sin confirmar"
}

// textoCapa compone la capa en la que se trabaja y las tareas grandes que le
// quedan.
func (p Panel) textoCapa() string {
	return valorODefecto(p.Capa) + fmt.Sprintf(" · %d tareas grandes", p.TareasGrandes)
}

// filaDeDato alinea en el sidebar la etiqueta de un dato y su valor.
func filaDeDato(etiqueta, valor string, ancho int) string {
	resto := ancho - len([]rune(etiqueta)) - 1
	if resto < 1 {
		resto = 1
	}
	return estiloEtiqueta.Render(etiqueta) + " " + truncar(valor, resto)
}

// Render pinta el sidebar de la sesión activa: el título de la conversación
// arriba, los bloques CONTEXTO y TODO, la lista de tareas del agente, el estado
// compacto (git, capa y cola, aprobaciones, agente y proyecto) y, pegado al pie,
// la ruta del proyecto. Recibe el alto de la columna para que el pie quede abajo
// aunque sobre espacio. Es de lectura: aquí no hay nada que pulsar.
func (p Panel) Render(ancho, alto int) string {
	if ancho < 1 {
		ancho = 1
	}
	var lineas []string
	add := func(l string) { lineas = append(lineas, l) }

	// Título: la sesión activa, con su estado entre paréntesis.
	add(estiloTitulo.Render(truncar(valorODefecto(p.Sesion)+estadoEntreParentesis(p.Estado), ancho)))
	add("")

	// CONTEXTO: tokens usados, porcentaje ocupado y, si toca, el aviso de límite.
	add(estiloSeccion.Render("CONTEXTO"))
	add("  " + truncar(p.textoContexto(), ancho-2))
	add("  " + truncar(fmt.Sprintf("%d%% usada", p.PorcentajeContexto()), ancho-2))
	if p.ContextoApretado() {
		add("  " + estiloAviso.Render("cerca del límite"))
	}
	add("")

	// TODO: el elemento en curso y cuántos quedan.
	add(estiloSeccion.Render("▾ TODO"))
	add("  " + truncar(valorODefecto(p.ElementoActual)+fmt.Sprintf(" · quedan %d", p.ElementosRestantes), ancho-2))
	add("")

	// LISTA DE TAREAS: los pasos del agente, si queda alguno accionable.
	if t := p.RenderTareas(ancho); t != "" {
		for _, l := range strings.Split(t, "\n") {
			add(l)
		}
		add("")
	}

	// ESTADO: el resto de los datos de la sesión, en filas etiqueta + valor.
	add(estiloSeccion.Render("ESTADO"))
	add(filaDeDato("Git", p.textoGit(), ancho))
	add(filaDeDato("Capa y cola", p.textoCapa(), ancho))
	add(filaDeDato("Aprobaciones", fmt.Sprintf("%d esperando decisión", p.Aprobaciones), ancho))
	add(filaDeDato("Agente", valorODefecto(p.Agente), ancho))
	add(filaDeDato("Proyecto", p.Proyecto+" · "+p.Version, ancho))

	// El pie va pegado abajo: la ruta del proyecto.
	if alto > 1 {
		for len(lineas) < alto-1 {
			add("")
		}
	}
	add(estiloSistema.Render(truncar("["+valorODefecto(p.Ruta)+"]", ancho)))
	return strings.Join(lineas, "\n")
}

// RenderTareas pinta la lista de pasos de la sesión: `[•]` en curso, `[✓]`
// hecha, `[x]` cancelada y `[ ]` pendiente. Se oculta cuando no queda nada
// accionable —sin pasos, o todos completados o cancelados—, como el panel de
// opencode: un checklist terminado solo estorba (SPEC-TOOLS).
func (p Panel) RenderTareas(ancho int) string {
	accionable := false
	for _, t := range p.Tareas {
		if t.Estado == "pendiente" || t.Estado == "en_progreso" {
			accionable = true
			break
		}
	}
	if !accionable {
		return ""
	}
	var b strings.Builder
	b.WriteString(estiloSeccion.Render("LISTA DE TAREAS") + "\n")
	for _, t := range p.Tareas {
		b.WriteString("  " + glifoDeTarea(t.Estado) + " " + truncar(t.Contenido, ancho-4) + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// glifoDeTarea es la marca de cada estado en el panel.
func glifoDeTarea(estado string) string {
	switch estado {
	case "en_progreso":
		return "[•]"
	case "completada":
		return "[✓]"
	case "cancelada":
		return "[x]"
	default:
		return "[ ]"
	}
}

// AvisoAprobaciones pinta la línea de aviso con el formato de notify.go. La
// visibilidad la decide la vista (app.go): con el panel de datos abierto el
// dato ya está en su fila, y el aviso es para cuando el panel está cerrado
// (T-F009-03).
func (p Panel) AvisoAprobaciones() string {
	aviso := formatoAvisoPendientes(p.Aprobaciones)
	if aviso == "" {
		return ""
	}
	return estiloAviso.Render(aviso)
}

// AnchoPanel es el ancho fijo del sidebar cuando está abierto.
const AnchoPanel = 38

// Los estilos de la vista viven en styles.go (T-F002): un solo sitio para que
// la pantalla no tenga colores sueltos por los archivos.
