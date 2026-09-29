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
//   - deja el ciclo de sesiones listo (sin crear ninguna hasta la primera petición) y reconstruye la conversación al responder,
//   - y hace vivir el flujo de aprobación real: cada petición de permiso llega
//     a la vista como `peticion_aprobacion` y la decisión del usuario vuelve al
//     motor como `aprobacion_resuelta` (el puente entre el `Ask` de `tools` y
//     el panel de la TUI).
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
	"strconv"
	"strings"
	"sync"
	"time"

	"localcli/internal/agent"
	tcontext "localcli/internal/context"
	lcexec "localcli/internal/exec"
	"localcli/internal/flow"
	gitrepo "localcli/internal/git"
	"localcli/internal/ollama"
	"localcli/internal/queue"
	"localcli/internal/session"
	"localcli/internal/store"
	"localcli/internal/task"
	"localcli/internal/tools"
	"localcli/internal/tui"
)

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
	// todos es el repositorio de la lista de pasos de la sesión (SPEC-TOOLS).
	todos *store.Todos
	// bloques es el repositorio del bloque de contexto de un flujo
	// (SPEC-MOTOR-FLUJOS).
	bloques *store.Bloques
	// chat es el repositorio del hilo de procesamiento (chat_evento): las líneas
	// de sub-proceso y de herramienta que se guardan como parte del chat.
	chat *store.Chat
	bus  *session.Bus
	dir  string
	// catalogo son los flujos del proyecto: los que declara
	// `.localcli/flows/*.json`. Lo arma el arranque; `Enviar` reconoce los
	// comandos contra él.
	catalogo *flow.Catalogo

	// cliente es Ollama: solo se usa para la lista del selector de modelos de
	// la bienvenida (SPEC-INTERFAZ §Reglas); el arranque no le escribe nada.
	cliente *ollama.Client
	// modeloMu protege el modelo elegido: lo fija la autodetección al arrancar
	// y lo cambia el usuario desde el selector (FijarModelo). El motor lo lee
	// en cada turno — la elección del usuario prevalece (SPEC-OLLAMA-PERFIL).
	modeloMu sync.Mutex
	modelo   string
	// agenteRecordado es el último agente usado, leído de las preferencias del
	// usuario al arrancar: la vista empieza en él (SPEC-OLLAMA-PERFIL).
	agenteRecordado string
	// agentes son los nombres de los agentes disponibles, en orden estable
	// (`plan`, `build` y el resto alfabético). Los carga `agenteBase` de
	// `.localcli/agents/*.json`; la vista cicla por esta lista.
	agentes []string

	mu         sync.Mutex
	sesionID   string
	pendientas map[string]chan string // approvalID -> canal de resolución
	// contextos cachea, por modelo, la ventana de contexto que declara
	// `/api/tags` (details.context_length), para no repetir el listado al
	// calcular `num_ctx` en cada turno. Un 0 cacheado significa «el modelo no la
	// declara»: se guarda igual para no volver a listar.
	contextos map[string]int
}

// modeloElegido es la lectura con mutex del modelo activo.
func (ad *Adaptador) modeloElegido() string {
	ad.modeloMu.Lock()
	defer ad.modeloMu.Unlock()
	return ad.modelo
}

// topeVentana es el techo de la ventana de contexto: LOCALCLI_CONTEXT_LIMIT si
// el usuario lo fijó, y si no el tope por defecto de `ollama`.
func topeVentana() int {
	if n := tcontext.LimiteDeEntorno(); n > 0 {
		return n
	}
	return ollama.TopeVentanaPorDefecto
}

// ventana devuelve el `num_ctx` a pedir para el modelo dado: el menor entre su
// ventana declarada y el tope. Es lo que evita que Ollama recorte la lista de
// mensajes y responda `no user query found in messages` en los turnos con
// herramientas (el servidor usa por defecto ~4096, que es demasiado poco).
func (ad *Adaptador) ventana(modelo string) int {
	return ollama.VentanaDeModelo(ollama.Modelo{ContextLength: ad.largoDeContexto(modelo)}, topeVentana())
}

