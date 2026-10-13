package tools

import "time"

// EventoInvocacion es lo que se emite antes de ejecutar: quién pidió qué, con su
// verbo y su tema.
type EventoInvocacion struct {
	Sesion string
	Agente string
	Nombre string
	Verbo  string
	Tema   string
}

// EventoResultado es lo que se emite al terminar: el desenlace, la medida, si se
// recortó y cuánto tardó el handler. No lleva la salida.
type EventoResultado struct {
	Sesion   string
	Nombre   string
	Verbo    string
	Tema     string
	OK       bool
	Error    string
	Medida   string
	Truncado bool
	Duracion time.Duration
}

// Eventos son los enganches de pantalla y auditoría de la capa universal. Sin
// ellos la ejecución funciona igual.
type Eventos struct {
	Invocada  func(EventoInvocacion)
	Resultado func(EventoResultado)
}

func (e Eventos) invocada(ev EventoInvocacion) {
	if e.Invocada != nil {
		e.Invocada(ev)
	}
}

func (e Eventos) resultado(ev EventoResultado) {
	if e.Resultado != nil {
		e.Resultado(ev)
	}
}
