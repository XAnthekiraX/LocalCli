package session

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"localcli/internal/agent"
	"localcli/internal/flow"
	"localcli/internal/llm/openai"
	"localcli/internal/store"
	"localcli/internal/tools"
	"localcli/internal/tui"
)

// flujoDePrueba es un flujo del esquema nuevo: dos etapas lineales, sin agente
// propio y sin entrega declarada — la última etapa entrega siempre.
func flujoDePrueba() flow.Flujo {
	return flow.Flujo{
		Comando: "/prueba",
		Reglas:  []string{"regla de prueba"},
		Etapas: []flow.Etapa{
			{ID: "uno", Nombre: "Uno", Orden: "primera etapa", Respuesta: "aporte de la etapa uno"},
			{ID: "fin", Nombre: "Fin", Orden: "última etapa", Respuesta: "cierre del flujo"},
		},
	}
}

// flujoEnLoop es el mismo flujo con su primera etapa como bucle.
func flujoEnLoop() flow.Flujo {
	return flow.Flujo{
		Comando: "/vueltas",
		Etapas: []flow.Etapa{
			{ID: "vuelta", Nombre: "Vueltas", Orden: "repite hasta que esté resuelto", Type: flow.TipoLoop, MaxIteraciones: 5},
			{ID: "fin", Nombre: "Fin", Orden: "resume lo conseguido", Respuesta: "resumen final"},
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

// servidorLoop responde `false` las primeras `fallos` veces y después `true`.
func servidorLoop(t *testing.T, fallos int32) *httptest.Server {
	t.Helper()
	var n int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cuerpo, _ := io.ReadAll(r.Body)
		if bytes.Contains(cuerpo, []byte(marcaTitulo)) {
			escribirSSE(w, "Titulo")
			return
		}
		intento := atomic.AddInt32(&n, 1)
		if intento <= fallos {
			escribirSSE(w, "intento " + fmt.Sprint(intento) + " falló\nfalse")
			return
		}
		escribirSSE(w, "resuelto en el intento " + fmt.Sprint(intento) + "\ntrue")
	}))
	t.Cleanup(srv.Close)
	return srv
}

func orquestadorConServidor(t *testing.T, srv *httptest.Server, c *flow.Catalogo) (*Orquestador, string) {
	t.Helper()
	db, limpiar, _ := store.CrearDBTemporal()
	t.Cleanup(func() { _ = limpiar() })
	cwd := t.TempDir()
	projectID, _ := store.ResolucionProyecto(db, cwd)
	o := Nuevo(db, projectID, cwd, agentesDePrueba(t), openai.NuevoCliente(srv.URL), "m", 100000, nil, nil)
	o.ConHerramientas(nil, nil)
	o.ConFlujos(c)
	sesion, err := o.Crear()
	if err != nil {
		t.Fatal(err)
	}
	return o, sesion.ID
}

func orquestadorDeFlujo(t *testing.T) (*Orquestador, string) {
	t.Helper()
	return orquestadorConServidor(t, servidorEtapas(t), catalogoConFlujo(flujoDePrueba()))
}

// TestElegirFlujoSoloReconoceFlow: un flujo se reconoce únicamente con el
// prefijo `/flow:` —el `/comando` crudo queda para los comandos de la interfaz—,
// y la petición es obligatoria: sin ella se avisa en vez de arrancar.
func TestElegirFlujoSoloReconoceFlow(t *testing.T) {
	o, _ := orquestadorDeFlujo(t)

	if e := o.elegirFlujo("/prueba algo"); e.esFlujo {
		t.Fatal("un `/comando` crudo no debería arrancar un flujo")
	}
	e := o.elegirFlujo("/flow:prueba algo")
	if !e.esFlujo || e.flujo.Comando != "/prueba" || e.peticion != "algo" || e.sinPeticion {
		t.Fatalf("`/flow:prueba algo` no resolvió bien: %+v", e)
	}
	sinPeticion := o.elegirFlujo("/flow:prueba")
	if !sinPeticion.esFlujo || !sinPeticion.sinPeticion {
		t.Fatalf("sin petición debería avisarse, no arrancar: %+v", sinPeticion)
	}
	if e := o.elegirFlujo("/flow:inexistente"); e.esFlujo {
		t.Fatal("un `/flow:` desconocido no debería resolverse")
	}
	if e := o.elegirFlujo("/flow:"); e.esFlujo {
		t.Fatal("`/flow:` sin nombre no debería resolverse")
	}
}

