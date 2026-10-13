package input

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"localcli/internal/tui/render"
)

// El conteo de tokens se acorta en la unidad que se lee de un vistazo: entero por
// debajo del millar, miles con un decimal hasta diez mil y miles enteros después.
func TestFormatearTokens(t *testing.T) {
	casos := []struct {
		n      int
		quiero string
	}{
		{0, "0"},
		{-5, "0"},
		{54, "54"},
		{999, "999"},
		{1000, "1.0k"},
		{1234, "1.2k"},
		{9999, "10.0k"},
		{54_000, "54k"},
	}
	for _, c := range casos {
		if got := formatearTokens(c.n); got != c.quiero {
			t.Errorf("formatearTokens(%d) = %q, quiero %q", c.n, got, c.quiero)
		}
	}
}

// La caja vacía enseña su placeholder entre comillas y sin cursor.
func TestLaCajaVaciaEnsenaSuPlaceholder(t *testing.T) {
	m := Nuevo("plan", "qwen3:8b", "llama.cpp", Capacidades{})
	got := render.SinANSI(m.View(40))
	if !strings.Contains(got, `"`+placeholder+`"`) {
		t.Fatalf("no se ve el placeholder:\n%s", got)
	}
	if strings.Contains(got, cursor) {
		t.Fatalf("sin texto no debería haber cursor:\n%s", got)
	}
}

// La línea crece en filas al desbordar el ancho, sin recortar lo escrito.
func TestLaLineaCreceEnFilasSinRecortar(t *testing.T) {
	m := Nuevo("plan", "qwen3:8b", "llama.cpp", Capacidades{})
	escrito := "una petición bastante larga que desborda el ancho de la caja"
	m.Escribir(escrito)

	filas, filaCursor := m.filasInteriores(20)
	texto := strings.ReplaceAll(render.SinANSI(strings.Join(filas, "")), cursor, "")
	if texto != escrito {
		t.Fatalf("el envoltorio perdió texto:\n%q\n%q", escrito, texto)
	}
	if len(filas) < 2 {
		t.Fatalf("el texto debería ocupar varias filas: %d", len(filas))
	}
	if filaCursor < 0 || filaCursor >= len(filas) {
		t.Fatalf("el cursor quedó fuera de las filas: %d", filaCursor)
	}
	if !strings.Contains(render.SinANSI(filas[filaCursor]), cursor) {
		t.Fatalf("el cursor no está en su fila:\n%s", render.SinANSI(filas[filaCursor]))
	}
}

// Al pasar del tope de filas la ventana se desplaza para que el cursor siga
// visible: la caja no crece sin fin ni pierde el cursor.
func TestLaVentanaSigueAlCursorAlPasarDelTope(t *testing.T) {
	filas := []string{"f0", "f1", "f2", "f3", "f4", "f5", "f6"}
	ventana, inicio := ventanaFilas(filas, 6, 3)
	if inicio != 4 || strings.Join(ventana, ",") != "f4,f5,f6" {
		t.Fatalf("la ventana con el cursor al final = %v desde %d", ventana, inicio)
	}
	// Todo lo que cabe no se toca.
	ventana, inicio = ventanaFilas(filas, 0, 20)
	if inicio != 0 || len(ventana) != len(filas) {
		t.Fatalf("una lista corta no debería desplazarse: %d desde %d", len(ventana), inicio)
	}
}

// Lo escrito se envía expandido: los tokens vuelven a su valor real y sin
// espacios sobrantes.
func TestLoEscritoSeEnviaExpandido(t *testing.T) {
	m := Nuevo("plan", "qwen3:8b", "llama.cpp", Capacidades{})
	m.Escribir("resume esto: ")
	m.InsertarPegado("primera línea\nsegunda línea")

	if !strings.Contains(m.Texto, "[PEGADO") {
		t.Fatalf("el texto pegado debería mostrarse como token: %q", m.Texto)
	}
	if got := m.Valor(); got != "resume esto: primera línea\nsegunda línea" {
		t.Fatalf("el valor enviado = %q", got)
	}
	if got := m.Peticion(); got != "resume esto: primera línea\nsegunda línea" {
		t.Fatalf("la petición = %q", got)
	}
}

// `Enter` con texto emite la petición; sin texto no emite nada. No inserta
// saltos de línea.
func TestEnterEmiteLaPeticion(t *testing.T) {
	m := Nuevo("plan", "qwen3:8b", "llama.cpp", Capacidades{})
	if _, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter}); cmd != nil {
		t.Fatal("sin texto, `enter` no debería emitir nada")
	}
	m.Escribir("hola")
	modelo, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("con texto, `enter` debería emitir la petición")
	}
	if msg, ok := cmd().(EnviadoMsg); !ok || msg.Texto != "hola" {
		t.Fatalf("el mensaje emitido = %#v", cmd())
	}
	if strings.Contains(modelo.Texto, "\n") {
		t.Fatalf("`enter` no debería insertar saltos: %q", modelo.Texto)
	}
}

// El interruptor de razonamiento solo existe si el modelo declara la capacidad, y
// el ratón solo lo encuentra entonces.
func TestElInterruptorSoloExisteSiElModeloRazona(t *testing.T) {
	sinCapacidad := Nuevo("plan", "m", "", Capacidades{Herramientas: Soportada})
	sinCapacidad.View(40)
	if sinCapacidad.pie(40).tienePensar {
		t.Fatal("sin la capacidad declarada no debería haber interruptor")
	}
	sinCapacidad.AlternarPensar()
	if sinCapacidad.Pensar {
		t.Fatal("sin la capacidad declarada el interruptor no debería cambiar")
	}

	conCapacidad := Nuevo("plan", "m", "", Capacidades{Razonamiento: Soportada})
	conCapacidad.View(40)
	conCapacidad.AlternarPensar()
	if !conCapacidad.Pensar {
		t.Fatal("el interruptor debería encenderse")
	}
	if !strings.Contains(render.SinANSI(conCapacidad.View(40)), "pensar [x]") {
		t.Fatal("encendido, el pie debería enseñarlo")
	}
	if !conCapacidad.SobrePensar(conCapacidad.rectPensar.Fila, conCapacidad.rectPensar.Columna) {
		t.Fatal("el interruptor debería quedar localizable para el ratón")
	}
}

