package agent

// handoff.go — T-B006-07: el relevo `plan`→`build`.
//
// Fuente de verdad: ai/docs/specs/SPEC-AGENTE-BASE.md §El relevo y
// ai/docs/backend/01-domain/BUSINESS_RULES.md §Agentes.
//
// El relevo es explícito y es el mecanismo central de la herramienta:
//
//	plan lee → plan propone → tú apruebas → cambias a build → build aplica
//
// Este tipo solo modela el paso: no lee ni escribe nada, no llama a nadie.
// Nada se aplica si no se propuso y se aprobó antes, y **una aprobación vale
// para el cambio propuesto, no para lo que siga**: por eso, tras aplicar, el
// relevo vuelve al estado de plan y el siguiente cambio necesita su propia
// propuesta y su propia aprobación. No hay interruptor que salte el paso.

import (
	"fmt"

	"localcli/internal/tools"
)

// Fase es el punto del relevo en el que está el trabajo.
type Fase int

const (
	// FasePlan: `plan` investiga. No hay nada aprobado que aplicar.
	FasePlan Fase = iota
	// FaseEsperandoAprobacion: `plan` propuso y el usuario aún no aprobó.
	FaseEsperandoAprobacion
	// FaseAplicando: lo propuesto se aprobó y `build` puede aplicarlo.
	FaseAplicando
)

func (f Fase) String() string {
	switch f {
	case FasePlan:
		return "plan"
	case FaseEsperandoAprobacion:
		return "esperando_aprobacion"
	case FaseAplicando:
		return "aplicando"
	}
	return "desconocida"
}

// Relevo controla el paso de `plan` a `build`. Empieza en plan: nada se puede
// aplicar hasta que alguien proponga y otro apruebe.
type Relevo struct {
	fase Fase
}

// NuevoRelevo crea un relevo en la fase de plan.
func NuevoRelevo() *Relevo { return &Relevo{fase: FasePlan} }

// Fase devuelve el punto actual del relevo.
func (r *Relevo) Fase() Fase { return r.fase }

// Agente devuelve el agente que puede actuar ahora mismo: `plan` mientras no
// haya algo aprobado, `build` solo cuando el cambio está aprobado.
func (r *Relevo) Agente() string {
	if r.fase == FaseAplicando {
		return tools.AgenteBuild
	}
	return tools.AgentePlan
}

// PuedeEscribir informa si `build` está habilitado para aplicar. Es false
// hasta que hay una propuesta aprobada.
func (r *Relevo) PuedeEscribir() bool { return r.fase == FaseAplicando }

// Proponer registra que `plan` ha propuesto un cambio concreto y que ahora
// espera aprobación. Solo tiene sentido cuando el trabajo está en plan: una
// propuesta no se propone dos veces.
func (r *Relevo) Proponer() error {
	if r.fase != FasePlan {
		return fmt.Errorf("relevo: no se puede proponer en la fase %s", r.fase)
	}
	r.fase = FaseEsperandoAprobacion
	return nil
}

// Aprobar es la decisión del usuario sobre lo propuesto. Sin propuesta previa
// no hay nada que aprobar: una aprobación no es un permiso general.
func (r *Relevo) Aprobar() error {
	if r.fase != FaseEsperandoAprobacion {
		return fmt.Errorf("relevo: no hay nada que aprobar en la fase %s", r.fase)
	}
	r.fase = FaseAplicando
	return nil
}

// Aplicar cierra el ciclo de `build` y devuelve el relevo a plan. La
// aprobación cubría solo el cambio propuesto, así que el siguiente cambio
// empieza de cero.
func (r *Relevo) Aplicar() error {
	if r.fase != FaseAplicando {
		return fmt.Errorf("relevo: no se puede aplicar en la fase %s", r.fase)
	}
	r.fase = FasePlan
	return nil
}
