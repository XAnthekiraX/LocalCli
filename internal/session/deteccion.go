package session

// deteccion.go — T-B019-01: detección de trabajo ordenado.
//
// Fuente de verdad: [[backend/01-domain/BUSINESS_RULES]] §Detección de trabajo
// ordenado y [[specs/SPEC-COLA-TAREAS]]. Una petición que implica una lista
// ordenada de trabajo **propone un TODO**, y quien la detecta es el flujo, no
// la cola. La propuesta no crea nada: viaja como el evento `propuesta_todo`
// ([[backend/04-infrastructure/EVENTS]]) y el TODO no existe hasta que el
// usuario confirma o escribe `/ejecutar`.
//
// El detector reconoce una enumeración explícita y no interpreta nada más:
//
//   - una lista en líneas propias, numerada ("1. …") o con viñetas ("- …"),
//     de dos elementos o más;
//   - una secuencia con conectores ("primero … luego …", "después …");
//   - ordinales de trabajo repetidos ("tarea 1, tarea 2, tarea 3").
//
// Y reconoce la mención "capa por capa" (o "por capas"), que propone un paso
// por capa del proyecto. Si no hay enumeración, no hay propuesta: la petición
// se responde en el chat como cualquier otra.

import (
	"regexp"
	"strconv"
	"strings"

	"localcli/internal/tui"
)

// CapasPorDefecto son las capas del proyecto, en el orden de construcción que
// fija la documentación. Es lo que propone "documentar capa por capa" cuando la
// petición no nombra capas.
var CapasPorDefecto = []string{"backend", "frontend", "database"}

// Propuesta es el TODO que propone una petición con lista ordenada: la petición
// que la disparó y los pasos, en orden.
type Propuesta struct {
	Peticion string
	Pasos    []string
}

var (
	// Un elemento de lista en su propia línea: "- algo", "1. algo", "2) algo".
	reItemLista = regexp.MustCompile(`^\s*(?:[-*•]|\d+[.)])\s+(\S.*)$`)
	// Conectores de secuencia, que parten la petición en pasos.
	reConector = regexp.MustCompile(`(?i)\s*(?:,\s*)?\b(?:y\s+)?(?:luego|después|despues|a continuación|mas\s+tarde|más\s+tarde)\b\s*`)
	// Ordinal de trabajo: "tarea 1", "paso 2", "fase 3".
	reOrdinal = regexp.MustCompile(`(?i)\b(tarea|paso|fase)\s+(\d+)\b`)
	// La mención "capa por capa" / "por capas".
	reCapaPorCapa = regexp.MustCompile(`(?i)\bcapa[s]?\s+por\s+capa\b|\bpor\s+capas\b`)
)

// DetectarTrabajoOrdenado decide si una petición implica una lista ordenada de
// trabajo y, si la implica, devuelve sus pasos en orden. `capas` son las capas
// del proyecto para la mención "capa por capa"; si viene vacío se usan
// CapasPorDefecto.
func DetectarTrabajoOrdenado(peticion string, capas []string) (Propuesta, bool) {
	texto := strings.TrimSpace(peticion)
	if texto == "" {
		return Propuesta{}, false
	}
	if pasos, ok := pasosDeLista(texto); ok {
		return Propuesta{Peticion: texto, Pasos: pasos}, true
	}
	if pasos, ok := pasosDeConectores(texto); ok {
		return Propuesta{Peticion: texto, Pasos: pasos}, true
	}
	if pasos, ok := pasosDeOrdinales(texto); ok {
		return Propuesta{Peticion: texto, Pasos: pasos}, true
	}
	if reCapaPorCapa.MatchString(texto) {
		if len(capas) == 0 {
			capas = CapasPorDefecto
		}
		pasos := append([]string(nil), capas...)
		return Propuesta{Peticion: texto, Pasos: pasos}, true
	}
	return Propuesta{}, false
}

// pasosDeLista lee una lista en líneas propias. Hacen falta dos elementos o
// más: uno solo no es una lista ordenada, es una frase.
func pasosDeLista(texto string) ([]string, bool) {
	var pasos []string
	for _, linea := range strings.Split(texto, "\n") {
		if m := reItemLista.FindStringSubmatch(linea); m != nil {
			pasos = append(pasos, strings.TrimSpace(m[1]))
		}
	}
	if len(pasos) < 2 {
		return nil, false
	}
	return pasos, true
}

// pasosDeConectores parte la petición por los conectores de secuencia. "primero
// documentar, luego implementar" son dos pasos; una petición con un solo
// conector no basta para proponer nada.
func pasosDeConectores(texto string) ([]string, bool) {
	if !reConector.MatchString(texto) {
		return nil, false
	}
	crudos := reConector.Split(texto, -1)
	var pasos []string
	for _, c := range crudos {
		c = strings.TrimSpace(strings.Trim(c, ",.;:"))
		c = strings.TrimSpace(strings.TrimPrefix(c, "y "))
		if c != "" {
			pasos = append(pasos, c)
		}
	}
	if len(pasos) < 2 {
		return nil, false
	}
	return pasos, true
}

// pasosDeOrdinales lee una enumeración del tipo "tarea 1, tarea 2, tarea 3":
// hacen falta dos ordinales o más, y cada paso es su ordinal con lo que le
// acompaña —el verbo que abre la petición vale para todos—, de modo que los
// pasos salen con la misma forma.
func pasosDeOrdinales(texto string) ([]string, bool) {
	marcas := reOrdinal.FindAllStringIndex(texto, -1)
	if len(marcas) < 2 {
		return nil, false
	}
	var pasos []string
	for i, m := range marcas {
		fin := len(texto)
		if i+1 < len(marcas) {
			fin = marcas[i+1][0]
		}
		trozo := strings.TrimSpace(strings.Trim(texto[m[0]:fin], ",;.: "))
		trozo = strings.TrimSpace(strings.TrimPrefix(trozo, "y "))
		if trozo != "" {
			pasos = append(pasos, trozo)
		}
	}
	if len(pasos) < 2 {
		return nil, false
	}
	return pasos, true
}

// proponerTodo detecta trabajo ordenado en una petición y, si lo hay, lo anuncia
// como propuesta. No crea la lista de pasos ni arranca la cola: eso ocurre
// cuando el usuario confirma o escribe `/ejecutar` (BUSINESS_RULES §Detección de
// trabajo ordenado). La propuesta es de la sesión que la pidió.
func (o *Orquestador) proponerTodo(sesionID, peticion string) {
	p, ok := DetectarTrabajoOrdenado(peticion, nil)
	if !ok {
		return
	}
	o.emitir(tui.Evento{Sesion: sesionID, Tipo: tui.EventoPropuestaTodo, Datos: map[string]string{
		"peticion": p.Peticion,
		"pasos":    strings.Join(p.Pasos, "\n"),
		"total":    strconv.Itoa(len(p.Pasos)),
	}})
}
