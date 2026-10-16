package tui

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"localcli/internal/tui/components/panel"
)

type eventoMsg struct{ evento Evento }

func (a *App) suscribir() {
	if a.puerto == nil || a.eventos != nil {
		return
	}
	canal, cerrar := a.puerto.Suscribir()
	a.eventos, a.cerrar = canal, cerrar
}

// escuchar espera el siguiente evento sin bloquear el bucle.
func escuchar(canal <-chan Evento) tea.Cmd {
	if canal == nil {
		return nil
	}
	return func() tea.Msg {
		evento, abierto := <-canal
		if !abierto {
			return nil
		}
		return eventoMsg{evento: evento}
	}
}

// aplicarEvento reparte el evento y traduce lo que el chat pinta: los fragmentos
// del streaming, las líneas de herramienta y de sub-proceso y los avisos. Los
// eventos de otra sesión no tocan la vista activa; `notificacion` es la excepción
// (INTERFACES §1). Devuelve el comando que corresponda: arrancar el latido del
// indicador si el turno sigue vivo.
func (a *App) aplicarEvento(evento Evento) tea.Cmd {
	switch evento.Tipo {
	case EventoPeticionAprobacion, EventoAprobacionResuelta:
		// El contador es de todas las sesiones: una que espera decisión no deja de
		// contar porque la activa sea otra (INTERFACES §1).
		a.contarAprobacion(evento)
		return nil
	case EventoCapacidadesModelo:
		// El estado de visión del modelo activo, resuelto por el motor: actualiza
		// el chip del pie de la caja.
		a.entrada.Capacidades.Vision = capacidadDeVista(capacidadDeTexto(evento.Datos["vision"]))
		return nil
	case EventoDescargaProgreso, EventoDescargaTerminada, EventoDescargaFallida:
		// La descarga no es de ninguna sesión: solo el modal que la lanzó la
		// pinta. Con el modal cerrado el evento se descarta.
		a.aplicarDescarga(evento)
		return nil
	}
	// El estado y el título se reflejan en la lista del modal aunque sean de otra
	// sesión: es la lista de todas, y una que trabaja en segundo plano cambia
	// delante del usuario (INTERFACES §1). El modal decide qué hacer con eso: si
	// no está abierto o la sesión no está en su lista, no toca nada.
	switch evento.Tipo {
	case EventoEstadoSesion:
		a.sesiones.ActualizarEstado(evento.Sesion, evento.Datos["estado"])
		// La sesión que termina —bien o mal— deja sus pendientes sin decisión
		// posible: se marcan obsoletas, aunque sea de otra sesión (INTERFACES §1).
		if estado := evento.Datos["estado"]; estado == EstadoTerminada || estado == EstadoError {
			a.aprobaciones.MarcarObsoleta(evento.Sesion)
			a.panel.Aprobaciones = a.aprobaciones.Pendientes()
		}
	case EventoTituloSesion:
		a.sesiones.ActualizarNombre(evento.Sesion, evento.Datos["titulo"])
	}
	if a.sesion == nil {
		return nil
	}
	if evento.Sesion != "" && evento.Sesion != a.sesion.ID && evento.Tipo != EventoNotificacion {
		return nil
	}
	switch evento.Tipo {
	case EventoTituloSesion:
		a.sesion.Nombre = evento.Datos["titulo"]
	case EventoEstadoSesion:
		a.sesion.Estado = evento.Datos["estado"]
		switch a.sesion.Estado {
		case EstadoTrabajando:
			// La sesión sigue trabajando: si la vista no lo estaba contando —al
			// abrir una sesión que ya estaba en marcha—, retoma su contador y su
			// glifo.
			if !a.chat.HayTurno() {
				a.chat.ReanudarTurno()
				return a.arrancarLatido()
			}
		case EstadoTerminada, EstadoError:
			// El turno se cierra: el contador se detiene y el hilo queda con el
			// tiempo que tardó.
			a.chat.CerrarTurno()
			a.olvidarCancelacion()
		case EstadoInactiva:
			// La sesión se detuvo sin terminar: fue una cancelación.
			a.chat.CancelarTurno()
			a.olvidarCancelacion()
		}
	case EventoToken:
		texto := evento.Datos["texto"]
		if texto == "" {
			texto = evento.Datos["content"]
		}
		if evento.Datos["razonamiento"] == "true" {
			a.chat.FragmentoRazonamiento(texto)
			return nil
		}
		a.chat.Fragmento(texto)
	case EventoHerramientaInvocada:
		// La herramienta se interpone en el turno: lo dicho hasta aquí se cierra
		// como un segmento (lo hace el propio chat) y la línea queda debajo, en el
		// orden en que ocurrió.
		a.chat.AnotarInvocacion(evento.Datos["herramienta"], evento.Datos["verbo"], evento.Datos["tema"])
	case EventoHerramientaResultado:
		// La duración viaja en milisegundos; sin ella la línea se pinta sin
		// tiempo, sin dejar hueco (INTERFACES §1.1). El bloque —solo `diagram`—
		// es el diagrama ya renderizado: se coloca bajo la línea como bloque
		// adicional (EVENTS §3).
		duracion := time.Duration(-1)
		if ms, err := enteroDe(evento.Datos["duracion"]); err == nil {
			duracion = time.Duration(ms) * time.Millisecond
		}
		a.chat.CerrarHerramienta(evento.Datos["herramienta"], evento.Datos["ok"] == "true",
			evento.Datos["truncado"] == "true", evento.Datos["medida"], evento.Datos["error"], duracion,
			evento.Datos["bloque"])
	case EventoEtapaIniciada:
		// Cada etapa que empieza deja una sola línea con su nombre.
		a.chat.AnotarEtapa(nombreDeEtapa(evento.Datos))
	case EventoEtapaTerminada:
		// La etapa cierra su línea; el resumen, si lo trae, se anota aparte.
		a.chat.AnotarAviso("etapa terminada: " + nombreDeEtapa(evento.Datos))
	case EventoEtapaFallida:
		a.chat.AnotarAviso("etapa fallida: " + nombreDeEtapa(evento.Datos))
	case EventoFlujoPausado:
		a.chat.AnotarAviso("flujo en pausa: esperando tu decisión")
	case EventoFlujoReanudado:
		a.chat.AnotarAviso("flujo reanudado")
	case EventoFlujoCancelado:
		a.chat.AnotarAviso("flujo cancelado")
	case EventoNotificacion:
		a.notificacionBajoInput = notificacionEnTexto(evento.Datos)
	case EventoCambioAplicado:
		a.chat.AnotarAviso("cambio aplicado en " + evento.Datos["archivo"])
		// El cambio ya está registrado en `change_history`: el dato de git del
		// panel suma un cambio al árbol, sin volver a llamar a git.
		a.panel.GitCambios++
	case EventoTokensTurno:
		// El total del contexto del chat y su límite alimentan la fila CONTEXTO del
		// panel; el consumo del turno, el conteo que va bajo la caja de entrada. El
		// contexto es una estimación sobre los mensajes de la sesión —no hay
		// tokenizador exacto por mensaje— y se marca como tal.
		if n, err := enteroDe(evento.Datos["contexto"]); err == nil {
			a.panel.ContextoTokens = n
			a.panel.TokensEstimados = true
		}
		if n, err := enteroDe(evento.Datos["limite"]); err == nil {
			a.panel.LimiteTokens = n
		}
		// El límite puede ser el tope del harness cuando el motor no declara su
		// ventana: el evento lo dice para que el panel no lo haga pasar por la
		// ventana del modelo (SPEC-MODELO-MOTOR §La ventana de contexto).
		a.panel.LimiteDelHarness = evento.Datos["limite_harness"] == "true"
		if n, err := enteroDe(evento.Datos["salida"]); err == nil {
			a.entrada.Tokens = n
		}
	case EventoTodoActualizada:
		// La lista de pasos es de la sesión activa, entera y en un solo evento: la
		// de otra sesión no entra al panel.
		if tareas := tareasDeEvento(evento.Datos["elementos"]); tareas != nil {
			a.panel.Tareas = tareas
		}
	case EventoColaActualizada:
		// La cola es del proyecto, no de una sesión: se ve desde cualquiera.
		if capa := evento.Datos["capa"]; capa != "" {
			a.panel.Capa = capa
		}
		if activa := evento.Datos["activa"]; activa != "" {
			a.panel.ElementoActual = activa
		}
		if n, err := enteroDe(evento.Datos["pendientes"]); err == nil {
			a.panel.ElementosRestantes = n
		}
	}
	return nil
}

