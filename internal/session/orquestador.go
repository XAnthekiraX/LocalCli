package session

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"localcli/internal/agent"
	"localcli/internal/arranque"
	"localcli/internal/flow"
	"localcli/internal/linea"
	"localcli/internal/llama"
	"localcli/internal/llm"
	"localcli/internal/llm/llamacpp"
	"localcli/internal/llm/openai"
	"localcli/internal/models"
	"localcli/internal/queue"
	"localcli/internal/store"
	"localcli/internal/task"
	"localcli/internal/tools"
	"localcli/internal/tui"
)

// Orquestador implementa `tui.Puerto`: es el único que habla con el modelo por
// una sesión y el único que escribe en `store` por ella.
type Orquestador struct {
	db        *sql.DB
	projectID string
	cwd       string
	agentes   *agent.Catalogo
	llm       *openai.Cliente
	runtime   *llama.Servidor
	catalogo  *models.Gestor

	mu      sync.Mutex
	modelo  string
	ventana int
	pensar  bool
	// capacidades es el estado declarado por el motor para el modelo activo.
	capacidades llm.Capacidades
	// resolverCapacidades vuelve a consultar al motor cuando cambia el modelo; lo
	// inyecta el cableado, que es quien conoce la dirección del motor.
	resolverCapacidades func() llm.Capacidades

	// deshechos guarda, por sesión, los turnos deshechos pendientes de rehacer, en
	// orden de deshacer: el último es el primero en rehacerse.
	muDeshechos sync.Mutex
	deshechos   map[string][]store.Turno

	eventos   chan tui.Evento
	terminado chan struct{}
	cerrar    sync.Once

	// herramientas es el registro del catálogo con sus handlers ya inyectados y
	// catalogoHerramientas su vista para el agente. Sin ellos, el turno es de
	// conversación.
	herramientas         *tools.Registro
	catalogoHerramientas []agent.Herramienta

	// flows es el catálogo de flujos cargado; sin él, un `/comando` se responde
	// como chat.
	flows *flow.Catalogo

	// pendientes son las aprobaciones que esperan la decisión del usuario: su
	// canal recibe `true` para aprobar y `false` para declinar.
	muAprob    sync.Mutex
	pendientes map[string]chan bool

	// colas son las colas derivadas del TODO del proyecto, por capa, y su
	// derivación al arrancar (T-B018).
	muCola sync.Mutex
	colas  map[task.Capa]*queue.Cola

	// descargador baja modelos desde una URL directa al directorio global. Se
	// construye junto al catálogo; sin él no se puede descargar.
	descargador *models.Descargador

	// muDescarga protege el cancelador de la descarga en curso y su generación:
	// cancelar o reemplazar una descarga no debe poder tocar a la siguiente.
	muDescarga   sync.Mutex
	cancelarDesc context.CancelFunc
	descargaGen  uint64
}

// Nuevo construye el orquestador con el modelo activo y su ventana ya resueltos.
// `runtime` y `catalogo` sirven para arrancar `llama-server` al cambiar de
// modelo; pueden ir a nil cuando no se gestiona el servidor.
func Nuevo(db *sql.DB, projectID, cwd string, agentes *agent.Catalogo, llm *openai.Cliente, modelo string, ventana int, runtime *llama.Servidor, catalogo *models.Gestor) *Orquestador {
	o := &Orquestador{
		db:        db,
		projectID: projectID,
		cwd:       cwd,
		agentes:   agentes,
		llm:       llm,
		runtime:   runtime,
		catalogo:  catalogo,
		modelo:    modelo,
		ventana:   ventana,
		eventos:   make(chan tui.Evento, 1024),
		terminado: make(chan struct{}),
	}
	if catalogo != nil {
		o.descargador = models.NuevoDescargador(models.NuevoHF(), catalogo.DirectorioModels())
	}
	return o
}

// ConHerramientas inyecta el registro del catálogo y su vista para el agente, y
// conecta los eventos de herramienta al bus de la vista.
func (o *Orquestador) ConHerramientas(registro *tools.Registro, catalogo []agent.Herramienta) {
	o.herramientas = registro
	o.catalogoHerramientas = catalogo
	if registro == nil {
		return
	}
	registro.Eventos = tools.Eventos{
		Invocada: func(e tools.EventoInvocacion) {
			o.emitir(tui.Evento{Sesion: e.Sesion, Tipo: tui.EventoHerramientaInvocada, Datos: map[string]string{
				"herramienta": e.Nombre, "verbo": e.Verbo, "tema": e.Tema,
			}})
		},
		Resultado: func(e tools.EventoResultado) {
			datos := map[string]string{
				"herramienta": e.Nombre, "verbo": e.Verbo, "tema": e.Tema,
				"ok":       strconv.FormatBool(e.OK),
				"truncado": strconv.FormatBool(e.Truncado),
				"medida":   e.Medida,
				"duracion": strconv.Itoa(int(e.Duracion.Milliseconds())),
			}
			if e.Error != "" {
				datos["error"] = e.Error
			}
			if e.Bloque != "" {
				datos["bloque"] = e.Bloque
				datos["bloque_tipo"] = e.BloqueTipo
			}
			o.emitir(tui.Evento{Sesion: e.Sesion, Tipo: tui.EventoHerramientaResultado, Datos: datos})
			o.guardarLineaHerramienta(e)
		},
	}
	// Los handlers de la lista de pasos son de la sesión: cierran sobre el
	// orquestador para escribir la lista y publicar el evento que la repinta.
	registro.Instalar("crear_todo", o.crearTodo)
	registro.Instalar("actualizar_todo", o.actualizarTodo)
	// `ejecutar_flujo` también es de la sesión: su handler corre el flujo.
	registro.Instalar("ejecutar_flujo", o.ejecutarFlujo)
}

// guardarEventoChat guarda una pieza de pantalla de la sesión —una herramienta,
// una etapa de un flujo o la respuesta de una etapa que pidió verse en el chat—
// para que el hilo se recargue igual al retomarla. El hilo
// en vivo no depende de que se guarde, así que el fallo se avisa y el turno sigue
// (DATA_FLOW §Creación: «Crear una línea de procesamiento»).
func (o *Orquestador) guardarEventoChat(sesionID, tipo, contenido string, durationMs *int, bloque store.Bloque) {
	if _, err := store.CrearChatEventoConBloque(o.db, o.projectID, sesionID, tipo, contenido, durationMs, bloque); err != nil {
		o.avisar(sesionID, "no se pudo guardar una línea del hilo: "+err.Error(), "")
	}
}