// largoDeContexto devuelve la ventana que declara el modelo, cacheada. Si no
// está en la caché, lista los modelos una vez y rellena toda la caché; un fallo
// de Ollama se trata como «no la declara» (0) y el tope manda.
func (ad *Adaptador) largoDeContexto(modelo string) int {
	ad.mu.Lock()
	if n, ok := ad.contextos[modelo]; ok {
		ad.mu.Unlock()
		return n
	}
	ad.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), timeoutDeteccion)
	defer cancel()
	modelos, err := ad.cliente.ListarModelos(ctx)
	if err != nil {
		return 0
	}
	ad.mu.Lock()
	defer ad.mu.Unlock()
	largo := 0
	for _, m := range modelos {
		ad.contextos[m.Nombre] = m.ContextLength
		if m.Nombre == modelo {
			largo = m.ContextLength
		}
	}
	return largo
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

	// El último modelo y el último agente que el usuario usó se recuerdan entre
	// ejecuciones (config.json, preferencia global del usuario). El modelo se
	// reutiliza si sigue instalado; el agente lo aplica la vista al construirse.
	prefs, _ := tui.CargarPreferencias()
	modelo, err := elegirModelo(cliente, prefs.Modelo)
	if err != nil {
		fmt.Fprintln(os.Stderr, "aviso:", err, "(la interfaz arranca sin modelo)")
	}

	// El adaptador nace aquí: los handlers de herramientas lo enlazan por
	// referencia (aprobarComando lee gest y sesionActual al invocarse, nunca al
	// montarse), así que el registro puede construirse antes de que el gestor
	// exista.
	cambios := store.NuevosCambios(conexion)
	ad := &Adaptador{aprobar: aprobar, cerrar: turnos, auditar: store.NuevasAuditorias(conexion), todos: store.NuevosTodos(conexion), bloques: store.NuevosBloques(conexion), chat: store.NuevoChat(conexion), bus: bus, dir: carpeta, pendientas: map[string]chan string{}, contextos: map[string]int{}, cliente: cliente}
	ad.modelo = modelo
	ad.agenteRecordado = prefs.Agente
	// El ejecutor externo corre las herramientas del usuario: el mismo módulo
	// `exec` con la misma frontera (Landlock, límites y recorte).
	ejecutorExterno := &lcexec.Ejecutor{Proyecto: carpeta}
	registro, err := registroDeHerramientas(ad, carpeta, cambios, ad.todos, ejecutorExterno)
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

	// Los flujos del proyecto son los archivos de `.localcli/flows/*.json`. Un
	// JSON roto no tumba el arranque; se arranca sin flujos (catálogo vacío) y se
	// avisa.
	catalogo, catErr := flow.CargarFlujos(carpeta)
	if catErr != nil {
		fmt.Fprintln(os.Stderr, "aviso: no se pudieron cargar los flujos de .localcli/flows:", catErr)
		catalogo = flow.CatalogoPorDefecto()
	}
	ad.catalogo = catalogo

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
	// El título de una sesión y el resumen de su conversación son dos
	// generaciones cortas del mismo modelo local: viven en el arranque, que es
	// quien tiene el cliente y la cola. El presupuesto de historial lo fija el
	// usuario en sus preferencias (SPEC-HISTORIAL-CONVERSACION).
	gest.Titulador = &tituladorPorTurno{ad: ad, infer: infer}
	gest.Resumidor = &resumidorPorTurno{ad: ad, infer: infer}
	// El presupuesto del historial se queda por debajo de la ventana de contexto
	// para que el prompt entero (sistema + herramientas + historial + salidas de
	// herramienta) quepa: la preferencia del usuario manda; si no,
	// LOCALCLI_CONTEXT_LIMIT; si no, un cuarto de la ventana del modelo (4096
	// para la ventana por defecto de 16384).
	gest.Presupuesto = prefs.HistorialTokens
	if gest.Presupuesto <= 0 {
		gest.Presupuesto = tcontext.LimiteDeEntorno()
	}
	if gest.Presupuesto <= 0 {
		gest.Presupuesto = ad.ventana(modelo) / 4
	}
	ad.gest = gest

	a := &Arranque{cerrarDB: cerrarDB, Bus: bus, Gest: gest, Infer: infer}
	a.puerto = ad

	// El motor necesita contexto, agente y aprobador POR CADA SESIÓN (el nodo
	// lleva el id de sesión y etapa para auditar). Se los inyectamos mediante
	// una función que reconstruye esas piezas sobre el mismo motor antes de
	// cada turno: es el sitio del arranque, no de session ni de flow.
	motorFlows.Contexto = &nodoPorTurno{ad: ad, grafo: grafo, cliente: cliente, modelo: modelo}
	// El bucle conversacional (agente → modelo → herramientas → modelo) vive en
	// `agent`; aquí solo se le dan sus dependencias reales y el mapa de agentes.
	//
	// El turno de inferencia se toma POR PETICIÓN al modelo (`Testigo`), no por
	// la ejecución entera: entre pasada y pasada, mientras el usuario decide una
	// aprobación, el modelo queda libre para las demás sesiones
	// (SPEC-OLLAMA-PERFIL).
	ejecutor := &agent.Ejecutor{
		Runner:     agenteRunner,
		Despachar:  despachador,
		MaxPasadas: 3,
		Testigo: func(ctx context.Context, fn func(context.Context) error) error {
			return infer.Encolar(ctx, ollama.OpcionesEncolar{IdSesion: ad.sesionActual()}, fn)
		},
		PuedeHerramientas: func(m string) bool {
			ctx, cancel := context.WithTimeout(context.Background(), timeoutDeteccion)
			defer cancel()
			caps, cErr := cliente.Capacidades(ctx, m)
			if cErr != nil {
				// Sin dato no se degrada: mejor conversar con herramientas que
				// quitarle el trabajo al modelo por una ficha que no llegó.
				return true
			}
			return ollama.PuedeUsarHerramientas(caps)
		},
	}
	agentes := agenteBase(carpeta)
	ad.agentes = agent.OrdenarNombres(agentes)
	motorFlows.Agente = &ejecutorPorTurno{ad: ad, ejecutor: ejecutor, agente: agentes}
	motorFlows.Aprobador = &aprobadorPorTurno{ad: ad}
	// El bloque de contexto de un flujo: cada etapa guarda su aportación
	// optimizada en `store` y la etapa `entrega` la lee entera
	// (SPEC-MOTOR-FLUJOS §Bloque de contexto).
	motorFlows.Bloque = &bloquePorTurno{ad: ad}
	motorFlows.Optimizador = &optimizadorPorTurno{ad: ad, infer: infer}
	// El hilo del chat guarda el sub-proceso de cada etapa: la línea queda en el
	// historial además de verse en vivo.
	motorFlows.Registro = &registroPorTurno{ad: ad}

	// Arranque de sesión: NO se crea ninguna. La sesión nace con la primera
	// petición desde la bienvenida (SPEC-SESIONES); hasta entonces no hay sesión
	// activa. Cada `Enviar` fija la suya antes de arrancar.
	ad.fijarSesion("")

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