// aplicarDescarga vuelca un evento de descarga en el modal de descarga. Si el
// modal no está abierto se descarta: una descarga terminada con la pantalla en
// otra parte no tiene dónde pintarse.
func (a *App) aplicarDescarga(evento Evento) {
	if a.modal != ModalDescargar {
		return
	}
	switch evento.Tipo {
	case EventoDescargaProgreso:
		hecho, err := entero64De(evento.Datos["hecho"])
		if err != nil {
			return
		}
		total, _ := entero64De(evento.Datos["total"])
		a.descarga.Progreso(hecho, total)
	case EventoDescargaTerminada:
		a.descarga.Terminada(evento.Datos["nombre"])
	case EventoDescargaFallida:
		a.descarga.Fallida(evento.Datos["error"])
	}
}

// contarAprobacion mantiene el panel de aprobaciones pendientes con lo que llega
// por el bus: la petición añade la línea y su resolución la retira. El número que
// el panel pinta en su fila ESTADO sale de las líneas que aún esperan decisión.
func (a *App) contarAprobacion(evento Evento) {
	switch evento.Tipo {
	case EventoPeticionAprobacion:
		a.aprobaciones.Anotar(Aprobacion{
			ID:     evento.Datos["aprobacion"],
			Sesion: evento.Sesion,
			Accion: evento.Datos["descripcion"],
		})
	case EventoAprobacionResuelta:
		a.aprobaciones.Retirar(evento.Datos["aprobacion"])
	}
	a.panel.Aprobaciones = a.aprobaciones.Pendientes()
	a.sincronizarFocoAprobaciones()
}

