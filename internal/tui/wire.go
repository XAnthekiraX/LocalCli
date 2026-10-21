package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Eventos que consume la vista (INTERFACES §1, EVENTS §1).
const (
	EventoToken                = "token"
	EventoEstadoSesion         = "estado_sesion"
	EventoTituloSesion         = "titulo_sesion"
	EventoNotificacion         = "notificacion"
	EventoPeticionAprobacion   = "peticion_aprobacion"
	EventoAprobacionResuelta   = "aprobacion_resuelta"
	EventoHerramientaInvocada  = "herramienta_invocada"
	EventoHerramientaResultado = "herramienta_resultado"
	EventoTokensTurno          = "tokens_turno"
	EventoEtapaIniciada        = "etapa_iniciada"
	EventoEtapaTerminada       = "etapa_terminada"
	EventoEtapaFallida         = "etapa_fallida"
	// EventoEtapaRespuesta publica la respuesta completa de una etapa de flujo
	// con `respuesta_en_chat`. La vista la pinta como bloque de chat en su sitio
	// cronológico: después de lo que la etapa ya dejó en el hilo y antes del
	// aviso de etapa terminada. Se guarda aparte, en `chat_evento`, y no es
	// conversación: no entra al contexto del modelo.
	EventoEtapaRespuesta    = "etapa_respuesta"
	EventoFlujoPausado      = "flujo_pausado"
	EventoFlujoReanudado    = "flujo_reanudado"
	EventoFlujoCancelado    = "flujo_cancelado"
	EventoPropuestaTodo     = "propuesta_todo"
	EventoColaActualizada   = "cola_actualizada"
	EventoElementoBloqueado = "elemento_bloqueado"
	EventoTodoActualizada   = "todo_actualizada"
	EventoCambioAplicado    = "cambio_aplicado"
	EventoContextoAuditado  = "contexto_auditado"
	// EventoMensajePersistido publica el id del mensaje del usuario recién
	// guardado en la base. La vista se lo cierre a la burbuja viva que pintó al
	// enviar, que nace sin id: sin él el menú del globo no podría revertir ni
	// bifurcar sobre ella.
	EventoMensajePersistido = "mensaje_persistido"
	// EventoCapacidadesModelo publica el estado de las capacidades del modelo
	// activo cuando el motor las resuelve (al cambiar de modelo).
	EventoCapacidadesModelo = "capacidades_modelo"
	// Eventos de la descarga de modelos por URL. No van ligados a ninguna sesión:
	// los dispara el modal de descarga y solo él los consume.
	EventoDescargaProgreso  = "descarga_progreso"
	EventoDescargaTerminada = "descarga_terminada"
	EventoDescargaFallida   = "descarga_fallida"
)

// Agentes base que conoce la vista de fábrica.
const (
	AgentePlan  = "plan"
	AgenteBuild = "build"
)

// Estados de la sesión que la vista reconoce ([[database/01-schema/ENUMS]]
// §Estado de sesión). La TUI no inventa estados ni transiciones: pinta los que
// le llegan.
const (
	EstadoInactiva         = "inactiva"
	EstadoTrabajando       = "trabajando"
	EstadoEsperandoPermiso = "esperando_permiso"
	EstadoTerminada        = "terminada"
	EstadoError            = "error"
)

// Evento es una notificación del orquestador: la sesión que lo produjo, su tipo y sus datos.
type Evento struct {
	Sesion string
	Tipo   string
	Datos  map[string]string
}

// Sesion es lo que la vista conoce de una sesión: identificador, nombre y estado.
type Sesion struct {
	ID     string
	Nombre string
	Estado string
}

// PiezaHistorial es un tramo del hilo guardado, ya traducido para que la vista lo
// pinte al retomar la sesión: o un turno de la conversación o una línea del
// sistema (una herramienta o una etapa de un flujo). La vista no importa el
// paquete del almacén: lo guardado llega traducido a estos datos mínimos.
type PiezaHistorial interface{ esPiezaHistorial() }

