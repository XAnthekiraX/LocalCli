// arranque.go — el cableado de producción: la única pieza que conoce todas las
// capas a la vez (BACKEND.md, wire.go: "El adaptador de producción vive en el
// arranque").
//
// Fuente de verdad: ai/docs/backend/04-infrastructure/EVENTS.md §4 ("la TUI no
// pregunta nada: se le notifica"), ai/docs/database/02-rules/DATA_FLOW.md §2 y
// §7 (petición de permiso, espera y resolución en la misma transacción) y
// ai/docs/backend/DECISIONS.md ("store es el único que escribe en SQLite").
//
// La vista (`tui`) solo conoce `tui.Puerto`; el motor (`session`) solo conoce su
// `Almacen`, su `Motor` y su bus; `flow` pide contexto, agente y aprobador por
// interfaces. Nada de eso sabe de lo demás. Este archivo los ata:
//   - adapta `session.Gestor` + `store` al `Puerto` que espera la vista,
//   - conecta Ollama (modelo local detectado, serializado por la cola FIFO),
//   - crea la sesión activa del proyecto y retoma la conversación,
//   - y hace vivir el flujo de aprobación real: cada petición de permiso llega
//     a la vista como `peticion_aprobacion` y la decisión del usuario vuelve al
//     motor como `aprobacion_resuelta` (el puente entre fileops/AprobadorSQLite
//     y el panel de la TUI).
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"localcli/internal/agent"
	tcontext "localcli/internal/context"
	lcexec "localcli/internal/exec"
	"localcli/internal/fileops"
	"localcli/internal/flow"
	"localcli/internal/ollama"
	"localcli/internal/queue"
	"localcli/internal/session"
	"localcli/internal/store"
	"localcli/internal/task"
	"localcli/internal/tools"
	"localcli/internal/tui"
)

// nombreSesionActiva es la sesión del proyecto que la TUI retoma al arrancar
// (SPEC-INTERFAZ §Pantalla de bienvenida: "la primera pantalla pertenece a la
// sesión activa del proyecto").
const nombreSesionActiva = "principal"

// timeoutDeteccion acota cuánto espera el arranque a que Ollama responda con
// su lista de modelos. Si no responde en ese tiempo, la interfaz arranca igual
// y avisa (CONFIGURATION.md §5: un fallo de modelo no puede tumbar la vista).
const timeoutDeteccion = 3 * time.Second

// Arranque es el cableado completo: base, bus de eventos, gestor de sesiones y
// cola de inferencia. `main` lo construye, saca la TUI y lo cierra al salir.
type Arranque struct {
	// cerrarDB baja la conexión al salir; es un cierre opaco, no una consulta:
	// el arranque no abre SQLite por su cuenta ni importa database/sql
	// (invariante de DECISIONS.md: store es el único que toca la base).
	cerrarDB func() error
	Bus      *session.Bus
	Gest     *session.Gestor
	Infer    *ollama.ColaInferencia

	puerto *Adaptador
}

// Adaptador cumple `tui.Puerto` sobre el gestor de sesiones y la base del
// proyecto. Es el único sitio donde la vista toca el mundo real; todo lo que la
// TUI puede hacer está limitado por lo que hay aquí (wire.go).
type Adaptador struct {
	gest *session.Gestor
	// aprobar y cerrar son repositorios de `store`: las operaciones de
	// aprobación y de turno salen de aquí, nunca de SQL directo.
	aprobar *store.Aprobaciones
	cerrar  *store.Turnos
	auditar *store.Auditorias
	bus     *session.Bus
	dir     string

	mu         sync.Mutex
	sesionID   string
	pendientas map[string]chan string // approvalID -> canal de resolución
}

var _ tui.Puerto = (*Adaptador)(nil)

