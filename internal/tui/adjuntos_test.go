package tui

// Tests de la detección de imágenes en el texto del turno (adjuntos.go). Una
// prueba, una regla: se adjunta lo que existe y parece imagen; lo que no, se
// ignora o se avisa, pero nunca corta el envío.

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestAdjuntosDeLeeImagenesYLasCodifica(t *testing.T) {
	dir := t.TempDir()
	img := filepath.Join(dir, "foto.png")
	contenido := []byte{0x89, 'P', 'N', 'G', 1, 2, 3}
	if err := os.WriteFile(img, contenido, 0o600); err != nil {
		t.Fatal(err)
	}

	imgs, avisos := AdjuntosDe("mira " + img + " y dime qué ves")
	if len(avisos) != 0 {
		t.Fatalf("sin avisos, quiero 0: %v", avisos)
	}
	if len(imgs) != 1 {
		t.Fatalf("imágenes = %d, quiero 1", len(imgs))
	}
	if imgs[0] != base64.StdEncoding.EncodeToString(contenido) {
		t.Errorf("base64 = %q", imgs[0])
	}
}

// Una ruta de imagen que no existe se ignora en silencio (evita falsos
// positivos en prosa), y una extensión que no es de imagen no se mira.
func TestAdjuntosDeIgnoraRutasInexistentesYNoImagenes(t *testing.T) {
	imgs, avisos := AdjuntosDe("renombra inexistente-xyz.png y revisa notas.txt")
	if len(imgs) != 0 || len(avisos) != 0 {
		t.Fatalf("nada que adjuntar: imgs=%v avisos=%v", imgs, avisos)
	}
}

// Un fallo de lectura distinto de «no existe» (aquí, una carpeta con extensión
// de imagen) se avisa sin adjuntar y sin cortar el envío.
func TestAdjuntosDeAvisaSiNoPuedeLeer(t *testing.T) {
	dir := t.TempDir()
	carpeta := filepath.Join(dir, "carpeta.png")
	if err := os.Mkdir(carpeta, 0o700); err != nil {
		t.Fatal(err)
	}

	imgs, avisos := AdjuntosDe("abre " + carpeta)
	if len(imgs) != 0 {
		t.Fatalf("una carpeta no es una imagen: %v", imgs)
	}
	if len(avisos) != 1 {
		t.Fatalf("avisos = %d, quiero 1: %v", len(avisos), avisos)
	}
}

// Al enviar, la ruta de imagen del texto viaja al puerto codificada en base64.
func TestEnviarAdjuntaLasImagenesDelTexto(t *testing.T) {
	dir := t.TempDir()
	img := filepath.Join(dir, "foto.png")
	contenido := []byte("png")
	if err := os.WriteFile(img, contenido, 0o600); err != nil {
		t.Fatal(err)
	}

	a, p := mainConSesion(t)
	teclear(t, a, "describe "+img)
	ejecuta(t, a, pulsa(t, a, tea.KeyMsg{Type: tea.KeyEnter}))

	if len(p.imagenesEnviadas) != 1 || len(p.imagenesEnviadas[0]) != 1 {
		t.Fatalf("imágenes enviadas = %+v", p.imagenesEnviadas)
	}
	if p.imagenesEnviadas[0][0] != base64.StdEncoding.EncodeToString(contenido) {
		t.Errorf("base64 = %q", p.imagenesEnviadas[0][0])
	}
}

// Adjuntar una imagen a un modelo que no declara visión avisa en el chat, sin
// bloquear el envío.
func TestAvisoDeVisionAlEnviarImagenSinVision(t *testing.T) {
	dir := t.TempDir()
	img := filepath.Join(dir, "foto.png")
	if err := os.WriteFile(img, []byte("png"), 0o600); err != nil {
		t.Fatal(err)
	}

	a, p := mainConSesion(t)
	a.Modelo = "llama3.2"
	a.CapVisionConocida = true
	a.ModeloVision = false
	teclear(t, a, "mira "+img)
	ejecuta(t, a, pulsa(t, a, tea.KeyMsg{Type: tea.KeyEnter}))

	if len(p.enviados) != 1 {
		t.Fatalf("el envío no se bloquea: enviados=%v", p.enviados)
	}
	if !chatContieneTexto(a, "no declara visión") {
		t.Error("debe avisar de que el modelo no interpreta imágenes")
	}
}

// Un modelo que sí declara visión no genera el aviso.
func TestSinAvisoCuandoElModeloVe(t *testing.T) {
	dir := t.TempDir()
	img := filepath.Join(dir, "foto.png")
	if err := os.WriteFile(img, []byte("png"), 0o600); err != nil {
		t.Fatal(err)
	}

	a, _ := mainConSesion(t)
	a.Modelo = "llava"
	a.CapVisionConocida = true
	a.ModeloVision = true
	teclear(t, a, "mira "+img)
	ejecuta(t, a, pulsa(t, a, tea.KeyMsg{Type: tea.KeyEnter}))

	if chatContieneTexto(a, "no declara visión") {
		t.Error("con visión no debe avisar")
	}
}

