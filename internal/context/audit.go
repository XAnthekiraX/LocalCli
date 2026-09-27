package context

// audit.go — T-B011-05: registrar en context_audit qué entró y qué salió.
//
// Fuente de verdad: [[specs/SPEC-NODO-CONTEXTO]] ("Todo lo que se descartó
// queda registrado con el motivo") y ai/docs/database/01-schema/TABLES.md (una
// fila por documento y etapa).
//
// El nodo no importa `database/sql` (invariante TestStoreEsElUnicoEscritorDeSQLite):
// escribe a través de una interfaz, que implementa `store.Auditoria`.

import (
	"errors"

	"localcli/internal/store"
)

// Auditor registra una fila de auditoría. Lo implementa `store.Auditoria`.
type Auditor interface {
	Registrar(a *store.ContextAudit) error
}

// Auditar escribe una fila por documento incluido (sin motivo) y una por
// documento descartado (con su motivo). Un auditor nil no es un error: el
// contexto se arma igual y simplemente no queda traza. Tampoco lo es la terna
// (sesión, etapa, documento) ya registrada: repetir una etapa —reintento,
// segundo turno del chat— no debe tumbar el turno, y el UNIQUE de la base
// sigue vivo (idx_context_audit_unico: una sola auditoría por terna).
func Auditar(a Auditor, sessionID, etapa string, incluidos []DocumentoSeleccionado, descartados []Descarte) error {
	if a == nil {
		return nil
	}
	for _, d := range incluidos {
		err := a.Registrar(&store.ContextAudit{
			SessionID: sessionID,
			Stage:     etapa,
			Document:  d.Ruta,
			Decision:  store.AuditIncluido,
			Tokens:    d.Tokens,
		})
		if err != nil && !esDuplicado(err) {
			return err
		}
	}
	for _, d := range descartados {
		err := a.Registrar(&store.ContextAudit{
			SessionID: sessionID,
			Stage:     etapa,
			Document:  d.Ruta,
			Decision:  store.AuditDescartado,
			Reason:    d.Motivo,
			Tokens:    -1, // -1 = NULL: no se estimó
		})
		if err != nil && !esDuplicado(err) {
			return err
		}
	}
	return nil
}

// esDuplicado reconoce la violación de unicidad de la terna: la fila ya
// existe, la etapa quedó registrada y no hay nada que corregir.
func esDuplicado(err error) bool { return errors.Is(err, store.ErrConflictivo) }
