package modals

import (
	"errors"
	"strings"
	"testing"

	"localcli/internal/tui/render"
)

func modelosDePrueba() []Modelo {
	return []Modelo{
		{Nombre: "qwen2.5", Motor: "Ollama local"},
		{Nombre: "llama3.2", Motor: "Ollama local", SinHerramientas: true},
		{Nombre: "mistral", Motor: "llama.cpp 8080"},
	}
}

// El modal se abre pidiendo la lista: se ve «cargando…» y nada está resaltado
// todavía.
func TestAbrirPideLaLista(t *testing.T) {
	var mm Modelos
	mm.Abrir("qwen2.5")

	if !mm.Abierto || mm.Aviso != AvisoCargando || len(mm.Lista) != 0 {
		t.Fatalf("abrir dejó el modal en %+v", mm)
	}
	if _, ok := mm.Elegido(); ok {
		t.Fatal("sin lista no hay nada que elegir")
	}
	if mm.Pie != PieModal {
		t.Fatalf("el pie = %q", mm.Pie)
	}
}

// Al llegar la lista, el resaltado arranca en el modelo en uso.
func TestElResaltadoArrancaEnElModeloEnUso(t *testing.T) {
	var mm Modelos
	mm.Abrir("llama3.2")
	mm.Fijar(modelosDePrueba(), nil)

	if mm.Indice != 1 {
		t.Fatalf("el resaltado quedó en %d, quiero 1", mm.Indice)
	}
	elegido, ok := mm.Elegido()
	if !ok || elegido.Nombre != "llama3.2" {
		t.Fatalf("el elegido = %+v", elegido)
	}
	// Un modelo en uso que no está en la lista deja el resaltado en el primero.
	var otro Modelos
	otro.Abrir("desaparecido")
	otro.Fijar(modelosDePrueba(), nil)
	if otro.Indice != 0 {
		t.Fatalf("con un modelo ausente el resaltado quedó en %d, quiero 0", otro.Indice)
	}
}

// Cada fila rotula el motor que sirve el modelo, y los que no declaran
// herramientas van marcados: se sabe antes de elegirlos.
func TestCadaFilaRotulaSuMotorYMarcaSinHerramientas(t *testing.T) {
	var mm Modelos
	mm.Abrir("")
	mm.Fijar(modelosDePrueba(), nil)

	plano := strings.Join(mm.Lineas, "\n")
	if !strings.Contains(plano, "Ollama local / qwen2.5") {
		t.Fatalf("la fila no rotula su motor:\n%s", plano)
	}
	if !strings.Contains(plano, "llama.cpp 8080 / mistral") {
		t.Fatalf("la fila no rotula su motor:\n%s", plano)
	}
	if !strings.Contains(plano, "llama3.2  (sin herramientas)") {
		t.Fatalf("no se marca el modelo sin herramientas:\n%s", plano)
	}
	if strings.Contains(plano, "qwen2.5  (sin herramientas)") {
		t.Fatalf("no se debería marcar un modelo con herramientas:\n%s", plano)
	}
}

// Sin motor que responda queda el aviso «sin modelos» y el modal se cierra con
// `esc`: nada se bloquea.
func TestSinMotorQuedaElAvisoSinModelos(t *testing.T) {
	var mm Modelos
	mm.Abrir("qwen2.5")
	mm.Fijar(nil, errors.New("el motor no responde"))

	if mm.Aviso != AvisoSinModelos || len(mm.Lista) != 0 {
		t.Fatalf("el aviso = %q con %d modelos", mm.Aviso, len(mm.Lista))
	}
	if _, ok := mm.Elegido(); ok {
		t.Fatal("sin lista no debería haber nada que elegir")
	}
	// Una lista que llega tarde no reabre un modal ya cerrado.
	var tarde Modelos
	tarde.Abrir("")
	tarde.Cerrar()
	tarde.Fijar(modelosDePrueba(), nil)
	if tarde.Abierto || len(tarde.Lista) != 0 {
		t.Fatalf("una lista tardía no debería rellenar nada: %+v", tarde)
	}
}

// El render del modal de modelos se pinta centrado y con su pie.
func TestElModalDeModelosSePintaCentrado(t *testing.T) {
	var mm Modelos
	mm.Abrir("qwen2.5")
	mm.Fijar(modelosDePrueba(), nil)

	plano := render.SinANSI(mm.Render(100, 20))
	if !strings.Contains(plano, "MODELOS") || !strings.Contains(plano, "› Ollama local / qwen2.5") {
		t.Fatalf("el render del modal:\n%s", plano)
	}
	if !strings.Contains(plano, PieModal) {
		t.Fatalf("falta el pie:\n%s", plano)
	}
}