// El pie de la caja enseña el agente, el modelo con su motor y las capacidades.
func TestElPieEnsenaAgenteModeloYCapacidades(t *testing.T) {
	m := Nuevo("build", "qwen3:8b", "llama.cpp", Capacidades{
		Herramientas: Soportada,
		Vision:       NoSoportada,
	})
	got := render.SinANSI(m.View(60))
	for _, quiero := range []string{"[build]", "qwen3:8b (llama.cpp)", "tool [*]", "[v]", "[T]"} {
		if !strings.Contains(got, quiero) {
			t.Errorf("el pie no enseña %q:\n%s", quiero, got)
		}
	}
	// Lo que aún no se sabe se marca, en vez de afirmar que no puede.
	m.Capacidades.Vision = Desconocida
	if got := render.SinANSI(m.View(60)); !strings.Contains(got, "[v?]") {
		t.Errorf("una capacidad sin dato debería marcarse:\n%s", got)
	}
}

// El conteo de tokens del turno se ve bajo la caja solo cuando hay consumo.
func TestElConteoDeTokensVaBajoLaCaja(t *testing.T) {
	m := Nuevo("plan", "qwen3:8b", "llama.cpp", Capacidades{})
	if got := render.SinANSI(m.View(40)); strings.Contains(got, "tokens:") {
		t.Fatalf("sin consumo no debería haber conteo:\n%s", got)
	}
	m.Tokens = 1234
	if got := render.SinANSI(m.View(40)); !strings.Contains(got, "tokens: 1.2k") {
		t.Fatalf("con consumo debería verse el conteo:\n%s", got)
	}
}

// Un pegado llega como un único `KeyRunes` con `Paste` en alto (bracketed paste):
// se anota y se muestra como token, no como la ruta o el texto crudos.
func TestElPegadoSeConvierteEnToken(t *testing.T) {
	m := Nuevo("plan", "qwen3:8b", "llama.cpp", Capacidades{})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Paste: true, Runes: []rune("primera\nsegunda")})

	if !strings.Contains(m.Texto, "[PEGADO 2 líneas]") {
		t.Fatalf("el pegado debería mostrarse como token: %q", m.Texto)
	}
	if got := m.Valor(); got != "primera\nsegunda" {
		t.Fatalf("el valor enviado = %q", got)
	}
}

// La ruta de un archivo pegado se ve como su nombre y se envía entera.
func TestElPegadoDeUnaRutaSeMuestraComoToken(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "foto.png")
	if err := os.WriteFile(ruta, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	m := Nuevo("plan", "qwen3:8b", "llama.cpp", Capacidades{})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Paste: true, Runes: []rune(ruta)})

	if strings.Contains(m.Texto, ruta) || !strings.Contains(m.Texto, "[foto.png]") {
		t.Fatalf("la ruta debería verse como token: %q", m.Texto)
	}
	if got := m.Valor(); got != ruta {
		t.Fatalf("el valor enviado = %q, quiero la ruta real %q", got, ruta)
	}
}

// Una carpeta pegada se resume con su número de elementos.
func TestElPegadoDeUnaCarpetaSeMuestraComoToken(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "uno.txt"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	m := Nuevo("plan", "qwen3:8b", "llama.cpp", Capacidades{})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Paste: true, Runes: []rune(dir)})

	if !strings.Contains(m.Texto, "[CARPETA 1 elemento]") {
		t.Fatalf("la carpeta debería verse como token: %q", m.Texto)
	}
	if got := m.Valor(); got != dir {
		t.Fatalf("el valor enviado = %q, quiero la ruta real %q", got, dir)
	}
}

// Al pintar la línea, cada token se tiñe con el estilo de adjunto y el resto con
// el estilo base, sin perder el texto.
func TestLosTokensSeResaltanAlPintar(t *testing.T) {
	m := Nuevo("plan", "qwen3:8b", "llama.cpp", Capacidades{})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Paste: true, Runes: []rune("primera\nsegunda")})
	m.Escribir(" revisa ")

	resaltada := m.adjuntos.Estilizar(m.Texto)
	if !strings.Contains(resaltada, render.EstiloAdjunto.Render("[PEGADO 2 líneas]")) {
		t.Fatalf("el token debería ir con el estilo de adjunto:\n%q", resaltada)
	}
	if texto := render.SinANSI(resaltada); texto != m.Texto {
		t.Fatalf("resaltar alteró el texto: %q", texto)
	}
}

// Limpiar vacía la línea y olvida los adjuntos: lo siguiente que se escriba no
// arrastra el valor de un token anterior.
func TestLimpiarOlvidaLosAdjuntos(t *testing.T) {
	m := Nuevo("plan", "m", "", Capacidades{})
	m.InsertarPegado("uno\ndos")
	m.Limpiar()

	if m.Texto != "" || m.Pos != 0 {
		t.Fatalf("limpiar dejó la línea con %q en %d", m.Texto, m.Pos)
	}
	if got := m.Valor(); got != "" {
		t.Fatalf("los adjuntos sobrevivieron a limpiar: %q", got)
	}
}