// nuevoArranque abre la base del proyecto (via `store`, el único escritor),
// monta el motor real contra Ollama local y deja la sesión activa lista. Un
// fallo de Ollama NO impide arrancar: la interfaz sigue viva y avisa cuando no
// hay modelo (CONFIGURATION.md §5).
func nuevoArranque(carpeta string) (*Arranque, error) {
	conexion, err := store.Open(carpeta)
	if err != nil {
		return nil, fmt.Errorf("arranque: %w", err)
	}
	// El arranque no manipula la conexión: la envuelve en los repositorios de
	// `store` y conserva solo un cierre opaco (invariante: ningún paquete
	// fuera de internal/store importa database/sql).
	aprobar := store.NuevasAprobaciones(conexion)
	turnos := store.NuevosTurnos(conexion)
	cerrarDB := conexion.Close

	bus := session.NuevoBus()
	infer := ollama.NewColaInferencia()
	cliente := ollama.NewClient("")

	modelo, err := elegirModelo(cliente)
	if err != nil {
		fmt.Fprintln(os.Stderr, "aviso:", err, "(la interfaz arranca sin modelo)")
	}

	// El adaptador nace aquí: los handlers de herramientas lo enlazan por
	// referencia (aprobarComando lee gest y sesionActual al invocarse, nunca al
	// montarse), así que el registro puede construirse antes de que el gestor
	// exista.
	cambios := store.NuevosCambios(conexion)
	ad := &Adaptador{aprobar: aprobar, cerrar: turnos, auditar: store.NuevasAuditorias(conexion), bus: bus, dir: carpeta, pendientas: map[string]chan string{}}
	aprobadorTerminal := func(ctx context.Context, descripcion string) (bool, error) {
		return ad.aprobarComando(ctx, descripcion)
	}
	registro, err := tools.NuevoRegistro(tools.Destinos{
		Archivos: handlerArchivos(aprobar, cambios, carpeta, ad),
		Terminal: handlerTerminal(aprobadorTerminal),
		Internet: handlerInternet(),
	})
	if err != nil {
		conexion.Close()
		return nil, err
	}
	despachador := agent.NuevoDespachador(registro)

	agenteRunner := agent.Runner{Cliente: cliente}
	grafo, gErr := tcontext.GrafoDeDocumentos(carpeta)
	if gErr != nil {
		// Sin grafo no hay nodo de contexto; el motor fallará con un mensaje
		// claro por etapa. No se inventa un grafo vacío.
		fmt.Fprintln(os.Stderr, "aviso: sin grafo de documentos:", gErr)
	}

	motorFlows := &flow.Motor{Eventos: bus}

	alcance, err := session.NuevoAlcance(carpeta, store.Sesiones{DB: conexion})
	if err != nil {
		conexion.Close()
		return nil, err
	}
	gest, err := session.NuevoGestor(alcance, motorFlows, bus)
	if err != nil {
		conexion.Close()
		return nil, err
	}
	// El flujo por defecto es el ciclo de trabajo (SPEC-CICLO-TRABAJO); la
	// detección de trabajo ordenado decide si además se consume la cola.
	gest.Flujo = session.FlujoPorDefecto()
	ad.gest = gest

	a := &Arranque{cerrarDB: cerrarDB, Bus: bus, Gest: gest, Infer: infer}
	a.puerto = ad

	// El motor necesita contexto, agente y aprobador POR CADA SESIÓN (el nodo
	// lleva el id de sesión y etapa para auditar). Se los inyectamos mediante
	// una función que reconstruye esas piezas sobre el mismo motor antes de
	// cada turno: es el sitio del arranque, no de session ni de flow.
	motorFlows.Contexto = &nodoPorTurno{ad: ad, grafo: grafo, cliente: cliente, modelo: modelo}
	motorFlows.Agente = &ejecutorPorTurno{ad: ad, runner: agenteRunner, despachar: despachador, modelo: modelo, infer: infer, agente: agenteBase(carpeta)}
	motorFlows.Aprobador = &aprobadorPorTurno{ad: ad}

	// Arranque de sesión: retomar la activa del proyecto o crearla
	// (SPEC-INTERFAZ §Pantalla de bienvenida).
	ses, err := ad.ResolverActiva()
	if err != nil {
		bus.Cerrar()
		conexion.Close()
		return nil, err
	}
	ad.fijarSesion(ses.ID)

	// Las aprobaciones que quedaron pendientes de otra ejecución vuelven a
	// tener dueño: se re-emiten como peticiones al bus (DATA_FLOW.md §7).
	reemitirPendientes(aprobar, bus)

	return a, nil
}

// Cerrar suelta recursos en orden: el bus primero (los canales de eventos se
// cierran y ningún consumidor queda colgado) y la base después.
func (a *Arranque) Cerrar() {
	if a == nil {
		return
	}
	if a.Bus != nil {
		a.Bus.Cerrar()
	}
	if a.cerrarDB != nil {
		_ = a.cerrarDB()
	}
}

// nuevaApp arma la TUI real sobre el puerto de producción. Devuelve el modelo
// listo para `tea.NewProgram`.
func nuevaApp(a *Arranque) *tui.App { return tui.Nuevo(a.puerto) }

// --- tui.Puerto -------------------------------------------------------------

// ResolverActiva retoma la sesión activa del proyecto o la crea. Es la única
// lectura de arranque que hace la vista a través del puerto.
func (ad *Adaptador) ResolverActiva() (*session.Sesion, error) {
	sesiones, err := ad.gest.Listar()
	if err != nil {
		return nil, err
	}
	for _, s := range sesiones {
		if s.Nombre == nombreSesionActiva && s.Estado != session.EstadoTerminada && s.Estado != session.EstadoError {
			return ad.gest.Estado(s.ID)
		}
	}
	for _, s := range sesiones {
		if s.Nombre == nombreSesionActiva {
			// Terminó o falló: se retoma pasando por inactiva (ENUMS.md §3),
			// que es legal desde cualquiera de los dos.
			if err := ad.gest.Cerrar(s.ID, true); err != nil {
				return nil, err
			}
			return ad.gest.Crear(nombreSesionActiva, "")
		}
	}
	return ad.gest.Crear(nombreSesionActiva, "")
}