// MensajeHistorial es un turno guardado de una sesión: quién habló, su texto, el
// razonamiento del turno y lo que tardó.
type MensajeHistorial struct {
	// ID es el identificador del mensaje en la base. Permite referirse a un turno
	// concreto —revertir hasta él o bifurcar desde él— sin depender de su posición.
	ID string
	// Rol es quién produjo el turno: "user" o "agent", como en la base.
	Rol   string
	Texto string
	// Razonamiento es el texto crudo del turno, si se guardó.
	Razonamiento string
	// Duracion es lo que tardó la respuesta; cero = no se midió.
	Duracion time.Duration
}

func (*MensajeHistorial) esPiezaHistorial() {}

// LineaHistorial es una línea del sistema guardada del hilo: una herramienta, una
// etapa de un flujo o la respuesta de una etapa con `respuesta_en_chat`. `Texto`
// llega ya compuesto, tal como se vio en vivo. `Bloque`
// es el diagrama renderizado de una herramienta `diagram`, ya en líneas Unicode
// de ancho calculado: la TUI no lo interpreta, solo lo coloca bajo la línea
// (INTERFACES §1.1, EVENTS §3). Vacío para el resto de herramientas.
type LineaHistorial struct {
	// Tipo es el tipo de línea guardada: "herramienta", "proceso" o "respuesta",
	// como en la base. "respuesta" no se pinta como línea suelta: la vista la
	// levanta a bloque del agente.
	Tipo string
	// Texto es la línea tal como se pintó.
	Texto string
	// Bloque es el bloque visual ya renderizado —un diagrama o una diferencia—
	// que se pinta bajo la línea; `BloqueTipo` dice cuál es para que la TUI sepa
	// cómo pintarlo (INTERFACES §1.1, EVENTS §3). Vacío en el resto de líneas.
	Bloque     string
	BloqueTipo string
}

func (*LineaHistorial) esPiezaHistorial() {}

// ModeloLocal es una entrada del modal de modelos: el nombre que reporta el
// motor, el motor que lo sirve y si declara capacidad de herramientas. La vista
// no importa el paquete del motor: lo que él reporta llega traducido a este dato
// mínimo.
type ModeloLocal struct {
	Nombre string
	// Motor es el nombre visible de la instancia que sirve el modelo, no el del
	// tipo: dos motores del mismo tipo se distinguen así.
	Motor string
	// SinHerramientas marca los modelos que no declaran capacidad de usar
	// herramientas. El valor cero es «sí puede»: un fallo de detección no debe
	// alarmar.
	SinHerramientas bool
}

// FlujoVista es un flujo que la paleta de la entrada ofrece: el identificador
// que va tras `flow:` (el comando del flujo sin la barra). La vista no conoce el
// catálogo de flujos: lo sirve el orquestador.
type FlujoVista struct {
	// Comando es lo que sigue a `flow:` en la invocación, p. ej. "planificar".
	Comando string
}

// Git es el estado del repositorio del proyecto, leído una sola vez al arrancar
// (INTERFACES §3.1): ni el panel ni nadie lo revalida, porque una sesión trabaja
// siempre en la misma rama. Una rama vacía dice que el proyecto no tiene git
// inicializado —o que no hay git— y el pie lo pinta como `sin iniciar`.
type Git struct {
	Rama    string
	Cambios int
}

// Capacidad es el estado de una capacidad del modelo en la frontera de la vista:
// soportada, no soportada o todavía desconocida. La vista no lo calcula: recibe
// el estado ya resuelto del puerto y lo pinta.
type Capacidad int

const (
	CapacidadDesconocida Capacidad = iota
	CapacidadSoportada
	CapacidadNoSoportada
)

// Capacidades reúne lo que la vista sabe del modelo activo.
type Capacidades struct {
	Herramientas Capacidad
	Vision       Capacidad
	Razonamiento Capacidad
}

