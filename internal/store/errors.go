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
var (
	// ErrRestriccion: un CHECK, UNIQUE o NOT NULL rompió la escritura. El
	// catálogo cerrado de valores lo impone la base (CONSTRAINTS.md §3); el
	// código lo reporta como argumento inválido: E_BAD_ARGS.
	ErrRestriccion = errors.New("E_BAD_ARGS: valor que no encaja con las restricciones de la base")

	// ErrClaveForanea: se escribió un hijo sin sesión o sin mensaje padre
	// (CONSTRAINTS.md §3). También es un argumento que no encaja.
	ErrClaveForanea = errors.New("E_BAD_ARGS: la clave foránea no encuentra su fila padre")

	// ErrConflictivo: intento de duplicar una fila con la misma PK o un
	// segundo razonamiento para el mismo mensaje (UNIQUE, CONSTRAINTS.md §1).
	ErrConflictivo = errors.New("E_BAD_ARGS: la fila ya existe (violación de unicidad)")

	// ErrBloqueado: busy_timeout agotado; otra conexión sostiene la escritura.
	// No hay código E_ dedicado en ERRORES.md: es un fallo transitorio que el
	// llamador puede reintentar.
	ErrBloqueado = errors.New("la base está bloqueada por otra escritura; reintente")

	// ErrEstadoIlegal: una transición de estado no permitida por ENUMS.md §3.
	// La legalidad la aplica el código, no un CHECK (CONSTRAINTS.md §2).
	ErrEstadoIlegal = errors.New("transición de estado no permitida")

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
		return fmt.Errorf("error de la base: %w", err)
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
