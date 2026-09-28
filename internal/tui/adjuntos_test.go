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

// --- T-F038: tokens de cualquier archivo y de pegado multilínea -------------

// Un archivo que no es imagen también se tokeniza y se expande igual.
func TestAdjuntosAnotaUnArchivoNoImagen(t *testing.T) {
	dir := t.TempDir()
	archivo := filepath.Join(dir, "codigo.go")
	if err := os.WriteFile(archivo, []byte("package x"), 0o600); err != nil {
		t.Fatal(err)
	}

	var ad adjuntos
	nota := ad.Anotar(archivo)
	if nota != "[codigo.go]" {
		t.Fatalf("anotado = %q, quiero [codigo.go]", nota)
	}
	if got := ad.Expandir(nota); got != archivo {
		t.Fatalf("expandido = %q, quiero %q", got, archivo)
	}
}

// Una ruta que no existe no se tokeniza.
func TestAdjuntosDejaLaRutaInexistenteTalCual(t *testing.T) {
	var ad adjuntos
	ruta := filepath.Join(t.TempDir(), "no-existe.go")
	if got := ad.Anotar(ruta); got != ruta {
		t.Fatalf("una ruta inexistente no se tokeniza: %q", got)
	}
}

// Un pegado de varias líneas se resume en [PEGADO N líneas] y expande al texto entero.
func TestAdjuntosPegadoMultilineaMuestraElToken(t *testing.T) {
	var ad adjuntos
	texto := "primera línea\nsegunda línea\ntercera línea"
	nota := ad.Anotar(texto)
	if nota != "[PEGADO 3 líneas]" {
		t.Fatalf("anotado = %q, quiero [PEGADO 3 líneas]", nota)
	}
	if got := ad.Expandir(nota); got != texto {
		t.Fatalf("expandido = %q, quiero el texto entero", got)
	}
}

// Un salto de línea final no convierte un pegado de una línea en pegado de dos.
func TestAdjuntosUnSaltoFinalNoEsPegado(t *testing.T) {
	dir := t.TempDir()
	archivo := filepath.Join(dir, "notas.txt")
	if err := os.WriteFile(archivo, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	var ad adjuntos
	if nota := ad.Anotar(archivo + "\n"); nota != "[notas.txt]" {
		t.Fatalf("un salto final no es multilínea: %q", nota)
	}
}

// Arrastrar varios archivos deja un token por archivo, no un resumen.
func TestAdjuntosVariosArchivosUnTokenCadaUno(t *testing.T) {
	dir := t.TempDir()
	uno := filepath.Join(dir, "uno.go")
	dos := filepath.Join(dir, "dos.go")
	for _, p := range []string{uno, dos} {
		if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	var ad adjuntos
	nota := ad.Anotar(uno + "\n" + dos)
	if nota != "[uno.go] [dos.go]" {
		t.Fatalf("anotado = %q, quiero [uno.go] [dos.go]", nota)
	}
	if got := ad.Expandir(nota); got != uno+" "+dos {
		t.Fatalf("expandido = %q", got)
	}
}

// Dos pegados distintos del mismo número de líneas no comparten token.
func TestAdjuntosDosPegadosIgualesNoSePisan(t *testing.T) {
	var ad adjuntos
	a1 := ad.Anotar("uno\ndos")
	a2 := ad.Anotar("tres\ncuatro")
	if a1 == a2 {
		t.Fatalf("dos pegados distintos no comparten token: %q", a1)
	}
	if ad.Expandir(a1) != "uno\ndos" || ad.Expandir(a2) != "tres\ncuatro" {
		t.Errorf("cada token expande a su texto: %q / %q", ad.Expandir(a1), ad.Expandir(a2))
	}
}

// Pegar un archivo que no es imagen muestra su token y al enviar viaja la ruta
// real, sin adjuntar nada en base64.
func TestPegarUnArchivoMuestraElTokenYEnviaLaRuta(t *testing.T) {
	dir := t.TempDir()
	archivo := filepath.Join(dir, "main.go")
	if err := os.WriteFile(archivo, []byte("package main"), 0o600); err != nil {
		t.Fatal(err)
	}

	a, p := mainConSesion(t)
	pulsa(t, a, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(archivo), Paste: true})

	if v := sinEstilo(a.View()); !strings.Contains(v, "[main.go]") {
		t.Fatalf("la entrada debe mostrar el token:\n%s", v)
	}
	if got := a.Entrada.Texto(); got != archivo {
		t.Fatalf("Texto() expande a la ruta real: %q", got)
	}

	ejecuta(t, a, pulsa(t, a, tea.KeyMsg{Type: tea.KeyEnter}))
	if len(p.enviados) != 1 || p.enviados[0] != "s1|"+archivo {
		t.Fatalf("enviados = %v, quiero la ruta real", p.enviados)
	}
	if len(p.imagenesEnviadas) != 1 || len(p.imagenesEnviadas[0]) != 0 {
		t.Errorf("un archivo que no es imagen no se adjunta en base64: %+v", p.imagenesEnviadas)
	}
}

// Pegar texto de varias líneas se resume en la entrada y viaja entero al enviar.
func TestPegarTextoMultilineaMuestraElTokenYEnviaElTexto(t *testing.T) {
	a, p := mainConSesion(t)
	texto := "primera\nsegunda\ntercera"
	pulsa(t, a, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(texto), Paste: true})

	if !strings.Contains(sinEstilo(a.View()), "[PEGADO 3 líneas]") {
		t.Fatalf("la entrada debe resumir el pegado:\n%s", sinEstilo(a.View()))
	}
	ejecuta(t, a, pulsa(t, a, tea.KeyMsg{Type: tea.KeyEnter}))
	if len(p.enviados) != 1 || p.enviados[0] != "s1|"+texto {
		t.Fatalf("al enviar viaja el texto entero: %v", p.enviados)
	}
}

