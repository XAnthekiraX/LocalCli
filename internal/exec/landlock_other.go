//go:build !linux

package exec

// landlock_other.go — T-B009-06: degradación sin Landlock.
//
// Fuente de verdad: ai/docs/backend/03-security/SECURITY.md §6 ("Aislamiento en
// sistemas sin Landlock: en sistemas que no son Linux... la terminal no tiene el
// bloqueo estructural. La garantía es más débil y queda documentada como tal; el
// proyecto lo asume") y ai/docs/backend/04-infrastructure/CONFIGURATION.md §5
// ("Si no lo está, el harness lo avisa y la garantía de la terminal es más
// débil, sin bloquear el uso").
//
// En un sistema que no es Linux no hay Landlock. El módulo sigue funcionando: el
// comando se lanza directamente y el harness muestra el aviso de que la garantía
// es más débil. No se finge un aislamiento que no existe.

// soportaLandlock es siempre false fuera de Linux.
func soportaLandlock() bool { return false }

// avisoDegradacion explica la garantía reducida.
func avisoDegradacion() string {
	return "la terminal no está bloqueada estructuralmente: este sistema no tiene Landlock (E_NO_LANDLOCK)"
}
