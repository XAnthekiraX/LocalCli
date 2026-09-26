package flow

// detect.go — T-B010-02: detectar "trabajo ordenado" en una petición.
//
// Fuente de verdad: ai/docs/backend/01-domain/BUSINESS_RULES.md §Detección de
// trabajo ordenado y [[specs/SPEC-COLA-TAREAS]] ("Documentar capa por capa",
// "ejecutar tarea 1, tarea 2, tarea 3", "primero esto, luego esto" crean un
// TODO; una petición sin orden ni lista no lo crea y se responde en el chat).
//
// Quien detecta es `flow`, no `queue`: esto es una heurística de lectura, no
// una decisión de permisos. Si acierta de más, el TODO se crea y se ve; si
// acierta de menos, la petición se responde en el chat.

import (
	"regexp"
	"strings"
)

// marcadores de secuencia explícita.
var patronesOrden = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bcapa por capa\b`),
	regexp.MustCompile(`(?i)\bpaso a paso\b`),
	regexp.MustCompile(`(?i)\buno por uno\b`),
	regexp.MustCompile(`(?i)\ben orden\b`),
	regexp.MustCompile(`(?i)\bprimero\b[\s\S]*\b(luego|despu[eé]s)\b`),
	regexp.MustCompile(`(?i)\ba continuaci[oó]n\b`),
}

// listaNumerada reconoce viñetas numeradas en el texto ("1. ...", "2) ...").
var listaNumerada = regexp.MustCompile(`(?m)^\s*[0-9]{1,2}[.)]\s+\S`)

// tareaNumerada reconoce referencias a tareas numeradas ("tarea 1", "tarea 2").
var tareaNumerada = regexp.MustCompile(`(?i)\btarea\s+[0-9]{1,2}\b`)

// EsTrabajoOrdenado informa si la petición implica una lista ordenada de
// trabajo y por tanto debe crear un TODO.
func EsTrabajoOrdenado(peticion string) bool {
	if strings.TrimSpace(peticion) == "" {
		return false
	}
	for _, re := range patronesOrden {
		if re.MatchString(peticion) {
			return true
		}
	}
	if listaNumerada.MatchString(peticion) {
		return true
	}
	// Al menos dos tareas numeradas distintas ("tarea 1, tarea 2").
	if len(tareaNumerada.FindAllString(peticion, -1)) >= 2 {
		return true
	}
	return false
}