func (ad *Adaptador) Listar() ([]session.Sesion, error) { return ad.gest.Listar() }

// Historial traduce la conversación guardada a lo que la vista pinta. La vista
// nunca lee la base: llega como dato (INTERFACES §3).
func (ad *Adaptador) Historial(sesionID string) ([]tui.MensajeHistorial, error) {
	ms, err := ad.gest.Historial(sesionID)
	if err != nil {
		return nil, err
	}
	out := make([]tui.MensajeHistorial, 0, len(ms))
	for _, m := range ms {
		out = append(out, tui.MensajeHistorial{Rol: m.Role, Texto: m.Content, Razonamiento: m.Reasoning})
	}
	return out, nil
}

// Enviar manda el mensaje al gestor y, tras el turno, intenta consumir la cola
// si la petición era trabajo ordenado (SPEC-COLA-TAREAS). La vista recibe el
// desenlace por eventos, no por el retorno.
func (ad *Adaptador) Enviar(ctx context.Context, sesionID, texto string) error {
	ad.fijarSesion(sesionID)
	ordenado := flow.EsTrabajoOrdenado(texto)
	err := ad.gest.Enviar(ctx, sesionID, texto)
	if err != nil {
		return err
	}
	if ordenado {
		go ad.consumirCola(sesionID)
	}
	return nil
}

// consumirCola espera a que termine el turno del mensaje y entonces deriva la
// cola del TODO de la capa y la entrega al motor. La espera es necesaria:
// `Enviar` corre el flujo en segundo plano y dos ejecuciones simultáneas de la
// misma sesión no están permitidas (SPEC-SESIONES).
func (ad *Adaptador) consumirCola(sesionID string) {
	ctx := context.Background()
	ad.gest.Esperar(sesionID)
	capa := task.Capa(strings.TrimSpace(ad.capaDe(sesionID)))
	if !task.EsCapa(string(capa)) {
		capa = task.CapaBackend
	}
	q, err := queue.Reconstruir(ad.dir, capa)
	if err != nil {
		return
	}
	consumidor, err := q.Consumir(queue.PermitirAlMotor())
	if err != nil {
		return
	}
	_ = ad.gest.ConsumirCola(ctx, sesionID, consumidor)
}

func (ad *Adaptador) Pendientes() ([]tui.Aprobacion, error) {
	items, err := ad.aprobar.Pendientes()
	if err != nil {
		return nil, err
	}
	out := make([]tui.Aprobacion, 0, len(items))
	for _, p := range items {
		out = append(out, tui.Aprobacion{ID: p.ID, Sesion: p.SessionID, Descripcion: p.Description})
	}
	return out, nil
}

// Resolver aplica la decisión del usuario en la transacción compuesta de
// DATA_FLOW.md §2 (aprobar + devolver la sesión a su estado) y notifica al
// motor por el bus: el evento `aprobacion_resuelta` libera la goroutine que
// estaba esperando en AprobadorSQLite y la línea sale del panel.
func (ad *Adaptador) Resolver(aprobacionID string, aprobar bool) error {
	ap, err := ad.aprobar.Obtener(aprobacionID)
	if err != nil {
		return err
	}
	if ap.Status != store.ApprovalPendiente {
		return fmt.Errorf("arranque: la aprobación %s ya está resuelta (%s)", aprobacionID, ap.Status)
	}
	estado := store.ApprovalDeclinada
	if aprobar {
		estado = store.ApprovalAprobada
	}
	// Al resolver, la sesión vuelve a su estado anterior: esperando_permiso solo
	// se alcanza desde trabajando o inactiva (ENUMS.md §3), así que ese era el
	// origen. Si algo cambió mientras tanto, ResolverPermiso falla y se informa.
	destino := session.EstadoTrabajando
	if ses, e := ad.gest.Estado(ap.SessionID); e == nil && ses != nil && ses.Estado == session.EstadoInactiva {
		destino = session.EstadoInactiva
	}
	if err := ad.aprobar.ResolverPermiso(aprobacionID, estado, destino); err != nil {
		return err
	}
	ad.bus.Emitir(tui.Evento{Nombre: tui.EventoAprobacionResuelta, Datos: map[string]string{
		"aprobacion": aprobacionID,
		"sesion":     ap.SessionID,
		"decision":   estado,
	}})
	ad.resolverCanal(aprobacionID, estado)
	return nil
}

