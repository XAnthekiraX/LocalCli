package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// Errores internos del store, con el código de ERRORES.md §3 en el mensaje.
// "Los errores de la base se traducen antes de salir de store, para que el
// resto del motor no sepa si fue un bloqueo, una restricción o una conexión"
// (ERRORES.md §4).
//
// Ninguno reutiliza E_BAD_ARGS: ese código describe el contrato de una
// herramienta, no una restricción de la base. Mezclarlos hacía que tres fallos
// distintos —un CHECK, una clave foránea y una fila duplicada— fueran
// indistinguibles para quien los recibiera (ERRORES.md §3 y §5).
var (
	// ErrRestriccion: un CHECK o un NOT NULL rompió la escritura. El
	// catálogo cerrado de valores lo impone la base (CONSTRAINTS.md §3).
	ErrRestriccion = errors.New("E_DB_CONSTRAINT: valor que no encaja con las restricciones de la base")

	// ErrClaveForanea: se escribió un hijo sin sesión o sin mensaje padre
	// (CONSTRAINTS.md §3).
	ErrClaveForanea = errors.New("E_DB_FOREIGN_KEY: la clave foránea no encuentra su fila padre")

	// ErrConflictivo: intento de duplicar una fila con la misma PK, un
	// segundo razonamiento para el mismo mensaje o una segunda fila de
	// auditoría para la misma etapa y documento (UNIQUE, CONSTRAINTS.md §1).
	ErrConflictivo = errors.New("E_DB_CONFLICT: la fila ya existe (violación de unicidad)")

	// ErrBloqueado: busy_timeout agotado; otra conexión sostiene la escritura.
	// Es transitorio: el llamador puede reintentar (ERRORES.md §3).
	ErrBloqueado = errors.New("E_DB_LOCKED: la base está bloqueada por otra escritura; reintente")

	// ErrNoDisponible: no se pudo abrir o leer la base. No distingue el motivo
	// porque el usuario solo puede actuar sobre "no hay base utilizable".
	ErrNoDisponible = errors.New("E_DB_UNAVAILABLE: no se pudo usar la base de datos del proyecto")

	// ErrEstadoIlegal: una transición de estado no permitida por ENUMS.md §3.
	// La legalidad la aplica el código, no un CHECK (CONSTRAINTS.md §2).
	ErrEstadoIlegal = errors.New("E_STAGE_FAILED: transición de estado no permitida")

	// ErrNoProyecto es el alias empaquetado de E_NOT_A_PROJECT (ERRORES.md §3),
	// para quien trapée errores del store sin importar db.go.
	ErrNoProyecto = ErrNotAProject
)

// traducirError mapea errores del driver modernc.org/sqlite a los códigos
// internos documentados. Es el único punto donde un error de SQLite sale del
// paquete: todo repositorio pasa sus errores por aquí.
func traducirError(err error) error {
	if err == nil {
		return nil
	}
	// sql.ErrNoRows no es un fallo de la base: es "no existe", y el llamador
	// lo compara directamente.
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNoEncontrado
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "constraint failed"):
		// SQLite agrupa todos los CHECK/UNIQUE/FK/NOT NULL bajo "constraint
		// failed"; el detalle va en el sufijo.
		switch {
		case strings.Contains(msg, "FOREIGN KEY constraint"):
			return fmt.Errorf("%w: %s", ErrClaveForanea, msg)
		case strings.Contains(msg, "UNIQUE constraint"):
			return fmt.Errorf("%w: %s", ErrConflictivo, msg)
		default: // CHECK, NOT NULL
			return fmt.Errorf("%w: %s", ErrRestriccion, msg)
		}
	case strings.Contains(msg, "database is locked"), strings.Contains(msg, "busy"):
		return fmt.Errorf("%w: %s", ErrBloqueado, msg)
	default:
		// Cualquier otro error de infraestructura sube tal cual, envuelto, sin
		// fingir un código que no le corresponde.
		return fmt.Errorf("%w: %v", ErrNoDisponible, err)
	}
}

// filasAfectadas comprueba que una escritura tocó exactamente `want` filas.
// Un UPDATE/DELETE sobre 0 filas significa que la id no existía: se traduce a
// ErrNoEncontrado en vez de devolver éxito en silencio.
func filasAfectadas(res sql.Result, want int64, qué string) error {
	n, err := res.RowsAffected()
	if err != nil {
		return traducirError(err)
	}
	if n != want {
		return fmt.Errorf("%w: %s (%d filas afectadas)", ErrNoEncontrado, qué, n)
	}
	return nil
}
