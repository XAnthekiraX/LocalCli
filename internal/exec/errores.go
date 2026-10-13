package exec

// Error es un error de terminal con su código interno.
type Error struct {
	Codigo  string
	Mensaje string
}

func (e *Error) Error() string { return e.Codigo + ": " + e.Mensaje }

func nuevaError(codigo, mensaje string) *Error {
	return &Error{Codigo: codigo, Mensaje: mensaje}
}