// aprobarComando pide decisión humana para un comando de terminal: registra la
// aprobación, emite la petición al panel y espera el veredicto. Es el puente
// entre `exec` y la vista (DATA_FLOW.md §7).
func (ad *Adaptador) aprobarComando(ctx context.Context, descripcion string) (bool, error) {
	id, ch, err := ad.pedirPermiso(descripcion)
	if err != nil {
		return false, err
	}
	var estado string
	select {
	case estado = <-ch:
	case <-ctx.Done():
		ad.resolverCanal(id, store.ApprovalObsoleta)
		return false, nil
	}
	return estado == store.ApprovalAprobada, nil
}

func (ad *Adaptador) Pausar(sesionID string) error { return ad.gest.Pausar(sesionID) }

func (ad *Adaptador) Cancelar(sesionID string) { ad.gest.Cancelar(sesionID) }

// Suscribir delega en el gestor: un solo canal para la vista, con su baja.
func (ad *Adaptador) Suscribir() (<-chan tui.Evento, func()) { return ad.gest.Suscribir() }

// --- estado interno del adaptador -------------------------------------------

func (ad *Adaptador) fijarSesion(id string) {
	ad.mu.Lock()
	defer ad.mu.Unlock()
	ad.sesionID = id
}

func (ad *Adaptador) sesionActual() string {
	ad.mu.Lock()
	defer ad.mu.Unlock()
	return ad.sesionID
}

func (ad *Adaptador) capaDe(sesionID string) string {
	ses, err := ad.gest.Estado(sesionID)
	if err != nil || ses == nil {
		return ""
	}
	return ses.Capa
}

// pedirPermiso registra la aprobación (transacción compuesta de store: crea la
// fila y pasa la sesión a esperando_permiso), emite `peticion_aprobacion` y
// devuelve el canal donde el usuario resolverá. Es el lado productor del flujo
// de aprobación (DATA_FLOW.md §7 paso 3).
func (ad *Adaptador) pedirPermiso(descripcion string) (approvalID string, resolucion <-chan string, err error) {
	// Con la sesión en esperando_permiso, store rechazaría una segunda
	// transición; si llega otro pedido mientras tanto, se registra sin tocar
	// el estado (ya está esperando).
	ses := ad.sesionActual()
	extra := ""
	if st, e := ad.gest.Estado(ses); e == nil && st != nil && st.Estado == session.EstadoEsperandoPermiso {
		ses, extra = "", " (pendiente de resolución)"
	}
	ap, err := ad.aprobar.PedirPermiso(ses, descripcion+extra)
	if err != nil {
		return "", nil, err
	}
	ch := make(chan string, 1)
	ad.mu.Lock()
	ad.pendientas[ap.ID] = ch
	ad.mu.Unlock()
	ad.bus.Emitir(tui.Evento{Nombre: tui.EventoPeticionAprobacion, Datos: map[string]string{
		"aprobacion":  ap.ID,
		"sesion":      ap.SessionID,
		"descripcion": ap.Description,
	}})
	return ap.ID, ch, nil
}

// resolverCanal entrega el estado final a quien esperaba (la goroutine del
// motor bloqueada en AprobadorSQLite).
func (ad *Adaptador) resolverCanal(aprobacionID, estado string) {
	ad.mu.Lock()
	ch, ok := ad.pendientas[aprobacionID]
	if ok {
		delete(ad.pendientas, aprobacionID)
	}
	ad.mu.Unlock()
	if ok {
		ch <- estado
	}
}

// esperarResolucion bloquea hasta que el usuario decida (o ctx se cancele o la
// aprobación quede obsoleta porque la sesión terminó). Implementa `fileops.Espera`.
func (ad *Adaptador) esperarResolucion(ctx context.Context, aprobacionID string) (string, error) {
	ad.mu.Lock()
	ch, ok := ad.pendientas[aprobacionID]
	ad.mu.Unlock()
	if !ok {
		// Nadie la pidió desde este proceso: se consulta el estado en la base.
		ap, err := ad.aprobar.Obtener(aprobacionID)
		if err != nil {
			return "", err
		}
		return ap.Status, nil
	}
	select {
	case estado := <-ch:
		return estado, nil
	case <-ctx.Done():
		// Cancelada: la pendiente pasa a obsoleta (BUSINESS_RULES.md).
		_ = ad.aprobar.Resolver(aprobacionID, store.ApprovalObsoleta)
		ad.resolverCanal(aprobacionID, store.ApprovalObsoleta)
		return store.ApprovalObsoleta, nil
	}
}

// reemitirPendientes vuelve a publicar las aprobaciones que quedaron
// pendientes de una ejecución anterior, para que el panel las muestre.
func reemitirPendientes(aprobar *store.Aprobaciones, bus *session.Bus) {
	items, err := aprobar.Pendientes()
	if err != nil {
		return
	}
	for _, p := range items {
		bus.Emitir(tui.Evento{Nombre: tui.EventoPeticionAprobacion, Datos: map[string]string{
			"aprobacion":  p.ID,
			"sesion":      p.SessionID,
			"descripcion": p.Description,
		}})
	}
}