// tareaDeEvento es la forma mínima de un paso en `todo_actualizada`: lo que el
// panel necesita para pintarlo. No lleva prioridad porque no se muestra.
type tareaDeEvento struct {
	Contenido string `json:"contenido"`
	Estado    string `json:"estado"`
}

// tareasDeEvento lee la lista completa de pasos que viaja en el evento. Devuelve
// nil si no hay lista o si no se puede leer: quien llama se queda entonces con lo
// que ya tenía, en vez de vaciar el panel por un dato ilegible.
func tareasDeEvento(elementos string) []panel.Tarea {
	if elementos == "" {
		return nil
	}
	var vista []tareaDeEvento
	if err := json.Unmarshal([]byte(elementos), &vista); err != nil {
		return nil
	}
	tareas := make([]panel.Tarea, len(vista))
	for i, t := range vista {
		tareas[i] = panel.Tarea{Contenido: t.Contenido, Estado: t.Estado}
	}
	return tareas
}

// nombreDeEtapa lee el nombre visible de la etapa del payload del evento.
func nombreDeEtapa(datos map[string]string) string {
	if nombre := datos["nombre"]; nombre != "" {
		return nombre
	}
	return datos["etapa"]
}

// notificacionEnTexto compone la línea del aviso con su motivo y qué hacer a
// continuación: el contenido no viaja en el evento, que ya está en su sitio.
func notificacionEnTexto(datos map[string]string) string {
	texto := datos["motivo"]
	if siguiente := datos["siguiente"]; siguiente != "" {
		texto += " — " + siguiente
	}
	return texto
}

// enteroDe lee un número que viaja como texto en el payload del evento.
func enteroDe(s string) (int, error) {
	var n int
	if s == "" {
		return 0, fmt.Errorf("tui: sin número")
	}
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

// entero64De lee un entero de 64 bits del payload del evento (tamaños de
// descarga).
func entero64De(s string) (int64, error) {
	if s == "" {
		return 0, fmt.Errorf("tui: sin número")
	}
	return strconv.ParseInt(s, 10, 64)
}