// ResolverActiva retoma la sesión más reciente del proyecto sin crear ninguna:
// es lo que hace la vista al cambiar a una sesión existente (por ejemplo tras
// borrar la activa). Devuelve `nil, nil` si el proyecto todavía no tiene
// sesiones: en ese caso la bienvenida crea una al enviar la primera petición
// (SPEC-SESIONES: no hay una sesión "principal" que exista antes de usarse).
func (ad *Adaptador) ResolverActiva() (*session.Sesion, error) {
	sesiones, err := ad.gest.Listar()
	if err != nil {
		return nil, err
	}
	if len(sesiones) == 0 {
		return nil, nil
	}
	// Listar ordena por actividad reciente: la primera es la más reciente.
	reciente := sesiones[0]
	return ad.gest.Estado(reciente.ID)
}

func (ad *Adaptador) Listar() ([]session.Sesion, error) { return ad.gest.Listar() }

// Crear deja una sesión nueva activa (SPEC-SESIONES): es lo que hace `Ctrl+X n`
// desde la vista principal y lo que crea la primera petición desde la
// bienvenida. Nace con el nombre provisional; su título lo genera el modelo a
// partir de la primera petición. La capa queda vacía porque es una sesión
// general (ENUMS.md: layer NULL).
func (ad *Adaptador) Crear() (*session.Sesion, error) {
	ses, err := ad.gest.Crear(session.NombreProvisional, "")
	if err != nil {
		return nil, err
	}
	ad.fijarSesion(ses.ID)
	return ses, nil
}

// Eliminar borra una sesión y todo lo suyo en cascada (RELATIONSHIPS.md §3):
// mensajes, razonamiento y aprobaciones caen con ella. Es lo que hace `Ctrl+D`
// en el modal de sesiones; la confirmación cuando está trabajando la pide la
// vista antes de llegar aquí.
func (ad *Adaptador) Eliminar(sesionID string) error {
	if err := ad.gest.Cerrar(sesionID, true); err != nil {
		return err
	}
	return nil
}

// Modelos da al modal de modelos la lista que reporta Ollama, ya traducida a
// `tui.ModeloLocal`: la vista no importa el paquete ollama ni toca HTTP
// (wire.go). Se pide al abrir el modal, no al arrancar (SPEC-INTERFAZ §Modal de
// modelos). Si Ollama no responde, el error sube tal cual y la TUI lo pinta como
// aviso «sin modelos» sin bloquear nada; el tiempo de espera es el del
// arranque, acotado por timeoutDeteccion.
//
// Cada modelo trae además si declara capacidad de herramientas (`/api/show`),
// para que el usuario sepa cuáles sirven antes de elegirlos.
func (ad *Adaptador) Modelos() ([]tui.ModeloLocal, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeoutDeteccion)
	defer cancel()
	modelos, err := ad.cliente.ListarModelos(ctx)
	if err != nil {
		return nil, err
	}
	// Aprovechar el listado para calentar la caché de ventanas de contexto: al
	// elegir un modelo no hará falta otra consulta a /api/tags.
	ad.mu.Lock()
	for _, m := range modelos {
		ad.contextos[m.Nombre] = m.ContextLength
	}
	ad.mu.Unlock()
	out := make([]tui.ModeloLocal, len(modelos))
	// La ficha de cada modelo se pide en paralelo, con un tope de 4 a la vez:
	// listar no debe tardar lo que la suma de las fichas.
	sem := make(chan struct{}, 4)
	var wg sync.WaitGroup
	for i, m := range modelos {
		wg.Add(1)
		go func(i int, nombre string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			caps, conocido := ad.capacidadesDe(ctx, nombre)
			out[i] = tui.ModeloLocal{Nombre: nombre, CapacidadesSinDato: !conocido}
			if conocido {
				out[i].SinHerramientas = !ollama.PuedeUsarHerramientas(caps)
				out[i].SinVision = !ollama.PuedeVer(caps)
			}
		}(i, m.Nombre)
	}
	wg.Wait()
	return out, nil
}