// --- handlers de herramientas ------------------------------------------------

// handlerArchivos enruta las once herramientas de archivo hacia `fileops`. Las
// escrituras pasan por el aprobador de producción (AprobadorSQLite): cada
// petición llega al panel como `peticion_aprobacion` y solo se aplica si el
// usuario resuelve (SPEC-ARCHIVOS: toda escritura requiere aprobación).
func handlerArchivos(aprobar *store.Aprobaciones, cambios *store.Cambios, carpeta string, ad *Adaptador) tools.Handler {
	return func(ctx context.Context, p tools.Peticion) (any, error) {
		ops := &fileops.Ops{
			Proyecto:  carpeta,
			Historial: cambios,
			SesionID:  ad.sesionActual(),
			Aprobador: &fileops.AprobadorSQLite{
				Permisos: aprobar,
				SesionID: ad.sesionActual(),
				Espera:   esperarDesdeBase(aprobar),
			},
		}
		switch h := p.Argumentos.(type) {
		case *tools.PeticionLeerArchivo:
			return fileops.LeerArchivo(carpeta, h.Ruta)
		case *tools.PeticionListarCarpeta:
			return fileops.ListarCarpeta(carpeta, h.Ruta)
		case *tools.PeticionBuscarArchivos:
			return fileops.BuscarArchivos(carpeta, h.Patron)
		case *tools.PeticionBuscarEnArchivos:
			return fileops.BuscarEnArchivos(carpeta, h.Patron, h.Ruta)
		case *tools.PeticionCrearArchivo:
			return ops.CrearArchivo(ctx, h.Ruta, h.Contenido)
		case *tools.PeticionEscribirArchivo:
			return ops.EscribirArchivo(ctx, h.Ruta, h.Contenido)
		case *tools.PeticionEditarArchivo:
			return ops.EditarArchivo(ctx, h.Ruta, h.Cambio)
		case *tools.PeticionEliminarArchivo:
			return ops.EliminarArchivo(ctx, h.Ruta)
		case *tools.PeticionCrearCarpeta:
			return ops.CrearCarpeta(ctx, h.Ruta)
		case *tools.PeticionEliminarCarpeta:
			return ops.EliminarCarpeta(ctx, h.Ruta)
		default:
			return nil, fmt.Errorf("arranque: petición de archivo desconocida %T", p.Argumentos)
		}
	}
}

// esperarDesdeBase es la `fileops.Espera` que usa el aprobador cuando una
// escritura llega por una ruta sin canal vivo (una petición de herramienta
// dentro del turno): sondea la base hasta que la aprobación se resuelve o el
// contexto se cancela. La resolución la pone el panel vía `Adaptador.Resolver`.
func esperarDesdeBase(aprobar *store.Aprobaciones) fileops.Espera {
	return func(ctx context.Context, aprobacionID string) (string, error) {
		ticker := time.NewTicker(250 * time.Millisecond)
		defer ticker.Stop()
		for {
			ap, err := aprobar.Obtener(aprobacionID)
			if err != nil {
				return "", err
			}
			if ap.Status != store.ApprovalPendiente {
				return ap.Status, nil
			}
			select {
			case <-ctx.Done():
				_ = aprobar.Resolver(aprobacionID, store.ApprovalObsoleta)
				return store.ApprovalObsoleta, nil
			case <-ticker.C:
			}
		}
	}
}

// handlerTerminal enruta `ejecutar_comando` hacia `exec`: lista blanca,
// límites y Landlock los decide el módulo. El aprobador es una función ligada
// al adaptador vivo (se inyecta en nuevoArranque): cada comando que necesite
// decisión humana llega al panel como `peticion_aprobacion`.
func handlerTerminal(aprobar func(context.Context, string) (bool, error)) tools.Handler {
	return func(ctx context.Context, p tools.Peticion) (any, error) {
		h, ok := p.Argumentos.(*tools.PeticionEjecutarComando)
		if !ok {
			return nil, fmt.Errorf("arranque: petición de terminal desconocida %T", p.Argumentos)
		}
		ejecutor := &lcexec.Ejecutor{Proyecto: h.Carpeta, Aprobador: lcexec.AprobadorFunc(
			func(c context.Context, descripcion string) (bool, error) { return aprobar(c, descripcion) })}
		return ejecutor.Ejecutar(ctx, h.Comando, h.Carpeta)
	}
}

// handlerInternet cubre `buscar_en_internet` y `abrir_pagina`. `tools` ya
// exige LOCALCLI_ALLOW_INTERNET antes de enrutar aquí (SECURITY.md §4): sin la
// variable ninguna petición sale de la máquina. Lo que devuelve entra al
// contexto como contenido sin confianza, recortado y auditado como cualquier
// documento.
func handlerInternet() tools.Handler {
	return func(ctx context.Context, p tools.Peticion) (any, error) {
		switch h := p.Argumentos.(type) {
		case *tools.PeticionBuscarInternet:
			return buscarEnInternet(ctx, h.Consulta)
		case *tools.PeticionAbrirPagina:
			return abrirPagina(ctx, h.Direccion)
		default:
			return nil, fmt.Errorf("arranque: petición de internet desconocida %T", p.Argumentos)
		}
	}
}

