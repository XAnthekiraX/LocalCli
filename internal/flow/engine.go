package flow

// engine.go — T-B010-07 y T-B010-08: el motor.
//
// Fuente de verdad: ai/docs/backend/04-infrastructure/EVENTS.md §1 y §3 (los
// eventos del motor y sus payloads) y
// ai/docs/backend/01-domain/BUSINESS_RULES.md §Motor de etapas y §Cola ("se
// ejecuta un elemento del TODO por iteración, nunca varios a la vez").
//
// El motor encadena etapas, decide seguir/parar/esperar tras cada una y emite
// el evento que corresponde. No habla con el modelo ni con las herramientas por
// su cuenta: pide contexto y ejecución a través de sus dependencias (`context`
// y `agent`), que se conectan como interfaces para poder probar el motor con
// stubs.

import (
	"context"
	"fmt"
	"strings"

	"localcli/internal/task"
	"localcli/internal/tools"
)

// Nombres de los eventos que emite el motor (EVENTS.md §1). No se inventa
// ninguno: lo que no está aquí, no se emite.
const (
	EventoEtapaIniciada  = "etapa_iniciada"
	EventoEtapaTerminada = "etapa_terminada"
	EventoEtapaFallida   = "etapa_fallida"
	EventoFlujoPausado   = "flujo_pausado"
	EventoFlujoReanudado = "flujo_reanudado"
	EventoFlujoCancelado = "flujo_cancelado"
)

// Evento es una notificación del motor. El payload lleva lo mínimo para pintar
// o decidir (EVENTS.md §3).
type Evento struct {
	Nombre string
	Datos  map[string]string
}

// Emisor publica eventos. Un consumidor lento no bloquea al productor: quien lo
// implemente decide cómo (un canal con cola).
type Emisor interface {
	Emitir(Evento)
}

// EmisorFunc adapta una función a Emisor.
type EmisorFunc func(Evento)

func (f EmisorFunc) Emitir(e Evento) { f(e) }

// Resultado es lo que produce una etapa.
type Resultado struct {
	Texto       string
	Herramienta string
}

// Roles de un mensaje de conversación. Son los literales que entiende el modelo
// (`ollama`), expuestos aquí para que `session` no dependa de `ollama`.
const (
	RolSistema   = "system"
	RolUsuario   = "user"
	RolAsistente = "assistant"
)

// Mensaje es un turno de la conversación que se le entrega al modelo. `flow` no
// interpreta su contenido: solo lo transporta desde `session` hasta `agent`.
type Mensaje struct {
	Rol   string
	Texto string
}

// Contexto entrega a una etapa solo el contexto que necesita. Lo implementa
// `context` (T-B011); el motor no lo arma por su cuenta. La etapa viaja con la
// petición: es lo que permite que `context_audit` distinga una etapa de otra
// (una fila por documento y etapa; TABLES.md §3).
type Contexto interface {
	ContextoPara(ctx context.Context, etapa, objetivo string) (string, error)
}

// EtapaChat es la etapa bajo la que se arma y se audita el contexto de una
// respuesta del chat: no es una etapa de flujo, pero el nodo de contexto
// necesita saber que no vino de una. `context` mantiene el mismo literal
// (context.EtapaChat) para decidir su fallback; tests de integración de ambas
// capas comprueban que los dos literales coinciden.
const EtapaChat = "chat"

// PeticionEtapa es lo que el motor le pide a un agente para correr una etapa (o
// el chat). Un solo contrato para los dos caminos: `Silenciosa` distingue una
// etapa intermedia de un flujo —cuyo texto no se muestra ni se persiste, solo
// alimenta la cadena— de la última etapa y del chat. `Etapa` es el nombre de la
// etapa (vacío en el chat); la vista lo usa en su línea de progreso.
// `SinHerramientas` corre el turno sin presentar herramientas al modelo: lo usa
// la etapa de composición, que solo redacta la entrega a partir del bloque.
type PeticionEtapa struct {
	Agente          string
	Contexto        string
	Historial       []Mensaje
	Imagenes        []string
	Etapa           string
	Silenciosa      bool
	SinHerramientas bool
}

