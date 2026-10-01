package tools

import (
	"strings"
	"testing"
)

// TestEsquemaDesdeDTO — el esquema que ve el modelo se deriva del DTO: tipo por
// campo, obligatoriedad por `omitempty` y descripción por el tag `desc`.
func TestEsquemaDesdeDTO(t *testing.T) {
	h, ok := Buscar("leer_archivo")
	if !ok {
		t.Fatal("falta leer_archivo")
	}
	if h.Esquema == nil || h.Esquema.Type != "object" {
		t.Fatalf("esquema = %+v, quiero un objeto", h.Esquema)
	}
	prop, ok := h.Esquema.Properties["ruta"]
	if !ok {
		t.Fatal("el esquema debe exponer el campo `ruta`")
	}
	if prop.Type != "string" {
		t.Errorf("tipo de ruta = %q, quiero string", prop.Type)
	}
	if prop.Description == "" {
		t.Error("un campo con tag `desc` debe aparecer con `description`")
	}
	if len(h.Esquema.Required) != 1 || h.Esquema.Required[0] != "ruta" {
		t.Errorf("required = %v, quiero [ruta]", h.Esquema.Required)
	}
}

// TestEsquemaOpcionalNoVaEnRequired — un campo con `omitempty` es opcional.
func TestEsquemaOpcionalNoVaEnRequired(t *testing.T) {
	h, _ := Buscar("buscar_en_archivos")
	if _, ok := h.Esquema.Properties["ruta"]; !ok {
		t.Fatal("el esquema debe exponer `ruta`")
	}
	for _, r := range h.Esquema.Required {
		if r == "ruta" {
			t.Error("`ruta` lleva omitempty: no puede ser obligatoria")
		}
	}
	found := false
	for _, r := range h.Esquema.Required {
		if r == "patron" {
			found = true
		}
	}
	if !found {
		t.Errorf("required = %v, quiero incluir `patron`", h.Esquema.Required)
	}
}

// `motivo` viaja en el esquema como opcional: solo hace falta cuando la ruta
// sale de la carpeta del proyecto (SPEC-ARCHIVOS §Reglas), y eso lo decide el
// momento de usarla, no la obligatoriedad del esquema.
func TestElEsquemaLlevaElMotivoOpcional(t *testing.T) {
	h, ok := NuevaHerramienta("leer_archivo", nil)
	if !ok {
		t.Fatal("leer_archivo está en el catálogo")
	}
	if _, hay := h.Esquema.Properties["motivo"]; !hay {
		t.Fatal("el esquema declara el motivo")
	}
	for _, r := range h.Esquema.Required {
		if r == "motivo" {
			t.Error("el motivo no es obligatorio en el esquema")
		}
	}
}

// TestEsquemaDeTodasLasHerramientas — las quince tienen esquema y ninguna queda
// sin describir.
func TestEsquemaDeTodasLasHerramientas(t *testing.T) {
	for _, h := range Herramientas() {
		if h.Esquema == nil || h.Esquema.Type != "object" {
			t.Errorf("%s: sin esquema derivado", h.Nombre)
		}
	}
}

// TestTruncadoRespetaElLimite — el recorte deja la salida dentro del límite y
// avisa de que se cortó.
func TestTruncadoRespetaElLimite(t *testing.T) {
	larga := strings.Repeat("palabra ", 5000) // ~40 KB
	corta, cortado := Recortar(larga, 100)
	if !cortado {
		t.Fatal("una salida enorme debe recortarse")
	}
	if len(corta) >= len(larga) {
		t.Error("el recorte no acortó nada")
	}
	if !strings.Contains(corta, "recortada") {
		t.Error("el recorte debe avisar de que se cortó")
	}
	if got := EstimarTokens(corta); got > 130 {
		t.Errorf("la salida recortada sigue siendo grande: %d tokens", got)
	}
	if _, cortado := Recortar("corto", 100); cortado {
		t.Error("una salida corta no paga el recorte")
	}
}
