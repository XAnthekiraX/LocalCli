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
	"strings"
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
	Agente     string // nombre del agente que la corre (p. ej. `plan` o `build`)
	Aprobacion bool   // el efecto de la etapa pasa por aprobación
	// Instruccion es lo que la etapa pide al agente. Se une a las Reglas del
	// flujo y viaja con el contexto de la etapa: es cómo un flujo declara sus
	// reglas (por ejemplo, cómo descubrir la documentación) sin tocar el código.
	Instruccion string
}

// Flujo es una secuencia ordenada de etapas con nombre. Puede venir de los
// flujos oficiales (los constructores de este paquete) o de un JSON del
// proyecto (ver flujo.go y catalogo.go).
type Flujo struct {
	Nombre string
	// Comando es el comando explícito que lo arranca, p. ej. "/resolver".
	// Vacío en un flujo que no se arranca por comando.
	Comando string
	// Descripcion dice, en minúscula, para qué es el flujo.
	Descripcion string
	// Peticion es la petición por defecto del flujo: el objetivo que se usa
	// cuando el comando se escribe sin texto detrás.
	Peticion string
	// Reglas son las reglas del flujo (no del agente): se anteponen al contexto
	// de cada etapa. Aquí viven las reglas de descubrimiento de documentación.
	Reglas []string
	Etapas []Etapa
}

// Validar comprueba que el flujo es encadenable: nombre, al menos una etapa y
// cada etapa con identificador y un agente con nombre. Qué agentes existen lo
// sabe el arranque (los carga de `ai/agents/*.json`); aquí solo se exige que la
// etapa declare uno, y el ejecutor rechaza un nombre desconocido al correrla.
func (f Flujo) Validar() error {
	if f.Nombre == "" {
		return fmt.Errorf("flow: el flujo no tiene nombre")
	}
	if f.Comando != "" && !strings.HasPrefix(f.Comando, "/") {
		return fmt.Errorf("flow: el comando del flujo %s no empieza por / (%q)", f.Nombre, f.Comando)
	}
	if len(f.Etapas) == 0 {
		return fmt.Errorf("flow: el flujo %s no tiene etapas", f.Nombre)
	}
	for i, e := range f.Etapas {
		if e.ID == "" {
			return fmt.Errorf("flow: la etapa %d del flujo %s no tiene id", i, f.Nombre)
		}
		if strings.TrimSpace(e.Agente) == "" {
			return fmt.Errorf("flow: la etapa %s no declara agente", e.ID)
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
