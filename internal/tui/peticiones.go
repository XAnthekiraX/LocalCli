package tui

import (
	"context"
	"errors"

	tea "github.com/charmbracelet/bubbletea"
)

type errorMsg struct{ err error }

// enviar manda lo escrito. La primera petición de la bienvenida crea la sesión.
func (a *App) enviar() tea.Cmd {
	texto := a.bienvenida.Peticion()
	if texto == "" {
		return nil
	}
	if cmd, consumido := a.ejecutarComandoLocal(texto); consumido {
		a.bienvenida.Limpiar()
		return cmd
	}
	if a.puerto == nil {
		a.pasoALaPrincipal()
		return nil
	}
	if a.sesion == nil {
		sesion, err := a.puerto.Crear()
		if err != nil {
			a.aviso = "no se pudo crear la sesión: " + err.Error()
			return nil
		}
		a.sesion = sesion
	}
	sesionID, agente := a.sesion.ID, a.agente
	// Lo escrito es la primera petición y aparece como primer mensaje del chat,
	// sin repetirla ni pedir confirmación.
	a.ultimaPeticion = texto
	a.chat.AñadirUsuario(texto)
	a.pasoALaPrincipal()
	return tea.Batch(a.lanzarTurno(sesionID, agente, texto), a.arrancarLatido())
}

// lanzarTurno arranca un turno cancelable y guarda su cancelador, que `esc` usa
// para detenerlo.
func (a *App) lanzarTurno(sesionID, agente, texto string) tea.Cmd {
	ctx, cancel := context.WithCancel(context.Background())
	a.cancelarTurno = cancel
	a.cancelarArmado = false
	return enviarCmd(ctx, a.puerto, sesionID, agente, texto)
}

func enviarCmd(ctx context.Context, puerto Puerto, sesionID, agente, texto string) tea.Cmd {
	return func() tea.Msg {
		if err := puerto.Enviar(ctx, sesionID, agente, texto); err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return errorMsg{err: err}
		}
		return nil
	}
}

// pasoALaPrincipal cambia de vista sin repetir la petición ni pedir confirmación.
func (a *App) pasoALaPrincipal() {
	a.bienvenida.Limpiar()
	a.entrada.Limpiar()
	a.sel.limpiar()
	a.aviso = ""
	a.vista = VistaPrincipal
}
