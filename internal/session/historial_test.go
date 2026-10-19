package session

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"localcli/internal/llm/openai"
	"localcli/internal/store"
	"localcli/internal/tools"
	"localcli/internal/tui"
)

// sesionDePrueba devuelve un orquestador y el id de una sesión suya, con la base
// y la carpeta en directorios temporales.
func sesionDePrueba(t *testing.T, url string) (*Orquestador, string) {
	t.Helper()
	db, limpiar, err := store.CrearDBTemporal()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = limpiar() })
	cwd := t.TempDir()
	projectID, err := store.ResolucionProyecto(db, cwd)
	if err != nil {
		t.Fatal(err)
	}
	o := Nuevo(db, projectID, cwd, agentesDePrueba(t), openai.NuevoCliente(url), "m", 4096, nil, nil)
	s, err := o.Crear()
	if err != nil {
		t.Fatal(err)
	}
	return o, s.ID
}

// sembrarMensajes deja `n` turnos cerrados, alternando usuario y agente.
func sembrarMensajes(t *testing.T, o *Orquestador, sesionID string, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		rol := "user"
		if i%2 == 1 {
			rol = "agent"
		}
		if _, err := store.CrearMensaje(o.db, o.projectID, sesionID, rol, strings.Repeat("x", 40), nil, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
}

func TestHistorialCortoViajaEntero(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("LOCALCLI_CONTEXT_LIMIT", "100000")
	o, sesionID := sesionDePrueba(t, "http://127.0.0.1:1")
	sembrarMensajes(t, o, sesionID, 4)

	msgs, err := o.historialPara(context.Background(), sesionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 4 {
		t.Fatalf("esperaba 4 mensajes, hay %d", len(msgs))
	}
	for _, m := range msgs {
		if m.Rol != "user" && m.Rol != "assistant" {
			t.Fatalf("rol inesperado en el historial: %q", m.Rol)
		}
	}
}

func TestHistorialLargoSeResumeYSeReutiliza(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("LOCALCLI_CONTEXT_LIMIT", "40")
	var llamadas int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&llamadas, 1)
		escribirSSE(w, "RESUMEN")
	}))
	t.Cleanup(srv.Close)
	o, sesionID := sesionDePrueba(t, srv.URL)
	sembrarMensajes(t, o, sesionID, 10)

	ctx := context.Background()
	msgs, err := o.historialPara(ctx, sesionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) < 2 || len(msgs) >= 10 {
		t.Fatalf("el historial no se compactó: %d mensajes", len(msgs))
	}
	if msgs[0].Rol != "system" || !strings.Contains(msgs[0].Contenido, "RESUMEN") {
		t.Fatalf("el primer mensaje debería ser el resumen: %+v", msgs[0])
	}
	if n := atomic.LoadInt32(&llamadas); n != 1 {
		t.Fatalf("el resumen se pidió %d veces, quiero 1", n)
	}

	// El resumen se reutiliza mientras el tramo reciente no cambie.
	if _, err := o.historialPara(ctx, sesionID); err != nil {
		t.Fatal(err)
	}
	if n := atomic.LoadInt32(&llamadas); n != 1 {
		t.Fatalf("el resumen no se reutilizó: %d llamadas", n)
	}
}

// La compactación es incremental: cuando llegan mensajes nuevos que no caben, el
// resumen persistido se amplía una sola vez y se reutiliza mientras el tramo
// reciente siga cabiendo.
func TestCompactacionIncrementalSeAmpliaConMensajesNuevos(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("LOCALCLI_CONTEXT_LIMIT", "40")
	var llamadas int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&llamadas, 1)
		escribirSSE(w, "RESUMEN")
	}))
	t.Cleanup(srv.Close)
	o, sesionID := sesionDePrueba(t, srv.URL)
	sembrarMensajes(t, o, sesionID, 10)
	ctx := context.Background()

	if _, err := o.historialPara(ctx, sesionID); err != nil {
		t.Fatal(err)
	}
	if n := atomic.LoadInt32(&llamadas); n != 1 {
		t.Fatalf("primer resumen: %d llamadas, quiero 1", n)
	}

	// Llegan mensajes nuevos que ya no caben: el resumen se amplía.
	sembrarMensajes(t, o, sesionID, 4)
	msgs, err := o.historialPara(ctx, sesionID)
	if err != nil {
		t.Fatal(err)
	}
	if n := atomic.LoadInt32(&llamadas); n != 2 {
		t.Fatalf("ampliación: %d llamadas, quiero 2", n)
	}
	if msgs[0].Rol != "system" || !strings.Contains(msgs[0].Contenido, "RESUMEN") {
		t.Fatalf("el primer mensaje debería ser el resumen: %+v", msgs[0])
	}

	// Sin mensajes nuevos, se reutiliza: no hay tercera llamada.
	if _, err := o.historialPara(ctx, sesionID); err != nil {
		t.Fatal(err)
	}
	if n := atomic.LoadInt32(&llamadas); n != 2 {
		t.Fatalf("reutilización: %d llamadas, quiero 2", n)
	}
}

