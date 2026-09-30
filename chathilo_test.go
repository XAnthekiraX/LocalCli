// chathilo_test.go — el hilo del chat: las líneas de procesamiento que el
// arranque guarda en `chat_evento` (sub-procesos de un flujo y líneas de
// herramienta) para que sobrevivan al cambio de sesión.
package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"localcli/internal/session"
	"localcli/internal/store"
	"localcli/internal/tools"
)

// adaptadorConSesion deja un Adaptador con base real y el id de una sesión, que
// es lo que necesitan el publicador y el registro para atribuir la línea. La
// sesión ya no vive en el adaptador: viaja en cada llamada (y en el contexto,
// para el registro), que es lo que la ata a su turno.
func adaptadorConSesion(t *testing.T) (*Adaptador, string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "ai", "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ai", "docs", "PROJECT.md"), []byte("# prueba\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	conexion, err := store.Open(dir)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = conexion.Close() })
	ses, err := store.CrearSesion(conexion, "chat", "")
	if err != nil {
		t.Fatalf("CrearSesion: %v", err)
	}
	ad := &Adaptador{bus: session.NuevoBus(), chat: store.NuevoChat(conexion)}
	return ad, ses.ID
}

// La línea de herramienta se guarda cerrada (con su marca, su medida y su
// tiempo), que es justo lo que la vista pinta: así el hilo la recuerda al
// volver a la sesión.
func TestElPublicadorGuardaLaLineaDeHerramientaEnElHilo(t *testing.T) {
	ad, sesion := adaptadorConSesion(t)
	pub := &publicadorBus{bus: ad.bus, ad: ad}
	pub.HerramientaInvocada(sesion, "leer_archivo", "plan", "LEER", "AGENTS.md")
	pub.HerramientaResultado(sesion, "leer_archivo", true, "", false, "93 líneas", 400*time.Millisecond)

	hilo, err := ad.chat.Hilo(sesion)
	if err != nil {
		t.Fatal(err)
	}
	if len(hilo) != 1 {
		t.Fatalf("una línea de herramienta en el hilo: %+v", hilo)
	}
	if hilo[0].Rol != "proceso" {
		t.Errorf("la línea se pinta como procesamiento: %q", hilo[0].Rol)
	}
	if !strings.Contains(hilo[0].Content, "LEER [AGENTS.md] · 93 líneas") {
		t.Errorf("la línea guardada lleva el verbo, el tema y la medida: %q", hilo[0].Content)
	}
	// El tiempo se pinta dentro de la línea (atenuado: los códigos de estilo van
	// alrededor del texto, que queda contiguo) y además se guarda como número.
	if !strings.Contains(hilo[0].Content, "400 ms") {
		t.Errorf("la línea guardada lleva su duración: %q", hilo[0].Content)
	}
	if hilo[0].DuracionMS != 400 {
		t.Errorf("la duración se guarda como columna: %d", hilo[0].DuracionMS)
	}
}

// El sub-proceso de una etapa se guarda con la misma forma que la TUI pinta.
func TestElRegistroGuardaElSubProcesoEnElHilo(t *testing.T) {
	ad, sesion := adaptadorConSesion(t)
	reg := &registroPorTurno{ad: ad}
	reg.ProcesoEtapa(tools.ConSesion(context.Background(), sesion), "Entender el problema", false)

	hilo, err := ad.chat.Hilo(sesion)
	if err != nil {
		t.Fatal(err)
	}
	if len(hilo) != 1 || hilo[0].Content != "[Sub Proceso] Entender el problema" {
		t.Fatalf("el sub-proceso debe quedar en el hilo: %+v", hilo)
	}
}