// capacidadesDe pide la ficha del modelo (/api/show) y dice si se pudo leer. Sin
// ficha no se declara ninguna capacidad: la vista no avisa ni asegura nada
// (mejor callar que alarmar), y por eso el «conocido» viaja aparte.
func (ad *Adaptador) capacidadesDe(ctx context.Context, nombre string) ([]string, bool) {
	caps, err := ad.cliente.Capacidades(ctx, nombre)
	if err != nil {
		return nil, false
	}
	return caps, true
}

// ModeloActual devuelve el modelo con el que trabaja el motor ahora mismo: el
// que detectó el arranque o el último que eligió el usuario en el modal de
// modelos. Es lo que la línea de modelo de la bienvenida muestra
// (SPEC-INTERFAZ §Línea de modelo); es una lectura en memoria, sin llamar a
// Ollama, así que la pantalla se pinta sin esperar a nada externo.
func (ad *Adaptador) ModeloActual() string { return ad.modeloElegido() }

// Carpeta devuelve la carpeta del proyecto —la desde la que se ejecutó
// `localcli`—, que el arranque ya tenía resuelta al abrir la base. Es el dato
// de «Ruta» del pie del panel (SPEC-INTERFAZ §Zonas 3). No consulta nada: es
// el valor con el que se construyó el adaptador.
func (ad *Adaptador) Carpeta() string { return ad.dir }

// Git devuelve la rama activa del repositorio del proyecto y si el árbol tiene
// cambios sin confirmar, que es lo que pinta el pie del panel
// (SPEC-INTERFAZ §Zonas 3, dato «Git»). La rama vacía significa que el proyecto
// no tiene git inicializado, o que no hay git instalado, y se muestra como
// «sin iniciar». La lectura la hace `internal/git`, que llama al binario: aquí
// no hay aprobación porque el harness lee su propio repositorio, no ejecuta
// nada del proyecto.
func (ad *Adaptador) Git() (string, int) { return gitrepo.Estado(ad.dir) }

// CapacidadesModelo dice qué declara capaz de hacer el modelo indicado (usar
// herramientas e interpretar imágenes), para la línea de estado bajo el input
// (SPEC-OLLAMA-PERFIL). Un fallo sube tal cual; la vista lo trata como dato
// desconocido y no bloquea.
func (ad *Adaptador) CapacidadesModelo(nombre string) (tui.Capacidades, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeoutDeteccion)
	defer cancel()
	caps, err := ad.cliente.Capacidades(ctx, nombre)
	if err != nil {
		return tui.Capacidades{}, err
	}
	return tui.Capacidades{
		Herramientas: ollama.PuedeUsarHerramientas(caps),
		Vision:       ollama.PuedeVer(caps),
	}, nil
}

// AgenteRecordado devuelve el último agente con el que trabajó el usuario,
// leído de sus preferencias al arrancar: la vista empieza en él
// (SPEC-OLLAMA-PERFIL). Vacío significa «sin preferencia».
func (ad *Adaptador) AgenteRecordado() string { return ad.agenteRecordado }

// Agentes devuelve los nombres de los agentes disponibles, en orden estable
// (`plan`, `build` y el resto alfabético). Los carga el arranque de
// `.localcli/agents/*.json`; la vista cicla por esta lista y el motor rechaza cualquier
// nombre que no esté en ella.
func (ad *Adaptador) Agentes() []string { return ad.agentes }

// FijarModelo guarda la elección del usuario: a partir de ahora cada turno del
// motor usa este modelo, aunque la autodetección del arranque hubiera tomado
// otro (SPEC-OLLAMA-PERFIL: el modelo lo elige el usuario).
func (ad *Adaptador) FijarModelo(nombre string) {
	ad.modeloMu.Lock()
	defer ad.modeloMu.Unlock()
	ad.modelo = nombre
}

// Historial traduce el hilo guardado de una sesión a lo que la vista pinta:
// los turnos de la conversación y las líneas de procesamiento, con su tiempo,
// más los números del panel de contexto. La vista nunca lee la base: llega como
// dato (INTERFACES §3).
func (ad *Adaptador) Historial(sesionID string) (tui.HistorialSesion, error) {
	hilo, err := ad.chat.Hilo(sesionID)
	if err != nil {
		return tui.HistorialSesion{}, err
	}
	out := make([]tui.MensajeHistorial, 0, len(hilo))
	for _, l := range hilo {
		out = append(out, tui.MensajeHistorial{
			Rol:          l.Rol,
			Texto:        l.Content,
			Razonamiento: l.Razonamiento,
			Duracion:     duracionDeMS(l.DuracionMS),
		})
	}
	usado, limite := ad.contextoSesion(sesionID)
	return tui.HistorialSesion{Mensajes: out, ContextoTokens: usado, LimiteTokens: limite}, nil
}

// duracionDeMS convierte la duración guardada (ms) al tipo de la vista. -1 =
// no se midió, que la vista trata como cero.
func duracionDeMS(ms int) time.Duration {
	if ms < 0 {
		return 0
	}
	return time.Duration(ms) * time.Millisecond
}

