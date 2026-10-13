package context

import (
	"errors"
	"fmt"
)

// Decision es el desenlace de un documento en la auditoría.
type Decision string

const (
	Incluido   Decision = "incluido"
	Descartado Decision = "descartado"
)

// Auditoria registra por terna (sesión, etapa, documento). Repetir una etapa
// conserva la fila ya registrada: la implementación usa inserción idempotente.
type Auditoria interface {
	Registrar(sesion, etapa, documento string, decision Decision, motivo string, tokens int) error
}

func (n *Nodo) registrar(p Peticion, documento string, decision Decision, motivo string, tokens int) {
	if n.Auditoria == nil {
		return
	}
	_ = n.Auditoria.Registrar(p.Sesion, p.Etapa, documento, decision, motivo, tokens)
}

// emitir publica `contexto_auditado` al terminar de armar el contexto.
func (n *Nodo) emitir(p Peticion) {
	if n.Evento != nil {
		n.Evento(p.Sesion, p.Etapa)
	}
}

func (n *Nodo) avisar(motivo string) {
	if n.Aviso != nil {
		n.Aviso(motivo)
	}
}

func errorf(format string, args ...any) error {
	return errors.New("context: " + fmt.Sprintf(format, args...))
}