// La bienvenida resume igual el pegado multilínea.
func TestPegarMultilineaEnLaBienvenidaMuestraElToken(t *testing.T) {
	a := Nuevo(&puertoStub{})
	pulsa(t, a, tea.WindowSizeMsg{Width: 100, Height: 30})
	pulsa(t, a, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("uno\ndos"), Paste: true})

	if a.Bienvenida.Texto != "[PEGADO 2 líneas]" {
		t.Fatalf("la bienvenida guarda el token: %q", a.Bienvenida.Texto)
	}
	if got := a.Bienvenida.TextoExpandido(); got != "uno\ndos" {
		t.Fatalf("TextoExpandido devuelve el texto: %q", got)
	}
}

// --- T-F038: carpetas -------------------------------------------------------

// Arrastrar una carpeta muestra [CARPETA N elementos] y expande a su ruta.
func TestAdjuntosAnotaUnaCarpeta(t *testing.T) {
	carpeta := filepath.Join(t.TempDir(), "proyecto")
	if err := os.Mkdir(carpeta, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"a.go", "b.go", "c.go"} {
		if err := os.WriteFile(filepath.Join(carpeta, n), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	var ad adjuntos
	nota := ad.Anotar(carpeta)
	if nota != "[CARPETA 3 elementos]" {
		t.Fatalf("anotado = %q, quiero [CARPETA 3 elementos]", nota)
	}
	if got := ad.Expandir(nota); got != carpeta {
		t.Fatalf("expandido = %q, quiero la ruta", got)
	}
}

// El conteo va en singular con un solo elemento y en plural con cero o varios.
func TestAdjuntosCarpetaConteoEnSingularYPlural(t *testing.T) {
	vacia := t.TempDir()
	if nota := (&adjuntos{}).Anotar(vacia); nota != "[CARPETA 0 elementos]" {
		t.Fatalf("carpeta vacía = %q", nota)
	}

	una := filepath.Join(t.TempDir(), "pkg")
	if err := os.Mkdir(una, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(una, "a.go"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if nota := (&adjuntos{}).Anotar(una); nota != "[CARPETA 1 elemento]" {
		t.Fatalf("carpeta de uno = %q", nota)
	}
}

// Varias líneas con archivos y carpetas dejan un token por elemento.
func TestAdjuntosMultilineaConCarpeta(t *testing.T) {
	dir := t.TempDir()
	archivo := filepath.Join(dir, "main.go")
	carpeta := filepath.Join(dir, "pkg")
	if err := os.WriteFile(archivo, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(carpeta, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(carpeta, "a.go"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	var ad adjuntos
	nota := ad.Anotar(archivo + "\n" + carpeta)
	if nota != "[main.go] [CARPETA 1 elemento]" {
		t.Fatalf("anotado = %q", nota)
	}
	if got := ad.Expandir(nota); got != archivo+" "+carpeta {
		t.Fatalf("expandido = %q", got)
	}
}

// Dos carpetas con el mismo número de elementos no comparten token.
func TestAdjuntosDosCarpetasIgualesNoSePisan(t *testing.T) {
	a1 := filepath.Join(t.TempDir(), "uno")
	a2 := filepath.Join(t.TempDir(), "dos")
	for _, d := range []string{a1, a2} {
		if err := os.Mkdir(d, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(d, "x"), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	var ad adjuntos
	t1 := ad.Anotar(a1)
	t2 := ad.Anotar(a2)
	if t1 == t2 {
		t.Fatalf("dos carpetas distintas no comparten token: %q", t1)
	}
	if ad.Expandir(t1) != a1 || ad.Expandir(t2) != a2 {
		t.Errorf("cada token expande a su carpeta: %q / %q", ad.Expandir(t1), ad.Expandir(t2))
	}
}

// Arrastrar una carpeta en la vista principal muestra el token y envía su ruta.
func TestPegarUnaCarpetaMuestraElTokenYEnviaLaRuta(t *testing.T) {
	carpeta := filepath.Join(t.TempDir(), "proyecto")
	if err := os.Mkdir(carpeta, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(carpeta, "a.go"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	a, p := mainConSesion(t)
	pulsa(t, a, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(carpeta), Paste: true})

	if v := sinEstilo(a.View()); !strings.Contains(v, "[CARPETA 1 elemento]") {
		t.Fatalf("la entrada debe mostrar el token de la carpeta:\n%s", v)
	}
	if got := a.Entrada.Texto(); got != carpeta {
		t.Fatalf("Texto() expande a la ruta real: %q", got)
	}

	ejecuta(t, a, pulsa(t, a, tea.KeyMsg{Type: tea.KeyEnter}))
	if len(p.enviados) != 1 || p.enviados[0] != "s1|"+carpeta {
		t.Fatalf("enviados = %v, quiero la ruta de la carpeta", p.enviados)
	}
}