// guardarLineaHerramienta guarda la línea cerrada de una herramienta, tal como se
// pintó, con la duración de su ejecución cuando se midió, y la llamada con su
// salida para reenviarla al modelo en los próximos turnos.
func (o *Orquestador) guardarLineaHerramienta(e tools.EventoResultado) {
	base := linea.HerramientaAbierta(e.Verbo, e.Tema)
	texto := linea.HerramientaCerrada(base, e.OK, e.Truncado, e.Medida, e.Error, e.Duracion)
	var ms *int
	if e.Duracion > 0 {
		n := int(e.Duracion.Milliseconds())
		ms = &n
	}
	if e.EnFlujo {
		// La herramienta interna de una etapa se pinta en el sub-proceso, pero
		// no se reenvía: el resultado del `ejecutar_flujo` ya la resume y su
		// llamada no pertenece al turno del agente principal.
		o.guardarEventoChat(e.Sesion, store.ChatEventoHerramienta, texto, ms, bloqueDeEvento(e))
		return
	}
	ejecucion := store.EjecucionHerramienta{
		CallID:      e.IDLlamada,
		Herramienta: e.Nombre,
		Argumentos:  e.Argumentos,
		Salida:      e.Salida,
		Error:       e.Error,
		OK:          e.OK,
		Truncado:    e.Truncado,
	}
	if _, err := store.CrearChatEventoToolConBloque(o.db, o.projectID, e.Sesion, texto, ms, bloqueDeEvento(e), ejecucion); err != nil {
		o.avisar(e.Sesion, "no se pudo guardar una línea del hilo: "+err.Error(), "")
	}
}

// bloqueDeEvento traduce el bloque visual del evento al tipo del almacén.
func bloqueDeEvento(e tools.EventoResultado) store.Bloque {
	return store.Bloque{Texto: e.Bloque, Tipo: e.BloqueTipo}
}

// pintarRespuestaEtapa deja la respuesta completa de una etapa de un flujo en el
// hilo de la sesión: la guarda en `chat_evento` con su tipo propio —para que el
// hilo se recupere igual al retomarlo— y publica el evento que la vista pinta
// como bloque de chat, después de lo que la etapa ya dejó en el hilo y antes del
// aviso de etapa terminada. No es conversación: su texto nunca entra al contexto
// del modelo (SPEC-ORQUESTADOR-FLUJOS §Visibilidad de una respuesta).
func (o *Orquestador) pintarRespuestaEtapa(sesionID, texto string) {
	o.guardarEventoChat(sesionID, store.ChatEventoRespuesta, texto, nil, store.Bloque{})
	o.emitir(tui.Evento{Sesion: sesionID, Tipo: tui.EventoEtapaRespuesta, Datos: map[string]string{"texto": texto}})
}

// ConFlujos inyecta el catálogo de flujos cargado del disco.
func (o *Orquestador) ConFlujos(flows *flow.Catalogo) { o.flows = flows }

// Flujos sirve el catálogo de flujos a la paleta de la vista (tui.Puerto): el
// comando sin la barra —lo que sigue a `flow:`—.
func (o *Orquestador) Flujos() []tui.FlujoVista {
	if o.flows == nil {
		return nil
	}
	flujos := o.flows.Flujos()
	vistas := make([]tui.FlujoVista, 0, len(flujos))
	for _, f := range flujos {
		vistas = append(vistas, tui.FlujoVista{Comando: strings.TrimPrefix(f.Comando, "/")})
	}
	return vistas
}

// catalogoDelTurno deriva el catálogo efectivo del agente y, si puede lanzar
// algún flujo, añade `ejecutar_flujo` con el enum de comandos disponibles.
func (o *Orquestador) catalogoDelTurno(agenteActivo agent.Agente) []agent.Herramienta {
	efectivo := agent.CatalogoEfectivo(agenteActivo, o.catalogoHerramientas)
	comandos := o.comandosLanzables(agenteActivo.Permisos())
	if len(comandos) == 0 {
		return efectivo
	}
	for _, h := range o.catalogoHerramientas {
		if h.Nombre != "ejecutar_flujo" {
			continue
		}
		h.Esquema = esquemaConEnum(h.Esquema, comandos)
		efectivo = append(efectivo, h)
	}
	return efectivo
}

// comandosLanzables lista los comandos de los flujos que un agente con esos
// permisos puede lanzar. Como los flujos ya no declaran agente por etapa, no hay
// forma de clasificarlos por permisos de antemano: cualquier agente que pueda
// usar `ejecutar_flujo` ve todos, y los problemas de permiso se resuelven en la
// etapa que los necesita.
func (o *Orquestador) comandosLanzables(permisos []string) []string {
	if o.flows == nil || len(permisos) == 0 {
		return nil
	}
	var comandos []string
	for _, f := range o.flows.Flujos() {
		comandos = append(comandos, f.Comando)
	}
	return comandos
}

// esquemaConEnum fija el enum de comandos en la propiedad `flujo` del esquema de
// `ejecutar_flujo`, que se arma por turno.
func esquemaConEnum(crudo json.RawMessage, comandos []string) json.RawMessage {
	var e tools.Esquema
	if err := json.Unmarshal(crudo, &e); err != nil || e.Propiedades == nil {
		return crudo
	}
	prop, ok := e.Propiedades["flujo"]
	if !ok {
		return crudo
	}
	prop.Enum = comandos
	e.Propiedades["flujo"] = prop
	datos, err := json.Marshal(e)
	if err != nil {
		return crudo
	}
	return datos
}

// adaptadorHerramientas conecta la capa universal de tools con el bucle del
// agente. Se construye por turno con la sesión, los permisos del agente activo y
// el canal de aprobación. `Observar` —opcional— ve cada resultado tal cual salió
// del registro, para que un flujo pueda reaccionar a una herramienta denegada.
type adaptadorHerramientas struct {
	registro *tools.Registro
	sesion   string
	permisos []string
	Ask      func(context.Context, tools.Solicitud) (tools.Decision, error)
	Observar func(nombre string, res agent.Resultado)
	// enFlujo marca las herramientas que corren dentro de una etapa: se pintan,
	// pero no se persisten como ejecución reenviable.
	enFlujo bool
}

