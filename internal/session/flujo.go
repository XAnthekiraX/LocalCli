package session

import (
	"context"
	"errors"
	"fmt"
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

// elegirFlujo reconoce un flujo al principio de una línea con la forma
// `/flow:<comando>`: devuelve el flujo, el objetivo (el resto de la línea o la
// petición por defecto) y si lo hubo. Un `/comando` sin el prefijo `flow:` no
// arranca un flujo.
func (o *Orquestador) elegirFlujo(linea string) (flow.Flujo, string, bool) {
	if o.flows == nil {
		return flow.Flujo{}, "", false
	}
	recortada := strings.TrimSpace(linea)
	if !strings.HasPrefix(recortada, prefijoFlujo) {
		return flow.Flujo{}, "", false
	}
	resto := strings.TrimSpace(strings.TrimPrefix(recortada, prefijoFlujo))
	nombre, objetivo, _ := strings.Cut(resto, " ")
	nombre = strings.TrimSpace(nombre)
	if nombre == "" {
		return flow.Flujo{}, "", false
	}
	f, ok := o.flows.PorComando("/" + nombre)
	if !ok {
		return flow.Flujo{}, "", false
	}
	objetivo = strings.TrimSpace(objetivo)
	if objetivo == "" {
		objetivo = f.Peticion
	}
	return f, objetivo, true
}

// correrFlujo ejecuta todas las etapas de un flujo como sub-proceso del turno:
// compone la ventana de cada etapa, corre su agente, respeta las pausas de
// `continuacion` y `aprobacion` y publica los eventos. Devuelve la entrega y el
// bloque resumido, que es lo único que cruza hacia quien lo invocó.
func (o *Orquestador) correrFlujo(ctx context.Context, sesionID string, f flow.Flujo, peticion string, alDelta func(agent.Delta) error) (string, string, error) {
	if err := store.VaciarFlowContext(o.db, o.projectID, sesionID, f.Nombre); err != nil {
		return "", "", err
	}

	entrega := ""
	for i, etapa := range f.Etapas {
		if etapa.Continuacion {
			ok, err := o.pausarEtapa(ctx, sesionID, "¿continuamos con la etapa "+etapa.Nombre+"?")
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
		o.guardarEventoChat(sesionID, store.ChatEventoProceso, linea.Proceso(etapa.Nombre), nil)

		agenteEtapa, ok := o.agentes.Buscar(etapa.Agente)
		if !ok {
			o.emitir(tui.Evento{Sesion: sesionID, Tipo: tui.EventoEtapaFallida, Datos: map[string]string{"etapa": etapa.ID, "nombre": etapa.Nombre}})
			return "", "", fmt.Errorf("session: el flujo %s pide el agente %q, que no existe", f.Nombre, etapa.Agente)
		}

		ventana, err := o.ventanaDeEtapa(sesionID, f, etapa, peticion)
		if err != nil {
			return "", "", err
		}

		delta := func(agent.Delta) error { return nil }
		if etapa.Entrega && alDelta != nil {
			delta = alDelta
		}
		respuesta, err := o.turnoEtapa(ctx, sesionID, agenteEtapa, ventana, etapa.Entrega, delta)
		if err != nil {
			o.emitir(tui.Evento{Sesion: sesionID, Tipo: tui.EventoEtapaFallida, Datos: map[string]string{"etapa": etapa.ID, "nombre": etapa.Nombre, "motivo": err.Error()}})
			return "", "", err
		}
		texto := strings.TrimSpace(respuesta.Texto)
		if texto == "" {
			o.emitir(tui.Evento{Sesion: sesionID, Tipo: tui.EventoEtapaFallida, Datos: map[string]string{"etapa": etapa.ID, "nombre": etapa.Nombre, "motivo": "sin respuesta"}})
			return "", "", fmt.Errorf("flujo: la etapa %s no entregó texto", etapa.ID)
		}

		if err := store.GuardarFlowContext(o.db, o.projectID, sesionID, f.Nombre, etapa.ID, etapa.Nombre, i, texto); err != nil {
			return "", "", err
		}
		if etapa.RespuestaEnChat {
			o.avisar(sesionID, texto, "")
		}
		if etapa.Aprobacion {
			ok, err := o.pausarEtapa(ctx, sesionID, "aprobar la etapa "+etapa.Nombre)
			if err != nil {
				return "", "", err
			}
			if !ok {
				o.emitir(tui.Evento{Sesion: sesionID, Tipo: tui.EventoFlujoCancelado})
				return "", "", nil
			}
		}

		o.emitir(tui.Evento{Sesion: sesionID, Tipo: tui.EventoEtapaTerminada, Datos: map[string]string{
			"etapa": etapa.ID, "nombre": etapa.Nombre, "resumen": resumenCorto(texto),
		}})

		if etapa.Entrega {
			entrega = texto
		}
	}

	bloque, err := o.bloqueResumido(sesionID, f.Nombre)
	if err != nil {
		return "", "", err
	}
	return entrega, bloque, nil
}

// ventanaDeEtapa compone el texto que ve la etapa: la petición original, las
// reglas del flujo, la contribución de las etapas anteriores y su propia pregunta.
func (o *Orquestador) ventanaDeEtapa(sesionID string, f flow.Flujo, etapa flow.Etapa, peticion string) ([]agent.Mensaje, error) {
	previos, err := store.ListarFlowContext(o.db, o.projectID, sesionID, f.Nombre)
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
	b.WriteString("\n## ")
	b.WriteString(etapa.Nombre)
	b.WriteString("\nPregunta: ")
	b.WriteString(etapa.Pregunta)
	if etapa.Instruccion != "" {
		b.WriteString("\n")
		b.WriteString(etapa.Instruccion)
	}

	return []agent.Mensaje{{Rol: agent.RolUsuario, Contenido: b.String()}}, nil
}

// turnoEtapa corre el turno de una etapa con su agente y sus permisos. La etapa
// `entrega` no usa herramientas; ninguna etapa ve `ejecutar_flujo` (profundidad
// uno).
func (o *Orquestador) turnoEtapa(ctx context.Context, sesionID string, agenteEtapa agent.Agente, ventana []agent.Mensaje, entrega bool, alDelta func(agent.Delta) error) (agent.Respuesta, error) {
	ejecutor := &agent.Ejecutor{
		Agente:    agenteEtapa,
		Motor:     motorOpenAI{cliente: o.llm, modelo: o.modeloActual, sinRazonamiento: func() bool { return !o.pensarActual() }},
		Capacidad: llm.Soportada,
		Ventana:   o.ventanaActual(),
		Avisar:    func(motivo, siguiente string) { o.avisar(sesionID, motivo, siguiente) },
	}
	if o.herramientas != nil && !entrega {
		ejecutor.Cerrado = agent.CatalogoEfectivo(agenteEtapa, o.catalogoHerramientas)
		ejecutor.Herramientas = adaptadorHerramientas{
			registro: o.herramientas,
			sesion:   sesionID,
			permisos: agenteEtapa.Permisos(),
			Ask: func(ctx context.Context, s tools.Solicitud) (tools.Decision, error) {
				return o.pedirAprobacion(ctx, sesionID, s)
			},
		}
	}
	return ejecutor.Turno(ctx, ventana, alDelta)
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
// flujo pedido, comprueba el permiso derivado y lo corre como sub-proceso.
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
	if !flow.PuedeLanzar(f, c.Permisos) {
		return tools.Resultado{Error: tools.CodFlowNotAllowed + ": el agente no puede lanzar el flujo `" + peticion.Flujo + "`"}, nil
	}

	entrega, bloque, err := o.correrFlujo(ctx, c.SesionID, f, strings.TrimSpace(peticion.Peticion), nil)
	if err != nil {
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
func (o *Orquestador) turnoDeFlujo(ctx context.Context, sesionID string, f flow.Flujo, objetivo string) error {
	if f.Comando == ComandoEjecutar {
		return o.arrancarLaCola(ctx, sesionID, f)
	}
	if err := o.turnoDeEtapas(ctx, sesionID, f, objetivo); err != nil {
		return err
	}
	// Lo que el ciclo deja hecho: las fases de la planificación, o la tarea que
	// entra en la cola (T-B019-05/06).
	o.cerrarCiclo(sesionID, f)
	return nil
}

// turnoDeEtapas corre las etapas de un flujo una vez: la etapa de entrega se
// emite token a token y su texto se guarda como la respuesta del turno. Es
// también el trabajo de **un** elemento de la cola.
func (o *Orquestador) turnoDeEtapas(ctx context.Context, sesionID string, f flow.Flujo, objetivo string) error {
	inicio := time.Now()
	entrega, _, err := o.correrFlujo(ctx, sesionID, f, objetivo, func(d agent.Delta) error {
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
