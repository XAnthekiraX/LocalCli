// notify.go — T-F009: la línea de aviso de aprobaciones pendientes.
//
// Fuente de verdad: SPEC-INTERFAZ §"El aviso que no se puede ocultar" ("si
// alguna sesión está esperando tu aprobación, aparece una línea discreta
// indicando cuántas hay"), frontend/01-domain/DOMAIN.md §1 (notify: "el único
// dato que se muestra fuera del panel") y §2 ("con el panel cerrado, el
// contador de aprobaciones pendientes sigue visible y se actualiza con cada
// evento").
//
// El contador lo alimentan los eventos (`peticion_aprobacion` suma,
// `aprobacion_resuelta` resta, wire.go) y llega aquí como número: esta pieza
// solo decide cómo se ve el número. Con el panel de datos abierto el dato ya
// está en su fila del panel, y una línea duplicada solo estorba.
package tui

import "fmt"

// formatoAvisoPendientes compone el texto del aviso: nada si no hay nada que
// decidir, singular con una y plural con más. Es la parte del formato, pura,
// que se prueba sin pantalla (T-F009-01).
func formatoAvisoPendientes(n int) string {
	if n <= 0 {
		return ""
	}
	if n == 1 {
		return "1 aprobación esperando tu decisión"
	}
	return fmt.Sprintf("%d aprobaciones esperando tu decisión", n)
}
