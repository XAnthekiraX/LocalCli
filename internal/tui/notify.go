package tui

import "fmt"

// formatoAvisoPendientes compone la línea de aviso con el número de aprobaciones
// que esperan decisión: nada con cero —no hay nada que avisar—, singular con una
// y plural con más. Es la parte pura del aviso, la que se prueba sin pantalla
// (DOMAIN §1 `notify`: es el único dato que se muestra fuera del panel de
// aprobaciones, para que con el panel cerrado el contador siga visible).
func formatoAvisoPendientes(n int) string {
	if n <= 0 {
		return ""
	}
	if n == 1 {
		return "1 aprobación esperando tu decisión"
	}
	return fmt.Sprintf("%d aprobaciones esperando tu decisión", n)
}