func (a adaptadorHerramientas) Ejecutar(ctx context.Context, agente string, l agent.Llamada) (agent.Resultado, error) {
	res := a.registro.Ejecutar(ctx, tools.Peticion{
		Sesion:     a.sesion,
		Agente:     agente,
		Permisos:   a.permisos,
		Ask:        a.Ask,
		IDLlamada:  l.ID,
		EnFlujo:    a.enFlujo,
		Nombre:     l.Nombre,
		Argumentos: json.RawMessage(l.Argumentos),
	})
	resultado := agent.Resultado{Salida: res.Salida, Error: res.Error}
	if a.Observar != nil {
		a.Observar(l.Nombre, resultado)
	}
	return resultado, nil
}

// ResponderAprobacion entrega la decisión del usuario a la sesión que espera esa
// aprobación. Una aprobación que ya no espera —porque se canceló o se resolvió—
// no hace nada.
func (o *Orquestador) ResponderAprobacion(sesionID, aprobacionID string, aprobar bool) error {
	o.muAprob.Lock()
	canal, ok := o.pendientes[aprobacionID]
	o.muAprob.Unlock()
	if !ok {
		return fmt.Errorf("session: la aprobación %s ya no espera decisión", aprobacionID)
	}
	select {
	case canal <- aprobar:
	default:
	}
	return nil
}

// pedirAprobacion pausa el turno hasta que el usuario decide: registra la
// aprobación, avisa a la vista, espera la decisión y deja la sesión como estaba.
func (o *Orquestador) pedirAprobacion(ctx context.Context, sesionID string, s tools.Solicitud) (tools.Decision, error) {
	id := uuid.New().String()
	descripcion := s.Titulo
	if s.Descripcion != "" {
		descripcion += ": " + s.Descripcion
	}

	canal := o.registrarPendiente(id)
	defer o.soltarPendiente(id)

	if _, err := store.CrearAprobacion(o.db, o.projectID, sesionID, descripcion); err != nil {
		return tools.Declinar, err
	}
	_ = o.marcarEstado(sesionID, EstadoEsperandoPermiso)
	o.emitir(tui.Evento{Sesion: sesionID, Tipo: tui.EventoPeticionAprobacion, Datos: map[string]string{
		"aprobacion": id, "descripcion": descripcion,
	}})

	var aprobar bool
	select {
	case aprobar = <-canal:
	case <-ctx.Done():
		o.cerrarAprobacion(sesionID, id, "declinada")
		return tools.Declinar, ctx.Err()
	}

	estado := "declinada"
	if aprobar {
		estado = "aprobada"
	}
	o.cerrarAprobacion(sesionID, id, estado)
	_ = o.marcarEstado(sesionID, EstadoTrabajando)
	if aprobar {
		return tools.Aprobar, nil
	}
	return tools.Declinar, nil
}

// cerrarAprobacion persiste el desenlace y retira la línea de la vista.
func (o *Orquestador) cerrarAprobacion(sesionID, id, estado string) {
	_ = store.ResolverAprobacion(o.db, o.projectID, sesionID, id, estado)
	o.emitir(tui.Evento{Sesion: sesionID, Tipo: tui.EventoAprobacionResuelta, Datos: map[string]string{"aprobacion": id}})
}

// registrarPendiente da de alta el canal de decisión de una aprobación.
func (o *Orquestador) registrarPendiente(id string) chan bool {
	o.muAprob.Lock()
	defer o.muAprob.Unlock()
	if o.pendientes == nil {
		o.pendientes = map[string]chan bool{}
	}
	canal := make(chan bool, 1)
	o.pendientes[id] = canal
	return canal
}

// soltarPendiente retira el canal: si llegara una decisión tardía, no encuentra
// a quién entregársela.
func (o *Orquestador) soltarPendiente(id string) {
	o.muAprob.Lock()
	delete(o.pendientes, id)
	o.muAprob.Unlock()
}

// Agentes devuelve los nombres de los agentes disponibles.
func (o *Orquestador) Agentes() []string { return o.agentes.Nombres() }

// AgenteRecordado devuelve el último agente elegido por el usuario.
func (o *Orquestador) AgenteRecordado() string {
	prefs, err := tui.CargarPreferencias()
	if err != nil {
		return ""
	}
	return prefs.Agente
}

// ModeloActual es el modelo activo.
func (o *Orquestador) ModeloActual() string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.modelo
}

// MotorActual es el runtime que sirve el modelo.
func (o *Orquestador) MotorActual() string { return "llama.cpp" }

// ConCapacidades fija el estado inicial de las capacidades del modelo activo.
func (o *Orquestador) ConCapacidades(c llm.Capacidades) {
	o.mu.Lock()
	o.capacidades = c
	o.mu.Unlock()
}

// ConResolucionCapacidades inyecta el resolutor que vuelve a consultar al motor
// cuando el modelo cambia.
func (o *Orquestador) ConResolucionCapacidades(fn func() llm.Capacidades) {
	o.resolverCapacidades = fn
}

// Capacidades devuelve las capacidades conocidas del modelo activo para la vista.
func (o *Orquestador) Capacidades() tui.Capacidades {
	return tui.Capacidades{Vision: capacidadAVista(o.capacidadesActuales().Vision)}
}

// capacidadesActuales lee el estado de capacidades del modelo activo.
func (o *Orquestador) capacidadesActuales() llm.Capacidades {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.capacidades
}

// refrescarCapacidades vuelve a consultar al motor y publica el estado de visión
// del modelo para que la vista actualice su chip. No pasa nada si no hay
// resolutor: las capacidades quedan desconocidas.
func (o *Orquestador) refrescarCapacidades() {
	if o.resolverCapacidades == nil {
		return
	}
	c := o.resolverCapacidades()
	o.mu.Lock()
	o.capacidades = c
	o.mu.Unlock()
	o.emitir(tui.Evento{Tipo: tui.EventoCapacidadesModelo, Datos: map[string]string{
		"vision": estadoCapacidad(c.Vision),
	}})
}

