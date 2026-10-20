package session

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"localcli/internal/agent"
	"localcli/internal/flow"
	"localcli/internal/linea"
	"localcli/internal/llm"
	"localcli/internal/store"
	"localcli/internal/tools"
	"localcli/internal/tui"
)

// prefijoFlujo es la marca con la que el usuario invoca un flujo explícito
// (SPEC-INTERFAZ §Sintaxis de comandos slash): `/flow:<comando>`. El comando del
// flujo ya no arranca un flujo por sí solo —el `/comando` crudo queda para los
// comandos de la interfaz—.
const prefijoFlujo = "/flow:"

// eleccionFlujo es lo que resulta de mirar una línea: no es un flujo, es un
// flujo con su petición, o es un flujo al que le falta la petición.
type eleccionFlujo struct {
	flujo       flow.Flujo
	peticion    string
	esFlujo     bool
	sinPeticion bool
}

// elegirFlujo reconoce un flujo al principio de una línea con la forma
// `/flow:<comando>`. La petición —el resto de la línea— es lo que el flujo debe
// resolver: sin ella no hay nada que correr y se pide. La única excepción es la
// cola (`/ejecutar`), que trae el objetivo de cada elemento. Un `/comando` sin el
// prefijo `flow:` no arranca un flujo.
func (o *Orquestador) elegirFlujo(linea string) eleccionFlujo {
	if o.flows == nil {
		return eleccionFlujo{}
	}
	recortada := strings.TrimSpace(linea)
	if !strings.HasPrefix(recortada, prefijoFlujo) {
		return eleccionFlujo{}
	}
	resto := strings.TrimSpace(strings.TrimPrefix(recortada, prefijoFlujo))
	nombre, objetivo, _ := strings.Cut(resto, " ")
	nombre = strings.TrimSpace(nombre)
	if nombre == "" {
		return eleccionFlujo{}
	}
	f, ok := o.flows.PorComando("/" + nombre)
	if !ok {
		return eleccionFlujo{}
	}
	objetivo = strings.TrimSpace(objetivo)
	if objetivo == "" && f.Comando != ComandoEjecutar {
		return eleccionFlujo{flujo: f, esFlujo: true, sinPeticion: true}
	}
	return eleccionFlujo{flujo: f, peticion: objetivo, esFlujo: true}
}

// correrFlujo ejecuta todas las etapas de un flujo como sub-proceso del turno:
// compone la ventana de cada etapa, pide su aprobación antes de correrla, corre
// su agente y publica los eventos. Todas las etapas corren con el agente activo
// de la sesión y ven el historial con el que se invocó el flujo, junto a la
// petición. Devuelve la entrega —lo que deja la última etapa— y el bloque
// resumido, que es lo único que cruza hacia quien lo invocó.
func (o *Orquestador) correrFlujo(ctx context.Context, sesionID string, agenteActivo agent.Agente, f flow.Flujo, peticion string, historial []agent.Mensaje, alDelta func(agent.Delta) error) (string, string, error) {
	if err := store.VaciarFlowContext(o.db, o.projectID, sesionID, f.Comando); err != nil {
		return "", "", err
	}

	entrega := ""
	for i, etapa := range f.Etapas {
		ultima := i == len(f.Etapas)-1
		if etapa.Aprobacion {
			// La aprobación se pide ANTES de correr la etapa
			// (SPEC-ORQUESTADOR-FLUJOS §Flujo principal): hasta que el
			// usuario no aprueba, la etapa no llega a ejecutarse.
			ok, err := o.pausarEtapa(ctx, sesionID, "aprobar la etapa "+etapa.Nombre)
			if err != nil {
				return "", "", err
			}
			if !ok {
				o.emitir(tui.Evento{Sesion: sesionID, Tipo: tui.EventoFlujoCancelado})
				return "", "", nil
			}
		}
		o.emitir(tui.Evento{Sesion: sesionID, Tipo: tui.EventoEtapaIniciada, Datos: map[string]string{
			"etapa": etapa.ID, "nombre": etapa.Nombre,
		}})
		o.guardarEventoChat(sesionID, store.ChatEventoProceso, linea.Proceso(etapa.Nombre), nil, store.Bloque{})

		texto, err := o.correrEtapa(ctx, sesionID, agenteActivo, f, etapa, peticion, historial, ultima, alDelta)
		if err != nil {
			o.emitir(tui.Evento{Sesion: sesionID, Tipo: tui.EventoEtapaFallida, Datos: map[string]string{
				"etapa": etapa.ID, "nombre": etapa.Nombre, "motivo": err.Error(),
			}})
			return "", "", err
		}
		texto = strings.TrimSpace(texto)
		if texto == "" {
			o.emitir(tui.Evento{Sesion: sesionID, Tipo: tui.EventoEtapaFallida, Datos: map[string]string{
				"etapa": etapa.ID, "nombre": etapa.Nombre, "motivo": "sin respuesta",
			}})
			return "", "", fmt.Errorf("flujo: la etapa %s no entregó texto", etapa.ID)
		}

		if err := store.GuardarFlowContext(o.db, o.projectID, sesionID, f.Comando, etapa.ID, etapa.Nombre, i, texto); err != nil {
			return "", "", err
		}
		if etapa.RespuestaEnChat && !yaViajaAlTurno(ultima, alDelta) {
			// La respuesta se pinta como bloque del hilo y se guarda en
			// `chat_evento`, nunca en el contexto del modelo
			// (SPEC-ORQUESTADOR-FLUJOS §Visibilidad de una respuesta).
			o.pintarRespuestaEtapa(sesionID, texto)
		}
		o.emitir(tui.Evento{Sesion: sesionID, Tipo: tui.EventoEtapaTerminada, Datos: map[string]string{
			"etapa": etapa.ID, "nombre": etapa.Nombre, "resumen": resumenCorto(texto),
		}})

		if ultima {
			entrega = texto
		}
	}

	bloque, err := o.bloqueResumido(sesionID, f.Comando)
	if err != nil {
		return "", "", err
	}
	return entrega, bloque, nil
}