// Agente ejecuta una etapa con el agente indicado (`plan` o `build`) sobre el
// contexto que recibe, precedido del historial de conversación de la sesión
// (vacío en una etapa de flujo; completo o compactado en el chat). Lo implementa
// `agent` (T-B006). No ejecuta herramientas aquí: eso es cosa de `agent` →
// `tools`.
type Agente interface {
	Ejecutar(ctx context.Context, p PeticionEtapa) (Resultado, error)
}

// Aprobador pide la decisión del usuario para una etapa que la requiere.
type Aprobador interface {
	Aprobar(ctx context.Context, descripcion string) (bool, error)
}

// Registro guarda en el hilo del chat las líneas de procesamiento que la vista
// muestra —el sub-proceso de una etapa— para que sobrevivan al cambio de
// sesión. Es opcional: sin él, la línea solo se ve en vivo. Lo implementa el
// arranque; un fallo al registrar no detiene el flujo: es un dato de pantalla,
// no una regla de negocio.
type Registro interface {
	ProcesoEtapa(nombre string, fallida bool)
}

// Motor encadena las etapas de un flujo.
type Motor struct {
	Contexto Contexto
	Agente   Agente
	// Aprobador es necesario solo si el flujo tiene etapas con Aprobacion.
	Aprobador Aprobador
	// Eventos es opcional: sin emisor, el motor trabaja igual.
	Eventos Emisor
	// Bloque y Optimizador son opcionales: sin ellos, un flujo con
	// `BloqueContexto` cae al encadenado de resúmenes mecánicos y el bloque vive
	// solo en memoria. Con ellos se activa el pipeline del bloque de contexto
	// (persistencia + optimización por etapa). Ver bloque.go.
	Bloque      Bloque
	Optimizador Optimizador
	// Registro es opcional: con él, el sub-proceso de cada etapa queda guardado
	// en el hilo del chat además de verse en vivo.
	Registro Registro
}

func (m *Motor) emitir(nombre string, datos map[string]string) {
	if m.Eventos == nil {
		return
	}
	m.Eventos.Emitir(Evento{Nombre: nombre, Datos: datos})
}

// registrarProceso guarda la línea de procesamiento de una etapa. Sin registro
// conectado no hace nada; un registro que falla no interrumpe el flujo.
func (m *Motor) registrarProceso(nombre string, fallida bool) {
	if m.Registro == nil {
		return
	}
	m.Registro.ProcesoEtapa(nombre, fallida)
}