// La cola es la excepción a la petición obligatoria: su objetivo lo trae cada
// elemento de la cola, no la línea.
func TestLaColaArrancaSinPeticion(t *testing.T) {
	o, _ := orquestadorDeFlujo(t)
	c := flow.NuevoCatalogo()
	c.Agregar(flujoDePrueba())
	c.Agregar(flow.Flujo{Comando: ComandoEjecutar, Etapas: []flow.Etapa{
		{ID: "e", Nombre: "Ejecutar", Orden: "recorre la cola", Respuesta: "estado de la cola"},
	}})
	o.ConFlujos(c)

	e := o.elegirFlujo("/flow:ejecutar")
	if !e.esFlujo || e.sinPeticion {
		t.Fatalf("la cola debería arrancar sin petición: %+v", e)
	}
}

// TestUnComandoCorreElFlujoYEntrega: un `/flow:<comando>` no responde como chat,
// corre las etapas, guarda el bloque y deja la entrega —la última etapa— como
// respuesta del turno.
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

	bloque, err := store.ListarFlowContext(o.db, o.projectID, sesion, "/prueba")
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

// TestUnFlujoSinPeticionNoArranca: se avisa y no se corre ninguna etapa; el
// mensaje no se manda al modelo.
func TestUnFlujoSinPeticionNoArranca(t *testing.T) {
	o, sesion := orquestadorDeFlujo(t)
	canal, cerrar := o.Suscribir()
	defer cerrar()

	if err := o.Enviar(context.Background(), sesion, "plan", "/flow:prueba"); err != nil {
		t.Fatalf("enviar: %v", err)
	}

	var avisos []string
	var etapas int
	for _, e := range drenar(canal) {
		switch e.Tipo {
		case tui.EventoNotificacion:
			avisos = append(avisos, e.Datos["motivo"])
		case tui.EventoEtapaIniciada:
			etapas++
		}
	}
	if etapas != 0 {
		t.Fatalf("no debería haber corrido etapas; inició %d", etapas)
	}
	if !contiene(avisos, "necesita una petición") {
		t.Fatalf("debería avisar que falta la petición: %q", avisos)
	}
}

// TestUnFlujoConHistorialSeLeEnvia: el historial de la sesión viaja junto a la
// petición en todas las etapas, y el propio mensaje `/flow:` no se duplica.
func TestUnFlujoConHistorialSeLeEnvia(t *testing.T) {
	var mu sync.Mutex
	var peticiones []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cuerpo, _ := io.ReadAll(r.Body)
		if !bytes.Contains(cuerpo, []byte(marcaTitulo)) {
			mu.Lock()
			peticiones = append(peticiones, string(cuerpo))
			mu.Unlock()
		}
		escribirSSE(w, "respuesta")
	}))
	t.Cleanup(srv.Close)
	o, sesion := orquestadorConServidor(t, srv, catalogoConFlujo(flujoDePrueba()))

	if err := o.Enviar(context.Background(), sesion, "plan", "hola, esto es una conversación previa"); err != nil {
		t.Fatalf("enviar: %v", err)
	}
	if err := o.Enviar(context.Background(), sesion, "plan", "/flow:prueba algo"); err != nil {
		t.Fatalf("enviar el flujo: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(peticiones) < 2 {
		t.Fatalf("las dos etapas deberían haber pedido turno: %d", len(peticiones))
	}
	ultima := peticiones[len(peticiones)-1]
	if !strings.Contains(ultima, "conversación previa") {
		t.Error("la etapa debería ver el historial de la sesión")
	}
	if !strings.Contains(ultima, "Petición original: algo") {
		t.Error("la etapa debería ver la petición original")
	}
	if strings.Contains(ultima, "/flow:prueba") {
		t.Error("el mensaje `/flow:` no debe duplicarse: la petición ya viaja en la ventana")
	}
}

