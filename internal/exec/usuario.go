package exec

import (
	"context"
	"encoding/json"
	"time"

	"localcli/internal/tools"
)

// HandlerUsuario es el adaptador de una herramienta del usuario sobre el
// ejecutor. Inyecta los argumentos por la entrada estándar y ejecuta `equipo`
// como lista fija, sin shell: los argumentos del modelo no llegan a la lista.
func (e *Ejecutor) HandlerUsuario(u tools.HerramientaUsuario) tools.Ejecutar {
	limite := time.Duration(u.TimeoutSegundos) * time.Second
	equipo := append([]string(nil), u.Equipo...)
	return func(ctx context.Context, args any, c tools.Contexto) (tools.Resultado, error) {
		entrada, err := json.Marshal(args)
		if err != nil {
			return tools.Resultado{Error: tools.CodBadArgs + ": los argumentos no se pudieron serializar"}, nil
		}
		salida, err := e.EjecutarEquipo(ctx, equipo, string(entrada), "", limite, c)
		if err != nil {
			return resultadoDeError(err)
		}
		return tools.Resultado{Salida: salida.Texto(), Truncado: salida.Truncado}, nil
	}
}