// capacidadAVista traduce el estado de una capacidad de la frontera del motor a
// la de la vista.
func capacidadAVista(c llm.Capacidad) tui.Capacidad {
	switch c {
	case llm.Soportada:
		return tui.CapacidadSoportada
	case llm.NoSoportada:
		return tui.CapacidadNoSoportada
	default:
		return tui.CapacidadDesconocida
	}
}

// estadoCapacidad rotula el estado de una capacidad para el payload del evento.
func estadoCapacidad(c llm.Capacidad) string {
	switch c {
	case llm.Soportada:
		return "soportada"
	case llm.NoSoportada:
		return "no_soportada"
	default:
		return "desconocida"
	}
}

// FijarPensar deja si el chat pide al modelo que razone antes de responder.
func (o *Orquestador) FijarPensar(pensar bool) {
	o.mu.Lock()
	o.pensar = pensar
	o.mu.Unlock()
}

// pensarActual dice si el chat deja razonar al modelo.
func (o *Orquestador) pensarActual() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.pensar
}

// Carpeta es la carpeta del proyecto abierta.
func (o *Orquestador) Carpeta() string { return o.cwd }

// Git es el estado del repositorio del proyecto.
func (o *Orquestador) Git() tui.Git { return leerGit(o.cwd) }

// Modelos lista los modelos descargados del catálogo, por su nombre (carpeta).
// Solo se ofrecen los que tienen su GGUF en disco: un modelo catalogado con el
// archivo borrado no es usable.
func (o *Orquestador) Modelos() ([]tui.ModeloLocal, error) {
	if o.catalogo == nil {
		return nil, nil
	}
	c, err := o.catalogo.Cargar()
	if err != nil {
		return nil, err
	}
	salida := make([]tui.ModeloLocal, 0, len(c.Modelos))
	for _, e := range c.Modelos {
		if !e.Usable() {
			continue
		}
		salida = append(salida, tui.ModeloLocal{Nombre: e.ID, Motor: "llama.cpp"})
	}
	return salida, nil
}

