// localcli arranca la interfaz, ya cableada al modelo, a los agentes y a la base.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"localcli/internal/agent"
	"localcli/internal/arranque"
	"localcli/internal/exec"
	"localcli/internal/fileops"
	"localcli/internal/flow"
	"localcli/internal/llama"
	"localcli/internal/llm"
	"localcli/internal/llm/llamacpp"
	"localcli/internal/llm/openai"
	"localcli/internal/models"
	"localcli/internal/queue"
	"localcli/internal/session"
	"localcli/internal/store"
	"localcli/internal/task"
	"localcli/internal/tools"
	"localcli/internal/tui"
)

// version es la versión del binario. Se puede inyectar en compilación con
// -ldflags "-X main.version=<version>"; sin flags conserva el valor por defecto.
var version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		if err := correr(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	switch cmd := os.Args[1]; cmd {
	case "-v":
		fmt.Println("localcli version " + version)
	case "-h":
		fmt.Println("comandos disponibles:")
		fmt.Println("  localcli        arranca la interfaz")
		fmt.Println("  localcli init    inicializa la raíz global (~/.config/localcli/)")
		fmt.Println("  localcli create  crea configuración del proyecto (.localcli/)")
		fmt.Println("  localcli -h      muestra esta ayuda")
		fmt.Println("  localcli -v      muestra la versión")
	case "init":
		if err := arranque.Init(); err != nil {
			fmt.Fprintf(os.Stderr, "error al inicializar: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("localcli inicializado correctamente")
	case "create":
		if err := arranque.Create(""); err != nil {
			fmt.Fprintf(os.Stderr, "error al crear configuración del proyecto: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("configuración del proyecto creada correctamente")
	default:
		fmt.Fprintf(os.Stderr, "comando desconocido: %s\n", cmd)
		os.Exit(1)
	}
}

// correr cablea la interfaz: base global, agentes y cliente del motor.
func correr() error {
	if !arranque.EstaInicializada() {
		return fmt.Errorf("localcli no iniciada, ejecuta el comando `localcli init`")
	}

	db, err := store.AbrirDB()
	if err != nil {
		return fmt.Errorf("no se pudo abrir la base de datos: %w", err)
	}
	defer db.Close()

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	projectID, err := store.ResolucionProyecto(db, cwd)
	if err != nil {
		return err
	}

	raiz, err := arranque.RaizMaquina()
	if err != nil {
		return err
	}
	agentes, avisos := agent.CargarAditivo(filepath.Join(raiz, "agents"), filepath.Join(cwd, ".localcli", "agents"))
	for _, aviso := range avisos {
		fmt.Fprintln(os.Stderr, aviso)
	}

	flujos, avisosFlujos := flow.CargarAditivo(filepath.Join(raiz, "flows"), filepath.Join(cwd, ".localcli", "flows"))
	for _, aviso := range avisosFlujos {
		fmt.Fprintln(os.Stderr, aviso)
	}

	cfg, err := arranque.LeerConfiguracion()
	if err != nil {
		return err
	}

	gestor := models.NuevoGestor(filepath.Join(raiz, "models.json"))
	catalogo, err := gestor.Cargar()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	if catalogo != nil {
		for _, m := range catalogo.Modelos {
			if m.Usable() && m.ContextoReal() <= 0 {
				fmt.Fprintln(os.Stderr, "modelo "+m.ID+": sin contexto declarado (ventana 0); se usará el defecto del harness")
			}
		}
	}
	activo, hayActivo := catalogo.Activo()

	// El contexto con el que se arranca el servidor sale del modelo (su GGUF),
	// no de un valor fijo del harness: así la ventana es la real del modelo.
	contextoDeseado := contextoDeArranque(activo, hayActivo)

	runtime, err := llama.Nuevo("", cfg.API, contextoDeseado)
	if err != nil {
		return err
	}
	defer runtime.Detener()
	if hayActivo {
		for _, aviso := range arrancarServidor(runtime, activo, contextoDeseado) {
			fmt.Fprintln(os.Stderr, aviso)
		}
	}

	cliente := openai.NuevoCliente(cfg.API)
	modelo := modeloElegido(activo, hayActivo, cliente)
	ventana := ventanaEfectiva(cfg.API, contextoDeseado)

	orquestador := session.Nuevo(db, projectID, cwd, agentes, cliente, modelo, ventana, runtime, gestor)
	orquestador.ConCapacidades(resolverCapacidades(cfg.API))
	orquestador.ConResolucionCapacidades(func() llm.Capacidades { return resolverCapacidades(cfg.API) })
	orquestador.ConHerramientas(montarHerramientas(db, projectID, cwd))
	orquestador.ConFlujos(flujos)
	derivarColas(orquestador, cwd)
	return tui.Correr(orquestador)
}

// derivarColas deriva al arrancar la cola del TODO de cada capa presente en
// `ai/tasks/`, y la entrega al orquestador. Una capa sin TODO se salta: la cola
// no se inventa.
func derivarColas(orquestador *session.Orquestador, cwd string) {
	for _, capa := range []task.Capa{task.CapaBackend, task.CapaFrontend, task.CapaDatabase} {
		cola, err := queue.Reconstruir(cwd, capa)
		if err != nil {
			continue
		}
		orquestador.ConCola(cola)
	}
}

// montarHerramientas construye el catálogo con los handlers de fileops y exec ya
// inyectados, y devuelve su vista para el agente.
func montarHerramientas(db *sql.DB, projectID, cwd string) (*tools.Registro, []agent.Herramienta) {
	registro := tools.NuevoRegistro(tools.Catalogo())

	operaciones := &fileops.Operaciones{Raiz: cwd, DB: db, ProjectID: projectID}
	registro.Instalar("leer_archivo", operaciones.LeerHandler())
	registro.Instalar("listar_carpeta", operaciones.ListarHandler())
	registro.Instalar("buscar_archivos", operaciones.BuscarArchivosHandler())
	registro.Instalar("buscar_en_archivos", operaciones.BuscarEnArchivosHandler())
	for nombre, handler := range operaciones.EscribirHandlers() {
		registro.Instalar(nombre, handler)
	}

	ejecutor := &exec.Ejecutor{Proyecto: cwd}
	registro.Instalar("ejecutar_comando", ejecutor.Handler())

	return registro, adaptarDefiniciones(registro.Definiciones())
}

func adaptarDefiniciones(definiciones []tools.Definicion) []agent.Herramienta {
	salida := make([]agent.Herramienta, 0, len(definiciones))
	for _, d := range definiciones {
		salida = append(salida, agent.Herramienta{
			Nombre: d.Nombre, Descripcion: d.Descripcion, Permiso: string(d.Permiso), Esquema: d.Esquema,
		})
	}
	return salida
}

// arrancarServidor asegura el servidor con el modelo del catálogo y recoge los
// avisos; un fallo de arranque no detiene el harness.
func arrancarServidor(runtime *llama.Servidor, activo models.Entrada, contexto int) []string {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	avisos, err := runtime.Asegurar(ctx, llama.Modelo{
		Nombre: activo.ID, Ruta: activo.RutaReal(), MMProj: activo.Proyector(), Contexto: contexto,
	})
	if err != nil {
		avisos = append(avisos, err.Error())
	}
	return avisos
}

// modeloElegido es el nombre del modelo activo: el del catálogo cuando su GGUF
// sigue en disco; si no, el que reporte el servidor.
func modeloElegido(e models.Entrada, hay bool, cliente *openai.Cliente) string {
	if hay && e.Usable() {
		return e.ID
	}
	nombre, _ := primerModelo(cliente)
	return nombre
}

// contextoDeArranque resuelve el contexto con el que se arranca `llama-server`:
// el override del usuario, si no el contexto real del modelo (su GGUF) y, si no
// lo declara, el defecto del harness.
func contextoDeArranque(activo models.Entrada, hay bool) int {
	contextoModelo := 0
	if hay && activo.Usable() {
		contextoModelo = activo.ContextoReal()
	}
	return llm.VentanaDeArranque(arranque.LimiteContextoOverride(), contextoModelo, arranque.LimiteContextoPorDefecto)
}

// ventanaEfectiva resuelve la ventana que el turno debe respetar: la que de
// verdad tiene el servidor (`props` de llama.cpp), que es la que rechaza con 400
// lo que no cabe. Si no se puede leer, el contexto con el que se arrancó.
func ventanaEfectiva(api string, contextoDeseado int) int {
	if p, err := llamacpp.Leer(context.Background(), api); err == nil && p.Ventana > 0 {
		return p.Ventana
	}
	return contextoDeseado
}

// resolverCapacidades lee del motor el estado de visión del modelo activo. Sin
// dato queda desconocida: una extensión ausente no es una capacidad ausente.
func resolverCapacidades(api string) llm.Capacidades {
	caps := llm.TDesconocidas()
	if p, err := llamacpp.Leer(context.Background(), api); err == nil {
		caps.Vision = p.Vision
	}
	return caps
}

// primerModelo resuelve el modelo activo y su ventana con una lectura del motor.
// Si el motor no responde, se arranca sin modelo.
func primerModelo(cliente *openai.Cliente) (string, int) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	modelos, err := cliente.Modelos(ctx)
	if err != nil || len(modelos) == 0 {
		return "", 0
	}
	return modelos[0].ID, modelos[0].Ventana
}
