package welcome

import (
	"strings"

	"localcli/internal/tui/render"
)

type atajo struct {
	tecla  string
	accion string
}

// atajos son las pistas que anuncia la bienvenida; el mapa central de teclas los generará aquí.
var atajos = []atajo{
	{tecla: "TAB", accion: "agentes"},
	{tecla: "Ctrl+P", accion: "comandos"},
	{tecla: "Ctrl+X M", accion: "modelos"},
}

func barraPistas() string {
	pistas := make([]string, 0, len(atajos))
	for _, a := range atajos {
		pistas = append(pistas, render.EstiloBlanco.Render(a.tecla)+" "+render.EstiloSutil.Render(a.accion))
	}
	return strings.Join(pistas, " · ")
}
