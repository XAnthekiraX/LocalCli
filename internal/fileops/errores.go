package fileops

import "localcli/internal/tools"

// Error es un error de archivos con su código interno.
type Error struct {
	Codigo  string
	Mensaje string
}

func (e *Error) Error() string { return e.Codigo + ": " + e.Mensaje }

func nuevaError(codigo, mensaje string) *Error {
	return &Error{Codigo: codigo, Mensaje: mensaje}
}

// corregible devuelve un resultado que el modelo puede leer y corregir.
func corregible(mensaje string) (tools.Resultado, error) {
	return tools.Resultado{Error: mensaje}, nil
}

// resultado traduce un error: corregible vuelve al modelo, de harness corta.
func resultado(err error) (tools.Resultado, error) {
	if te, ok := err.(*Error); ok {
		return tools.Resultado{Error: te.Error()}, nil
	}
	return tools.Resultado{}, err
}
