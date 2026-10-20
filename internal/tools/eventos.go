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
// recortó y cuánto tardó el handler. Lleva la salida y la llamada que la pidió
// para que la capa de sesión pueda persistirlas y reenviarlas al modelo en los
// turnos siguientes.
type EventoResultado struct {
	Sesion     string
	IDLlamada  string
	EnFlujo    bool
	Nombre     string
	Verbo      string
	Tema       string
	Argumentos string
	Salida     string
	OK         bool
	Error      string
	Medida     string
	Truncado   bool
	Duracion   time.Duration
	// Bloque es el bloque visual ya renderizado que se pinta bajo la línea
	// —el diagrama o la diferencia de un cambio— y BloqueTipo dice cuál es
	// (`diagrama` o `diferencia`). No viaja al modelo: es dato de pantalla
	// (EVENTS §3).
	Bloque     string
	BloqueTipo string
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
