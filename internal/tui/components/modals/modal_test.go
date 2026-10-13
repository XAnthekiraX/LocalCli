package modals

import (
	"strings"
	"testing"

	"localcli/internal/tui/render"
)

// Abrir muestra el modal con su lista y su aviso; cerrar no cambia nada de lo que
// había dentro.
func TestAbrirYCerrarNoCambianLaLista(t *testing.T) {
	var m Modal
	if m.Render(80, 20) != "" {
		t.Fatal("un modal cerrado no debería pintar nada")
	}
	m.Abrir("MODAL", []string{"uno", "dos"}, AvisoCargando)
	if !m.Abierto || m.Titulo != "MODAL" || m.Aviso != AvisoCargando {
		t.Fatalf("abrir dejó el modal en %+v", m)
	}
	m.Cerrar()
	if m.Abierto || len(m.Lineas) != 2 {
		t.Fatalf("cerrar no debería tocar la lista: %+v", m)
	}
}

// El resaltado recorre la lista y da la vuelta por los extremos; sin lista no hay
// a dónde moverse.
func TestElResaltadoRecorreLaListaYDavueltaLaVuelta(t *testing.T) {
	var m Modal
	m.Abrir("MODAL", []string{"uno", "dos", "tres"}, "")
	if indice, ok := m.Elegida(); !ok || indice != 0 {
		t.Fatalf("el resaltado debería arrancar en el primero: %d %v", indice, ok)
	}
	m.Mover(-1)
	if m.Indice != 2 {
		t.Fatalf("subir desde el primero = %d, quiero 2", m.Indice)
	}
	m.Mover(1)
	if m.Indice != 0 {
		t.Fatalf("bajar desde el último = %d, quiero 0", m.Indice)
	}
	vacio := Modal{}
	vacio.Abrir("MODAL", nil, AvisoSinModelos)
	vacio.Mover(1)
	if _, ok := vacio.Elegida(); ok {
		t.Fatal("sin lista no hay nada que elegir")
	}
}

// El render deja el bloque centrado en la ventana, con el título, el resaltado y
// el pie, y todas las filas de la misma anchura.
func TestElRenderCentraElBloque(t *testing.T) {
	var m Modal
	m.Abrir("MODAL", []string{"corto", "una línea bastante más larga que la otra"}, "")
	marco := m.Render(60, 12)
	lineas := strings.Split(render.SinANSI(marco), "\n")
	if len(lineas) != 12 {
		t.Fatalf("el marco tiene %d filas, quiero 12", len(lineas))
	}
	for i, linea := range lineas {
		if ancho := render.Ancho(linea); ancho != 60 {
			t.Fatalf("la fila %d mide %d columnas, quiero 60", i, ancho)
		}
	}
	plano := strings.Join(lineas, "\n")
	if !strings.Contains(plano, "MODAL") {
		t.Fatalf("falta el título:\n%s", plano)
	}
	if !strings.Contains(plano, "› corto") {
		t.Fatalf("falta la marca del resaltado:\n%s", plano)
	}
	if !strings.Contains(plano, PieModal) {
		t.Fatalf("falta el pie de teclas:\n%s", plano)
	}
	// El aviso sustituye a la lista cuando no hay nada que listar.
	var soloAviso Modal
	soloAviso.Abrir("MODAL", nil, AvisoSinModelos)
	if plano := render.SinANSI(soloAviso.Render(60, 12)); !strings.Contains(plano, AvisoSinModelos) {
		t.Fatalf("falta el aviso:\n%s", plano)
	}
}

// Un modal más alto que la ventana se recorta por el final y lo dice: lo que no
// se puede perder es la cabecera.
func TestUnaListaLargaNoSeComeLaCabecera(t *testing.T) {
	var m Modal
	var filas []string
	for i := 0; i < 40; i++ {
		filas = append(filas, "fila")
	}
	m.Abrir("MODAL", filas, "")

	plano := render.SinANSI(m.Render(60, 12))
	lineas := strings.Split(plano, "\n")
	if len(lineas) != 12 {
		t.Fatalf("el marco tiene %d filas, quiero 12", len(lineas))
	}
	if !strings.Contains(plano, "MODAL") {
		t.Fatalf("el recorte se comió el título:\n%s", plano)
	}
	if !strings.Contains(plano, "líneas más") {
		t.Fatalf("el recorte debería decir cuánto quedó fuera:\n%s", plano)
	}
	if !strings.Contains(plano, PieModal) {
		t.Fatalf("el recorte se comió el pie:\n%s", plano)
	}
}

// Un pie propio manda sobre el de fábrica: el modal de atajos no aplica nada y lo
// dice.
func TestElPiePropioMandaSobreElDeFabrica(t *testing.T) {
	m := Modal{Pie: PieAtajos}
	if got := m.PieTexto(); got != PieAtajos {
		t.Fatalf("el pie = %q, quiero %q", got, PieAtajos)
	}
	sinPie := Modal{}
	if got := sinPie.PieTexto(); got != PieModal {
		t.Fatalf("sin pie propio = %q, quiero %q", got, PieModal)
	}
}