// EjecutarFlujo corre las etapas en orden. Devuelve el estado final y, si algo
// falló o se canceló, el error correspondiente (E_STAGE_FAILED o
// E_FLOW_CANCELLED). Un flujo con una etapa que falla NO encadena la siguiente
// (BUSINESS_RULES.md §Invariantes).
func (m *Motor) EjecutarFlujo(ctx context.Context, f Flujo, objetivo string) (EstadoFlujo, error) {
	if err := f.Validar(); err != nil {
		return EstadoConError, err
	}
	if m.Contexto == nil || m.Agente == nil {
		return EstadoConError, fmt.Errorf("flow: el motor no tiene contexto y agente conectados")
	}

	estado := EstadoPendiente
	var err error
	if estado, err = EstadoTrasDecision(estado, Seguir); err != nil {
		return EstadoConError, err
	}

	// `acumulado` guarda el resumen corto de cada etapa ya corrida. Se inyecta en
	// el contexto de la etapa siguiente: es el encadenado de SPEC-MOTOR-FLUJOS
	// ("el resultado pasa a la etapa siguiente"), sin arrastrar la salida
	// completa. En un flujo con bloque de contexto cada entrada es la aportación
	// ya optimizada por el modelo.
	var acumulado []string

	// Un flujo con bloque de contexto empieza con el bloque vacío: la ejecución
	// nueva no hereda las aportaciones de la anterior.
	if f.BloqueContexto && m.Bloque != nil {
		if err := m.Bloque.Limpiar(ctx, f.Nombre); err != nil {
			return EstadoConError, fmt.Errorf("flow: no se pudo limpiar el bloque del flujo %s: %w", f.Nombre, err)
		}
	}

	for i, etapa := range f.Etapas {
		ultima := i == len(f.Etapas)-1
		// La etapa de composición de un flujo con bloque corre sin herramientas:
		// solo redacta la entrega a partir del bloque, así el cierre es
		// determinista y no depende de que el modelo deje de pedir herramientas.
		composicion := f.BloqueContexto && ultima
		if cerr := ctx.Err(); cerr != nil {
			m.emitir(EventoFlujoCancelado, map[string]string{"flujo": f.Nombre, "etapa": etapa.ID})
			return EstadoDetenido, nuevoError(CodigoFlujoCancelado,
				"el flujo "+f.Nombre+" se canceló en la etapa "+etapa.ID)
		}

		m.emitir(EventoEtapaIniciada, map[string]string{
			"flujo": f.Nombre, "etapa": etapa.ID, "nombre": etapa.Nombre,
		})
		m.registrarProceso(etapa.Nombre, false)

		contexto, cErr := m.Contexto.ContextoPara(ctx, etapa.ID, objetivo)
		if cErr != nil {
			m.emitir(EventoEtapaFallida, map[string]string{"flujo": f.Nombre, "etapa": etapa.ID})
			m.registrarProceso(etapa.Nombre, true)
			return EstadoConError, fmt.Errorf("%w: %s: %v", ErrEtapaFallida, etapa.ID, cErr)
		}
		contexto = componerBrief(f.Reglas, etapa.Instruccion, contexto)
		if composicion {
			// La composición recibe el bloque entero: el estado del turno vive
			// en `store`, no en la memoria del motor.
			if seccion := m.bloqueParaComposicion(ctx, f, acumulado); seccion != "" {
				contexto += "\n\n" + seccion
			}
		} else if len(acumulado) > 0 {
			contexto += "\n\n## Resultados de los pasos anteriores\n" + strings.Join(acumulado, "\n\n")
		}
		// Una etapa intermedia que no pide aprobación corre en silencio: su texto
		// no se muestra en el chat ni se guarda, solo alimenta la cadena. La
		// última etapa y las que piden aprobación se muestran: la entrega final y
		// lo que el usuario debe ver para poder aprobar.
		silenciosa := !ultima && !etapa.Aprobacion
		res, aErr := m.Agente.Ejecutar(ctx, PeticionEtapa{
			Agente:          etapa.Agente,
			Contexto:        contexto,
			Etapa:           etapa.Nombre,
			Silenciosa:      silenciosa,
			SinHerramientas: composicion,
		})
		if aErr != nil {
			m.emitir(EventoEtapaFallida, map[string]string{"flujo": f.Nombre, "etapa": etapa.ID})
			m.registrarProceso(etapa.Nombre, true)
			return EstadoConError, fmt.Errorf("%w: %s: %v", ErrEtapaFallida, etapa.ID, aErr)
		}

		if etapa.Aprobacion {
			// El flujo se pausa y se retoma en la MISMA etapa.
			estado, err = EstadoTrasDecision(estado, EsperarAprobacion)
			if err != nil {
				return EstadoConError, err
			}
			m.emitir(EventoFlujoPausado, map[string]string{"flujo": f.Nombre, "etapa": etapa.ID})
			if m.Aprobador == nil {
				return EstadoPausadoPermiso, nuevoError(CodigoFlujoCancelado,
					"la etapa "+etapa.ID+" necesita aprobación y no hay aprobador conectado")
			}
			ok, pErr := m.Aprobador.Aprobar(ctx, "aprobar la etapa "+etapa.Nombre)
			if pErr != nil {
				return EstadoConError, pErr
			}
			if !ok {
				m.emitir(EventoFlujoCancelado, map[string]string{"flujo": f.Nombre, "etapa": etapa.ID})
				return EstadoDetenido, nuevoError(CodigoFlujoCancelado,
					"el usuario declinó la etapa "+etapa.ID)
			}
			estado, err = EstadoTrasDecision(estado, Seguir)
			if err != nil {
				return EstadoConError, err
			}
			m.emitir(EventoFlujoReanudado, map[string]string{"flujo": f.Nombre, "etapa": etapa.ID})
		}

		if f.BloqueContexto && !ultima {
			// Cada etapa intermedia deja su aportación optimizada en el bloque.
			aportacion := m.optimizarEtapa(ctx, f, etapa, res.Texto)
			if err := m.guardarAportacion(ctx, f, etapa, i, aportacion); err != nil {
				return EstadoConError, err
			}
			acumulado = append(acumulado, bloqueResumen(etapa.Nombre, aportacion))
		} else {
			acumulado = append(acumulado, bloqueResumen(etapa.Nombre, res.Texto))
		}
		m.emitir(EventoEtapaTerminada, map[string]string{
			"flujo":   f.Nombre,
			"etapa":   etapa.ID,
			"nombre":  etapa.Nombre,
			"resumen": res.Texto,
		})
	}

	estado = EstadoTerminado
	return estado, nil
}