// yaViajaAlTurno dice si la salida de una etapa se transmite token a token al
// chat: es la última etapa del flujo y quien la corrió pidió avance en vivo. En
// ese caso la respuesta ya aparece en el hilo como la respuesta del turno, así
// que no se vuelve a pintar ni a guardar como bloque.
func yaViajaAlTurno(ultima bool, alDelta func(agent.Delta) error) bool {
	return ultima && alDelta != nil
}

// correrEtapa corre una etapa: una vez si es `linear`, o en bucle hasta que
// responda `true` si es `loop`. Solo la última etapa del flujo —la entrega—
// transmite su salida token a token al chat.
func (o *Orquestador) correrEtapa(ctx context.Context, sesionID string, agenteActivo agent.Agente, f flow.Flujo, etapa flow.Etapa, peticion string, historial []agent.Mensaje, ultima bool, alDelta func(agent.Delta) error) (string, error) {
	if etapa.EsLoop() {
		return o.correrEtapaLoop(ctx, sesionID, agenteActivo, f, etapa, peticion, historial, ultima, alDelta)
	}

	delta := func(agent.Delta) error { return nil }
	if yaViajaAlTurno(ultima, alDelta) {
		delta = alDelta
	}
	ventana, err := o.ventanaDeEtapa(sesionID, f, etapa, peticion, historial, nil)
	if err != nil {
		return "", err
	}
	respuesta, err := o.turnoEtapa(ctx, sesionID, agenteActivo, etapa, ventana, delta)
	if err != nil {
		return "", err
	}
	return respuesta.Texto, nil
}

// correrEtapaLoop repite la etapa hasta que el agente cierre con `true` o se
// agote `max_iteraciones`. Los intentos fallidos no se guardan ni se muestran al
// chat: solo alimentan la ventana del intento siguiente. El último acierto es la
// contribución que recibe la etapa siguiente.
func (o *Orquestador) correrEtapaLoop(ctx context.Context, sesionID string, agenteActivo agent.Agente, f flow.Flujo, etapa flow.Etapa, peticion string, historial []agent.Mensaje, ultima bool, alDelta func(agent.Delta) error) (string, error) {
	var intentos []string
	var diferidos []agent.Delta
	delta := func(agent.Delta) error { return nil }
	if ultima && alDelta != nil {
		// La entrega de un bucle se transmite solo cuando queda resuelta: lo que
		// se va generando se guarda y se reproduce al cerrar.
		delta = func(d agent.Delta) error { diferidos = append(diferidos, d); return nil }
	}

	for n := 1; n <= etapa.MaxIteraciones; n++ {
		diferidos = nil
		ventana, err := o.ventanaDeEtapa(sesionID, f, etapa, peticion, historial, intentos)
		if err != nil {
			return "", err
		}
		respuesta, err := o.turnoEtapa(ctx, sesionID, agenteActivo, etapa, ventana, delta)
		if err != nil {
			return "", err
		}
		resumen, listo, ok := partirVeredicto(respuesta.Texto)
		if !ok {
			intentos = append(intentos, "Intento "+strconv.Itoa(n)+": no cerró con la línea `true` o `false`.")
			continue
		}
		if listo {
			for _, d := range diferidos {
				_ = alDelta(d)
			}
			return textoDelLoop(etapa, resumen, n), nil
		}
		intentos = append(intentos, intentoFallido(n, resumen))
	}

	return "", fmt.Errorf("flujo: la etapa %q no respondió `true` en %d intentos", etapa.ID, etapa.MaxIteraciones)
}