func chatContieneTexto(a *App, sub string) bool {
	for _, m := range a.Chat.Mensajes() {
		if strings.Contains(m.Texto, sub) {
			return true
		}
	}
	return false
}

// --- token [archivo.ext] al pegar o arrastrar -------------------------------

func TestAdjuntosAnotaYExpandeVariasRutas(t *testing.T) {
	dir := t.TempDir()
	uno := filepath.Join(dir, "uno.png")
	dos := filepath.Join(dir, "dos.png")
	if err := os.WriteFile(uno, []byte("1"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dos, []byte("2"), 0o600); err != nil {
		t.Fatal(err)
	}

	var ad adjuntos
	pegado := uno + " y " + dos
	nota := ad.Anotar(pegado)
	if nota != "[uno.png] y [dos.png]" {
		t.Fatalf("anotado = %q", nota)
	}
	if got := ad.Expandir(nota); got != pegado {
		t.Fatalf("expandido = %q, quiero %q", got, pegado)
	}
}

// Dos archivos con el mismo nombre no se confunden: el token apunta al primero
// y el segundo conserva su ruta.
func TestAdjuntosNoMezclaDosArchivosConElMismoNombre(t *testing.T) {
	d1, d2 := t.TempDir(), t.TempDir()
	p1 := filepath.Join(d1, "foto.png")
	p2 := filepath.Join(d2, "foto.png")
	if err := os.WriteFile(p1, []byte("1"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p2, []byte("2"), 0o600); err != nil {
		t.Fatal(err)
	}

	var ad adjuntos
	nota := ad.Anotar(p1 + " " + p2)
	if !strings.Contains(nota, "[foto.png]") || !strings.Contains(nota, p2) {
		t.Fatalf("nota = %q", nota)
	}
	if ad.rutas["[foto.png]"] != p1 {
		t.Errorf("el token debe apuntar al primero: %q", ad.rutas["[foto.png]"])
	}
}

// Pegar o arrastrar una imagen en la interfaz principal muestra el token, no la
// ruta, y al enviar viaja la ruta real con la imagen adjunta.
func TestPegarUnaImagenMuestraElTokenYEnviaLaRuta(t *testing.T) {
	dir := t.TempDir()
	img := filepath.Join(dir, "foto.png")
	contenido := []byte("png")
	if err := os.WriteFile(img, contenido, 0o600); err != nil {
		t.Fatal(err)
	}

	a, p := mainConSesion(t)
	pulsa(t, a, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(img), Paste: true})

	v := sinEstilo(a.View())
	if !strings.Contains(v, "[foto.png]") {
		t.Fatalf("la entrada debe mostrar el token de la imagen:\n%s", v)
	}
	if strings.Contains(v, img) {
		t.Errorf("la ruta cruda no debe verse en la entrada:\n%s", v)
	}
	if got := a.Entrada.Texto(); got != img {
		t.Fatalf("Texto() expande a la ruta real: %q", got)
	}

	ejecuta(t, a, pulsa(t, a, tea.KeyMsg{Type: tea.KeyEnter}))
	if len(p.imagenesEnviadas) != 1 || len(p.imagenesEnviadas[0]) != 1 {
		t.Fatalf("imágenes enviadas = %+v", p.imagenesEnviadas)
	}
	if p.imagenesEnviadas[0][0] != base64.StdEncoding.EncodeToString(contenido) {
		t.Errorf("base64 = %q", p.imagenesEnviadas[0][0])
	}
}

// La bienvenida también muestra el token al pegar y expande al enviar.
func TestPegarUnaImagenEnLaBienvenidaMuestraElToken(t *testing.T) {
	dir := t.TempDir()
	img := filepath.Join(dir, "gato.jpg")
	if err := os.WriteFile(img, []byte("jpg"), 0o600); err != nil {
		t.Fatal(err)
	}

	a := Nuevo(&puertoStub{})
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	pulsa(t, a, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(img), Paste: true})

	if a.Bienvenida.Texto != "[gato.jpg]" {
		t.Fatalf("la línea guarda el token: %q", a.Bienvenida.Texto)
	}
	if got := a.Bienvenida.TextoExpandido(); got != img {
		t.Fatalf("TextoExpandido devuelve la ruta: %q", got)
	}
	if !strings.Contains(sinEstilo(a.View()), "[gato.jpg]") {
		t.Errorf("la bienvenida muestra el token:\n%s", sinEstilo(a.View()))
	}
}