// optimizarEtapa condensa el resultado de una etapa con el modelo. Si no hay
// optimizador conectado o la generación falla, cae al recorte mecánico de
// siempre: el flujo nunca se detiene por una optimización que no llegó
// (SPEC-MOTOR-FLUJOS §Reglas de negocio).
func (m *Motor) optimizarEtapa(ctx context.Context, f Flujo, etapa Etapa, texto string) string {
	if m.Optimizador != nil {
		if opt, err := m.Optimizador.Optimizar(ctx, f.Nombre, etapa.Nombre, texto); err == nil {
			if s := strings.TrimSpace(opt); s != "" {
				return s
			}
		}
	}
	rec, _ := tools.Recortar(strings.TrimSpace(texto), limiteTokensResumenEtapa)
	if s := strings.TrimSpace(rec); s == "" {
		return "(sin resultado)"
	}
	return strings.TrimSpace(rec)
}

// guardarAportacion persiste la aportación de una etapa en el bloque. Con un
// bloque conectado, un fallo de escritura detiene el flujo: la entrega final
// depende de que todo lo anterior esté en el bloque.
func (m *Motor) guardarAportacion(ctx context.Context, f Flujo, etapa Etapa, posicion int, contenido string) error {
	if m.Bloque == nil {
		return nil
	}
	if err := m.Bloque.Guardar(ctx, f.Nombre, EntradaBloque{
		Etapa: etapa.ID, Nombre: etapa.Nombre, Posicion: posicion, Contenido: contenido,
	}); err != nil {
		return fmt.Errorf("flow: no se pudo guardar la aportación de la etapa %s: %w", etapa.ID, err)
	}
	return nil
}

// bloqueParaComposicion arma la sección «Bloque de contexto» que recibe la etapa
// de composición. Lee el bloque persistido (todas las etapas anteriores); si no
// hay bloque conectado, compone con lo que quedó en memoria.
func (m *Motor) bloqueParaComposicion(ctx context.Context, f Flujo, acumulado []string) string {
	var entradas []EntradaBloque
	if m.Bloque != nil {
		if leidas, err := m.Bloque.Leer(ctx, f.Nombre); err == nil {
			entradas = leidas
		}
	}
	if len(entradas) > 0 {
		partes := make([]string, 0, len(entradas))
		for _, e := range entradas {
			partes = append(partes, "### "+e.Nombre+"\n"+e.Contenido)
		}
		return "## Bloque de contexto\n" + strings.Join(partes, "\n\n")
	}
	if len(acumulado) > 0 {
		return "## Bloque de contexto\n" + strings.Join(acumulado, "\n\n")
	}
	return ""
}

// Conversar responde una petición como chat: arma el contexto del objetivo y
// lo corre con el agente activo, sin etapas ni aprobaciones. Es el camino por
// defecto de la vista principal (SPEC-INTERFAZ §Reglas: "La vista principal es
// un chat… Ningún texto arranca un flujo de trabajo por sí solo"); el flujo
// solo arranca con un comando explícito.
//
// El agente llega como argumento porque lo elige el usuario con el indicador de
// la TUI: cada agente responde con las herramientas declaradas en su JSON
// (SPEC-TOOLS §Reglas). El motor no inventa un catálogo ni decide el agente.
//
// `historial` es la conversación anterior de la sesión, tal como la arma
// `session` (completa o compactada). El motor no la interpreta: la pasa al
// agente junto con el contexto del turno. `imagenes` son las imágenes (base64)
// de ESTE turno; solo el chat las lleva —las etapas van sin imágenes—.
func (m *Motor) Conversar(ctx context.Context, agente, objetivo string, historial []Mensaje, imagenes []string) (Resultado, error) {
	// El motor no decide qué agentes existen: los carga el arranque de
	// `.localcli/agents/*.json`. Solo exige que la petición nombre uno; el ejecutor
	// rechaza un nombre desconocido al correr el turno.
	if strings.TrimSpace(agente) == "" {
		return Resultado{}, fmt.Errorf("flow: el chat necesita un agente")
	}
	if m.Contexto == nil || m.Agente == nil {
		return Resultado{}, fmt.Errorf("flow: el motor no tiene contexto y agente conectados")
	}
	contexto, err := m.Contexto.ContextoPara(ctx, EtapaChat, objetivo)
	if err != nil {
		return Resultado{}, err
	}
	return m.Agente.Ejecutar(ctx, PeticionEtapa{
		Agente:    agente,
		Contexto:  contexto,
		Historial: historial,
		Imagenes:  imagenes,
	})
}

