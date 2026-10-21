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

// `Ctrl+D` deja la pregunta con el nombre del resaltado y no entrega nada: el
// modal no borra ni decide; la lista y el resaltado siguen donde estaban.
func TestPedirEliminarPreguntaSinEntregarNada(t *testing.T) {
	var mm Modelos
	mm.Abrir("qwen2.5")
	mm.Fijar(modelosDePrueba(), nil)
	mm.Mover(1)

	mm.PedirEliminar()

	if !mm.Confirmando() {
		t.Fatal("`ctrl+d` no dejó la eliminación pendiente de confirmar")
	}
	plano := render.SinANSI(mm.Render(100, 20))
	if !strings.Contains(plano, "¿seguro que deseas eliminar `llama3.2`? (s/n)") {
		t.Fatalf("la pregunta no nombra el modelo resaltado:\n%s", plano)
	}
	if !strings.Contains(plano, PieModal) {
		t.Fatalf("con la pregunta pendiente el pie sigue siendo el de la lista:\n%s", plano)
	}
	if len(mm.Lista) != len(modelosDePrueba()) {
		t.Fatalf("preguntar no debe tocar la lista: %+v", mm.Lista)
	}
	if elegido, ok := mm.Elegido(); !ok || elegido.Nombre != "llama3.2" || mm.Indice != 1 {
		t.Fatalf("el resaltado se movió al preguntar: %d %+v", mm.Indice, elegido)
	}
}

// Mientras la pregunta está pendiente, las flechas no cambian el resaltado: `s`
// borra el modelo que nombra la pregunta.
func TestMoverNoCambiaElResaltadoMientrasConfirma(t *testing.T) {
	var mm Modelos
	mm.Abrir("")
	mm.Fijar(modelosDePrueba(), nil)
	mm.PedirEliminar()

	mm.Mover(1)

	if mm.Indice != 0 {
		t.Fatalf("el resaltado se movió con la pregunta pendiente: %d", mm.Indice)
	}
	if elegido, _ := mm.Elegido(); elegido.Nombre != "qwen2.5" {
		t.Fatalf("el modelo confirmado dejó de ser el resaltado: %+v", elegido)
	}
}

// Cancelar la pregunta la retira y deja la lista como estaba: no cambia nada.
func TestCancelarConfirmacionQuitaLaPregunta(t *testing.T) {
	var mm Modelos
	mm.Abrir("")
	mm.Fijar(modelosDePrueba(), nil)
	mm.PedirEliminar()

	mm.CancelarConfirmacion()

	if mm.Confirmando() {
		t.Fatal("la confirmación siguió pendiente después de cancelarla")
	}
	plano := render.SinANSI(mm.Render(100, 20))
	if strings.Contains(plano, "(s/n)") {
		t.Fatalf("la pregunta no desapareció:\n%s", plano)
	}
	if len(mm.Lista) != len(modelosDePrueba()) || mm.Indice != 0 {
		t.Fatalf("cancelar tocó la lista o el resaltado: %d %+v", mm.Indice, mm.Lista)
	}
}

// El error del puerto se pinta sobre la lista sin perderla: se lee por qué no se
// borró y se puede seguir eligiendo.
func TestElAvisoSePintaSinPerderLaLista(t *testing.T) {
	var mm Modelos
	mm.Abrir("")
	mm.Fijar(modelosDePrueba(), nil)

	mm.Avisar("session: el modelo \"qwen2.5\" está en uso; cambia de modelo antes de eliminarlo")

	plano := render.SinANSI(mm.Render(100, 20))
	if !strings.Contains(plano, "está en uso") {
		t.Fatalf("el aviso del puerto no se pintó:\n%s", plano)
	}
	if !strings.Contains(plano, "› Ollama local / qwen2.5") || len(mm.Lista) != len(modelosDePrueba()) {
		t.Fatalf("el aviso hizo perder la lista:\n%s", plano)
	}
}

// Una apertura nueva no arrastra la pregunta de la anterior.
func TestAbrirLimpiaLaConfirmacion(t *testing.T) {
	var mm Modelos
	mm.Abrir("")
	mm.Fijar(modelosDePrueba(), nil)
	mm.PedirEliminar()

	mm.Abrir("")

	if mm.Confirmando() || len(mm.Lista) != 0 {
		t.Fatalf("abrir no dejó el modal limpio: %+v", mm)
	}
}

// Colocar recorta el índice a la lista recargada: tras borrar el último, el
// resaltado cae en el nuevo último.
func TestColocarRecortaElIndiceALaLista(t *testing.T) {
	var mm Modelos
	mm.Abrir("")
	mm.Fijar(modelosDePrueba(), nil)

	mm.Colocar(99)
	if mm.Indice != len(mm.Lista)-1 {
		t.Fatalf("colocar al final dejó el índice en %d", mm.Indice)
	}
	mm.Colocar(-3)
	if mm.Indice != 0 {
		t.Fatalf("colocar al principio dejó el índice en %d", mm.Indice)
	}
}