const (
	internetMaxBody       = 512 << 10 // medio MiB de respuesta: suficiente y acotado
	internetMaxResultados = 8
)

// buscarEnInternet usa la API de DuckDuckGo (sin clave, JSON puro). Es lo
// único que sale de la máquina: solo viaja la consulta, nunca contenido del
// proyecto (SECURITY.md §4).
func buscarEnInternet(ctx context.Context, consulta string) (tools.RespuestaBuscarInternet, error) {
	u := "https://api.duckduckgo.com/?q=" + httpQuery(consulta) + "&format=json&no_html=1"
	var cuerpo struct {
		RelatedTopic []struct {
			Name     string `json:"Name"`
			Text     string `json:"Text"`
			FirstURL string `json:"FirstURL"`
			Topics   []struct {
				Name     string `json:"Name"`
				FirstURL string `json:"FirstURL"`
				Text     string `json:"Text"`
			} `json:"Topics"`
		} `json:"RelatedTopic"`
	}
	if err := getJSON(ctx, u, &cuerpo); err != nil {
		return tools.RespuestaBuscarInternet{}, err
	}
	out := tools.RespuestaBuscarInternet{}
	add := func(titulo, fragmento, direccion string) {
		if strings.TrimSpace(titulo) == "" || len(out.Resultados) >= internetMaxResultados {
			return
		}
		out.Resultados = append(out.Resultados, tools.ResultadoInternet{
			Titulo: titulo, Fragmento: fragmento, Direccion: direccion})
	}
	for _, tema := range cuerpo.RelatedTopic {
		add(tema.Name, tema.Text, tema.FirstURL)
		for _, sub := range tema.Topics {
			add(sub.Name, sub.Text, sub.FirstURL)
		}
	}
	return out, nil
}

// abrirPagina descarga una página y devuelve su texto sin marcado. El esquema
// se fuerza a https: una dirección en claro sería exposición accidental.
func abrirPagina(ctx context.Context, direccion string) (tools.RespuestaAbrirPagina, error) {
	if !strings.Contains(direccion, "://") {
		direccion = "https://" + direccion
	}
	resp, err := httpGet(ctx, direccion)
	if err != nil {
		return tools.RespuestaAbrirPagina{}, err
	}
	defer resp.Body.Close()
	datos, err := io.ReadAll(io.LimitReader(resp.Body, internetMaxBody))
	if err != nil {
		return tools.RespuestaAbrirPagina{}, err
	}
	return tools.RespuestaAbrirPagina{Contenido: textoPlano(string(datos))}, nil
}

func httpQuery(s string) string {
	var b strings.Builder
	for _, c := range []byte(s) {
		switch {
		case c == ' ':
			b.WriteByte('+')
		case 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9' || c == '-' || c == '_' || c == '.' || c == '~':
			b.WriteByte(c)
		default:
			b.WriteString(fmt.Sprintf("%%%02X", c))
		}
	}
	return b.String()
}

func httpGet(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "LocalCli/0.1 (busqueda local; contenido no confiable)")
	return http.DefaultClient.Do(req)
}

func getJSON(ctx context.Context, url string, destino any) error {
	resp, err := httpGet(ctx, url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	datos, err := io.ReadAll(io.LimitReader(resp.Body, internetMaxBody))
	if err != nil {
		return err
	}
	return json.Unmarshal(datos, destino)
}

// textoPlano quita etiquetas y entidades más comunes: lo que entra al contexto
// es texto, no HTML (y nunca se interpreta).
func textoPlano(html string) string {
	i := strings.Index(html, "<body")
	if i >= 0 {
		if j := strings.Index(html[i:], ">"); j >= 0 {
			html = html[i+j+1:]
		}
	}
	if k := strings.LastIndex(html, "</body>"); k >= 0 {
		html = html[:k]
	}
	var b strings.Builder
	enEtiqueta := false
	for _, r := range html {
		switch {
		case r == '<':
			enEtiqueta = true
		case r == '>':
			enEtiqueta = false
			b.WriteRune(' ')
		case !enEtiqueta:
			b.WriteRune(r)
		}
	}
	texto := b.String()
	for _, par := range [][2]string{{"&amp;", "&"}, {"&lt;", "<"}, {"&gt;", ">"}, {"&quot;", "\""}, {"&#39;", "'"}, {"&nbsp;", " "}} {
		texto = strings.ReplaceAll(texto, par[0], par[1])
	}
	texto = strings.Join(strings.Fields(texto), " ")
	if len(texto) > 4000 {
		texto = texto[:4000] + "…"
	}
	return texto
}

// --- piezas del motor, ligadas a la sesión del turno -------------------------

// elegirModelo detecta el modelo local (CONFIGURATION.md §3: "se detecta, no se
// configura"): lista los modelos de Ollama y toma el primero que cabe en la
// máquina. Si Ollama no responde, devuelve error y el arranque sigue sin modelo.
func elegirModelo(cliente *ollama.Client) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeoutDeteccion)
	defer cancel()
	modelos, err := cliente.ListarModelos(ctx)
	if err != nil {
		return "", fmt.Errorf("no hay conexión con Ollama: %w", err)
	}
	hw := ollama.PerfilPorDefecto()
	caben, _, _ := hw.ClasificarModelos(modelos)
	if len(caben) > 0 {
		return caben[0].Nombre, nil
	}
	if len(modelos) > 0 {
		// Sin clasificación clara se usa el primero: el usuario eligió ese modelo
		// en Ollama y el aviso de tamaño ya se mostró al listar.
		return modelos[0].Nombre, nil
	}
	return "", errors.New("Ollama no tiene modelos instalados")
}