// contextoSesion estima los tokens que el chat de una sesión ocupa en el
// contexto: la suma de sus mensajes de usuario y de agente. Las líneas de
// procesamiento NO cuentan: no se le envían al modelo. El límite es la ventana
// del modelo en uso. Es una estimación: no hay tokenizador exacto por mensaje.
func (ad *Adaptador) contextoSesion(sesionID string) (usado, limite int) {
	limite = ad.ventana(ad.modeloElegido())
	if sesionID == "" {
		return 0, limite
	}
	ms, err := ad.gest.Historial(sesionID)
	if err != nil {
		return 0, limite
	}
	for _, m := range ms {
		if m.Role == "user" || m.Role == "agent" {
			usado += tcontext.EstimarTokens(m.Content)
		}
	}
	return usado, limite
}

// Tareas devuelve la lista de pasos de una sesión para que la vista la pinte al
// cargarla (SPEC-TOOLS). Igual que el historial, la vista no toca la base: llega
// como dato.
func (ad *Adaptador) Tareas(sesionID string) ([]tui.TareaPanel, error) {
	if ad.todos == nil || sesionID == "" {
		return nil, nil
	}
	items, err := ad.todos.Leer(sesionID)
	if err != nil {
		return nil, err
	}
	out := make([]tui.TareaPanel, 0, len(items))
	for _, it := range items {
		out = append(out, tui.TareaPanel{Contenido: it.Contenido, Estado: it.Estado})
	}
	return out, nil
}

// Enviar manda el mensaje al gestor. La vista principal responde como chat por
// defecto; solo un comando explícito arranca un flujo (SPEC-INTERFAZ §Reglas:
// "/planificar, /crear, /actualizar, /eliminar, /resolver o /ejecutar"). La
// vista recibe el desenlace por eventos, no por el retorno.
//
// El agente viaja con la petición (T-F015-04): lo elige el usuario con el
// indicador y con él responde el chat. La detección de trabajo ordenado ya no
// arranca nada: se propone en `session` (SPEC-COLA-TAREAS).
//
// `imagenes` son las imágenes (base64) que la vista detectó en el texto: solo
// acompañan al chat; un comando de flujo las ignora (los flujos no adjuntan).
func (ad *Adaptador) Enviar(ctx context.Context, sesionID, agente, texto string, imagenes []string) error {
	ad.fijarSesion(sesionID)
	if cmd, ok := ad.catalogo.De(texto); ok {
		if cmd.Consumir {
			return ad.ejecutarCola(sesionID, texto)
		}
		return ad.gest.ArrancarFlujo(ctx, sesionID, cmd.Flujo, cmd.Objetivo(texto))
	}
	return ad.gest.Conversar(ctx, sesionID, agente, texto, imagenes)
}

// Comandos devuelve los comandos de flujo del proyecto para la paleta de la
// vista: los que declara `.localcli/flows/*.json`, y `/ejecutar` al final (que no
// es un flujo: consume la cola). La vista no importa `flow`; el catálogo llega
// ya traducido a su tipo.
func (ad *Adaptador) Comandos() []tui.ComandoFlujo {
	if ad.catalogo == nil {
		return nil
	}
	flujos := ad.catalogo.Flujos()
	out := make([]tui.ComandoFlujo, 0, len(flujos)+1)
	for _, f := range flujos {
		out = append(out, tui.ComandoFlujo{Nombre: f.Comando, Descripcion: f.Descripcion})
	}
	return append(out, tui.ComandoFlujo{Nombre: "/ejecutar", Descripcion: "ejecutar la cola de tareas"})
}

// ejecutarCola atiende `/ejecutar`: deja constancia del comando en el historial
// y consume la cola del TODO en segundo plano. Es el único camino que arranca
// la cola (SPEC-COLA-TAREAS: "El usuario decide si se ejecuta").
func (ad *Adaptador) ejecutarCola(sesionID, texto string) error {
	if err := ad.gest.Anotar(sesionID, texto); err != nil {
		return err
	}
	go ad.consumirCola(sesionID)
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
// estaba esperando en `pedirAprobacion` y la línea sale del panel.
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

// pedirAprobacion implementa `tools.Contexto.Ask`: es el ÚNICO puente entre la
// capa universal de herramientas y la vista. Registra la aprobación, emite
// `peticion_aprobacion` al panel y espera el veredicto, sea cual sea la
// herramienta que lo pida (un comando, una escritura o una del usuario). Es el
// mecanismo único de aprobación (DECISIONS.md).
func (ad *Adaptador) pedirAprobacion(ctx context.Context, s tools.Solicitud) (tools.Decision, error) {
	descripcion := s.Descripcion
	if descripcion == "" {
		descripcion = "aprobar una acción"
	}
	id, ch, err := ad.pedirPermiso(descripcion, s.Motivo)
	if err != nil {
		return tools.Decision{}, err
	}
	var estado string
	select {
	case estado = <-ch:
	case <-ctx.Done():
		ad.resolverCanal(id, store.ApprovalObsoleta)
		return tools.Decision{}, nil
	}
	switch estado {
	case store.ApprovalAprobada:
		// Resolver el borrado en el panel ES la confirmación explícita.
		return tools.Decision{Aprobada: true, Explicita: true}, nil
	default:
		return tools.Decision{}, nil
	}
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
func (ad *Adaptador) pedirPermiso(descripcion, motivo string) (approvalID string, resolucion <-chan string, err error) {
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
		"motivo":      motivo,
	}})
	return ap.ID, ch, nil
}

