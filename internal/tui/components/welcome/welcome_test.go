package welcome

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"localcli/internal/tui/render"
)

// arteDePrueba es un logotipo corto: las pruebas del reparto no dependen del arte
// real, que tiene su propia prueba byte a byte. arteDePruebaCompacto es su versión
// estrecha.
const (
	arteDePrueba         = "AAA\nBBB"
	arteDePruebaCompacto = "cc"
)

func datos() Datos {
	return Datos{Agente: "plan", Modelo: "qwen3:8b", Motor: "llama.cpp"}
}

// La bienvenida se centra en la terminal y su caja de entrada sabe qué
// rectángulo ocupa: es su propia zona de selección.
func TestElRepartoCentraLaCajaYSusLimites(t *testing.T) {
	m := Nuevo(arteDePrueba, arteDePruebaCompacto)
	pantalla, limites := m.VistaLimites(100, 30, datos())

	lineas := strings.Split(pantalla, "\n")
	if len(lineas) != 30 {
		t.Fatalf("el marco tiene %d filas, quiero 30", len(lineas))
	}
	for i, linea := range lineas {
		if ancho := render.Ancho(linea); ancho != 100 {
			t.Fatalf("la fila %d mide %d columnas, quiero 100", i, ancho)
		}
	}
	if limites.Alto <= 0 || limites.Ancho <= 0 {
		t.Fatalf("la caja no expuso sus límites: %+v", limites)
	}
	if limites.Fila+limites.Alto > 30 || limites.Columna+limites.Ancho > 100 {
		t.Fatalf("la caja se sale del marco: %+v", limites)
	}
	// El placeholder cae dentro de la caja: el reparto de filas es el que dice.
	filaTexto := -1
	for i, linea := range lineas {
		if strings.Contains(linea, placeholder) {
			filaTexto = i
		}
	}
	if filaTexto != limites.Fila+1 {
		t.Fatalf("el texto está en la fila %d y el rectángulo empieza en %d", filaTexto, limites.Fila)
	}
}

// La caja nunca se compone por debajo del ancho del arte, y cede antes que
// recortar el logotipo.
func TestLaCajaNoBajaDelAnchoDelArte(t *testing.T) {
	m := Nuevo(arteDePrueba, arteDePruebaCompacto)
	_, limites := m.VistaLimites(100, 30, datos())
	if limites.Ancho < m.anchoArte() {
		t.Fatalf("la caja mide %d columnas y el arte %d", limites.Ancho, m.anchoArte())
	}
}

// En una terminal estrecha la bienvenida se ajusta: usa el logotipo compacto y
// ninguna fila supera el ancho disponible (SPEC-INTERFAZ §Arte canónico).
func TestLaBienvenidaSeAjustaAlAnchoEstrecho(t *testing.T) {
	m := Nuevo(strings.Repeat("█", 90), arteDePruebaCompacto)
	for _, ancho := range []int{40, 60, 69} {
		pantalla, limites := m.VistaLimites(ancho, 24, datos())
		for i, linea := range strings.Split(pantalla, "\n") {
			if w := render.Ancho(linea); w > ancho {
				t.Fatalf("a %d columnas la fila %d mide %d", ancho, i, w)
			}
		}
		if !strings.Contains(render.SinANSI(pantalla), arteDePruebaCompacto) {
			t.Fatalf("a %d columnas no se pintó el logotipo compacto:\n%s", ancho, render.SinANSI(pantalla))
		}
		if limites.Columna+limites.Ancho > ancho {
			t.Fatalf("la caja se sale a %d columnas: %+v", ancho, limites)
		}
	}
}

// La línea de estado rotula el modelo con el motor que lo sirve, y sin motor no
// deja paréntesis colgando.
func TestLaLineaDeEstadoRotulaElModelo(t *testing.T) {
	lineaEstado := func(d Datos) string {
		filas, _ := estado(d, 200)
		return strings.Join(filas, " ")
	}
	if got := render.SinANSI(lineaEstado(datos())); !strings.Contains(got, "[plan] • qwen3:8b (llama.cpp)") {
		t.Errorf("la línea de estado = %q", got)
	}
	if got := lineaEstado(Datos{Agente: "plan", Modelo: "qwen3:8b"}); !strings.Contains(render.SinANSI(got), "qwen3:8b") ||
		strings.Contains(render.SinANSI(got), "(") {
		t.Errorf("sin motor la línea = %q", render.SinANSI(got))
	}
	if got := render.SinANSI(lineaEstado(Datos{Agente: "plan"})); !strings.Contains(got, "—") {
		t.Errorf("sin modelo la línea = %q", got)
	}
}

// El aviso del arranque se pinta bajo las pistas, sin romper el centrado.
func TestElAvisoSePintaBajoLasPistas(t *testing.T) {
	m := Nuevo(arteDePrueba, arteDePruebaCompacto)
	conAviso, _ := m.VistaLimites(100, 30, Datos{Agente: "plan", Aviso: "no se pudo leer las preferencias"})
	if !strings.Contains(render.SinANSI(conAviso), "no se pudo leer las preferencias") {
		t.Fatalf("el aviso no se pintó:\n%s", render.SinANSI(conAviso))
	}
}

