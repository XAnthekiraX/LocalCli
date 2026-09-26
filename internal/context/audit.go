package context

// audit.go — T-B011-05: registrar en context_audit qué entró y qué salió.
//
// Fuente de verdad: [[specs/SPEC-NODO-CONTEXTO]] ("Todo lo que se descartó
// queda registrado con el motivo") y ai/docs/database/01-schema/TABLES.md (una
// fila por documento y etapa).
//
// El nodo no importa `database/sql` (invariante TestStoreEsElUnicoEscritorDeSQLite):
// escribe a través de una interfaz, que implementa `store.Auditoria`.

import "localcli/internal/store"

// Auditor registra una fila de auditoría. Lo implementa `store.Auditoria`.
type Auditor interface {
	Registrar(a *store.ContextAudit) error
}

// Auditar escribe una fila por documento incluido (sin motivo) y una por
// documento descartado (con su motivo). Un auditor nil no es un error: el
// contexto se arma igual y simplemente no queda traza.
func Auditar(a Auditor, sessionID, etapa string, incluidos []DocumentoSeleccionado, descartados []Descarte) error {
	if a == nil {
		return nil
	}
	for _, d := range incluidos {
		if err := a.Registrar(&store.ContextAudit{
			SessionID: sessionID,
			Stage:     etapa,
			Document:  d.Ruta,
			Decision:  store.AuditIncluido,
			Tokens:    d.Tokens,
		}); err != nil {
			return err
		}
	}
	for _, d := range descartados {
		if err := a.Registrar(&store.ContextAudit{
			SessionID: sessionID,
			Stage:     etapa,
			Document:  d.Ruta,
			Decision:  store.AuditDescartado,
			Reason:    d.Motivo,
			Tokens:    -1, // -1 = NULL: no se estimó
		}); err != nil {
			return err
		}
	}
	return nil
}