// resolverCanal entrega el estado final a quien esperaba (la goroutine del
// motor bloqueada en `pedirAprobacion`).
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

// --- internet ----------------------------------------------------------------

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
// configura"): si el usuario ya usó uno y sigue instalado, ese manda; si no,
// lista los modelos de Ollama y toma el primero que cabe en la máquina. Si
// Ollama no responde, devuelve error y el arranque sigue sin modelo.
func elegirModelo(cliente *ollama.Client, preferido string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeoutDeteccion)
	defer cancel()
	modelos, err := cliente.ListarModelos(ctx)
	if err != nil {
		return "", fmt.Errorf("no hay conexión con Ollama: %w", err)
	}
	// El último modelo usado prevalece si sigue disponible (SPEC-OLLAMA-PERFIL:
	// el modelo lo elige el usuario y su elección se recuerda).
	if preferido != "" {
		for _, m := range modelos {
			if m.Nombre == preferido {
				return preferido, nil
			}
		}
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

func (n *nodoPorTurno) ContextoPara(ctx context.Context, etapa, objetivo string) (string, error) {
	modelo := n.ad.modeloElegido()
	if n.grafo == nil || modelo == "" {
		return "", fmt.Errorf("arranque: sin grafo o modelo no hay contexto que entregar")
	}
	nodo := &tcontext.Nodo{
		Grafo:     n.grafo,
		Modelo:    &tcontext.ModeloOllama{Cliente: n.cliente, Modelo: modelo},
		Auditor:   n.ad.auditar,
		SessionID: n.ad.sesionActual(),
		// El bloque de contexto se queda por debajo de la ventana del modelo:
		// sin este límite, el nodo no recortaba nada (limite 0 = sin tope) y el
		// contexto podía desbordar la petición. Un cuarto de la ventana.
		Limite: n.ad.ventana(modelo) / 4,
	}
	return nodo.ContextoPara(ctx, etapa, objetivo)
}

// ejecutorPorTurno implementa flow.Agente: corre la etapa con el agente base
// (`plan` o `build`) cargado de .localcli/agents. Serializa la inferencia en la FIFO,
// delega el ciclo conversacional (LLM → herramienta → resultado → LLM) en
// `agent.Ejecutor` —el bucle único que comparten los dos agentes—, traduce los
// tokens a eventos de la TUI (EVENTS.md §1, DECISIONS.md [25]) y guarda el
// turno al terminar.
type ejecutorPorTurno struct {
	ad       *Adaptador
	ejecutor *agent.Ejecutor
	agente   map[string]agent.Agente
}

func (e *ejecutorPorTurno) Ejecutar(ctx context.Context, p flow.PeticionEtapa) (flow.Resultado, error) {
	// El modelo se lee en cada turno: si el usuario lo cambió desde el selector
	// de la bienvenida, esa elección es la que corre (SPEC-OLLAMA-PERFIL).
	modelo := e.ad.modeloElegido()
	if modelo == "" {
		return flow.Resultado{}, errors.New("arranque: no hay modelo de Ollama disponible")
	}
	agente, ok := e.agente[p.Agente]
	if !ok {
		return flow.Resultado{}, fmt.Errorf("arranque: agente base desconocido %q", p.Agente)
	}
	sesionID := e.ad.sesionActual()

	// Una etapa silenciosa (intermedia de un flujo) no se muestra ni se guarda:
	// su texto solo alimenta la cadena del motor. Sin sink no hay eventos de
	// token y, más abajo, el turno no se persiste.
	var sink agent.Sink
	if !p.Silenciosa {
		sink = sinkBus{ad: e.ad, sesionID: sesionID}
	}

	// El turno de inferencia NO se toma aquí: el bucle de `agent` lo toma por
	// PETICIÓN al modelo, así que una sesión esperando una aprobación deja libre
	// el modelo para las demás (SPEC-OLLAMA-PERFIL, T-B024-14). El tiempo que se
	// mide es el del turno entero (modelo + herramientas).
	inicio := time.Now()
	salida, err := e.ejecutor.Ejecutar(ctx, agente, modelo, p.Contexto, mensajesDeOllama(p.Historial), p.Imagenes, e.ad.ventana(modelo), p.SinHerramientas, sink)
	if err != nil {
		return flow.Resultado{}, err
	}
	duracion := int(time.Since(inicio).Milliseconds())

	resumen := strings.TrimSpace(salida.Texto)
	if resumen == "" {
		resumen = "(respuesta vacía)"
	}
	// El turno del agente se guarda en la misma transacción que su razonamiento
	// y su estado (DATA_FLOW.md), con sus tokens y su duración: el historial que
	// verá la próxima vez incluye esta respuesta. Una etapa silenciosa no se
	// persiste: solo la entrega final (o el chat) queda en el historial.
	if sesionID != "" && !p.Silenciosa {
		if _, cErr := cerrarTurnoGuardado(e.ad, sesionID, resumen, salida.Razonamiento,
			int(salida.TokensEntrada), int(salida.TokensSalida), duracion); cErr != nil {
			return flow.Resultado{}, cErr
		}
	}
	// El consumo del turno y el contexto del chat alimentan el panel; solo la
	// etapa visible (o el chat) lo reporta: el de un sub-proceso no interrumpe la
	// lectura. El contexto se calcula DESPUÉS de guardar el turno, para que lo
	// incluya.
	if !p.Silenciosa {
		contexto, limite := e.ad.contextoSesion(sesionID)
		e.ad.bus.Emitir(tui.Evento{Nombre: tui.EventoTokensTurno, Datos: map[string]string{
			"entrada":  strconv.FormatUint(salida.TokensEntrada, 10),
			"salida":   strconv.FormatUint(salida.TokensSalida, 10),
			"contexto": strconv.Itoa(contexto),
			"limite":   strconv.Itoa(limite),
		}})
	}
	return flow.Resultado{Texto: resumen}, nil
}

// sinkBus traduce los fragmentos que produce el modelo a eventos de token de
// la TUI. Es el único punto donde el ciclo de `agent` toca el bus.
type sinkBus struct {
	ad       *Adaptador
	sesionID string
}

func (s sinkBus) Token(texto string, esRazon bool) {
	if texto == "" {
		return
	}
	datos := map[string]string{"sesion": s.sesionID, "texto": texto}
	if esRazon {
		datos["razonamiento"] = "true"
	}
	s.ad.bus.Emitir(tui.Evento{Nombre: tui.EventoToken, Datos: datos})
}

// Aviso publica un aviso del turno (por ejemplo, que el agente ha caído a modo
// conversación porque el modelo no declara herramientas) como una notificación
// de sesión, que es como llega al usuario sin bloquear nada.
func (s sinkBus) Aviso(texto string) {
	if texto == "" {
		return
	}
	s.ad.bus.Emitir(tui.Evento{Nombre: session.EventoNotificacion, Datos: map[string]string{
		"sesion": s.sesionID,
		"motivo": texto,
	}})
}

// cerrarTurnoGuardado persiste la respuesta del agente con su razonamiento,
// sus tokens y su duración, dejando el estado de la sesión como está (lo mueve
// `session` al terminar el flujo). Los tokens y la duración llegan como -1
// cuando el modelo no los reportó.
func cerrarTurnoGuardado(ad *Adaptador, sesionID, contenido, razon string, in, out, duracionMs int) (*store.Message, error) {
	return ad.cerrar.CerrarAgente(sesionID, contenido, in, out, duracionMs, razon, "")
}

// aprobadorPorTurno implementa flow.Aprobador con el aprobador de producción:
// registra la petición, espera la decisión del panel y traduce el resultado.
type aprobadorPorTurno struct{ ad *Adaptador }

func (p *aprobadorPorTurno) Aprobar(ctx context.Context, descripcion string) (bool, error) {
	id, ch, err := p.ad.pedirPermiso(descripcion, "")
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

// bloquePorTurno implementa flow.Bloque sobre el repositorio del arranque. El
// bloque está acotado a la sesión en curso: la lee del Adaptador en cada
// llamada, igual que el nodo de contexto.
type bloquePorTurno struct{ ad *Adaptador }

func (b *bloquePorTurno) Limpiar(ctx context.Context, flujo string) error {
	return b.ad.bloques.Limpiar(b.ad.sesionActual(), flujo)
}

func (b *bloquePorTurno) Guardar(ctx context.Context, flujo string, e flow.EntradaBloque) error {
	return b.ad.bloques.Guardar(b.ad.sesionActual(), &store.FlowContext{
		Flow:      flujo,
		Stage:     e.Etapa,
		StageName: e.Nombre,
		Position:  e.Posicion,
		Content:   e.Contenido,
	})
}

func (b *bloquePorTurno) Leer(ctx context.Context, flujo string) ([]flow.EntradaBloque, error) {
	filas, err := b.ad.bloques.Leer(b.ad.sesionActual(), flujo)
	if err != nil {
		return nil, err
	}
	out := make([]flow.EntradaBloque, 0, len(filas))
	for _, f := range filas {
		out = append(out, flow.EntradaBloque{
			Etapa: f.Stage, Nombre: f.StageName, Posicion: f.Position, Contenido: f.Content,
		})
	}
	return out, nil
}

// registroPorTurno implementa flow.Registro: guarda en el hilo del chat la
// línea del sub-proceso de cada etapa, con la misma forma que la pinta la TUI
// (LineaProceso). Es best-effort: si la escritura falla, el flujo sigue.
type registroPorTurno struct{ ad *Adaptador }

func (r *registroPorTurno) ProcesoEtapa(nombre string, fallida bool) {
	if r.ad.chat == nil {
		return
	}
	sesion := r.ad.sesionActual()
	if sesion == "" {
		return
	}
	_ = r.ad.chat.Registrar(sesion, store.ChatTipoProceso, tui.LineaProceso(nombre, fallida))
}

// optimizadorPorTurno implementa flow.Optimizador: condensa el resultado de una
// etapa con una generación corta del modelo local, sin herramientas ni
// streaming. Un fallo no detiene el flujo (el motor cae al resumen mecánico).
type optimizadorPorTurno struct {
	ad    *Adaptador
	infer *ollama.ColaInferencia
}

func (o *optimizadorPorTurno) Optimizar(ctx context.Context, flujo, etapa, resultado string) (string, error) {
	crudo, err := generarTextoCorrido(ctx, o.ad, o.infer, flow.PromptOptimizacion(flujo, etapa, resultado))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(crudo), nil
}

// agenteBase carga los agentes del proyecto desde `.localcli/agents/*.json`: los dos
// base (`plan`, `build`) y cualquier agente propio que el usuario deje ahí. El
// nombre de cada archivo no decide nada: manda el campo `nombre` del JSON.
//
// Un archivo inválido se ignora con un aviso por stderr, no tumba el arranque
// (CONFIGURATION.md 5: la vista sigue viva), y `plan` y `build` siempre quedan
// disponibles aunque su JSON falte o esté roto: son los que arrancan los flujos
// oficiales.
func agenteBase(raiz string) map[string]agent.Agente {
	dir := filepath.Join(raiz, ".localcli", "agents")
	out := map[string]agent.Agente{}
	if entradas, err := os.ReadDir(dir); err == nil {
		for _, e := range entradas {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
				continue
			}
			a, err := agent.Cargar(filepath.Join(dir, e.Name()))
			if err != nil {
				fmt.Fprintln(os.Stderr, "aviso: agente ignorado:", err)
				continue
			}
			out[a.Nombre] = a
		}
	}
	for _, nombre := range []string{tools.AgentePlan, tools.AgenteBuild} {
		if _, ok := out[nombre]; !ok {
			out[nombre] = agenteDeRespaldo(nombre)
		}
	}
	return out
}

// agenteDeRespaldo es el agente mínimo que se usa cuando falta o no carga su
// JSON: prompt de identidad y sin permisos (solo conversación). No inventa
// herramientas; si el usuario quiere que las tenga, escribe su `.localcli/agents`.
func agenteDeRespaldo(nombre string) agent.Agente {
	return agent.Agente{
		Nombre: nombre,
		Prompt: "Eres el agente `" + nombre + "` de LocalCli. Responde en español, concreto y breve.",
		Skills: []string{},
	}
}

// mensajesDeOllama traduce el historial neutro de `flow` al formato del cliente
// de Ollama. Es la frontera donde la conversación deja de ser un dato de
// `session`/`flow` y pasa a ser lo que se le envía al modelo.
func mensajesDeOllama(historial []flow.Mensaje) []ollama.Mensaje {
	if len(historial) == 0 {
		return nil
	}
	out := make([]ollama.Mensaje, 0, len(historial))
	for _, m := range historial {
		out = append(out, ollama.Mensaje{Role: m.Rol, Content: m.Texto})
	}
	return out
}

// generarTextoCorrido lanza una generación de un solo turno contra el modelo y
// devuelve el texto acumulado. Pasa por la FIFO para no solaparse con un turno
// en curso. La usan el título de sesión y el resumen de conversación, que no se
// pintan en la TUI: solo importa el texto final.
func generarTextoCorrido(ctx context.Context, ad *Adaptador, infer *ollama.ColaInferencia, prompt string) (string, error) {
	modelo := ad.modeloElegido()
	if modelo == "" {
		return "", errors.New("arranque: no hay modelo de Ollama disponible")
	}
	var b strings.Builder
	err := infer.Encolar(ctx, ollama.OpcionesEncolar{IdSesion: ad.sesionActual()}, func(c context.Context) error {
		ch, err := ad.cliente.Chat(c, ollama.GenerarRequest{
			Model:    modelo,
			Messages: []ollama.Mensaje{{Role: "user", Content: prompt}},
			NumCtx:   ad.ventana(modelo),
		})
		if err != nil {
			return err
		}
		for ev := range ch {
			switch ev.Tipo {
			case ollama.EventoToken:
				b.WriteString(ev.Texto)
			case ollama.EventoError:
				return ev.Error
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return b.String(), nil
}

// tituladorPorTurno implementa `session.Titulador`: pide al modelo un título
// para la sesión a partir de su primera petición. Es una generación corta que no
// se pinta en la TUI; el nombre final llega por el evento `titulo_sesion`.
type tituladorPorTurno struct {
	ad    *Adaptador
	infer *ollama.ColaInferencia
}

func (t *tituladorPorTurno) Titulo(ctx context.Context, texto string) (string, error) {
	crudo, err := generarTextoCorrido(ctx, t.ad, t.infer, session.PromptTitulo(texto))
	if err != nil {
		return "", err
	}
	return session.LimpiarTitulo(crudo), nil
}

// resumidorPorTurno implementa `session.Resumidor`: condensa la conversación
// antigua cuando no cabe en el presupuesto de contexto. Igual que el título, es
// una generación de un solo turno.
type resumidorPorTurno struct {
	ad    *Adaptador
	infer *ollama.ColaInferencia
}

func (r *resumidorPorTurno) Resumir(ctx context.Context, transcripcion string) (string, error) {
	crudo, err := generarTextoCorrido(ctx, r.ad, r.infer, session.PromptResumen(transcripcion))
	if err != nil {
		return "", err
	}
	return session.LimpiarResumen(crudo), nil
}