// `/compactar` resume y persiste aunque el historial quepa en el presupuesto; a
// partir de ahí el resumen manda y no se vuelve a llamar al modelo.
func TestCompactarManualPersisteYSeReutiliza(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	// Presupuesto holgado: la compactación automática no se dispararía.
	t.Setenv("LOCALCLI_CONTEXT_LIMIT", "100000")
	dir := filepath.Join(home, ".config", "localcli")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"),
		[]byte(`{"compactacion":{"conservar_tokens":40}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	var llamadas int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&llamadas, 1)
		escribirSSE(w, "RESUMEN")
	}))
	t.Cleanup(srv.Close)
	o, sesionID := sesionDePrueba(t, srv.URL)
	sembrarMensajes(t, o, sesionID, 10)
	ctx := context.Background()

	compactado, err := o.Compactar(ctx, sesionID)
	if err != nil {
		t.Fatal(err)
	}
	if !compactado {
		t.Fatal("Compactar debería haber persistido un resumen")
	}
	if _, ok, err := store.ObtenerResumenHistorial(o.db, o.projectID, sesionID); err != nil || !ok {
		t.Fatalf("el resumen no se persistió: ok=%v err=%v", ok, err)
	}

	msgs, err := o.historialPara(ctx, sesionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) == 0 || msgs[0].Rol != "system" || !strings.Contains(msgs[0].Contenido, "RESUMEN") {
		t.Fatalf("tras compactar, el historial debería empezar por el resumen: %+v", msgs)
	}
	if n := atomic.LoadInt32(&llamadas); n != 1 {
		t.Fatalf("el resumen se reutilizó mal: %d llamadas, quiero 1", n)
	}
}

// TestCompactarSinTramoAntiguoNoHaceNada: con todo el historial dentro del tramo
// reciente, `/compactar` no inventa un resumen —no llama al modelo, no persiste
// nada y lo reporta como `false`—. Es la regresión de la puerta que hacía un
// no-op silencioso y aun así informaba «contexto compactado».
func TestCompactarSinTramoAntiguoNoHaceNada(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	// Presupuesto holgado: la recencia (conservar por defecto, 15000) cubre todo.
	t.Setenv("LOCALCLI_CONTEXT_LIMIT", "100000")
	var llamadas int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&llamadas, 1)
		escribirSSE(w, "RESUMEN")
	}))
	t.Cleanup(srv.Close)
	o, sesionID := sesionDePrueba(t, srv.URL)
	sembrarMensajes(t, o, sesionID, 4)
	ctx := context.Background()

	compactado, err := o.Compactar(ctx, sesionID)
	if err != nil {
		t.Fatal(err)
	}
	if compactado {
		t.Fatal("sin tramo antiguo, Compactar no debería persistir un resumen")
	}
	if _, ok, err := store.ObtenerResumenHistorial(o.db, o.projectID, sesionID); err != nil || ok {
		t.Fatalf("no debería persistirse resumen: ok=%v err=%v", ok, err)
	}
	if n := atomic.LoadInt32(&llamadas); n != 0 {
		t.Fatalf("no debería llamarse al modelo: %d llamadas", n)
	}
}

// El historial entregado no supera el presupuesto aunque el resumen se sume al
// tramo reciente: la recencia reserva un hueco para él ([[specs/SPEC-HISTORIAL-CONVERSACION]]).
func TestLaEntregaNoSuperaElPresupuesto(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("LOCALCLI_CONTEXT_LIMIT", "3000")
	// `conservar_tokens` por defecto (15000) supera el presupuesto: sin la
	// reserva, el resumen se añadiría por encima del tramo reciente.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		escribirSSE(w, strings.Repeat("y", 8000))
	}))
	t.Cleanup(srv.Close)
	o, sesionID := sesionDePrueba(t, srv.URL)
	sembrarMensajes(t, o, sesionID, 300)

	msgs, err := o.historialPara(context.Background(), sesionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) == 0 || msgs[0].Rol != "system" {
		t.Fatalf("esperaba un resumen al frente del historial: %+v", msgs)
	}
	entregado := 0
	for _, m := range msgs {
		entregado += estimarTokens(m.Contenido) + 4
	}
	if entregado > 3000 {
		t.Fatalf("el historial entregado (%d tokens) supera el presupuesto (3000)", entregado)
	}
}

func TestFalloDelResumenNoBloquea(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("LOCALCLI_CONTEXT_LIMIT", "40")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "sin modelo", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	o, sesionID := sesionDePrueba(t, srv.URL)
	sembrarMensajes(t, o, sesionID, 10)

	msgs, err := o.historialPara(context.Background(), sesionID)
	if err != nil {
		t.Fatalf("un fallo del resumen no debe devolver error: %v", err)
	}
	if len(msgs) == 0 || len(msgs) >= 10 {
		t.Fatalf("debería entregarse solo el tramo reciente: %d mensajes", len(msgs))
	}
	for _, m := range msgs {
		if m.Rol == "system" {
			t.Fatalf("sin resumen no debería haber mensaje de sistema: %+v", m)
		}
	}
}

func TestPresupuestoHistorialCadena(t *testing.T) {
	t.Setenv("LOCALCLI_CONTEXT_LIMIT", "")

	if p := presupuestoHistorial(0, tui.Preferencias{}); p != presupuestoHistorialDefecto {
		t.Fatalf("sin ventana = %d, quiero %d", p, presupuestoHistorialDefecto)
	}

	// Con ventana conocida y sin preferencia ni entorno: el umbral por defecto
	// (60 %) de la ventana.
	ventana := 32768
	esperado := int(float64(ventana) * tui.UmbralCompactacionPorDefecto)
	if p := presupuestoHistorial(ventana, tui.Preferencias{}); p != esperado {
		t.Fatalf("con ventana = %d, quiero %d", p, esperado)
	}

	t.Setenv("LOCALCLI_CONTEXT_LIMIT", "2000")
	if p := presupuestoHistorial(32768, tui.Preferencias{}); p != 2000 {
		t.Fatalf("con el tope de entorno = %d, quiero 2000", p)
	}

	if p := presupuestoHistorial(32768, tui.Preferencias{HistorialTokens: 1500}); p != 1500 {
		t.Fatalf("con la preferencia = %d, quiero 1500", p)
	}
}

// TestHistorialReenviaHerramientasEjecutadas: una herramienta guardada viaja al
// modelo en el turno siguiente como el par assistant(tool_calls) + tool(resultado),
// intercalada entre el mensaje del usuario y la respuesta del agente. Las líneas
// antiguas sin `call_id` y los sub-procesos siguen siendo solo pantalla.
func TestHistorialReenviaHerramientasEjecutadas(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("LOCALCLI_CONTEXT_LIMIT", "100000")
	o, sesionID := sesionDePrueba(t, "http://127.0.0.1:1")

	if _, err := store.CrearMensaje(o.db, o.projectID, sesionID, "user", "revisa el proyecto", nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	// Línea antigua sin ejecución: no debe viajar.
	if _, err := store.CrearChatEvento(o.db, o.projectID, sesionID, store.ChatEventoHerramienta, "✓ LEER [viejo.go] · 3 líneas", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CrearChatEventoTool(o.db, o.projectID, sesionID, "✓ LEER [ir.go] · 2 líneas", nil, store.EjecucionHerramienta{
		CallID: "call-1", Herramienta: "leer_archivo", Argumentos: `{"ruta":"ir.go"}`,
		Salida: "package diagram\n", OK: true,
	}); err != nil {
		t.Fatal(err)
	}
	// Un fallo viaja compacto para que el modelo no repita el mismo tropiezo.
	if _, err := store.CrearChatEventoTool(o.db, o.projectID, sesionID, "✗ EJEC [go test] · falló", nil, store.EjecucionHerramienta{
		CallID: "call-2", Herramienta: "ejecutar_comando", Argumentos: `{"comando":"go test"}`,
		Error: "el comando falló", OK: false,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CrearMensaje(o.db, o.projectID, sesionID, "agent", "ya está revisado", nil, nil, nil); err != nil {
		t.Fatal(err)
	}

	msgs, err := o.historialPara(context.Background(), sesionID)
	if err != nil {
		t.Fatal(err)
	}
	// usuario + (assistant+tool) + (assistant+tool) + agente
	if len(msgs) != 6 {
		t.Fatalf("mensajes = %d, quiero 6: %+v", len(msgs), msgs)
	}
	if msgs[0].Rol != "user" || msgs[5].Rol != "assistant" {
		t.Fatalf("el hilo debería abrir con el usuario y cerrar con el agente: %+v", msgs)
	}
	if msgs[1].Rol != "assistant" || len(msgs[1].Llamadas) != 1 || msgs[1].Llamadas[0].ID != "call-1" {
		t.Fatalf("falta la llamada de la herramienta: %+v", msgs[1])
	}
	if msgs[2].Rol != "tool" || msgs[2].IDLlamada != "call-1" || msgs[2].Contenido != "package diagram\n" {
		t.Fatalf("falta el resultado de la herramienta: %+v", msgs[2])
	}
	if msgs[3].Rol != "assistant" || msgs[3].Llamadas[0].ID != "call-2" {
		t.Fatalf("falta la llamada del comando fallido: %+v", msgs[3])
	}
	if msgs[4].Rol != "tool" || msgs[4].Contenido != "el comando falló" {
		t.Fatalf("el fallo debería viajar compacto: %+v", msgs[4])
	}
	for _, m := range msgs {
		if strings.Contains(m.Contenido, "viejo.go") {
			t.Fatalf("una línea sin ejecución no debería viajar al modelo: %+v", m)
		}
	}
}

// TestHistorialRecortaSalidasLargas: el historial no reenvía una lectura entera;
// la salida se acota y se marca como recortada.
func TestHistorialRecortaSalidasLargas(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("LOCALCLI_CONTEXT_LIMIT", "100000")
	o, sesionID := sesionDePrueba(t, "http://127.0.0.1:1")

	if _, err := store.CrearMensaje(o.db, o.projectID, sesionID, "user", "lee el archivo", nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CrearChatEventoTool(o.db, o.projectID, sesionID, "✓ LEER [grande.go]", nil, store.EjecucionHerramienta{
		CallID: "call-grande", Herramienta: "leer_archivo", Argumentos: `{"ruta":"grande.go"}`,
		Salida: strings.Repeat("z", maxSalidaHistorial*2), OK: true,
	}); err != nil {
		t.Fatal(err)
	}

	msgs, err := o.historialPara(context.Background(), sesionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 3 {
		t.Fatalf("mensajes = %d, quiero 3: %+v", len(msgs), msgs)
	}
	salida := msgs[2].Contenido
	if !strings.HasSuffix(salida, marcaSalidaHistorial) {
		t.Fatalf("la salida recortada debería llevar su marca: %q", salida[len(salida)-40:])
	}
	if len([]rune(salida)) != maxSalidaHistorial+len([]rune(marcaSalidaHistorial)) {
		t.Fatalf("la salida recortada mide %d runas", len([]rune(salida)))
	}
}

// TestHerramientaDeFlujoNoSeReenvia: la línea de una herramienta que corre
// dentro de una etapa se pinta, pero no se persiste como ejecución reenviable:
// el resultado del `ejecutar_flujo` que la lanzó ya la resume.
func TestHerramientaDeFlujoNoSeReenvia(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	o, sesionID := sesionDePrueba(t, "http://127.0.0.1:1")

	o.guardarLineaHerramienta(tools.EventoResultado{
		Sesion: sesionID, IDLlamada: "call-flujo", EnFlujo: true,
		Nombre: "leer_archivo", Verbo: "LEER", Tema: "interno.go", OK: true,
		Salida: "contenido interno", Duracion: time.Millisecond,
	})

	ejecuciones, err := store.ListarEjecucionesHerramienta(o.db, o.projectID, sesionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(ejecuciones) != 0 {
		t.Fatalf("una herramienta de flujo no debería reenviarse: %+v", ejecuciones)
	}
	eventos, err := store.ListarChatEventos(o.db, o.projectID, sesionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(eventos) != 1 || eventos[0].Ejecucion != nil {
		t.Fatalf("la línea del flujo debería quedar solo para pintar: %+v", eventos)
	}
}

// TestEjecucionesDesdeDescartaLoResumido: una ejecución anterior a la marca del
// resumen no se reenvía; una posterior sí.
func TestEjecucionesDesdeDescartaLoResumido(t *testing.T) {
	conv := []mensajeConv{
		{id: "m1", createdAt: "2026-01-01T10:00:00.000000100Z"},
		{id: "m2", createdAt: "2026-01-01T10:00:01.000000000Z"},
	}
	ejecs := []ejecucionConv{
		{callID: "vieja", createdAt: "2026-01-01T10:00:00.500000000Z"},
		{callID: "nueva", createdAt: "2026-01-01T10:00:01.500000000Z"},
	}
	// El corte deja m1 y m2 en el tramo resumido: la ejecución anterior a m2 se
	// descarta y la posterior se conserva.
	quedan := ejecucionesDesde(conv, 2, ejecs)
	if len(quedan) != 1 || quedan[0].callID != "nueva" {
		t.Fatalf("ejecuciones del tramo reciente = %+v, quiero solo la nueva", quedan)
	}
	if todas := ejecucionesDesde(conv, 0, ejecs); len(todas) != 2 {
		t.Fatalf("sin corte deberían quedar todas: %+v", todas)
	}
}