// ElementoCola es un elemento del TODO listo para ejecutarse.
type ElementoCola struct {
	ID       string
	Objetivo string
}

// Cola es lo que el motor necesita para consumir el TODO. Lo implementa `queue`
// (T-B012), que deriva la cola de los archivos de tarea.
type Cola interface {
	// Siguiente devuelve el siguiente elemento elegible, o nil si la cola está
	// vacía.
	Siguiente(ctx context.Context) (*ElementoCola, error)
	// Marcar cambia el estado del elemento en su archivo de tarea.
	Marcar(id string, estado task.Estado) error
}

// ConsumirCola ejecuta la cola entera, un elemento por iteración. Cada elemento
// se marca en_progreso, se ejecuta como un flujo y se marca completada; nunca
// hay dos en_progreso a la vez. Al vaciarse, se detiene.
func (m *Motor) ConsumirCola(ctx context.Context, cola Cola, f Flujo) error {
	if cola == nil {
		return fmt.Errorf("flow: no hay cola que consumir")
	}
	enProgreso := ""
	for {
		if err := ctx.Err(); err != nil {
			return nuevoError(CodigoFlujoCancelado, "la cola se canceló")
		}
		elem, err := cola.Siguiente(ctx)
		if err != nil {
			return err
		}
		if elem == nil {
			return nil // cola vacía: se detiene sola
		}
		// Invariante de un elemento por iteración: no se arranca otro mientras
		// el anterior siga en progreso.
		if enProgreso != "" {
			return fmt.Errorf("flow: ya hay un elemento en progreso (%s); se ejecuta uno por iteración", enProgreso)
		}
		if err := cola.Marcar(elem.ID, task.EstadoEnProgreso); err != nil {
			return err
		}
		enProgreso = elem.ID

		if _, err := m.EjecutarFlujo(ctx, f, elem.Objetivo); err != nil {
			// El elemento queda bloqueado con lo que falta; la cola no sigue
			// con él, pero tampoco se marca completado.
			_ = cola.Marcar(elem.ID, task.EstadoBloqueada)
			return err
		}
		if err := cola.Marcar(elem.ID, task.EstadoCompletada); err != nil {
			return err
		}
		enProgreso = ""
	}
}

// limiteTokensResumenEtapa acota el resumen de una etapa antes de encadenarlo a
// la siguiente. Es corto a propósito: cada etapa recibe solo lo que necesita, no
// todo lo que produjo la anterior (SPEC-MOTOR-FLUJOS §Reglas de negocio).
const limiteTokensResumenEtapa = 400

// bloqueResumen compone el bloque que una etapa deja a la siguiente: su nombre y
// su resultado, recortado al límite. Lo usan el encadenado y el evento
// `etapa_terminada`.
func bloqueResumen(nombre, texto string) string {
	texto, _ = tools.Recortar(strings.TrimSpace(texto), limiteTokensResumenEtapa)
	if texto == "" {
		texto = "(sin resultado)"
	}
	return "### " + nombre + "\n" + texto
}

// componerBrief antepone las reglas del flujo y la instrucción de la etapa al
// contexto que recibe el agente. Es cómo un flujo declara sus reglas —por
// ejemplo, cómo descubrir la documentación— sin tocar el código: el brief viaja
// dentro del texto que ya recibe la etapa, sin cambiar las interfaces.
func componerBrief(reglas []string, instruccion, contexto string) string {
	var partes []string
	if len(reglas) > 0 {
		partes = append(partes, "Reglas del flujo:\n- "+strings.Join(reglas, "\n- "))
	}
	if instruccion != "" {
		partes = append(partes, instruccion)
	}
	if len(partes) == 0 {
		return contexto
	}
	brief := strings.Join(partes, "\n\n")
	if contexto == "" {
		return brief
	}
	return brief + "\n\n" + contexto
}