// nodoPorTurno implementa flow.Contexto: arma el nodo de contexto con la sesión
// del turno actual. El nodo es efímero por diseño (una selección por petición).
type nodoPorTurno struct {
	ad      *Adaptador
	grafo   tcontext.Grafo
	cliente *ollama.Client
	modelo  string
}

func (n *nodoPorTurno) ContextoPara(ctx context.Context, objetivo string) (string, error) {
	if n.grafo == nil || n.modelo == "" {
		return "", fmt.Errorf("arranque: sin grafo o modelo no hay contexto que entregar")
	}
	nodo := &tcontext.Nodo{
		Grafo:     n.grafo,
		Modelo:    &tcontext.ModeloOllama{Cliente: n.cliente, Modelo: n.modelo},
		Auditor:   n.ad.auditar,
		SessionID: n.ad.sesionActual(),
	}
	return nodo.ContextoPara(ctx, objetivo)
}

// ejecutorPorTurno implementa flow.Agente: corre la etapa con el agente base
// (`plan` o `build`) cargado de ai/agents, encolando la inferencia en la FIFO,
// transmitiendo tokens y razonamiento a la TUI como eventos (EVENTS.md §1,
// DECISIONS.md [25]) y despachando las herramientas que el modelo solicite a
// través de `agent` → `tools` (TOOLS.md §7).
type ejecutorPorTurno struct {
	ad        *Adaptador
	runner    agent.Runner
	despachar *agent.Despachador
	modelo    string
	infer     *ollama.ColaInferencia
	agente    map[string]agent.Agente
}

func (e *ejecutorPorTurno) Ejecutar(ctx context.Context, nombreAgente, contexto string) (flow.Resultado, error) {
	if e.modelo == "" {
		return flow.Resultado{}, errors.New("arranque: no hay modelo de Ollama disponible")
	}
	agente, ok := e.agente[nombreAgente]
	if !ok {
		return flow.Resultado{}, fmt.Errorf("arranque: agente base desconocido %q", nombreAgente)
	}
	sesionID := e.ad.sesionActual()

	var texto strings.Builder
	var razon strings.Builder
	err := e.infer.Encolar(ctx, ollama.OpcionesEncolar{IdSesion: sesionID}, func(c context.Context) error {
		texto.Reset()
		razon.Reset()
		mensajes := []ollama.Mensaje{{Role: "user", Content: contexto}}
		// Tres pasadas como máximo: el modelo responde, pide herramientas, se
		// ejecutan, y vuelve a responder con los resultados. Un turno no puede
		// quedar abierto indefinidamente.
		for pasada := 0; pasada < 3; pasada++ {
			ch, err := e.runner.Generar(c, agente, e.modelo, mensajes)
			if err != nil {
				e.ad.bus.Emitir(tui.Evento{Nombre: tui.EventoToken, Datos: map[string]string{
					"sesion": sesionID, "texto": "[sin respuesta del modelo]",
				}})
				return err
			}
			texto.Reset()
			razon.Reset()
			var pedidos []agent.SolicitudHerramienta
			for ev := range ch {
				switch ev.Tipo {
				case ollama.EventoToken:
					texto.WriteString(ev.Texto)
					e.emitirToken(sesionID, ev.Texto, false)
				case ollama.EventoRazonamiento:
					razon.WriteString(ev.Texto)
					e.emitirToken(sesionID, ev.Texto, true)
				case ollama.EventoError:
					return ev.Error
				}
			}
			pedidos = solicitudesDeHerramienta(texto.String())
			if len(pedidos) == 0 {
				return nil
			}
			mensajes = append(mensajes, ollama.Mensaje{Role: "assistant", Content: texto.String()})
			var informe strings.Builder
			for _, sol := range pedidos {
				resultado, err := e.despachar.Despachar(c, agente, sol)
				if err != nil {
					resultado = map[string]string{"error": err.Error()}
				}
				datos, mErr := json.Marshal(resultado)
				if mErr != nil {
					datos = []byte(`{"error":"respuesta no serializable"}`)
				}
				informe.WriteString(sol.Nombre + " → " + string(datos) + "\n")
			}
			mensajes = append(mensajes, ollama.Mensaje{Role: "user", Content: "Resultados de herramientas:\n" + informe.String()})
		}
		return nil
	})
	if err != nil {
		return flow.Resultado{}, err
	}
	resumen := strings.TrimSpace(texto.String())
	if resumen == "" {
		resumen = "(respuesta vacía)"
	}
	// El turno del agente se guarda en la misma transacción que su razonamiento
	// y su estado (DATA_FLOW.md): el historial que verá la próxima vez incluye
	// esta respuesta.
	if sesionID != "" {
		if _, cErr := cerrarTurnoGuardado(e.ad, sesionID, resumen, razon.String()); cErr != nil {
			return flow.Resultado{}, cErr
		}
	}
	return flow.Resultado{Texto: resumen}, nil
}