// Puerto es la frontera de la TUI con el orquestador: todo lo que la vista pide va por aquí.
type Puerto interface {
	Agentes() []string
	AgenteRecordado() string
	ModeloActual() string
	MotorActual() string
	// Capacidades devuelve el estado resuelto de las capacidades del modelo
	// activo. Se lee al arrancar y al cambiar de modelo, no en cada repintado.
	Capacidades() Capacidades
	// Carpeta es la carpeta del proyecto y Git el estado del repositorio: las dos
	// lecturas del pie del panel, que se hacen al arrancar y no se revalidan.
	Carpeta() string
	Git() Git
	// Modelos lista los modelos descargados (el catálogo) para el modal de
	// modelos. Se pide al abrir el modal, no en el arranque; si no hay ninguno,
	// el modal avisa «sin modelos» y nada se bloquea.
	Modelos() ([]ModeloLocal, error)
	// FijarModelo elige el modelo con el que trabajará el motor a partir de
	// ahora: lo elegido por el usuario prevalece sobre la autodetección del
	// arranque.
	FijarModelo(nombre string)
	// EliminarModelo borra el modelo indicado: su carpeta `models/<id>/` y su
	// entrada del catálogo. El modelo en uso no se elimina: devuelve un error
	// accionable y no toca ni el disco ni el catálogo.
	EliminarModelo(nombre string) error
	// DescargarModelo arranca la descarga de un modelo desde una URL directa de
	// GGUF y, opcionalmente, su proyector multimodal (mmproj). Valida las URLs y
	// devuelve el error de inmediato si no sirven; el avance y el desenlace llegan
	// por los eventos de descarga.
	DescargarModelo(ggufURL, mmprojURL string) error
	// CancelarDescarga corta la descarga en curso, si la hay.
	CancelarDescarga()
	// NombreDeURL deriva el nombre del modelo de una URL de descarga, para que el
	// modal lo anticipe antes de empezar.
	NombreDeURL(url string) string
	// FijarPensar deja si el chat deja razonar al modelo: apagado responde
	// directo, encendido piensa antes de contestar.
	FijarPensar(pensar bool)
	// Flujos lista los flujos cargados para la paleta de la entrada. El catálogo
	// de flujos lo sirve el orquestador: la vista no lo inventa ni lo mantiene.
	Flujos() []FlujoVista
	// Listar lista las sesiones del proyecto para el modal de sesiones: nombre y
	// estado de cada una, incluidas las de segundo plano.
	Listar() ([]Sesion, error)
	// Historial devuelve los mensajes guardados de una sesión, para pintarlos al
	// retomarla. Solo los de esa sesión: el contexto de una nunca se mezcla con el
	// de otra (SPEC-SESIONES).
	Historial(sesionID string) ([]PiezaHistorial, error)
	// SincronizarContexto calcula y emite el estado de tokens del contexto efectivo
	// de una sesión (resumen + tramo reciente) para sincronizar el panel al retomarla.
	SincronizarContexto(sesionID string) tea.Cmd
	// Deshacer deshace el último turno de la conversación de una sesión y
	// Rehacer lo vuelve a poner en pie; ninguno toca los archivos. Devuelven
	// cuántos mensajes cambiaron de estado: 0 es «no había nada que
	// deshacer/rehacer» y la vista solo avisa.
	Deshacer(sesionID string) (int, error)
	Rehacer(sesionID string) (int, error)
	// Eliminar borra una sesión y todo lo suyo en cascada; lo usa el modal de
	// sesiones con `Ctrl+D`. Si la sesión está trabajando, la vista pide
	// confirmación antes de llamar aquí.
	Eliminar(sesionID string) error
	Crear() (*Sesion, error)
	Enviar(ctx context.Context, sesionID, agente, texto string) error
	// Compactar fuerza la compactación del historial de una sesión: resume lo
	// antiguo y deja persistido el resumen. Lo usa el comando `/compactar`.
	// Devuelve `true` si persistió un resumen y `false` si no había nada que
	// resumir (el historial entero cabe en el tramo reciente).
	Compactar(ctx context.Context, sesionID string) (bool, error)
	// ResponderAprobacion entrega la decisión de una línea del panel de
	// aprobaciones: la sesión que la pidió deja de esperar y continúa o se corta.
	ResponderAprobacion(sesionID, aprobacionID string, aprobar bool) error
	Suscribir() (<-chan Evento, func())
	// DeshacerDesde revierte la conversación de una sesión desde el mensaje
	// indicado hasta el final. Devuelve cuántos mensajes se sacaron; 0 significa
	// que no había nada que revertir.
	DeshacerDesde(sesionID, mensajeID string) (int, error)
	// Bifurcar abre una sesión nueva con el historial de la sesión indicada hasta
	// el mensaje dado, incluido, y la devuelve para que la vista la adopte.
	Bifurcar(sesionID, mensajeID string) (*Sesion, error)
}