// partirVeredicto separa el resumen de trabajo de la última línea, que debe ser
// `true` (resuelto) o `false` (hay que repetir). Acepta comillas invertidas y
// mayúsculas: `True`, ```“true```“.
func partirVeredicto(texto string) (resumen string, listo bool, ok bool) {
	lineas := strings.Split(texto, "\n")
	i := len(lineas) - 1
	for i >= 0 && strings.TrimSpace(lineas[i]) == "" {
		i--
	}
	if i < 0 {
		return "", false, false
	}
	veredicto := strings.Trim(strings.TrimSpace(lineas[i]), "`")
	switch strings.ToLower(strings.TrimSpace(veredicto)) {
	case "true":
		listo = true
	case "false":
		listo = false
	default:
		return "", false, false
	}
	return strings.TrimSpace(strings.Join(lineas[:i], "\n")), listo, true
}

// intentoFallido resume en una línea un intento que no cerró el bucle.
func intentoFallido(n int, resumen string) string {
	if resumen == "" {
		return "Intento " + strconv.Itoa(n) + " (`false`): volvió a fallar."
	}
	return "Intento " + strconv.Itoa(n) + " (`false`): " + resumen
}

// textoDelLoop es la contribución que deja una etapa `loop`: el resumen del
// intento que la resolvió, o una línea que deja constancia de que se resolvió.
func textoDelLoop(etapa flow.Etapa, resumen string, n int) string {
	if resumen != "" {
		return resumen
	}
	return "«" + etapa.Nombre + "» se resolvió tras " + strconv.Itoa(n) + " intento(s)."
}

// ventanaDeEtapa compone el texto que ve la etapa: el historial con el que se
// invocó el flujo, la petición original, las reglas del flujo, la contribución de
// las etapas anteriores, los intentos previos de un bucle y su propio orden.
func (o *Orquestador) ventanaDeEtapa(sesionID string, f flow.Flujo, etapa flow.Etapa, peticion string, historial []agent.Mensaje, intentos []string) ([]agent.Mensaje, error) {
	previos, err := store.ListarFlowContext(o.db, o.projectID, sesionID, f.Comando)
	if err != nil {
		return nil, err
	}

	var b strings.Builder
	b.WriteString("Petición original: ")
	b.WriteString(peticion)
	b.WriteString("\n")
	if len(f.Reglas) > 0 {
		b.WriteString("\nReglas:\n")
		for _, r := range f.Reglas {
			b.WriteString("- ")
			b.WriteString(r)
			b.WriteString("\n")
		}
	}
	for _, p := range previos {
		b.WriteString("\n## ")
		b.WriteString(p.Nombre)
		b.WriteString("\n")
		b.WriteString(p.Contenido)
		b.WriteString("\n")
	}
	if len(intentos) > 0 {
		b.WriteString("\n## Intentos anteriores de ")
		b.WriteString(etapa.Nombre)
		b.WriteString("\n")
		for _, it := range intentos {
			b.WriteString(it)
			b.WriteString("\n")
		}
	}
	b.WriteString("\n## ")
	b.WriteString(etapa.Nombre)
	b.WriteString("\nOrden: ")
	b.WriteString(etapa.Orden)
	if etapa.EsLoop() {
		b.WriteString("\nCierra SIEMPRE con una única línea: `true` si el trabajo ya está resuelto, `false` si hay que volver a intentarlo. Lo que escribas antes de esa línea es el resumen de tu trabajo.")
	} else if etapa.Respuesta != "" {
		b.WriteString("\nSalida requerida: ")
		b.WriteString(etapa.Respuesta)
	}

	mensajes := make([]agent.Mensaje, 0, len(historial)+1)
	mensajes = append(mensajes, historial...)
	mensajes = append(mensajes, agent.Mensaje{Rol: agent.RolUsuario, Contenido: b.String()})
	return mensajes, nil
}

