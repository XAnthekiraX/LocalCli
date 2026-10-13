package session

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"localcli/internal/agent"
	"localcli/internal/llm"
	"localcli/internal/tui"
)

// pngMinimo es la firma PNG, suficiente para que `DetectContentType` la reconozca.
func pngMinimo() []byte {
	return []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a, 0, 0, 0, 0}
}

func TestRutasImagenDetectaYNoDuplica(t *testing.T) {
	dir := t.TempDir()
	img := filepath.Join(dir, "foto.png")
	if err := os.WriteFile(img, pngMinimo(), 0o600); err != nil {
		t.Fatal(err)
	}
	txt := filepath.Join(dir, "notas.txt")
	if err := os.WriteFile(txt, []byte("hola"), 0o600); err != nil {
		t.Fatal(err)
	}

	got := rutasImagen("mira " + img + " y otra vez " + img + " y también " + txt)
	if len(got) != 1 || got[0] != img {
		t.Fatalf("rutasImagen = %v, quiero solo %q", got, img)
	}
}

func TestRutasImagenConEspacios(t *testing.T) {
	dir := t.TempDir()
	// Crear archivo con espacios en el nombre
	img := filepath.Join(dir, "captura de pantalla.png")
	if err := os.WriteFile(img, pngMinimo(), 0o600); err != nil {
		t.Fatal(err)
	}

	got := rutasImagen("describe esta imagen " + img)
	if len(got) != 1 || got[0] != img {
		t.Fatalf("rutasImagen con espacios = %v, quiero %q", got, img)
	}
}

func TestRutasImagenConRutaAbsolutaEspacios(t *testing.T) {
	dir := t.TempDir()
	// Crear subdirectorio con espacios
	subdir := filepath.Join(dir, "Capturas de pantalla")
	if err := os.Mkdir(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	img := filepath.Join(subdir, "foto 1.png")
	if err := os.WriteFile(img, pngMinimo(), 0o600); err != nil {
		t.Fatal(err)
	}

	got := rutasImagen("mira " + img)
	if len(got) != 1 || got[0] != img {
		t.Fatalf("rutasImagen ruta absoluta con espacios = %v, quiero %q", got, img)
	}
}

func TestMimeDeImagenRechazaTexto(t *testing.T) {
	if _, ok := mimeDeImagen([]byte("esto no es una imagen")); ok {
		t.Fatal("el texto no debería pasar por imagen")
	}
	if mime, ok := mimeDeImagen(pngMinimo()); !ok || mime != "image/png" {
		t.Fatalf("png: mime=%q ok=%v", mime, ok)
	}
}

func TestAdjuntarAlUltimoUsuario(t *testing.T) {
	hist := []agent.Mensaje{
		{Rol: agent.RolSistema, Contenido: "sys"},
		{Rol: agent.RolUsuario, Contenido: "uno"},
		{Rol: agent.RolAsistente, Contenido: "dos"},
		{Rol: agent.RolUsuario, Contenido: "tres"},
	}
	imgs := []agent.Imagen{{Ruta: "x.png", MIME: "image/png"}}
	got := adjuntarAlUltimoUsuario(hist, imgs, "tres")

	if len(got[3].Imagenes) != 1 {
		t.Fatalf("las imágenes deberían ir al último usuario: %+v", got[3])
	}
	if len(got[1].Imagenes) != 0 {
		t.Fatalf("el usuario anterior no debería llevar imágenes: %+v", got[1])
	}
}

func TestAdjuntarAlUltimoUsuarioQuitaRutasImagen(t *testing.T) {
	dir := t.TempDir()
	img := filepath.Join(dir, "foto.png")
	if err := os.WriteFile(img, pngMinimo(), 0o600); err != nil {
		t.Fatal(err)
	}

	hist := []agent.Mensaje{
		{Rol: agent.RolUsuario, Contenido: "describe esta imagen " + img},
	}
	imgs := []agent.Imagen{{Ruta: img, MIME: "image/png"}}
	got := adjuntarAlUltimoUsuario(hist, imgs, "describe esta imagen "+img)

	// El contenido debería tener la ruta quitada
	contenido := got[0].Contenido
	if contenido == "describe esta imagen "+img {
		t.Fatalf("la ruta de imagen debería haberse quitado del contenido, pero sigue: %q", contenido)
	}
	esperado := "describe esta imagen"
	if contenido != esperado {
		t.Fatalf("contenido esperado %q, got %q", esperado, contenido)
	}
}

func orquestadorImagenes(cwd string, vision llm.Capacidad) *Orquestador {
	return &Orquestador{
		cwd:         cwd,
		eventos:     make(chan tui.Evento, 32),
		terminado:   make(chan struct{}),
		capacidades: llm.Capacidades{Vision: vision},
	}
}

func TestImagenesDelTurnoConVisionSoportada(t *testing.T) {
	dir := t.TempDir()
	img := filepath.Join(dir, "foto.png")
	if err := os.WriteFile(img, pngMinimo(), 0o600); err != nil {
		t.Fatal(err)
	}
	o := orquestadorImagenes(dir, llm.Soportada)

	got := o.imagenesDelTurno(context.Background(), "s1", "describe "+img)
	if len(got) != 1 || got[0].MIME != "image/png" || got[0].Ruta != img {
		t.Fatalf("imagenesDelTurno = %+v", got)
	}
}

func TestImagenesDelTurnoConVisionNoSoportadaNoAdjunta(t *testing.T) {
	dir := t.TempDir()
	img := filepath.Join(dir, "foto.png")
	if err := os.WriteFile(img, pngMinimo(), 0o600); err != nil {
		t.Fatal(err)
	}
	o := orquestadorImagenes(dir, llm.NoSoportada)

	if got := o.imagenesDelTurno(context.Background(), "s1", "describe "+img); len(got) != 0 {
		t.Fatalf("no debería adjuntar sin visión: %+v", got)
	}
	select {
	case e := <-o.eventos:
		if e.Tipo != tui.EventoNotificacion {
			t.Fatalf("esperaba una notificación, llegó %q", e.Tipo)
		}
	default:
		t.Fatal("debería avisar de que el modelo no declara visión")
	}
}
