package panel

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// valorODefecto dice que un dato vacío se pinta como ausente: el panel no deja
// huecos ni inventa un valor.
func valorODefecto(v string) string {
	if strings.TrimSpace(v) == "" {
		return "—"
	}
	return v
}

// estadoEntreParentesis acompaña al nombre de la sesión con su estado. Sin
// estado no se dejan los paréntesis colgando.
func estadoEntreParentesis(estado string) string {
	if estado == "" {
		return ""
	}
	return " (" + estado + ")"
}

// textoContexto compone el dato del contexto: los tokens que ocupa el chat en el
// contexto de la sesión, marcados como estimados cuando lo son (SPEC-PANEL-CONTEXTO
// §Reglas de negocio).
func (m Model) textoContexto() string {
	t := fmt.Sprintf("%d tokens", m.ContextoTokens)
	if m.TokensEstimados {
		t += " (estimado)"
	}
	return t
}

// textoPorcentajeContexto compone la ocupación sobre el límite. Cuando el motor
// no declaró su ventana, el límite es el tope del harness y eso se dice: no es la
// ventana del modelo, igual que una capacidad desconocida no se hace pasar por
// ausente (SPEC-MODELO-MOTOR §La ventana de contexto).
func (m Model) textoPorcentajeContexto() string {
	pc := fmt.Sprintf("%d%% usada", m.PorcentajeContexto())
	if m.LimiteDelHarness {
		pc += " · límite del harness"
	}
	return pc
}

// textoGit compone el estado del repositorio: la rama activa y el número de
// cambios sin confirmar. Sin rama no hay repositorio —proyecto sin git
// inicializar, o sin git instalado— y eso se dice en vez de dejar un hueco:
// «sin iniciar» es un dato, no un «no sé» (SPEC-INTERFAZ §Zonas 3).
func (m Model) textoGit() string {
	if m.GitRama == "" {
		return "sin iniciar"
	}
	return fmt.Sprintf("%s · %d", m.GitRama, m.GitCambios)
}

// firmaHarness compone la última fila del pie: el nombre de la herramienta y su
// versión. Si faltara alguno se pinta el que haya, sin dejar un separador
// colgando: el pie no inventa datos.
func (m Model) firmaHarness() string {
	switch {
	case m.Proyecto == "":
		return m.Version
	case m.Version == "":
		return m.Proyecto
	}
	return m.Proyecto + " · " + m.Version
}

// textoCapa compone la capa en la que se trabaja y las tareas grandes que le
// quedan.
func (m Model) textoCapa() string {
	return valorODefecto(m.Capa) + fmt.Sprintf(" · %d tareas grandes", m.TareasGrandes)
}

// glifoDeTarea es la marca de cada estado en el panel: `[•]` en curso, `[✓]`
// hecho, `[x]` cancelado y `[ ]` pendiente (SPEC-INTERFAZ §Zonas 3).
func glifoDeTarea(estado string) string {
	switch estado {
	case EnProgreso:
		return "[•]"
	case Completada:
		return "[✓]"
	case Cancelada:
		return "[x]"
	default:
		return "[ ]"
	}
}

// accionable dice si queda algún paso por hacer: con todos completados o
// cancelados la lista solo estorba y se oculta.
func accionable(tareas []Tarea) bool {
	for _, t := range tareas {
		if t.Estado == Pendiente || t.Estado == EnProgreso {
			return true
		}
	}
	return false
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
// proyecto, y repetir el home en cada repintado solo le quita sitio a la parte
// que dice en qué carpeta se está. Una ruta que no cuelga del home se devuelve
// tal cual.
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
