package exec

import (
	"context"
	"errors"

	"localcli/internal/tools"
)

// Handler es el `Ejecutar` de ejecutar_comando para inyectar en el catálogo.
func (e *Ejecutor) Handler() tools.Ejecutar {
	return func(ctx context.Context, args any, c tools.Contexto) (tools.Resultado, error) {
		peticion, ok := args.(tools.PeticionEjecutarComando)
		if !ok {
			return tools.Resultado{Error: tools.CodBadArgs + ": argumentos inesperados"}, nil
		}
		salida, err := e.Ejecutar(ctx, Solicitud{Comando: peticion.Comando, Carpeta: peticion.Carpeta}, c)
		if err != nil {
			return resultadoDeError(err)
		}
		return tools.Resultado{Salida: salida.Texto(), Truncado: salida.Truncado}, nil
	}
}

// resultadoDeError distingue un error corregible (de negocio) de un fallo del
// harness: el primero vuelve al modelo, el segundo corta el turno.
func resultadoDeError(err error) (tools.Resultado, error) {
	var te *Error
	if errors.As(err, &te) {
		return tools.Resultado{Error: te.Error()}, nil
	}
	return tools.Resultado{}, err
}
