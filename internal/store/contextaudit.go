package store

import "database/sql"

// ContextAudit es una fila de context_audit (TABLES.md §3): la traza de qué
// documentación recibió el modelo en una etapa y qué se descartó, con motivo.
// Hay un registro por documento y por etapa.
type ContextAudit struct {
	ID        string
	SessionID string
	Stage     string
	Document  string // ruta relativa al proyecto
	Decision  string // "incluido" | "descartado"
	Reason    string // obligatorio al descartar, NULL al incluir (CHECK en la base)
	Tokens    int    // -1 = NULL (tamaño no estimado)
	CreatedAt string
}

// Decisiones de auditoría de contexto.
const (
	AuditIncluido   = "incluido"
	AuditDescartado = "descartado"
)

// RegistrarAuditoria inserta la decisión sobre un documento en una etapa. El
// CHECK (decision = 'descartado') = (reason IS NOT NULL) de CONSTRAINTS.md §2
// rechaza en la base un descarte sin motivo o un incluido con él.
func RegistrarAuditoria(db *sql.DB, a *ContextAudit) error {
	return registrarAuditoria(db, a)
}

func registrarAuditoria(e ejecutor, a *ContextAudit) error {
	now := a.CreatedAt
	if now == "" {
		now = nowISO()
	}
	id := a.ID
	if id == "" {
		id = newID()
	}
	_, err := e.Exec(
		`INSERT INTO context_audit (id, session_id, stage, document, decision, reason, tokens, created_at)
 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, a.SessionID, a.Stage, a.Document, a.Decision, nullStr(a.Reason), nullInt(a.Tokens), now,
	)
	if err != nil {
		return traducirError(err)
	}
	a.ID, a.CreatedAt = id, now
	return nil
}

// AuditoriaDeEtapa es la consulta de QUERIES.md §1 ("Auditoría de una etapa
// concreta"): qué documentación recibió el modelo en una etapa, con lo incluido
// y lo descartado y su motivo. Acelera idx_context_audit_session_stage. Es con
// lo que el usuario audita por qué una respuesta fue mala.
func AuditoriaDeEtapa(db *sql.DB, sessionID, stage string) ([]ContextAudit, error) {
	rows, err := db.Query(
		`SELECT id, session_id, stage, document, decision, COALESCE(reason, ''), tokens, created_at
 FROM context_audit
 WHERE session_id = ? AND stage = ?
 ORDER BY decision, document`, sessionID, stage)
	if err != nil {
		return nil, traducirError(err)
	}
	defer rows.Close()
	var out []ContextAudit
	for rows.Next() {
		var (
			a  ContextAudit
			tk any
		)
		err := rows.Scan(&a.ID, &a.SessionID, &a.Stage, &a.Document, &a.Decision, &a.Reason, &tk, &a.CreatedAt)
		if err != nil {
			return nil, traducirError(err)
		}
		a.Tokens = intNull(tk)
		out = append(out, a)
	}
	return out, traducirError(rows.Err())
}