func (e *ejecutorPorTurno) emitirToken(sesionID, texto string, esRazon bool) {
	if texto == "" {
		return
	}
	datos := map[string]string{"sesion": sesionID, "texto": texto}
	if esRazon {
		datos["razonamiento"] = "true"
	}
	e.ad.bus.Emitir(tui.Evento{Nombre: tui.EventoToken, Datos: datos})
}

// solicitudesDeHerramienta extrae del texto del modelo los bloques
// ```herramienta ...``` con argumentos JSON. Es el contrato mínimo de llamada
// a herramientas para un modelo local sin function-calling nativo: lo que el
// modelo no declara en ese formato no se ejecuta, y `tools` valida el nombre y
// los argumentos contra el catálogo cerrado antes de enrutar (VALIDATION.md).
func solicitudesDeHerramienta(texto string) []agent.SolicitudHerramienta {
	var out []agent.SolicitudHerramienta
	resto := texto
	for {
		i := strings.Index(resto, "```herramienta")
		if i < 0 {
			break
		}
		resto = resto[i+len("```herramienta"):]
		j := strings.Index(resto, "\n```")
		if j < 0 {
			break
		}
		cuerpo := strings.TrimSpace(resto[:j])
		resto = resto[j+4:]
		nombre, args, ok := strings.Cut(cuerpo, "\n")
		nombre = strings.TrimSpace(nombre)
		if !ok || !tools.Existe(nombre) {
			continue
		}
		argJSON := strings.TrimSpace(args)
		if argJSON == "" {
			argJSON = "{}"
		}
		if !json.Valid([]byte(argJSON)) {
			continue
		}
		out = append(out, agent.SolicitudHerramienta{Nombre: nombre, Argumentos: json.RawMessage(argJSON)})
	}
	return out
}

// cerrarTurnoGuardado persiste la respuesta del agente con su razonamiento,
// dejando el estado de la sesión como está (lo mueve `session` al terminar el
// flujo).
func cerrarTurnoGuardado(ad *Adaptador, sesionID, contenido, razon string) (*store.Message, error) {
	return ad.cerrar.CerrarAgente(sesionID, contenido, -1, -1, razon, "")
}

// aprobadorPorTurno implementa flow.Aprobador con el aprobador de producción:
// registra la petición, espera la decisión del panel y traduce el resultado.
type aprobadorPorTurno struct{ ad *Adaptador }

func (p *aprobadorPorTurno) Aprobar(ctx context.Context, descripcion string) (bool, error) {
	id, ch, err := p.ad.pedirPermiso(descripcion)
	if err != nil {
		return false, err
	}
	_ = id
	var estado string
	select {
	case estado = <-ch:
	case <-ctx.Done():
		p.ad.resolverCanal(id, store.ApprovalObsoleta)
		return false, nil
	}
	return estado == store.ApprovalAprobada, nil
}

// agenteBase carga los agentes base del proyecto (ai/agents/*.json) con
// fallback a un prompt mínimo si falta el archivo: el arranque no puede morir
// por un JSON ausente (CONFIGURATION.md §5: la vista sigue viva).
func agenteBase(raiz string) map[string]agent.Agente {
	out := map[string]agent.Agente{}
	for _, nombre := range []string{tools.AgentePlan, tools.AgenteBuild} {
		ruta := filepath.Join(raiz, "ai", "agents", nombre+".json")
		a, err := agent.Cargar(ruta)
		if err != nil {
			a = agent.Agente{
				Nombre: nombre,
				Prompt: "Eres el agente `" + nombre + "` de LocalCli. Responde en español, concreto y breve.",
				Skills: []string{},
			}
		}
		out[nombre] = a
	}
	return out
}