// TestLaEtapaLoopRepiteHastaTrue: la etapa `loop` repite mientras responda
// `false`, los intentos fallidos no quedan en el bloque y el acierto sí.
func TestLaEtapaLoopRepiteHastaTrue(t *testing.T) {
	o, sesion := orquestadorConServidor(t, servidorLoop(t, 2), catalogoConFlujo(flujoEnLoop()))

	if err := o.Enviar(context.Background(), sesion, "plan", "/flow:vueltas deja todo en verde"); err != nil {
		t.Fatalf("enviar: %v", err)
	}

	bloque, err := store.ListarFlowContext(o.db, o.projectID, sesion, "/vueltas")
	if err != nil || len(bloque) != 2 {
		t.Fatalf("el bloque debería tener dos aportes: %+v (%v)", bloque, err)
	}
	if !strings.Contains(bloque[0].Contenido, "resuelto en el intento 3") {
		t.Fatalf("el aporte del bucle debe ser el intento resuelto: %q", bloque[0].Contenido)
	}
}

// TestLaEtapaLoopSeAgota: si nadie responde `true`, la etapa falla con un error
// que dice cuántos intentos tuvo.
func TestLaEtapaLoopSeAgota(t *testing.T) {
	o, sesion := orquestadorConServidor(t, servidorLoop(t, 999), catalogoConFlujo(flujoEnLoop()))

	err := o.Enviar(context.Background(), sesion, "plan", "/flow:vueltas nunca se pone verde")
	if err == nil {
		t.Fatal("una etapa que agota sus iteraciones debería fallar")
	}
	if !strings.Contains(err.Error(), "true") {
		t.Fatalf("el error debería decir que no respondió `true`: %v", err)
	}
}

func TestPartirVeredicto(t *testing.T) {
	casos := []struct {
		nombre     string
		texto     string
		resumen   string
		listo     bool
		valido    bool
	}{
		{"una línea true", "true", "", true, true},
		{"resumen más veredicto", "todo en verde\ntrue", "todo en verde", true, true},
		{"resumen más false", "aún no\ntodo falla\nfalse", "aún no\ntodo falla", false, true},
		{"comillas invertidas", "listo\n`true`", "listo", true, true},
		{"mayúsculas", "listo\nTrue", "listo", true, true},
		{"sin veredicto", "todo en verde", "", false, false},
		{"vacío", "   \n  ", "", false, false},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			resumen, listo, ok := partirVeredicto(c.texto)
			if ok != c.valido || listo != c.listo || resumen != c.resumen {
				t.Fatalf("partirVeredicto(%q) = (%q, %v, %v)", c.texto, resumen, listo, ok)
			}
		})
	}
}

// TestEjecutarFlujoDevuelveLaEntregaYElBloque: el handler de la herramienta corre
// el flujo con el agente que la invocó y devuelve entrega + bloque.
func TestEjecutarFlujoDevuelveLaEntregaYElBloque(t *testing.T) {
	o, sesion := orquestadorDeFlujo(t)

	res, err := o.ejecutarFlujo(context.Background(), tools.PeticionEjecutarFlujo{Flujo: "/prueba", Peticion: "algo"}, tools.Contexto{SesionID: sesion, Agente: "plan"})
	if err != nil || res.Error != "" {
		t.Fatalf("ejecutar_flujo: %+v %v", res, err)
	}
	if !strings.Contains(res.Salida, "respuesta-2") || !strings.Contains(res.Salida, "## Fin") {
		t.Fatalf("la salida debería traer la entrega y el bloque: %q", res.Salida)
	}

	// Un flujo desconocido se rechaza.
	res, _ = o.ejecutarFlujo(context.Background(), tools.PeticionEjecutarFlujo{Flujo: "/nada"}, tools.Contexto{SesionID: sesion, Agente: "plan"})
	if !strings.HasPrefix(res.Error, tools.CodFlowUnknown) {
		t.Fatalf("un flujo desconocido debería dar E_FLOW_UNKNOWN: %q", res.Error)
	}

	// Y un agente que no existe también.
	res, _ = o.ejecutarFlujo(context.Background(), tools.PeticionEjecutarFlujo{Flujo: "/prueba", Peticion: "algo"}, tools.Contexto{SesionID: sesion, Agente: "nadie"})
	if !strings.HasPrefix(res.Error, tools.CodFlowUnknown) {
		t.Fatalf("un agente desconocido debería dar E_FLOW_UNKNOWN: %q", res.Error)
	}
}

