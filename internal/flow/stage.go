package flow

// stage.go — T-B010-01: el tipo Etapa y la secuencia encadenable.
//
// Fuente de verdad: ai/docs/backend/01-domain/BUSINESS_RULES.md §Motor de etapas
// ("las etapas se ejecutan en orden y cada una arranca cuando la anterior
// terminó; si una etapa falla, el flujo se detiene; un flujo pausado por un
// permiso se retoma desde la misma etapa") y [[specs/SPEC-MOTOR-FLUJOS]].
//
// Una etapa declara quién la corre (agente `plan` o `build`) y si su efecto
// necesita aprobación. Tras cada etapa el motor decide una de tres cosas:
// seguir (avanza a la siguiente), parar (falla o se cancela) o esperar
// aprobación (pausa y se retoma en la misma etapa).

import (
	"fmt"

	"localcli/internal/tools"
)

// Decision es lo que el motor decide al terminar una etapa.
type Decision int

const (
	// Seguir: la etapa terminó bien; se pasa a la siguiente.
	Seguir Decision = iota
	// Parar: la etapa falló o el usuario canceló; el flujo se detiene.
	Parar
	// EsperarAprobacion: hay algo pendiente de decisión; el flujo se pausa y
	// se retoma en la misma etapa.
	EsperarAprobacion
)

func (d Decision) String() string {
	switch d {
	case Seguir:
		return "seguir"
	case Parar:
		return "parar"
	case EsperarAprobacion:
		return "esperar_aprobacion"
	}
	return "desconocida"
}

// Etapa es un paso encadenable de un flujo.
type Etapa struct {
	ID         string
	Nombre     string
	Agente     string // tools.AgentePlan | tools.AgenteBuild
	Aprobacion bool   // el efecto de la etapa pasa por aprobación
}

// Flujo es una secuencia ordenada de etapas con nombre.
type Flujo struct {
	Nombre string
	Etapas []Etapa
}

// Validar comprueba que el flujo es encadenable: nombre, al menos una etapa y
// cada etapa con identificador y un agente del reparto plan/build.
func (f Flujo) Validar() error {
	if f.Nombre == "" {
		return fmt.Errorf("flow: el flujo no tiene nombre")
	}
	if len(f.Etapas) == 0 {
		return fmt.Errorf("flow: el flujo %s no tiene etapas", f.Nombre)
	}
	for i, e := range f.Etapas {
		if e.ID == "" {
			return fmt.Errorf("flow: la etapa %d del flujo %s no tiene id", i, f.Nombre)
		}
		if e.Agente != tools.AgentePlan && e.Agente != tools.AgenteBuild {
			return fmt.Errorf("flow: la etapa %s no tiene un agente válido (%q)", e.ID, e.Agente)
		}
	}
	return nil
}

// EstadoFlujo es el punto en el que está un flujo en ejecución.
type EstadoFlujo int

const (
	EstadoPendiente EstadoFlujo = iota
	EstadoEnCurso
	EstadoPausadoPermiso
	EstadoTerminado
	EstadoDetenido
	EstadoConError
)

func (e EstadoFlujo) String() string {
	switch e {
	case EstadoPendiente:
		return "pendiente"
	case EstadoEnCurso:
		return "en_curso"
	case EstadoPausadoPermiso:
		return "pausado_permiso"
	case EstadoTerminado:
		return "terminado"
	case EstadoDetenido:
		return "detenido"
	case EstadoConError:
		return "con_error"
	}
	return "desconocido"
}

// TransicionValida dice si un flujo puede pasar de un estado a otro. Los
// estados finales (terminado, detenido, con error) no vuelven solos: un flujo
// detenido no se reanuda; un flujo pausado por permiso sí (se retoma desde la
// misma etapa).
func TransicionValida(desde, hasta EstadoFlujo) bool {
	switch desde {
	case EstadoPendiente:
		return hasta == EstadoEnCurso || hasta == EstadoDetenido || hasta == EstadoConError
	case EstadoEnCurso:
		return hasta == EstadoPausadoPermiso || hasta == EstadoTerminado ||
			hasta == EstadoDetenido || hasta == EstadoConError
	case EstadoPausadoPermiso:
		return hasta == EstadoEnCurso || hasta == EstadoDetenido || hasta == EstadoConError
	}
	return false
}

// EstadoTrasDecision aplica la decisión tomada al terminar una etapa y devuelve
// el estado siguiente. Rechaza una transición inválida.
func EstadoTrasDecision(desde EstadoFlujo, d Decision) (EstadoFlujo, error) {
	var hasta EstadoFlujo
	switch d {
	case Seguir:
		hasta = EstadoEnCurso
	case Parar:
		hasta = EstadoDetenido
	case EsperarAprobacion:
		hasta = EstadoPausadoPermiso
	default:
		return desde, fmt.Errorf("flow: decisión desconocida %d", d)
	}
	if !TransicionValida(desde, hasta) {
		return desde, fmt.Errorf("flow: transición inválida %s → %s", desde, hasta)
	}
	return hasta, nil
}
