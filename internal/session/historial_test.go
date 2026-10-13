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

	"localcli/internal/llm/openai"
	"localcli/internal/store"
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
	home := t.TempDir()
	t.Setenv("HOME", home)

	t.Setenv("LOCALCLI_CONTEXT_LIMIT", "")
	if p := presupuestoHistorial(0); p != presupuestoHistorialDefecto {
		t.Fatalf("sin ventana = %d, quiero %d", p, presupuestoHistorialDefecto)
	}

	// Con ventana conocida y sin preferencia ni entorno: la mitad de la ventana.
	if p := presupuestoHistorial(32768); p != 16384 {
		t.Fatalf("con ventana = %d, quiero 16384", p)
	}

	t.Setenv("LOCALCLI_CONTEXT_LIMIT", "2000")
	if p := presupuestoHistorial(32768); p != 2000 {
		t.Fatalf("con el tope de entorno = %d, quiero 2000", p)
	}

	dir := filepath.Join(home, ".config", "localcli")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{"historial_tokens": 1500}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if p := presupuestoHistorial(32768); p != 1500 {
		t.Fatalf("con la preferencia = %d, quiero 1500", p)
	}
}