// DescargarModelo arranca la descarga de un modelo desde una URL directa de
// GGUF y, opcionalmente, su proyector multimodal (mmproj). Valida las URLs y
// arranca el trabajo en segundo plano; el avance llega por el bus
// (`EventoDescargaProgreso`, `EventoDescargaTerminada`, `EventoDescargaFallida`).
// Una descarga en curso se cancela al arrancar otra.
func (o *Orquestador) DescargarModelo(raw, mmprojRaw string) error {
	if o.descargador == nil || o.catalogo == nil {
		return fmt.Errorf("session: no hay catálogo de modelos")
	}
	v, err := models.ParsearURL(raw)
	if err != nil {
		return err
	}
	nombre := models.NombreDeURL(raw)
	if nombre == "" {
		return fmt.Errorf("session: no se pudo derivar el nombre del modelo")
	}

	var mmprojNombre string
	if strings.TrimSpace(mmprojRaw) != "" {
		mmprojNombre = models.NombreDeURL(mmprojRaw)
		if mmprojNombre == "" {
			return fmt.Errorf("session: no se pudo derivar el nombre del mmproj")
		}
		if !esGGUFURL(mmprojRaw) {
			return fmt.Errorf("session: el mmproj debe ser un archivo .gguf")
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	o.muDescarga.Lock()
	if o.cancelarDesc != nil {
		o.cancelarDesc()
	}
	o.descargaGen++
	gen := o.descargaGen
	o.cancelarDesc = cancel
	o.muDescarga.Unlock()

	go o.correrDescarga(ctx, gen, v, nombre, mmprojRaw, mmprojNombre)
	return nil
}

// CancelarDescarga corta la descarga en curso, si la hay. No publica nada: la
// vista ya decidió cerrar y el goroutine de la descarga enmudece al dejar de ser
// la vigente.
func (o *Orquestador) CancelarDescarga() {
	o.muDescarga.Lock()
	cancel := o.cancelarDesc
	o.cancelarDesc = nil
	o.descargaGen++
	o.muDescarga.Unlock()
	if cancel != nil {
		cancel()
	}
}

// NombreDeURL deriva el nombre del modelo de una URL de descarga, para que la
// vista lo anticipe antes de empezar.
func (o *Orquestador) NombreDeURL(raw string) string { return models.NombreDeURL(raw) }

// correrDescarga baja la variante (y opcionalmente el mmproj) y registra el
// modelo al terminar. Solo publica eventos mientras siga siendo la descarga
// vigente: una cancelada o reemplazada calla.
func (o *Orquestador) correrDescarga(ctx context.Context, gen uint64, v models.Variante, nombre, mmprojRaw, mmprojNombre string) {
	ultimoPct := -2
	var ultimoHecho int64
	desc := models.NuevoDescargador(models.NuevoHF(), o.catalogo.DirectorioModels())
	desc.ConProgreso(func(hecho, total int64) {
		if !o.descargaVigente(gen) {
			return
		}
		pct := porcentajeDescarga(hecho, total)
		if pct >= 0 {
			if pct == ultimoPct {
				return
			}
			ultimoPct = pct
		} else if hecho-ultimoHecho < 1<<20 {
			// Sin total conocido se avisa cada mebibyte, no en cada bloque.
			return
		}
		ultimoHecho = hecho
		o.emitir(tui.Evento{Tipo: tui.EventoDescargaProgreso, Datos: map[string]string{
			"nombre": nombre, "hecho": strconv.FormatInt(hecho, 10), "total": strconv.FormatInt(total, 10),
		}})
	})
	ruta, err := desc.DescargarURL(ctx, v, nombre)
	if !o.descargaVigente(gen) {
		return
	}
	if err != nil {
		if ctx.Err() != nil {
			return // cancelada: la vista ya lo sabe
		}
		o.emitir(tui.Evento{Tipo: tui.EventoDescargaFallida, Datos: map[string]string{
			"nombre": nombre, "error": err.Error(),
		}})
		return
	}

	// Si hay mmproj, descargarlo en el mismo directorio del modelo
	var mmprojArchivo string
	if strings.TrimSpace(mmprojRaw) != "" {
		mmprojVariante := models.Variante{Archivo: mmprojNombre, URL: mmprojRaw}
		mmprojRuta, err := desc.DescargarURL(ctx, mmprojVariante, nombre)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			o.emitir(tui.Evento{Tipo: tui.EventoDescargaFallida, Datos: map[string]string{
				"nombre": nombre, "error": "mmproj: " + err.Error(),
			}})
			return
		}
		mmprojArchivo = filepath.Base(mmprojRuta)
	}

	if err := o.catalogo.AltaDescarga(nombre, v, ruta, 0, mmprojArchivo); err != nil {
		o.emitir(tui.Evento{Tipo: tui.EventoDescargaFallida, Datos: map[string]string{
			"nombre": nombre, "error": err.Error(),
		}})
		return
	}
	o.emitir(tui.Evento{Tipo: tui.EventoDescargaTerminada, Datos: map[string]string{
		"nombre": nombre, "ruta": ruta,
	}})
}

// descargaVigente dice si la descarga de esa generación sigue siendo la actual.
func (o *Orquestador) descargaVigente(gen uint64) bool {
	o.muDescarga.Lock()
	defer o.muDescarga.Unlock()
	return o.descargaGen == gen
}

// esGGUFURL verifica si la URL apunta a un archivo .gguf (ignora query params).
func esGGUFURL(raw string) bool {
	u := strings.TrimSpace(raw)
	if u == "" {
		return false
	}
	if i := strings.IndexByte(u, '?'); i >= 0 {
		u = u[:i]
	}
	if i := strings.IndexByte(u, '#'); i >= 0 {
		u = u[:i]
	}
	return strings.HasSuffix(strings.ToLower(u), ".gguf")
}

// porcentajeDescarga es el avance en tanto por ciento; -1 si no se conoce el
// total.
func porcentajeDescarga(hecho, total int64) int {
	if total <= 0 {
		return -1
	}
	return int(hecho * 100 / total)
}

// FijarModelo deja el modelo elegido como el activo y arranca `llama-server`
// con él. El reinicio corre en segundo plano y avisa por el bus: la interfaz no
// se bloquea mientras el servidor carga.
func (o *Orquestador) FijarModelo(nombre string) {
	o.mu.Lock()
	o.modelo = nombre
	o.mu.Unlock()
	// La elección vive en el catálogo, no en las preferencias de la vista: así el
	// próximo arranque resuelve el mismo modelo activo. Si no está en el catálogo,
	// `aplicarAlServidor` lo avisa y no se persiste nada.
	if o.catalogo != nil {
		_ = o.catalogo.Seleccionar(nombre)
	}
	o.aplicarAlServidor(nombre)
}

// aplicarAlServidor levanta el servidor con el modelo elegido. Si el modelo no
// está en el catálogo, lo avisa sin tocar el servidor.
func (o *Orquestador) aplicarAlServidor(nombre string) {
	if o.runtime == nil || o.catalogo == nil {
		return
	}
	entrada, ok := o.modeloDelCatalogo(nombre)
	if !ok {
		o.avisarSinSesion("el modelo "+nombre+" no está en el catálogo", "descárgalo para poder usarlo")
		return
	}
	if activo, ok := o.runtime.Activo(); ok && activo.Nombre == entrada.ID {
		return
	}
	contexto := contextoDeModelo(entrada)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		avisos, err := o.runtime.Reiniciar(ctx, llama.Modelo{
			Nombre: entrada.ID, Ruta: entrada.RutaReal(), MMProj: entrada.Proyector(), Contexto: contexto,
		})
		if err != nil {
			o.avisarSinSesion("no se pudo arrancar "+entrada.ID+": "+err.Error(), "revisa llama-server")
			return
		}
		// La ventana efectiva es la que el servidor acepta de verdad: si reutilizó
		// uno externo, su `n_ctx` manda sobre el contexto pedido.
		nueva := o.ventanaDelServidor(contexto)
		antigua := o.ventanaActual()
		o.fijarVentana(nueva)
		for _, aviso := range avisos {
			o.avisarSinSesion(aviso, "")
		}
		if nueva < antigua {
			o.avisarSinSesion("modelo "+entrada.ID+": ventana reducida a "+strconv.Itoa(nueva)+"; los historiales más largos se compactarán", "")
		}
		o.avisarSinSesion("modelo activo: "+entrada.ID, "")
		o.refrescarCapacidades()
	}()
}

// contextoDeModelo resuelve el contexto con el que se arranca el servidor para un
// modelo: el override del usuario, si no el contexto real de su GGUF y, si no lo
// declara, el defecto del harness.
func contextoDeModelo(e models.Entrada) int {
	return llm.VentanaDeArranque(arranque.LimiteContextoOverride(), e.ContextoReal(), arranque.LimiteContextoPorDefecto)
}

// ventanaDelServidor es la ventana que el servidor acepta de verdad (su `n_ctx`
// por `/props`); si no se puede leer, el contexto pedido.
func (o *Orquestador) ventanaDelServidor(contextoDeseado int) int {
	if o.llm != nil {
		if v := llamacpp.VentanaDe(context.Background(), o.llm.BaseURL()); v > 0 {
			return v
		}
	}
	return contextoDeseado
}

// modeloDelCatalogo busca el modelo por su nombre en el catálogo.
func (o *Orquestador) modeloDelCatalogo(nombre string) (models.Entrada, bool) {
	c, err := o.catalogo.Cargar()
	if err != nil {
		return models.Entrada{}, false
	}
	return c.Buscar(nombre)
}

// avisarSinSesion publica una notificación no ligada a una sesión concreta.
func (o *Orquestador) avisarSinSesion(motivo, siguiente string) {
	datos := map[string]string{"motivo": motivo}
	if siguiente != "" {
		datos["siguiente"] = siguiente
	}
	o.emitir(tui.Evento{Tipo: tui.EventoNotificacion, Datos: datos})
}

// Listar devuelve las sesiones del proyecto.
func (o *Orquestador) Listar() ([]tui.Sesion, error) {
	sesiones, err := store.ListarSesiones(o.db, o.projectID)
	if err != nil {
		return nil, err
	}
	salida := make([]tui.Sesion, len(sesiones))
	for i := range sesiones {
		salida[i] = DeStore(&sesiones[i]).AVista()
	}
	return salida, nil
}