// TestLaEtapaAbortaCuandoFaltaUnPermiso: las etapas corren con el agente activo;
// si piden una herramienta que ese agente no tiene concedida, la etapa se aborta
// y lo dice con nombre y herramienta, en vez de seguir a ciegas.
func TestLaEtapaAbortaCuandoFaltaUnPermiso(t *testing.T) {
	// El modelo pide crear un archivo, que es de permiso `write`.
	srv := servidorTool(t, "crear_archivo", map[string]string{"ruta": "x.txt", "contenido": "hola"})
	db, limpiar, _ := store.CrearDBTemporal()
	defer limpiar()
	cwd := t.TempDir()
	projectID, _ := store.ResolucionProyecto(db, cwd)
	registro := tools.NuevoRegistro(tools.Catalogo())
	o := Nuevo(db, projectID, cwd, agentesDePrueba(t), openai.NuevoCliente(srv.URL), "m", 0, nil, nil)
	o.ConHerramientas(registro, catalogoDelRegistro(registro))
	o.ConFlujos(catalogoConFlujo(flujoDePrueba()))
	sesion, err := o.Crear()
	if err != nil {
		t.Fatal(err)
	}

	// El agente activo es `plan`, de solo lectura.
	err = o.Enviar(context.Background(), sesion.ID, "plan", "/flow:prueba crea un archivo")
	if err == nil {
		t.Fatal("la etapa debería abortar cuando falta el permiso")
	}
	if !strings.Contains(err.Error(), "crear_archivo") || !strings.Contains(err.Error(), "plan") {
		t.Fatalf("el error debería nombrar la herramienta y el agente: %v", err)
	}

	// Una etapa abortada no deja aporte a medias en el bloque.
	bloque, _ := store.ListarFlowContext(o.db, o.projectID, sesion.ID, "/prueba")
	if len(bloque) != 0 {
		t.Fatalf("una etapa abortada no debería dejar aporte: %+v", bloque)
	}
}