// turnoEtapa corre el turno de una etapa con el agente activo de la sesión y sus
// permisos. Ninguna etapa ve `ejecutar_flujo` (profundidad uno). Si el agente
// intenta una herramienta que no tiene concedida, la etapa se aborta con un
// error que explica qué permiso falta.
func (o *Orquestador) turnoEtapa(ctx context.Context, sesionID string, agenteActivo agent.Agente, etapa flow.Etapa, ventana []agent.Mensaje, alDelta func(agent.Delta) error) (agent.Respuesta, error) {
	ejecutor := &agent.Ejecutor{
		Agente:    agenteActivo,
		Motor:     motorOpenAI{cliente: o.llm, modelo: o.modeloActual, sinRazonamiento: func() bool { return !o.pensarActual() }},
		Capacidad: llm.Soportada,
		Ventana:   o.ventanaActual(),
		Avisar:    func(motivo, siguiente string) { o.avisar(sesionID, motivo, siguiente) },
	}
	denegada := ""
	if o.herramientas != nil {
		ejecutor.Cerrado = agent.CatalogoEfectivo(agenteActivo, o.catalogoHerramientas)
		ejecutor.Herramientas = adaptadorHerramientas{
			registro: o.herramientas,
			sesion:   sesionID,
			permisos: agenteActivo.Permisos(),
			enFlujo:  true,
			Observar: func(nombre string, res agent.Resultado) {
				if denegada == "" && strings.HasPrefix(res.Error, tools.CodToolNotAllowed) {
					denegada = nombre
				}
			},
			Ask: func(ctx context.Context, s tools.Solicitud) (tools.Decision, error) {
				return o.pedirAprobacion(ctx, sesionID, s)
			},
		}
	}
	respuesta, err := ejecutor.Turno(ctx, ventana, alDelta)
	if err != nil {
		return respuesta, err
	}
	if denegada != "" {
		return respuesta, errPermisoEtapa{etapa: etapa.Nombre, herramienta: denegada, agente: agenteActivo.Nombre}
	}
	return respuesta, nil
}

// errPermisoEtapa explica que una etapa no puede continuar porque el agente
// activo no tiene concedida la herramienta que necesita.
type errPermisoEtapa struct {
	etapa       string
	herramienta string
	agente      string
}

func (e errPermisoEtapa) Error() string {
	return "la etapa «" + e.etapa + "» necesita la herramienta «" + e.herramienta +
		"», que el agente «" + e.agente + "» no tiene concedida; cambia de agente (Tab) o concede el permiso y vuelve a lanzar el flujo"
}

// pausarEtapa detiene el flujo hasta que el usuario decide: aprobar continúa.
func (o *Orquestador) pausarEtapa(ctx context.Context, sesionID, titulo string) (bool, error) {
	o.emitir(tui.Evento{Sesion: sesionID, Tipo: tui.EventoFlujoPausado})
	decision, err := o.pedirAprobacion(ctx, sesionID, tools.Solicitud{Titulo: titulo})
	if err != nil {
		return false, err
	}
	if decision != tools.Aprobar {
		return false, nil
	}
	o.emitir(tui.Evento{Sesion: sesionID, Tipo: tui.EventoFlujoReanudado})
	return true, nil
}

// bloqueResumido junta la contribución de cada etapa del bloque.
func (o *Orquestador) bloqueResumido(sesionID, flujo string) (string, error) {
	previos, err := store.ListarFlowContext(o.db, o.projectID, sesionID, flujo)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for i, p := range previos {
		if i > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString("## ")
		b.WriteString(p.Nombre)
		b.WriteString("\n")
		b.WriteString(p.Contenido)
	}
	return b.String(), nil
}

// ejecutarFlujo es el handler de la herramienta `ejecutar_flujo`: resuelve el
// flujo pedido y lo corre como sub-proceso con el agente que la invocó. Los
// flujos ya no se clasifican por permisos: si una etapa necesita una herramienta
// que el agente no tiene, la etapa falla y lo dice.
func (o *Orquestador) ejecutarFlujo(ctx context.Context, args any, c tools.Contexto) (tools.Resultado, error) {
	peticion, ok := args.(tools.PeticionEjecutarFlujo)
	if !ok {
		return tools.Resultado{Error: tools.CodBadArgs + ": argumentos inesperados"}, nil
	}
	if o.flows == nil {
		return tools.Resultado{Error: tools.CodFlowUnknown + ": el flujo `" + peticion.Flujo + "` no existe"}, nil
	}
	f, ok := o.flows.PorComando(peticion.Flujo)
	if !ok {
		return tools.Resultado{Error: tools.CodFlowUnknown + ": el flujo `" + peticion.Flujo + "` no existe"}, nil
	}
	agente, ok := o.agentes.Buscar(c.Agente)
	if !ok {
		return tools.Resultado{Error: tools.CodFlowUnknown + ": el agente `" + c.Agente + "` no existe"}, nil
	}

	entrega, bloque, err := o.correrFlujo(ctx, c.SesionID, agente, f, strings.TrimSpace(peticion.Peticion), nil, nil)
	if err != nil {
		var falta errPermisoEtapa
		if errors.As(err, &falta) {
			return tools.Resultado{Error: tools.CodFlowNotAllowed + ": " + falta.Error()}, nil
		}
		return tools.Resultado{}, err
	}
	salida := entrega
	if bloque != "" {
		if salida != "" {
			salida += "\n\n"
		}
		salida += bloque
	}
	return tools.Resultado{Salida: salida}, nil
}