// Eliminar borra una sesión del proyecto y, en cascada, todo lo suyo.
func (o *Orquestador) Eliminar(sesionID string) error {
	return store.EliminarSesion(o.db, o.projectID, sesionID)
}

// Historial devuelve el hilo guardado de una sesión, traducido a lo que la vista
// pinta al retomarla. Es la lectura que hace visible la conversación al reabrir la
// carpeta: los mensajes se guardan turno a turno en `Enviar` y las líneas del
// sistema —herramientas y sub-procesos— en `chat_evento`, y aquí se intercalan en
// su orden cronológico para que el hilo se lea igual que en vivo (DATA_FLOW
// §Creación, INTERFACES §1.1).
func (o *Orquestador) Historial(sesionID string) ([]tui.PiezaHistorial, error) {
	mensajes, err := store.Historial(o.db, o.projectID, sesionID)
	if err != nil {
		return nil, err
	}
	eventos, err := store.ListarChatEventos(o.db, o.projectID, sesionID)
	if err != nil {
		return nil, err
	}

	salida := make([]tui.PiezaHistorial, 0, len(mensajes)+len(eventos))
	i, j := 0, 0
	for i < len(mensajes) || j < len(eventos) {
		// Se elige el más antiguo; a igual instante gana el mensaje, porque la
		// línea se interpone entre el mensaje que la precede y el que la sigue.
		tomarMensaje := i < len(mensajes) && (j >= len(eventos) || !despues(mensajes[i].CreatedAt, eventos[j].CreatedAt))
		if tomarMensaje {
			f := mensajes[i]
			m := &tui.MensajeHistorial{ID: f.ID, Rol: f.Role, Texto: f.Content}
			if f.Razonamiento != nil {
				m.Razonamiento = *f.Razonamiento
			}
			if f.DurationMs != nil {
				m.Duracion = time.Duration(*f.DurationMs) * time.Millisecond
			}
			salida = append(salida, m)
			i++
			continue
		}
		e := eventos[j]
		salida = append(salida, &tui.LineaHistorial{
			Tipo: e.Tipo, Texto: e.Content, Bloque: e.Bloque, BloqueTipo: e.BloqueTipo,
		})
		j++
	}
	return salida, nil
}

// despues dice si el instante `a` es posterior a `b`. Los tiempos se guardan en
// ISO 8601 UTC; mensajes con precisión de segundo y líneas con nanosegundos, así
// que se comparan ya interpretados. Un instante ilegible cuenta como «no
// posterior», para no adelantar una línea sin motivo.
func despues(a, b string) bool {
	ta, errA := time.Parse(time.RFC3339Nano, a)
	tb, errB := time.Parse(time.RFC3339Nano, b)
	if errA != nil || errB != nil {
		return false
	}
	return ta.After(tb)
}

// Crear abre una sesión nueva. Nace con el nombre provisional; el modelo lo
// sustituye por un título con su primera petición.
func (o *Orquestador) Crear() (*tui.Sesion, error) {
	s, err := store.CrearSesion(o.db, o.projectID, nombreProvisional, nil)
	if err != nil {
		return nil, err
	}
	vista := DeStore(s).AVista()
	return &vista, nil
}

// Suscribir entrega el bus de eventos y la función que lo cierra.
func (o *Orquestador) Suscribir() (<-chan tui.Evento, func()) {
	return o.eventos, func() { o.cerrar.Do(func() { close(o.terminado) }) }
}

// Enviar ejecuta un turno: persiste lo escrito, arma el historial y corre el
// ciclo conversacional del agente, publicando el avance por eventos.
func (o *Orquestador) Enviar(ctx context.Context, sesionID, agenteNombre, texto string) error {
	if err := o.marcarEstado(sesionID, EstadoTrabajando); err != nil {
		o.avisar(sesionID, "no se pudo abrir el turno: "+err.Error(), "")
		return err
	}

	// Un turno nuevo invalida lo deshecho: rehacerlo lo dejaría intercalado.
	o.olvidarDeshechos(sesionID)

	agenteActivo, ok := o.agentes.Buscar(agenteNombre)
	if !ok {
		o.avisar(sesionID, "agente desconocido: "+agenteNombre, "elige otro con Tab")
		o.cerrarTurno(sesionID, true)
		return fmt.Errorf("session: agente desconocido %q", agenteNombre)
	}

	mensaje, err := store.CrearMensaje(o.db, o.projectID, sesionID, "user", texto, nil, nil, nil)
	if err != nil {
		o.avisar(sesionID, "no se pudo guardar el mensaje: "+err.Error(), "")
		o.cerrarTurno(sesionID, true)
		return err
	}
	// El id recién guardado viaja a la vista para que la burbuja viva del
	// usuario lo adopte: el menú del globo (revertir/copiar/bifurcar) solo se
	// abre sobre un mensaje con identidad en la base.
	o.emitir(tui.Evento{Sesion: sesionID, Tipo: tui.EventoMensajePersistido, Datos: map[string]string{
		"mensaje": mensaje.ID,
		"rol":     "user",
	}})
	o.titular(ctx, sesionID, texto)

	// Un comando de flujo arranca etapas en vez de responder como chat.
	eleccion := o.elegirFlujo(texto)
	if eleccion.sinPeticion {
		o.avisar(sesionID, "el flujo "+eleccion.flujo.Comando+" necesita una petición",
			"escríbelo así: "+prefijoFlujo+strings.TrimPrefix(eleccion.flujo.Comando, "/")+" <qué debe resolver>")
		o.cerrarTurno(sesionID, false)
		return nil
	}
	if eleccion.esFlujo {
		return o.turnoDeFlujo(ctx, sesionID, agenteActivo, eleccion.flujo, eleccion.peticion)
	}

	// Una petición con lista ordenada propone un TODO; no arranca la cola ni
	// crea la lista de pasos (BUSINESS_RULES §Detección de trabajo ordenado).
	o.proponerTodo(sesionID, texto)

	historial, err := o.historialPara(ctx, sesionID)
	if err != nil {
		o.avisar(sesionID, "no se pudo leer el historial: "+err.Error(), "")
		o.cerrarTurno(sesionID, true)
		return err
	}

	// Las imágenes que el usuario escribió en la petición acompañan a ese turno;
	// son efímeras y solo las ve el modelo (DATA_FLOW §Creación).
	imagenes := o.imagenesDelTurno(ctx, sesionID, texto)
	if len(imagenes) > 0 {
		historial = adjuntarAlUltimoUsuario(historial, imagenes, texto)
	}

	inicio := time.Now()
	ejecutor := o.nuevoEjecutorDeTurno(agenteActivo, sesionID)
	// Lo que el modelo vaya entregando se acumula por si el turno se detiene a
	// mitad: lo parcial no se pierde, se guarda como su respuesta.
	resultado, parcial, err := o.correrTurno(ctx, sesionID, ejecutor, historial)
	if err != nil && esErrorDeVentana(err) && o.compactacionAuto() {
		// El prompt no cupo en la ventana pese a la estimación: se compacta a la
		// fuerza y se reintenta una sola vez antes de darse por vencido.
		if hist2, ok := o.recompactar(ctx, sesionID, imagenes, texto); ok {
			ejecutor = o.nuevoEjecutorDeTurno(agenteActivo, sesionID)
			resultado, parcial, err = o.correrTurno(ctx, sesionID, ejecutor, hist2)
		}
	}
	duracion := int(time.Since(inicio).Milliseconds())
	if err != nil {
		if errors.Is(err, context.Canceled) {
			// El usuario detuvo el turno: no es un fallo, la sesión queda inactiva y
			// lo que el modelo ya entregó se conserva, para que al reabrir la sesión
			// el hilo siga mostrando lo que se vio en vivo.
			o.guardarParcial(sesionID, strings.Join(parcial, ""), duracion)
			o.avisar(sesionID, "turno detenido", "")
			_ = o.marcarEstado(sesionID, EstadoInactiva)
			return err
		}

		// Mejorar mensaje para errores del servidor LLM.
		motivo, siguiente := o.mejorarErrorLLM(err)
		o.avisar(sesionID, motivo, siguiente)
		o.cerrarTurno(sesionID, true)
		return err
	}

	prompt, completion := resultado.Uso.Entrada, resultado.Uso.Salida
	if _, err := store.CrearMensaje(o.db, o.projectID, sesionID, "agent", resultado.Texto, &prompt, &completion, &duracion); err != nil {
		o.avisar(sesionID, "no se pudo guardar la respuesta: "+err.Error(), "")
	}

	if completion > 0 || prompt > 0 {
		o.emitir(tui.Evento{Sesion: sesionID, Tipo: tui.EventoTokensTurno, Datos: map[string]string{
			"salida":   strconv.Itoa(completion),
			"contexto": strconv.Itoa(prompt),
			"limite":   strconv.Itoa(o.ventanaActual()),
		}})
	}

	o.cerrarTurno(sesionID, false)
	return nil
}

