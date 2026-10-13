package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// La entrada reconoce tres familias de invocación (SPEC-INTERFAZ §Sintaxis de
// comandos slash):
//
//   - `/comando`        comando de la interfaz: se resuelve aquí, como acción
//     local, y no se envía al agente.
//   - `/flow:<nombre>`  flujo concreto: lo resuelve el orquestador; aquí solo se
//     deja pasar.
//   - `/skill:<nombre>` skill concreta: reconocida, pero todavía sin motor.
//
// Un texto sin barra es una petición normal. Este archivo es la frontera donde
// se decide qué se consume localmente y qué sigue su camino hasta la sesión.
const (
	prefijoFlujo = "/flow:"
	prefijoSkill = "/skill:"
)

// ejecutarComandoLocal mira si lo escrito es un comando de la interfaz y, si lo
// es, lo ejecuta. Devuelve el comando de Bubble Tea y si lo consumió. Un flujo
// (`/flow:…`) no se consume: sigue hacia el orquestador. Un texto sin barra
// tampoco.
func (a *App) ejecutarComandoLocal(texto string) (tea.Cmd, bool) {
	recortado := strings.TrimSpace(texto)
	if !strings.HasPrefix(recortado, "/") {
		return nil, false
	}
	if strings.HasPrefix(recortado, prefijoFlujo) {
		return nil, false
	}
	if motivo, ok := a.avisoDeSkill(recortado); ok {
		a.notificarLocal(motivo)
		return nil, true
	}
	nombre, _, _ := strings.Cut(recortado, " ")
	switch nombre {
	case "/models":
		return a.abrirModalModelos(), true
	case "/descargar":
		return a.abrirModalDescargar(), true
	case "/sessions":
		return a.abrirModalSesiones(), true
	case "/undo":
		return a.deshacerTurno(), true
	case "/redo":
		return a.rehacerTurno(), true
	case "/retry":
		return a.repetirUltimaPeticion(), true
	case "/init":
		a.notificarLocal("/init todavía no está implementado")
		return nil, true
	default:
		a.notificarLocal("comando desconocido: " + nombre)
		return nil, true
	}
}

// avisoDeSkill reconoce una invocación `/skill:<nombre>` y devuelve el aviso de
// que aún no hay motor de skills. El prefijo desnudo también lo da.
func (a *App) avisoDeSkill(recortado string) (string, bool) {
	if !strings.HasPrefix(recortado, prefijoSkill) {
		return "", false
	}
	nombre := strings.TrimSpace(strings.TrimPrefix(recortado, prefijoSkill))
	nombre, _, _ = strings.Cut(nombre, " ")
	if nombre == "" {
		return "indica el nombre de la skill tras /skill:", true
	}
	return "las skills todavía no están implementadas: " + nombre, true
}

// deshacerTurno deshace el último turno de la conversación de la sesión activa y
// deja el hilo con lo que queda. Sin sesión solo avisa.
func (a *App) deshacerTurno() tea.Cmd {
	if a.sesion == nil {
		a.notificarLocal("no hay una sesión activa para deshacer")
		return nil
	}
	return turnoTocadoCmd(a.puerto, a.sesion.ID,
		"no se pudo deshacer", "turno deshecho", "no hay nada que deshacer", Puerto.Deshacer)
}

// rehacerTurno vuelve a poner en pie el último turno deshecho de la sesión activa.
func (a *App) rehacerTurno() tea.Cmd {
	if a.sesion == nil {
		a.notificarLocal("no hay una sesión activa para rehacer")
		return nil
	}
	return turnoTocadoCmd(a.puerto, a.sesion.ID,
		"no se pudo rehacer", "turno rehecho", "no hay nada que rehacer", Puerto.Rehacer)
}

// turnoTocadoCmd ejecuta el deshacer o el rehacer y trae de vuelta el hilo ya
// cambiado con su aviso: la operación y la recarga viajan juntas, así el mensaje
// pinta el estado nuevo y no uno a medias. Un turno que no cambió nada deja el
// aviso de vacío y no recarga.
func turnoTocadoCmd(p Puerto, sesionID, fallo, exito, vacio string, op func(Puerto, string) (int, error)) tea.Cmd {
	return func() tea.Msg {
		if p == nil {
			return nil
		}
		n, err := op(p, sesionID)
		if err != nil {
			return avisoTurnoMsg{ID: sesionID, Texto: fallo + ": " + err.Error()}
		}
		// Sin nada que deshacer o rehacer no se recarga el hilo: solo se avisa,
		// para no vaciarlo cuando no había cambio que pintar.
		if n == 0 {
			return avisoTurnoMsg{ID: sesionID, Texto: vacio}
		}
		mensajes, err := p.Historial(sesionID)
		if err != nil {
			return avisoTurnoMsg{ID: sesionID, Texto: "no se pudo leer el historial: " + err.Error()}
		}
		return historialMsg{ID: sesionID, Mensajes: mensajes, Aviso: exito}
	}
}

// repetirUltimaPeticion vuelve a lanzar la última petición enviada, sin volver a
// teclearla. Sin sesión ni petición previa solo deja el aviso.
func (a *App) repetirUltimaPeticion() tea.Cmd {
	if a.sesion == nil {
		a.notificarLocal("no hay una sesión activa para repetir la petición")
		return nil
	}
	if a.ultimaPeticion == "" {
		a.notificarLocal("no hay una petición previa que repetir")
		return nil
	}
	texto := a.ultimaPeticion
	a.chat.AñadirUsuario(texto)
	return tea.Batch(a.lanzarTurno(a.sesion.ID, a.agente, texto), a.arrancarLatido())
}

// notificarLocal deja un aviso de un comando local en la vista vigente: en la
// bienvenida va al aviso de la caja; en la principal, al hilo.
func (a *App) notificarLocal(motivo string) {
	if a.vista == VistaBienvenida {
		a.aviso = motivo
		return
	}
	a.chat.AnotarAviso(motivo)
}
