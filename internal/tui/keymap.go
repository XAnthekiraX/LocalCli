// keymap.go — T-B014-07: los atajos de teclado.
//
// Fuente de verdad: SPEC-INTERFAZ-ATAJOS §Reglas ("Trae atajos por defecto que
// se pueden cambiar sin reinstalar", "Un atajo no puede quedar asignado a dos
// acciones") y §Criterios de aceptación ("Se listan los atajos disponibles y la
// acción de cada uno").
//
// Todos los atajos de acción son combinaciones con `Ctrl` a propósito: el chat
// está escribiendo todo el rato, y una letra suelta como atajo haría imposible
// escribir la palabra que empieza por ella. `Enter` es la única tecla sin
// modificador, porque enviar es la acción principal de la pantalla.
//
// El mapa es un dato, no un switch enterrado: se valida (sin duplicados) y se
// puede sustituir por otro sin tocar la vista.
package tui

import (
	"fmt"
	"strings"
)

// Accion es lo que hace un atajo.
type Accion int

const (
	AccionNinguna Accion = iota
	AccionEnviar
	AccionSalir
	AccionPanel
	AccionSelector
	AccionRazonamiento
	AccionAprobar
	AccionDeclinar
	AccionPausar
	AccionCancelar
	AccionCerrarSelector
)

// Atajo une una tecla (en la notación de Bubble Tea: "ctrl+s", "enter") con su
// acción y con la descripción que se muestra en la ayuda.
type Atajo struct {
	Tecla       string
	Accion      Accion
	Descripcion string
}

// AtajosPorDefecto devuelve el mapa por defecto. Es una copia: cambiar el
// resultado no cambia los atajos del programa.
func AtajosPorDefecto() []Atajo {
	return []Atajo{
		{Tecla: "enter", Accion: AccionEnviar, Descripcion: "enviar la petición"},
		{Tecla: "ctrl+c", Accion: AccionSalir, Descripcion: "salir"},
		{Tecla: "ctrl+o", Accion: AccionPanel, Descripcion: "abrir o cerrar el panel de datos"},
		{Tecla: "ctrl+s", Accion: AccionSelector, Descripcion: "selector de sesiones"},
		{Tecla: "ctrl+r", Accion: AccionRazonamiento, Descripcion: "mostrar u ocultar el razonamiento"},
		{Tecla: "ctrl+a", Accion: AccionAprobar, Descripcion: "aprobar la propuesta seleccionada"},
		{Tecla: "ctrl+d", Accion: AccionDeclinar, Descripcion: "declinar la propuesta seleccionada"},
		{Tecla: "ctrl+p", Accion: AccionPausar, Descripcion: "pausar la cola en curso"},
		{Tecla: "ctrl+x", Accion: AccionCancelar, Descripcion: "cancelar el trabajo en curso"},
		{Tecla: "esc", Accion: AccionCerrarSelector, Descripcion: "cerrar el selector"},
	}
}

// ValidarAtajos rechaza dos atajos con la misma tecla (SPEC-INTERFAZ-ATAJOS:
// "Un atajo no puede quedar asignado a dos acciones") y teclas vacías.
func ValidarAtajos(atajos []Atajo) error {
	vistas := map[string]Accion{}
	for _, a := range atajos {
		if strings.TrimSpace(a.Tecla) == "" {
			return fmt.Errorf("tui: un atajo no puede tener la tecla vacía")
		}
		if a.Accion == AccionNinguna {
			return fmt.Errorf("tui: el atajo %s no tiene acción", a.Tecla)
		}
		if previa, ok := vistas[a.Tecla]; ok {
			return fmt.Errorf("tui: la tecla %s está asignada a dos acciones (%d y %d)", a.Tecla, previa, a.Accion)
		}
		vistas[a.Tecla] = a.Accion
	}
	return nil
}

// AccionDe devuelve la acción de una tecla. La segunda devolución distingue "no
// hay atajo para esa tecla" de "hay un atajo que no hace nada": la primera es el
// caso normal de estar escribiendo.
func AccionDe(atajos []Atajo, tecla string) (Accion, bool) {
	for _, a := range atajos {
		if a.Tecla == tecla {
			return a.Accion, true
		}
	}
	return AccionNinguna, false
}

// AyudaAtajos lista los atajos con lo que hace cada uno, en el orden del mapa.
func AyudaAtajos(atajos []Atajo) string {
	var b strings.Builder
	for _, a := range atajos {
		b.WriteString(fmt.Sprintf("%-9s %s\n", a.Tecla, a.Descripcion))
	}
	return b.String()
}