// Sin geometría —antes de la primera medida de la terminal— la bienvenida se
// pinta en bloque, sin centrar y sin perderse.
func TestSinGeometriaSePintaEnBloqueLlano(t *testing.T) {
	m := Nuevo(arteDePrueba, arteDePruebaCompacto)
	pantalla, limites := m.VistaLimites(0, 0, datos())

	if !strings.Contains(render.SinANSI(pantalla), "AAA") {
		t.Fatalf("sin geometría se perdió el arte:\n%s", render.SinANSI(pantalla))
	}
	if !strings.Contains(render.SinANSI(pantalla), "modelo: qwen3:8b (llama.cpp)") {
		t.Fatalf("sin geometría se perdió la línea de modelo:\n%s", render.SinANSI(pantalla))
	}
	if limites.Alto != 0 {
		t.Fatalf("sin geometría no debería haber límites: %+v", limites)
	}
}

// Las pistas de teclado son las de fábrica, con la tecla en blanco y la acción
// atenuada.
func TestLaBarraDePistas(t *testing.T) {
	got := render.SinANSI(barraPistas())
	if got != "TAB agentes · Ctrl+P comandos · Ctrl+X M modelos" {
		t.Errorf("la barra de pistas = %q", got)
	}
}

// La línea de entrada de la bienvenida se edita en cualquier punto, como la de la
// vista principal.
func TestLaLineaSeEditaEnCualquierPunto(t *testing.T) {
	m := Nuevo(arteDePrueba, arteDePruebaCompacto)
	m.Escribir("hola")
	m.Izquierda()
	m.Escribir("X")
	if m.Texto != "holXa" {
		t.Fatalf("escribir en medio = %q, quiero %q", m.Texto, "holXa")
	}
	m.Fin()
	m.Borrar()
	if m.Texto != "holX" {
		t.Fatalf("borrar al final = %q", m.Texto)
	}
	m.Inicio()
	m.BorrarAdelante()
	if m.Texto != "olX" {
		t.Fatalf("borrar adelante al principio = %q", m.Texto)
	}
	if got := m.Peticion(); got != "olX" {
		t.Fatalf("la petición = %q", got)
	}
	m.Limpiar()
	if m.Texto != "" || m.Pos != 0 || m.Peticion() != "" {
		t.Fatalf("limpiar dejó %q en %d", m.Texto, m.Pos)
	}
}

// Un pegado de varias líneas se ve como token y la petición va expandida.
func TestElPegadoMultilineaSeTokeniza(t *testing.T) {
	m := Nuevo(arteDePrueba, arteDePruebaCompacto)
	m.InsertarPegado("primera\nsegunda")

	if !strings.Contains(m.Texto, "[PEGADO 2 líneas]") {
		t.Fatalf("el pegado debería verse como token: %q", m.Texto)
	}
	if got := m.Peticion(); got != "primera\nsegunda" {
		t.Fatalf("la petición = %q", got)
	}
}

// Una ruta pegada se ve como su nombre y se expande entera al enviar.
func TestElPegadoDeUnaRutaSeTokeniza(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "foto.png")
	if err := os.WriteFile(ruta, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	m := Nuevo(arteDePrueba, arteDePruebaCompacto)
	m.InsertarPegado(ruta)

	if strings.Contains(m.Texto, ruta) || !strings.Contains(m.Texto, "[foto.png]") {
		t.Fatalf("la ruta debería verse como token: %q", m.Texto)
	}
	if got := m.Peticion(); got != ruta {
		t.Fatalf("la petición = %q, quiero %q", got, ruta)
	}
}

// Una carpeta pegada se resume con su número de elementos.
func TestElPegadoDeUnaCarpetaSeTokeniza(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "uno.txt"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	m := Nuevo(arteDePrueba, arteDePruebaCompacto)
	m.InsertarPegado(dir)

	if !strings.Contains(m.Texto, "[CARPETA 1 elemento]") {
		t.Fatalf("la carpeta debería verse como token: %q", m.Texto)
	}
	if got := m.Peticion(); got != dir {
		t.Fatalf("la petición = %q, quiero %q", got, dir)
	}
}

// Limpiar olvida los adjuntos: lo siguiente que se escriba no arrastra el valor
// de un token anterior.
func TestLimpiarOlvidaLosAdjuntos(t *testing.T) {
	m := Nuevo(arteDePrueba, arteDePruebaCompacto)
	m.InsertarPegado("uno\ndos")
	m.Limpiar()

	if m.Texto != "" || m.Pos != 0 || m.Peticion() != "" {
		t.Fatalf("limpiar dejó %q en %d", m.Texto, m.Pos)
	}
}
