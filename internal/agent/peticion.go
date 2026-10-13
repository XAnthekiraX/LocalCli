package agent

// Roles de un mensaje de la petición al modelo.
const (
	RolSistema     = "system"
	RolUsuario     = "user"
	RolAsistente   = "assistant"
	RolHerramienta = "tool"
)

// Llamada es una herramienta pedida por el modelo.
type Llamada struct {
	ID         string
	Nombre     string
	Argumentos string
}

// Imagen es una imagen adjunta a un mensaje del usuario. Es efímera: acompaña al
// turno y no se persiste; el núcleo común la serializa para el modelo.
type Imagen struct {
	Ruta  string
	MIME  string
	Datos []byte
}

// Mensaje es un turno de la petición al modelo. En el asistente llevan las
// llamadas pedidas; en la herramienta, el id de la llamada que responde. Las
// imágenes solo van en el mensaje del usuario que las aporta.
type Mensaje struct {
	Rol       string
	Contenido string
	Imagenes  []Imagen
	Llamadas  []Llamada
	IDLlamada string
}

// Peticion es lo que el agente entrega al motor en una ronda.
type Peticion struct {
	Mensajes     []Mensaje
	Herramientas []Herramienta
}

// Delta es un fragmento del modelo: respuesta final o razonamiento.
type Delta struct {
	Texto        string
	Razonamiento bool
}

// Uso es el consumo de tokens de una ronda.
type Uso struct {
	Entrada int
	Salida  int
}

// ConstruirPeticion arma la petición: el prompt del agente entero como mensaje de
// sistema, después el historial y, en el canal de herramientas, los esquemas de
// las concedidas. El catálogo no entra en el prompt.
func ConstruirPeticion(a Agente, historial []Mensaje, herramientas []Herramienta) Peticion {
	mensajes := make([]Mensaje, 0, len(historial)+1)
	mensajes = append(mensajes, Mensaje{Rol: RolSistema, Contenido: a.Prompt})
	mensajes = append(mensajes, historial...)
	return Peticion{Mensajes: mensajes, Herramientas: herramientas}
}
