package agent

import (
	"strings"
	"testing"
)

func TestLaPeticionLlevaElPromptIntegroYSoloLoConcedido(t *testing.T) {
	a := Agente{Nombre: "plan", Prompt: "eres plan\n\ncon salto de línea"}
	concedidas := []Herramienta{{Nombre: "leer_archivo", Permiso: PermisoRead}}

	p := ConstruirPeticion(a, []Mensaje{{Rol: RolUsuario, Contenido: "hola"}}, concedidas)

	if len(p.Mensajes) != 2 {
		t.Fatalf("la petición debería llevar prompt + historial: %+v", p.Mensajes)
	}
	if p.Mensajes[0].Rol != RolSistema || p.Mensajes[0].Contenido != a.Prompt {
		t.Fatalf("el prompt debería ir íntegro como mensaje de sistema: %+v", p.Mensajes[0])
	}
	if strings.Contains(p.Mensajes[0].Contenido, "leer_archivo") {
		t.Fatal("el catálogo no debería aparecer dentro del prompt")
	}
	if p.Mensajes[1].Rol != RolUsuario || p.Mensajes[1].Contenido != "hola" {
		t.Fatalf("el historial no se respetó: %+v", p.Mensajes[1])
	}
	if len(p.Herramientas) != 1 || p.Herramientas[0].Nombre != "leer_archivo" {
		t.Fatalf("el canal de herramientas debería llevar solo las concedidas: %+v", p.Herramientas)
	}
}