// turnoDeFlujo corre un flujo arrancado por comando del usuario. El flujo que
// arranca la cola (`/ejecutar`) no se corre una vez: recorre la cola y ejecuta
// cada elemento con sus etapas (T-B019); cualquier otro flujo corre sus etapas
// una vez.
func (o *Orquestador) turnoDeFlujo(ctx context.Context, sesionID string, agenteActivo agent.Agente, f flow.Flujo, objetivo string) error {
	if f.Comando == ComandoEjecutar {
		return o.arrancarLaCola(ctx, sesionID, agenteActivo, f)
	}
	if err := o.turnoDeEtapas(ctx, sesionID, agenteActivo, f, objetivo); err != nil {
		return err
	}
	// Lo que el ciclo deja hecho: las fases de la planificación, o la tarea que
	// entra en la cola (T-B019-05/06).
	o.cerrarCiclo(sesionID, f)
	return nil
}

// turnoDeEtapas corre las etapas de un flujo una vez: la última etapa se emite
// token a token y su texto se guarda como la respuesta del turno. Es también el
// trabajo de **un** elemento de la cola.
func (o *Orquestador) turnoDeEtapas(ctx context.Context, sesionID string, agenteActivo agent.Agente, f flow.Flujo, objetivo string) error {
	inicio := time.Now()
	historial, err := o.historialParaFlujo(ctx, sesionID)
	if err != nil {
		o.avisar(sesionID, "no se pudo armar el historial del flujo: "+err.Error(), "")
		o.cerrarTurno(sesionID, true)
		return err
	}
	entrega, _, err := o.correrFlujo(ctx, sesionID, agenteActivo, f, objetivo, historial, func(d agent.Delta) error {
		datos := map[string]string{"texto": d.Texto}
		if d.Razonamiento {
			datos["razonamiento"] = "true"
		}
		o.emitir(tui.Evento{Sesion: sesionID, Tipo: tui.EventoToken, Datos: datos})
		return nil
	})
	duracion := int(time.Since(inicio).Milliseconds())
	if err != nil {
		if errors.Is(err, context.Canceled) {
			o.avisar(sesionID, "turno detenido", "")
			_ = o.marcarEstado(sesionID, EstadoInactiva)
			return err
		}
		o.avisar(sesionID, err.Error(), "")
		o.cerrarTurno(sesionID, true)
		return err
	}
	if entrega != "" {
		if _, err := store.CrearMensaje(o.db, o.projectID, sesionID, "agent", entrega, nil, nil, &duracion); err != nil {
			o.avisar(sesionID, "no se pudo guardar la respuesta: "+err.Error(), "")
		}
	}
	o.cerrarTurno(sesionID, false)
	return nil
}

// historialParaFlujo arma el historial del modelo para un flujo: el de la sesión
// sin el propio mensaje `/flow:…` que acaba de persistirse —la petición ya viaja
// en la ventana de cada etapa—.
func (o *Orquestador) historialParaFlujo(ctx context.Context, sesionID string) ([]agent.Mensaje, error) {
	mensajes, err := o.historialPara(ctx, sesionID)
	if err != nil {
		return nil, err
	}
	if n := len(mensajes); n > 0 {
		ultimo := mensajes[n-1]
		if ultimo.Rol == agent.RolUsuario && strings.HasPrefix(strings.TrimSpace(ultimo.Contenido), prefijoFlujo) {
			mensajes = mensajes[:n-1]
		}
	}
	return mensajes, nil
}

// resumenCorto deja la primera línea del texto, acotada, para la línea de etapa.
func resumenCorto(texto string) string {
	linea := texto
	if i := strings.IndexByte(linea, '\n'); i >= 0 {
		linea = linea[:i]
	}
	linea = strings.TrimSpace(linea)
	if r := []rune(linea); len(r) > 120 {
		linea = string(r[:120]) + "…"
	}
	return linea
}
