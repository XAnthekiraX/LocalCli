package store

import "database/sql"

// Operaciones de archivo registradas (ENUMS.md §2): los seis valores coinciden
// con las seis herramientas de escritura del catálogo.
const (
	OpCrearArchivo    = "crear_archivo"
	OpEscribirArchivo = "escribir_archivo"
	OpEditarArchivo   = "editar_archivo"
	OpEliminarArchivo = "eliminar_archivo"
	OpCrearCarpeta    = "crear_carpeta"
	OpEliminarCarpeta = "eliminar_carpeta"
)

// ChangeHistory es una fila de change_history (TABLES.md §3). Esta tabla nunca
// se borra (QUERIES.md §5); su session_id usa ON DELETE SET NULL para sobrevivir
// a la sesión (RELATIONSHIPS.md §3).
type ChangeHistory struct {
	ID            string
	SessionID     string // "" = NULL (sesión ya eliminada, o cambio sin sesión)
	Operation     string
	FilePath      string // ruta relativa al proyecto
	BeforeContent string // "" = NULL
	AfterContent  string // "" = NULL
	CreatedAt     string
}

// AntesDespuesSegunOperacion codifica en código lo que el CHECK de la base ya
// impone (CONSTRAINTS.md §2), para fallar temprano con mensaje claro en vez de
// con el texto crudo de SQLite:
//
//	crear_*  → before NULL; eliminar_* → after NULL; escribir/editar → ambos.
func contenidoValido(operation string, before, after bool) bool {
	switch operation {
	case OpCrearArchivo, OpCrearCarpeta:
		return !before
	case OpEliminarArchivo, OpEliminarCarpeta:
		return !after
	case OpEscribirArchivo, OpEditarArchivo:
		return before && after
	}
	return false
}

// RegistrarCambio inserta un cambio aplicado a un archivo del proyecto. Solo el
// módulo de herramientas de archivo llama aquí, y solo tras una aprobación
// (QUERIES.md §5); store no decide esa política, solo la sostiene.
// DATA_FLOW.md: la escritura del archivo y la fila de change_history son un
// solo paso logico. El archivo no cabe en una transaccion SQL (lo escribe
// fileops), asi que el store expone dos vias: esta, para el caso simple, y
// RegistrarCambioEnTX, que mete la fila en la transaccion que el llamador
// abre para todo lo demas del paso.
func RegistrarCambio(db *sql.DB, h *ChangeHistory) error {
	return registrarCambio(db, h)
}

func registrarCambio(e ejecutor, h *ChangeHistory) error {
	if !contenidoValido(h.Operation, h.BeforeContent != "", h.AfterContent != "") {
		return ErrRestriccion
	}
	now := h.CreatedAt
	if now == "" {
		now = nowISO()
	}
	id := h.ID
	if id == "" {
		id = newID()
	}
	_, err := e.Exec(
		`INSERT INTO change_history (id, session_id, operation, file_path, before_content, after_content, created_at)
 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, nullStr(h.SessionID), h.Operation, h.FilePath, nullStr(h.BeforeContent), nullStr(h.AfterContent), now,
	)
	if err != nil {
		return traducirError(err)
	}
	h.ID, h.CreatedAt = id, now
	return nil
}

// HistorialDeArchivo es la consulta de QUERIES.md §1 para revertir o auditar un
// archivo concreto. Acelera idx_change_history_file. Deliberadamente NO hace
// JOIN con sessions (§5): si la sesión fue borrada, session_id es NULL y el
// registro sigue devolviéndose.
func HistorialDeArchivo(db *sql.DB, filePath string) ([]ChangeHistory, error) {
	rows, err := db.Query(
		`SELECT id, COALESCE(session_id, ''), operation, file_path,
        COALESCE(before_content, ''), COALESCE(after_content, ''), created_at
 FROM change_history
 WHERE file_path = ?
 ORDER BY created_at`, filePath)
	if err != nil {
		return nil, traducirError(err)
	}
	defer rows.Close()
	var out []ChangeHistory
	for rows.Next() {
		var h ChangeHistory
		err := rows.Scan(&h.ID, &h.SessionID, &h.Operation, &h.FilePath, &h.BeforeContent, &h.AfterContent, &h.CreatedAt)
		if err != nil {
			return nil, traducirError(err)
		}
		out = append(out, h)
	}
	return out, traducirError(rows.Err())
}