// La misma negación cruzada por la herramienta: el agente invocador no tiene los
// permisos que la etapa pide.
func TestEjecutarFlujoInformaLaFaltaDePermiso(t *testing.T) {
	srv := servidorTool(t, "crear_archivo", map[string]string{"ruta": "x.txt", "contenido": "hola"})
	db, limpiar, _ := store.CrearDBTemporal()
	defer limpiar()
	cwd := t.TempDir()
	projectID, _ := store.ResolucionProyecto(db, cwd)
	registro := tools.NuevoRegistro(tools.Catalogo())
	o := Nuevo(db, projectID, cwd, agentesDePrueba(t), openai.NuevoCliente(srv.URL), "m", 0, nil, nil)
	o.ConHerramientas(registro, catalogoDelRegistro(registro))
	o.ConFlujos(catalogoConFlujo(flujoDePrueba()))
	sesion, err := o.Crear()
	if err != nil {
		t.Fatal(err)
	}

	res, _ := o.ejecutarFlujo(context.Background(),
		tools.PeticionEjecutarFlujo{Flujo: "/prueba", Peticion: "crea un archivo"},
		tools.Contexto{SesionID: sesion.ID, Agente: "plan"})
	if !strings.HasPrefix(res.Error, tools.CodFlowNotAllowed) {
		t.Fatalf("una etapa sin permisos debería dar E_FLOW_NOT_ALLOWED: %q", res.Error)
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

// flujoVisible es un flujo cuyas dos etapas piden verse en el chat
// (`respuesta_en_chat`).
func flujoVisible() flow.Flujo {
	return flow.Flujo{
		Comando: "/visible",
		Etapas: []flow.Etapa{
			{ID: "uno", Nombre: "Uno", Orden: "primera etapa", Respuesta: "aporte de la etapa uno", RespuestaEnChat: true},
			{ID: "fin", Nombre: "Fin", Orden: "última etapa", Respuesta: "cierre del flujo", RespuestaEnChat: true},
		},
	}
}

// TestLaRespuestaEnChatSePintaYSeGuarda: una etapa con `respuesta_en_chat` deja su
// respuesta completa en el hilo —el evento que la vista pinta como bloque— y una
// fila en `chat_evento` con su tipo propio; ya no sale como aviso bajo el input
// (SPEC-ORQUESTADOR-FLUJOS §Visibilidad de una respuesta). La etapa cuyo texto ya
// viaja token a token al turno no se pinta dos veces.
func TestLaRespuestaEnChatSePintaYSeGuarda(t *testing.T) {
	o, sesion := orquestadorConServidor(t, servidorEtapas(t), catalogoConFlujo(flujoVisible()))
	canal, cerrar := o.Suscribir()
	defer cerrar()

	if err := o.Enviar(context.Background(), sesion, "plan", "/flow:visible algo"); err != nil {
		t.Fatalf("enviar: %v", err)
	}

	var respuestas, avisos []string
	for _, e := range drenar(canal) {
		switch e.Tipo {
		case tui.EventoEtapaRespuesta:
			respuestas = append(respuestas, e.Datos["texto"])
		case tui.EventoNotificacion:
			avisos = append(avisos, e.Datos["motivo"])
		}
	}
	if strings.Join(respuestas, "|") != "respuesta-1" {
		t.Fatalf("respuestas pintadas en el hilo = %v, quiero solo la de la primera etapa", respuestas)
	}
	if contiene(avisos, "respuesta-1") {
		t.Fatalf("la respuesta ya no debería salir como aviso bajo el input: %v", avisos)
	}

	eventos, err := store.ListarChatEventos(o.db, o.projectID, sesion)
	if err != nil {
		t.Fatal(err)
	}
	var guardadas []string
	for _, ev := range eventos {
		if ev.Tipo == store.ChatEventoRespuesta {
			guardadas = append(guardadas, ev.Content)
		}
	}
	if strings.Join(guardadas, "|") != "respuesta-1" {
		t.Fatalf("respuestas guardadas en chat_evento = %v", guardadas)
	}
}

// TestLaUltimaEtapaSePintaCuandoNoHayAvanceEnVivo: el flujo que lanza el agente con
// `ejecutar_flujo` no transmite token a token, así que también se pinta la
// respuesta de su última etapa, que si no quedaría invisible.
func TestLaUltimaEtapaSePintaCuandoNoHayAvanceEnVivo(t *testing.T) {
	o, sesion := orquestadorConServidor(t, servidorEtapas(t), catalogoConFlujo(flujoVisible()))
	canal, cerrar := o.Suscribir()
	defer cerrar()

	if _, err := o.ejecutarFlujo(context.Background(),
		tools.PeticionEjecutarFlujo{Flujo: "/visible", Peticion: "algo"},
		tools.Contexto{SesionID: sesion, Agente: "plan"}); err != nil {
		t.Fatalf("ejecutar el flujo: %v", err)
	}

	var respuestas []string
	for _, e := range drenar(canal) {
		if e.Tipo == tui.EventoEtapaRespuesta {
			respuestas = append(respuestas, e.Datos["texto"])
		}
	}
	if strings.Join(respuestas, "|") != "respuesta-1|respuesta-2" {
		t.Fatalf("respuestas pintadas en el hilo = %v, quiero las dos etapas", respuestas)
	}
}

// flujoProtegido deja su primera etapa con `aprobacion`: la pausa se pide antes
// de correrla (SPEC-ORQUESTADOR-FLUJOS §Flujo principal).
func flujoProtegido() flow.Flujo {
	return flow.Flujo{
		Comando: "/protegido",
		Etapas: []flow.Etapa{
			{ID: "uno", Nombre: "Uno", Orden: "etapa protegida", Respuesta: "aporte de la etapa uno", Aprobacion: true},
			{ID: "fin", Nombre: "Fin", Orden: "última etapa", Respuesta: "cierre del flujo"},
		},
	}
}

// servidorContado es como servidorEtapas, pero avisa por `llamado` cada turno
// de modelo que recibe —sin contar las peticiones de título— para poder comprobar
// si la etapa llegó a pedir turno.
func servidorContado(t *testing.T, llamado chan<- string) *httptest.Server {
	t.Helper()
	var n int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cuerpo, _ := io.ReadAll(r.Body)
		if bytes.Contains(cuerpo, []byte(marcaTitulo)) {
			escribirSSE(w, "Titulo")
			return
		}
		respuesta := fmt.Sprintf("respuesta-%d", atomic.AddInt32(&n, 1))
		llamado <- respuesta
		escribirSSE(w, respuesta)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// esperarAprobacionSinEtapas espera la petición de aprobación y devuelve su id,
// fallando si antes se anunció el inicio de alguna etapa.
func esperarAprobacionSinEtapas(t *testing.T, canal <-chan tui.Evento) string {
	t.Helper()
	for {
		select {
		case e := <-canal:
			if e.Tipo == tui.EventoEtapaIniciada {
				t.Fatalf("la etapa %q se inició antes de pedir la aprobación", e.Datos["nombre"])
			}
			if e.Tipo == tui.EventoPeticionAprobacion {
				return e.Datos["aprobacion"]
			}
		case <-time.After(5 * time.Second):
			t.Fatal("no llegó la petición de aprobación")
			return ""
		}
	}
}

// TestLaAprobacionDeUnaEtapaSePideAntesDeCorrerla: `aprobacion` pausa el flujo
// antes de ejecutar la etapa —la petición llega al panel sin que el modelo haya
// recibido ninguna llamada— y, al aprobar, la etapa corre y alimenta el bloque.
func TestLaAprobacionDeUnaEtapaSePideAntesDeCorrerla(t *testing.T) {
	llamado := make(chan string, 8)
	o, sesion := orquestadorConServidor(t, servidorContado(t, llamado), catalogoConFlujo(flujoProtegido()))
	canal, cerrar := o.Suscribir()
	defer cerrar()

	resultado := make(chan error, 1)
	go func() { resultado <- o.Enviar(context.Background(), sesion, "plan", "/flow:protegido algo") }()

	id := esperarAprobacionSinEtapas(t, canal)
	select {
	case llamada := <-llamado:
		t.Fatalf("la etapa pidió turno (%q) antes de la aprobación", llamada)
	default:
	}

	if err := o.ResponderAprobacion(sesion, id, true); err != nil {
		t.Fatalf("responder: %v", err)
	}
	if err := <-resultado; err != nil {
		t.Fatalf("el flujo no debería fallar tras aprobar: %v", err)
	}

	bloque, err := store.ListarFlowContext(o.db, o.projectID, sesion, "/protegido")
	if err != nil || len(bloque) != 2 {
		t.Fatalf("aprobada, las dos etapas deberían haber corrido: %+v (%v)", bloque, err)
	}
}

// TestDeclinarLaAprobacionCancelaElFlujoSinCorrerLaEtapa: declinada la pausa de
// una etapa, el flujo se cancela sin pedir turno al modelo y sin dejar ni aporte
// ni respuesta del turno.
func TestDeclinarLaAprobacionCancelaElFlujoSinCorrerLaEtapa(t *testing.T) {
	llamado := make(chan string, 8)
	o, sesion := orquestadorConServidor(t, servidorContado(t, llamado), catalogoConFlujo(flujoProtegido()))
	canal, cerrar := o.Suscribir()
	defer cerrar()

	resultado := make(chan error, 1)
	go func() { resultado <- o.Enviar(context.Background(), sesion, "plan", "/flow:protegido algo") }()

	id := esperarAprobacionSinEtapas(t, canal)
	if err := o.ResponderAprobacion(sesion, id, false); err != nil {
		t.Fatalf("responder: %v", err)
	}
	if err := <-resultado; err != nil {
		t.Fatalf("declinar cancela el flujo sin error: %v", err)
	}

	select {
	case llamada := <-llamado:
		t.Fatalf("una etapa declinada no debería haber pedido turno: %q", llamada)
	default:
	}
	bloque, _ := store.ListarFlowContext(o.db, o.projectID, sesion, "/protegido")
	if len(bloque) != 0 {
		t.Fatalf("una etapa declinada no debería dejar aporte: %+v", bloque)
	}
	mensajes, _ := store.ListarMensajes(o.db, o.projectID, sesion)
	if len(mensajes) != 1 {
		t.Fatalf("solo debería quedar el mensaje `/flow:` del usuario: %+v", mensajes)
	}
	var cancelado bool
	for _, e := range drenar(canal) {
		if e.Tipo == tui.EventoFlujoCancelado {
			cancelado = true
		}
	}
	if !cancelado {
		t.Fatal("el flujo debería avisar de que se canceló")
	}
}
