package modals

import (
	"strings"
	"testing"

	"localcli/internal/tui/render"
)

func filasDePrueba() []Fila {
	return []Fila{
		{Grupo: "General", Tecla: "ctrl+c", Accion: "salir"},
		{Grupo: "General", Tecla: "ctrl+x m", Accion: "modal de modelos"},
		{Grupo: "Chat", Tecla: "up", Accion: "subir por el historial del chat"},
		{Grupo: "Vista", Tecla: "(deshabilitada)", Accion: "pausar la cola en curso"},
	}
}

// La tabla agrupa por categorías y alinea la columna de teclas: cada línea lleva
// su tecla y su acción.
func TestLaTablaAgrupaYAlinea(t *testing.T) {
	var am Atajos
	am.Abrir(filasDePrueba())

	plano := strings.Join(am.Lineas, "\n")
	if !strings.Contains(plano, "General") || !strings.Contains(plano, "Chat") || !strings.Contains(plano, "Vista") {
		t.Fatalf("faltan categorías:\n%s", plano)
	}
	for _, quiero := range []string{"ctrl+c", "salir", "ctrl+x m", "modal de modelos"} {
		if !strings.Contains(plano, quiero) {
			t.Errorf("la tabla no trae %q:\n%s", quiero, plano)
		}
	}
	// La acción arranca en la misma columna en todas las filas de la tabla.
	columna := -1
	for _, fila := range filasDePrueba() {
		for _, linea := range am.Lineas {
			if !strings.Contains(linea, fila.Accion) {
				continue
			}
			pos := strings.Index(linea, fila.Accion)
			if columna == -1 {
				columna = pos
				continue
			}
			if pos != columna {
				t.Fatalf("la columna de acciones no está alineada: %d y %d en %q", columna, pos, linea)
			}
		}
	}
	if columna == -1 {
		t.Fatal("no se encontró ninguna acción en la tabla")
	}
}

// Es de solo lectura: no hay nada resaltado y `enter` no elige nada.
func TestElModalDeAtajosEsDeSoloLectura(t *testing.T) {
	var am Atajos
	am.Abrir(filasDePrueba())

	if am.Indice != -1 {
		t.Fatalf("el índice de un modal de solo lectura = %d, quiero -1", am.Indice)
	}
	if _, ok := am.Elegida(); ok {
		t.Fatal("sin resaltado no debería haber nada elegido")
	}
	if am.Pie != PieAtajos {
		t.Fatalf("el pie = %q, quiero %q", am.Pie, PieAtajos)
	}
	// Navegar no mueve nada: no hay lista que recorrer.
	am.Mover(1)
	if am.Indice != -1 {
		t.Fatalf("navegar movió el resaltado: %d", am.Indice)
	}
}

// El render del modal de atajos se centra con su título y su pie de solo lectura.
func TestElModalDeAtajosSePintaCentrado(t *testing.T) {
	var am Atajos
	am.Abrir(filasDePrueba())

	plano := render.SinANSI(am.Render(80, 20))
	if !strings.Contains(plano, "ATAJOS") || !strings.Contains(plano, PieAtajos) {
		t.Fatalf("el render del modal:\n%s", plano)
	}
	if strings.Contains(plano, "›") {
		t.Fatalf("un modal de solo lectura no debería resaltar filas:\n%s", plano)
	}
}

// Una acción sin teclas se lista como deshabilitada: la tabla no deja un hueco
// mudo.
func TestUnaAccionSinTeclasSeListaDeshabilitada(t *testing.T) {
	var am Atajos
	am.Abrir([]Fila{{Grupo: "Vista", Tecla: "(deshabilitada)", Accion: "pausar la cola en curso"}})

	if !strings.Contains(strings.Join(am.Lineas, "\n"), "(deshabilitada)") {
		t.Fatalf("la acción deshabilitada debería decirse:\n%s", strings.Join(am.Lineas, "\n"))
	}
}