// nuevoEjecutorDeTurno arma el ejecutor del agente activo para un turno de chat,
// con el catálogo de herramientas y el canal de aprobación de la sesión.
func (o *Orquestador) nuevoEjecutorDeTurno(agenteActivo agent.Agente, sesionID string) *agent.Ejecutor {
	ejecutor := &agent.Ejecutor{
		Agente:    agenteActivo,
		Motor:     motorOpenAI{cliente: o.llm, modelo: o.modeloActual, sinRazonamiento: func() bool { return !o.pensarActual() }},
		Capacidad: llm.Soportada,
		Ventana:   o.ventanaActual(),
		Avisar:    func(motivo, siguiente string) { o.avisar(sesionID, motivo, siguiente) },
	}
	if o.herramientas != nil {
		ejecutor.Cerrado = o.catalogoDelTurno(agenteActivo)
		ejecutor.Herramientas = adaptadorHerramientas{
			registro: o.herramientas,
			sesion:   sesionID,
			permisos: agenteActivo.Permisos(),
			Ask: func(ctx context.Context, s tools.Solicitud) (tools.Decision, error) {
				return o.pedirAprobacion(ctx, sesionID, s)
			},
		}
	}
	return ejecutor
}

// correrTurno ejecuta el ciclo del agente emitiendo lo que el modelo entrega y
// acumulando el texto parcial por si el turno se detiene a mitad.
func (o *Orquestador) correrTurno(ctx context.Context, sesionID string, ejecutor *agent.Ejecutor, historial []agent.Mensaje) (agent.Respuesta, []string, error) {
	var parcial []string
	resultado, err := ejecutor.Turno(ctx, historial, func(d agent.Delta) error {
		datos := map[string]string{"texto": d.Texto}
		if d.Razonamiento {
			datos["razonamiento"] = "true"
		} else {
			parcial = append(parcial, d.Texto)
		}
		o.emitir(tui.Evento{Sesion: sesionID, Tipo: tui.EventoToken, Datos: datos})
		return nil
	})
	return resultado, parcial, err
}

// recompactar fuerza una compactación más agresiva —la mitad del tramo reciente—
// para liberar ventana y volver a armar el historial del turno. Es el reintento
// ante un desbordamiento real del servidor ([[specs/SPEC-HISTORIAL-CONVERSACION]]).
func (o *Orquestador) recompactar(ctx context.Context, sesionID string, imagenes []agent.Imagen, texto string) ([]agent.Mensaje, bool) {
	filas, err := store.ListarMensajes(o.db, o.projectID, sesionID)
	if err != nil {
		return nil, false
	}
	conv := conversacionDe(filas)
	if len(conv) == 0 {
		return nil, false
	}
	ejecs, err := o.ejecucionesPara(sesionID)
	if err != nil {
		return nil, false
	}
	prefs := preferenciasSeguras()
	recencia := prefs.Compactacion.ConservarEfectivo() / 2
	if recencia < 1 {
		recencia = 1
	}
	corte, resumen := o.compactar(ctx, sesionID, conv, recencia, prefs)
	if corte <= 0 {
		return nil, false
	}
	historial := armarHistorialConHerramientas(conv, corte, resumen, ejecs)
	if len(imagenes) > 0 {
		historial = adjuntarAlUltimoUsuario(historial, imagenes, texto)
	}
	return historial, true
}

// compactacionAuto dice si el usuario tiene encendida la compactación automática.
func (o *Orquestador) compactacionAuto() bool {
	return preferenciasSeguras().Compactacion.AutoActivo()
}

