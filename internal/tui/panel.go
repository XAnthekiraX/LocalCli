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
// su cuenta ni consulta nada. Quien le pase un dato, manda. La única lectura que
// hace es la carpeta del usuario para abreviarla a `~` en el pie, y se lee una
// sola vez: a partir de ahí es texto puro.
package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
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
	// ContextoTokens es el total de tokens del chat que pertenece al contexto
	// (los mensajes del usuario y del agente de la sesión; las líneas de
	// procesamiento no cuentan). Es una estimación, así que se marca como tal.
	ContextoTokens int
	// Tokens es el consumo del turno en curso, el que se pinta bajo la entrada
	// («tokens: 54k»). No es el contexto.
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
	Ruta       string
	GitRama    string
	GitCambios int

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
	return Panel{Abierto: true, Proyecto: Nombre, Version: Version, Agente: "plan"}
}

// PorcentajeContexto devuelve qué parte del límite está ocupada por el contexto
// del chat (0 si no hay límite conocido).
func (p Panel) PorcentajeContexto() int {
	if p.LimiteTokens <= 0 {
		return 0
	}
	pc := p.ContextoTokens * 100 / p.LimiteTokens
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

// textoContexto compone el dato del contexto: los tokens que ocupa el chat en
// el contexto de la sesión, marcados como estimados cuando lo son. El
// porcentaje ocupado va en su propia línea.
func (p Panel) textoContexto() string {
	t := fmt.Sprintf("%d tokens", p.ContextoTokens)
	if p.TokensEstimados {
		t += " (estimado)"
	}
	return t
}

// textoGit compone el estado del repositorio: la rama activa y el número de
// cambios sin confirmar (0 si el árbol está limpio), sin texto de por medio. Sin
// rama no hay repositorio —el proyecto sin git inicializar, o sin git
// instalado—, y eso se dice en vez de dejar un hueco: «sin iniciar» es un dato,
// no un «no sé» (SPEC-INTERFAZ §Zonas 3).
func (p Panel) textoGit() string {
	if p.GitRama == "" {
		return "sin iniciar"
	}
	return fmt.Sprintf("%s · %d", p.GitRama, p.GitCambios)
}

// firmaHarness compone la última fila del pie: el nombre de la herramienta y su
// versión. En producción siempre están los dos, pero si faltara alguno se pinta
// el que haya en vez de dejar un separador colgando: el pie no inventa datos.
func (p Panel) firmaHarness() string {
	switch {
	case p.Proyecto == "":
		return p.Version
	case p.Version == "":
		return p.Proyecto
	}
	return p.Proyecto + " · " + p.Version
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
// arriba, los bloques CONTEXTO y TODO, la lista de tareas del agente y el estado
// compacto (capa y cola, aprobaciones y agente). Abajo, pegado al suelo de la
// columna, va el pie de tres filas: el estado de git, la ruta del proyecto y el
// nombre y la versión del harness.
//
// Recibe el alto de la columna porque el pie ocupa sus filas siempre: si la
// lista de tareas no cabe, se recorta y se resume, y si la terminal es más baja
// que el pie entero, se recorta el panel por arriba. Es de lectura: aquí no hay
// nada que pulsar.
func (p Panel) Render(ancho, alto int) string {
	if ancho < 1 {
		ancho = 1
	}
	// El texto del sidebar deja dos columnas de separación a cada lado: se
	// compone sobre el ancho interior y luego se sangra.
	sangria := 2
	ancho -= 2 * sangria
	if ancho < 1 {
		ancho = 1
	}

	// El pie, de arriba abajo: git, ruta y harness. La ruta va con el home
	// abreviado y recortada por la izquierda, porque el final es la parte que
	// la identifica.
	pie := []string{
		filaDeDato("Git", p.textoGit(), ancho),
		estiloSistema.Render(truncarPorLaIzquierda("["+rutaBreve(valorODefecto(p.Ruta))+"]", ancho)),
		estiloEtiqueta.Render(truncar(p.firmaHarness(), ancho)),
	}
	// `disponible` son las filas que quedan por encima del pie, descontando el
	// aire de arriba y el de abajo, y es el presupuesto de la lista de tareas. Si
	// no cabe, vale cero: el pie manda. En una columna más baja que el pie no
	// cabe nada de aire.
	aireAbajo := 1
	if alto < len(pie)+2 {
		aireAbajo = 0
	}
	disponible := max(alto-len(pie)-1-aireAbajo, 0)

	// PARTES FIJAS: su altura no depende de los datos, así que se componen
	// enteras y son las que el recorte final puede llegar a comerse.
	// Título: la sesión activa, con su estado entre paréntesis.
	cabeza := []string{
		estiloTitulo.Render(truncar(valorODefecto(p.Sesion)+estadoEntreParentesis(p.Estado), ancho)),
		"",
		// CONTEXTO: tokens usados, porcentaje ocupado y, si toca, el aviso de
		// límite.
		estiloSeccion.Render("CONTEXTO"),
		"  " + truncar(p.textoContexto(), ancho-2),
		"  " + truncar(fmt.Sprintf("%d%% usada", p.PorcentajeContexto()), ancho-2),
	}
	if p.ContextoApretado() {
		cabeza = append(cabeza, "  "+estiloAviso.Render("cerca del límite"))
	}
	// TODO: el elemento en curso y cuántos quedan.
	cabeza = append(cabeza,
		"",
		estiloSeccion.Render("▾ TODO"),
		"  "+truncar(valorODefecto(p.ElementoActual)+fmt.Sprintf(" · quedan %d", p.ElementosRestantes), ancho-2),
		"",
	)

	// ESTADO: el resto de los datos de la sesión, en filas etiqueta + valor. Git
	// y proyecto no están aquí: viven en el pie, que es donde se leen de un
	// vistazo sin tener que llegar hasta el bloque de arriba.
	estado := []string{
		estiloSeccion.Render("ESTADO"),
		filaDeDato("Capa y cola", p.textoCapa(), ancho),
		filaDeDato("Aprobaciones", fmt.Sprintf("%d esperando decisión", p.Aprobaciones), ancho),
		filaDeDato("Agente", valorODefecto(p.Agente), ancho),
	}

	// LISTA DE TAREAS: los pasos del agente, si queda alguno accionable. Es lo
	// único que crece sin límite, así que es lo que cede cuando la columna es
	// baja. La línea en blanco que la separa de ESTADO se reserva siempre, se
	// pinte lista o no, para que el presupuesto de filas sea el mismo.
	lineas := append([]string{}, cabeza...)
	// El presupuesto de filas de la lista es el hueco entre las dos partes fijas
	// más su separador en blanco. Sin altura conocida no hay tope: se pintan
	// todos los pasos.
	tope := len(p.Tareas) + 1
	if disponible > 0 {
		tope = disponible - len(cabeza) - len(estado) - 1
	}
	tareas := p.lineasDeTareas(ancho, tope)
	if len(tareas) > 0 {
		lineas = append(lineas, tareas...)
		lineas = append(lineas, "")
	}
	lineas = append(lineas, estado...)

	// El cuerpo se ajusta al hueco que deja el pie. Si sobran filas, el relleno
	// va después de ESTADO, que es donde estorban menos, para que el contenido se
	// quede arriba; si faltan, se recorta por abajo, que es donde está lo que
	// menos pesa.
	if disponible > 0 {
		if len(lineas) > disponible {
			lineas = lineas[:disponible]
		}
		for len(lineas) < disponible {
			lineas = append(lineas, "")
		}
	}
	lineas = append(lineas, pie...)

	// Aire vertical: una fila en blanco arriba del título y —si la columna tiene
	// sitio— otra debajo del pie, para que el texto no toque los bordes.
	lineas = append([]string{""}, lineas...)
	if aireAbajo > 0 {
		lineas = append(lineas, "")
	}

	// Una columna más baja que el pie entero es el único caso en que no hay
	// recorte limpio: se van las primeras filas del panel, porque lo que no
	// puede caerse es la ruta y la firma. Sin geometría no se toca nada.
	if alto > 0 && len(lineas) > alto {
		lineas = lineas[len(lineas)-alto:]
	}
	// Dos columnas de separación a la izquierda; a la derecha las deja libres el
	// relleno del layout (componerColumnas completa la fila a AnchoPanel).
	sangrado := strings.Repeat(" ", sangria)
	for i, l := range lineas {
		lineas[i] = sangrado + l
	}
	return strings.Join(lineas, "\n")
}

// lineasDeTareas pinta la lista de pasos de la sesión: `[•]` en curso, `[✓]`
// hecha, `[x]` cancelada y `[ ]` pendiente. Devuelve nil cuando no queda nada
// accionable —sin pasos, o todos completados o cancelados—, como el panel de
// opencode: un checklist terminado solo estorba (SPEC-TOOLS).
//
// `tope` es cuántas filas puede ocupar la lista como mucho, encabezado
// incluido. Lo que no cabe no se pierde: se resume en «N más», porque el pie no
// puede caerse por una lista larga.
func (p Panel) lineasDeTareas(ancho, tope int) []string {
	accionable := false
	for _, t := range p.Tareas {
		if t.Estado == "pendiente" || t.Estado == "en_progreso" {
			accionable = true
			break
		}
	}
	// Un tope de 1 o menos es que solo cabe el pie: ni siquiera el encabezado.
	if !accionable || tope < 2 {
		return nil
	}
	// El encabezado se queda con una de las filas del tope. Se pintan todos los
	// pasos, no solo los accionables: el glifo de cada uno dice si está hecho,
	// cancelado o pendiente.
	caben := min(len(p.Tareas), tope-1)
	lineas := make([]string, 0, tope)
	lineas = append(lineas, estiloSeccion.Render("LISTA DE TAREAS"))
	for _, t := range p.Tareas[:caben] {
		lineas = append(lineas, "  "+glifoDeTarea(t.Estado)+" "+truncar(t.Contenido, ancho-4))
	}
	if fuera := len(p.Tareas) - caben; fuera > 0 {
		if len(lineas) < tope {
			lineas = append(lineas, "  … "+strconv.Itoa(fuera)+" más")
		} else {
			// No queda fila para el resumen: le cede la suya al último paso, que
			// también cuenta como no pintado.
			lineas[len(lineas)-1] = "  … " + strconv.Itoa(fuera+1) + " más"
		}
	}
	return lineas
}

// rutaDelUsuario se lee la primera vez que hace falta y ya no cambia: el pie la
// necesita en cada repintado y la carpeta del usuario no se mueve mientras la
// sesión viva. Si el sistema no la dice, se queda vacía y cada ruta se pinta tal
// cual.
var rutaDelUsuario = sync.OnceValue(func() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
})

// rutaBreve abrevia la carpeta del usuario a `~`. La ruta del pie es la del
// proyecto, y repetir el home en cada arranque solo le quita sitio a la parte que
// dice en qué carpeta se está. Una ruta que no cuelga del home se devuelve tal
// cual.
func rutaBreve(ruta string) string {
	home := rutaDelUsuario()
	if ruta == "" || home == "" {
		return ruta
	}
	if ruta == home {
		return "~"
	}
	if prefijo := home + string(filepath.Separator); strings.HasPrefix(ruta, prefijo) {
		return "~" + ruta[len(home):]
	}
	return ruta
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

// AnchoPanel es el ancho fijo del sidebar cuando está abierto: 38 columnas de
// contenido más las dos de separación a cada lado (ver Render).
const AnchoPanel = 42

// Los estilos de la vista viven en styles.go (T-F002): un solo sitio para que
// la pantalla no tenga colores sueltos por los archivos.
