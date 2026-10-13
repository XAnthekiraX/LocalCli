package session

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"localcli/internal/agent"
	"localcli/internal/flow"
	"localcli/internal/llm/openai"
	"localcli/internal/store"
	"localcli/internal/tools"
	"localcli/internal/tui"
)

func flujoDePrueba() flow.Flujo {
	return flow.Flujo{
		Nombre: "prueba", Comando: "/prueba", Descripcion: "d", Peticion: "algo por defecto", Pregunta: "¿qué?",
		Etapas: []flow.Etapa{
			{ID: "uno", Nombre: "Uno", Agente: "plan", Pregunta: "primera"},
			{ID: "fin", Nombre: "Fin", Agente: "plan", Pregunta: "segunda", Entrega: true},
		},
	}
}

func catalogoConFlujo(f flow.Flujo) *flow.Catalogo {
	c := flow.NuevoCatalogo()
	c.Agregar(f)
	return c
}

// servidorEtapas responde cada turno con un texto numerado y las peticiones de
// título con un título fijo.
func servidorEtapas(t *testing.T) *httptest.Server {
	t.Helper()
	var n int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cuerpo, _ := io.ReadAll(r.Body)
		respuesta := "Titulo"
		if !bytes.Contains(cuerpo, []byte(marcaTitulo)) {
			respuesta = fmt.Sprintf("respuesta-%d", atomic.AddInt32(&n, 1))
		}
		escribirSSE(w, respuesta)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func orquestadorDeFlujo(t *testing.T) (*Orquestador, string) {
	t.Helper()
	srv := servidorEtapas(t)
	db, limpiar, _ := store.CrearDBTemporal()
	t.Cleanup(func() { _ = limpiar() })
	cwd := t.TempDir()
	projectID, _ := store.ResolucionProyecto(db, cwd)
	o := Nuevo(db, projectID, cwd, agentesDePrueba(t), openai.NuevoCliente(srv.URL), "m", 100000, nil, nil)
	o.ConHerramientas(nil, nil)
	o.ConFlujos(catalogoConFlujo(flujoDePrueba()))
	sesion, err := o.Crear()
	if err != nil {
		t.Fatal(err)
	}
	return o, sesion.ID
}

// TestElegirFlujoSoloReconoceFlow: un flujo se reconoce únicamente con el
// prefijo `/flow:` —el `/comando` crudo queda para los comandos de la
// interfaz—, y sin objetivo vale la petición por defecto del flujo.
func TestElegirFlujoSoloReconoceFlow(t *testing.T) {
	o, _ := orquestadorDeFlujo(t)

	if _, _, ok := o.elegirFlujo("/prueba algo"); ok {
		t.Fatal("un `/comando` crudo no debería arrancar un flujo")
	}
	f, objetivo, ok := o.elegirFlujo("/flow:prueba algo")
	if !ok || f.Comando != "/prueba" || objetivo != "algo" {
		t.Fatalf("`/flow:prueba` no resolvió: flujo=%+v objetivo=%q ok=%v", f, objetivo, ok)
	}
	if _, objetivo, ok := o.elegirFlujo("/flow:prueba"); !ok || objetivo != "algo por defecto" {
		t.Fatalf("sin objetivo debería valer la petición por defecto: %q (ok=%v)", objetivo, ok)
	}
	if _, _, ok := o.elegirFlujo("/flow:inexistente"); ok {
		t.Fatal("un `/flow:` desconocido no debería resolverse")
	}
	if _, _, ok := o.elegirFlujo("/flow:"); ok {
		t.Fatal("`/flow:` sin nombre no debería resolverse")
	}
}

// TestUnComandoCorreElFlujoYEntrega: un `/flow:<comando>` no responde como chat,
// corre las etapas, guarda el bloque y deja la entrega como respuesta del turno.
// El `/comando` crudo ya no arranca un flujo.
func TestUnComandoCorreElFlujoYEntrega(t *testing.T) {
	o, sesion := orquestadorDeFlujo(t)
	canal, cerrar := o.Suscribir()
	defer cerrar()

	if err := o.Enviar(context.Background(), sesion, "plan", "/flow:prueba algo"); err != nil {
		t.Fatalf("enviar: %v", err)
	}

	var iniciadas []string
	for _, e := range drenar(canal) {
		if e.Tipo == tui.EventoEtapaIniciada {
			iniciadas = append(iniciadas, e.Datos["nombre"])
		}
	}
	if strings.Join(iniciadas, "|") != "Uno|Fin" {
		t.Fatalf("etapas iniciadas = %v", iniciadas)
	}

	bloque, err := store.ListarFlowContext(o.db, o.projectID, sesion, "prueba")
	if err != nil || len(bloque) != 2 {
		t.Fatalf("el bloque debería tener dos aportes: %+v (%v)", bloque, err)
	}
	if bloque[0].Nombre != "Uno" || bloque[1].Nombre != "Fin" {
		t.Fatalf("bloque = %+v", bloque)
	}

	mensajes, _ := store.ListarMensajes(o.db, o.projectID, sesion)
	if len(mensajes) != 2 || mensajes[1].Role != "agent" {
		t.Fatalf("mensajes = %+v", mensajes)
	}
	if mensajes[1].Content != "respuesta-2" {
		t.Fatalf("la entrega debería ser la respuesta del turno: %q", mensajes[1].Content)
	}
}

// TestEjecutarFlujoDevuelveLaEntregaYElBloque: el handler de la herramienta corre
// el flujo y devuelve entrega + bloque, y respeta el permiso derivado.
func TestEjecutarFlujoDevuelveLaEntregaYElBloque(t *testing.T) {
	o, sesion := orquestadorDeFlujo(t)

	res, err := o.ejecutarFlujo(context.Background(), tools.PeticionEjecutarFlujo{Flujo: "/prueba", Peticion: "algo"}, tools.Contexto{SesionID: sesion, Permisos: []string{"read"}})
	if err != nil || res.Error != "" {
		t.Fatalf("ejecutar_flujo: %+v %v", res, err)
	}
	if !strings.Contains(res.Salida, "respuesta-2") || !strings.Contains(res.Salida, "## Fin") {
		t.Fatalf("la salida debería traer la entrega y el bloque: %q", res.Salida)
	}

	// Un flujo desconocido se rechaza.
	res, _ = o.ejecutarFlujo(context.Background(), tools.PeticionEjecutarFlujo{Flujo: "/nada"}, tools.Contexto{SesionID: sesion, Permisos: []string{"read"}})
	if !strings.HasPrefix(res.Error, tools.CodFlowUnknown) {
		t.Fatalf("un flujo desconocido debería dar E_FLOW_UNKNOWN: %q", res.Error)
	}
}

func TestEjecutarFlujoRechazaPermisoInsuficiente(t *testing.T) {
	o, sesion := orquestadorDeFlujo(t)
	f := flujoDePrueba()
	f.Nombre = "escribe"
	f.Comando = "/escribe"
	f.Etapas = append([]flow.Etapa{{ID: "b", Nombre: "Build", Agente: "build", Pregunta: "p"}}, f.Etapas...)
	c := flow.NuevoCatalogo()
	c.Agregar(f)
	o.ConFlujos(c)

	res, _ := o.ejecutarFlujo(context.Background(), tools.PeticionEjecutarFlujo{Flujo: "/escribe"}, tools.Contexto{SesionID: sesion, Permisos: []string{"read"}})
	if !strings.HasPrefix(res.Error, tools.CodFlowNotAllowed) {
		t.Fatalf("plan no debería lanzar un flujo que escribe: %q", res.Error)
	}
}

// TestLaEtapaNoVeEjecutarFlujo: el catálogo del turno incluye `ejecutar_flujo`
// con su enum, pero el de una etapa no (profundidad uno).
func TestLaEtapaNoVeEjecutarFlujo(t *testing.T) {
	db, limpiar, _ := store.CrearDBTemporal()
	defer limpiar()
	cwd := t.TempDir()
	projectID, _ := store.ResolucionProyecto(db, cwd)
	o := Nuevo(db, projectID, cwd, agentesConBuild(t), openai.NuevoCliente("http://127.0.0.1:1"), "m", 0, nil, nil)
	registro := tools.NuevoRegistro(tools.Catalogo())
	o.ConHerramientas(registro, catalogoDelRegistro(registro))
	o.ConFlujos(catalogoConFlujo(flujoDePrueba()))

	build, _ := o.agentes.Buscar("build")

	turno := o.catalogoDelTurno(build)
	if !tieneHerramienta(turno, "ejecutar_flujo") {
		t.Fatal("el catálogo del turno debería incluir `ejecutar_flujo`")
	}

	etapa := agent.CatalogoEfectivo(build, o.catalogoHerramientas)
	if tieneHerramienta(etapa, "ejecutar_flujo") {
		t.Fatal("una etapa no debería ver `ejecutar_flujo` (profundidad uno)")
	}
}

func tieneHerramienta(lista []agent.Herramienta, nombre string) bool {
	for _, h := range lista {
		if h.Nombre == nombre {
			return true
		}
	}
	return false
}