// esErrorDeVentana reconoce el rechazo del servidor por exceso de contexto: un
// 4xx cuyo mensaje habla de la ventana o del tamaño de la petición.
func esErrorDeVentana(err error) bool {
	var eErr *openai.ErrorServidor
	if !errors.As(err, &eErr) {
		return false
	}
	if eErr.Codigo < 400 || eErr.Codigo >= 500 {
		return false
	}
	m := strings.ToLower(eErr.Mensaje)
	return strings.Contains(m, "exceeds the available context") ||
		strings.Contains(m, "context length") ||
		strings.Contains(m, "context size") ||
		strings.Contains(m, "n_ctx")
}

// guardarParcial conserva lo que el modelo entregó antes de detenerse: lo guarda
// como su respuesta, con el tiempo que duró el turno. Sin texto no hay nada que
// guardar, y un fallo al escribir no tumba el turno ya detenido.
func (o *Orquestador) guardarParcial(sesionID, texto string, duracionMs int) {
	if strings.TrimSpace(texto) == "" {
		return
	}
	if _, err := store.CrearMensaje(o.db, o.projectID, sesionID, "agent", texto, nil, nil, &duracionMs); err != nil {
		o.avisar(sesionID, "no se pudo guardar la respuesta parcial: "+err.Error(), "")
	}
}

// estadoDe lee el estado actual de una sesión del proyecto.
func (o *Orquestador) estadoDe(sesionID string) (string, error) {
	s, err := store.ObtenerSesion(o.db, o.projectID, sesionID)
	if err != nil {
		return "", err
	}
	return s.Status, nil
}

// emitirEstado publica el cambio de estado de una sesión.
func (o *Orquestador) emitirEstado(sesionID, estado string) {
	o.emitir(tui.Evento{Sesion: sesionID, Tipo: tui.EventoEstadoSesion, Datos: map[string]string{"estado": estado}})
}

// marcarEstado valida, persiste y publica el estado; desde un final pasa por inactiva.
func (o *Orquestador) marcarEstado(sesionID, nuevo string) error {
	actual, err := o.estadoDe(sesionID)
	if err != nil {
		return err
	}
	if actual == nuevo {
		return nil
	}
	if !TransicionValida(actual, nuevo) && EsEstadoFinal(actual) && TransicionValida(EstadoInactiva, nuevo) {
		if err := store.ActualizarEstadoSesion(o.db, o.projectID, sesionID, EstadoInactiva); err != nil {
			return err
		}
		actual = EstadoInactiva
	}
	if !TransicionValida(actual, nuevo) {
		return errorTransicion(actual, nuevo)
	}
	if err := store.ActualizarEstadoSesion(o.db, o.projectID, sesionID, nuevo); err != nil {
		return err
	}
	o.emitirEstado(sesionID, nuevo)
	return nil
}

// cerrarTurno persiste el estado final y notifica el cierre de un turno de chat.
func (o *Orquestador) cerrarTurno(sesionID string, fallo bool) {
	estado := EstadoTerminada
	if fallo {
		estado = EstadoError
	}
	if err := o.marcarEstado(sesionID, estado); err != nil {
		return
	}
	if !fallo {
		o.avisar(sesionID, "la sesión terminó de responder", "revisa su respuesta en el historial")
	}
}

// avisar publica una notificación: lo que pasó y qué hacer.
func (o *Orquestador) avisar(sesionID, motivo, siguiente string) {
	datos := map[string]string{"motivo": motivo}
	if siguiente != "" {
		datos["siguiente"] = siguiente
	}
	o.emitir(tui.Evento{Sesion: sesionID, Tipo: tui.EventoNotificacion, Datos: datos})
}

// emitir publica un evento sin bloquearse si la vista ya cerró.
func (o *Orquestador) emitir(e tui.Evento) {
	select {
	case o.eventos <- e:
	case <-o.terminado:
	}
}

func (o *Orquestador) modeloActual() string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.modelo
}

func (o *Orquestador) ventanaActual() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.ventana
}

// fijarVentana actualiza la ventana efectiva del turno tras releer el contexto
// real del servidor. Un valor no positivo no toca la vigente.
func (o *Orquestador) fijarVentana(ventana int) {
	if ventana <= 0 {
		return
	}
	o.mu.Lock()
	o.ventana = ventana
	o.mu.Unlock()
}

// mejorarErrorLLM clasifica el error del modelo y devuelve un mensaje accionable.
func (o *Orquestador) mejorarErrorLLM(err error) (motivo, siguiente string) {
	var eErr *openai.ErrorServidor
	if errors.As(err, &eErr) {
		switch eErr.Tipo {
		case "servidor":
			if strings.Contains(strings.ToLower(eErr.Mensaje), "callexpr") ||
				strings.Contains(strings.ToLower(eErr.Mensaje), "call expr") ||
				strings.Contains(strings.ToLower(eErr.Mensaje), "function call") {
				return "el servidor LLM falló al ejecutar una llamada a herramienta (CallExpr)",
					"reintenta la petición; si persiste, reinicia llama-server o reduce la complejidad de la tarea"
			}
			return fmt.Sprintf("error interno del servidor LLM (%d)", eErr.Codigo),
				"reintenta en unos segundos; si persiste, reinicia llama-server"
		case "rate_limit":
			return "límite de tasa del servidor LLM excedido",
				"espera unos segundos y reintenta"
		case "cliente":
			return fmt.Sprintf("petición inválida al servidor LLM (%d)", eErr.Codigo),
				"revisa la configuración del modelo y la carpeta del proyecto"
		}
		return fmt.Sprintf("error del servidor LLM (%d): %s", eErr.Codigo, eErr.Mensaje),
			"revisa que llama-server esté levantado en la dirección de localcli.json"
	}

	// Errores de conexión/red.
	if strings.Contains(err.Error(), "no se pudo conectar") ||
		strings.Contains(err.Error(), "connection refused") ||
		strings.Contains(err.Error(), "i/o timeout") {
		return "no se pudo conectar con llama-server",
			"verifica que llama-server esté corriendo en la dirección de localcli.json"
	}

	// Fallback genérico.
	return err.Error(), "revisa que llama-server esté levantado en la dirección de localcli.json"
}
