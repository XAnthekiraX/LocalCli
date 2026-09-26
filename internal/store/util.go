package store

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// ejecutor es lo mínimo que necesita una sentencia de escritura o lectura
// simple. Lo satisfacen tanto *sql.DB como *sql.Tx, y esa es la razón de que
// exista: una escritura que DATA_FLOW.md exige atómica con otra solo puede
// componerse si ambas aceptan la transacción que las envuelve. Las funciones
// de repositorio son envoltorios sobre una versión que recibe `ejecutor`, de
// modo que el llamador puede elegir si escribe suelto o dentro de una TX.
type ejecutor interface {
	Exec(query string, args ...any) (sql.Result, error)
	QueryRow(query string, args ...any) *sql.Row
}

// newID genera el identificador de fila: UUID v4 en texto (SCHEMA.md §1,
// "Todas las tablas usan id de tipo TEXT con un UUID v4").
func newID() string {
	return uuid.NewString()
}

// nowISO devuelve la hora actual en ISO 8601 UTC, el formato de todas las
// columnas de fecha (SCHEMA.md §1: "created_at y updated_at son TEXT en
// ISO 8601 con zona UTC").
func nowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// isoTime formatea un instante concreto como hace la base (helper para quien
// inserta filas con fechas que ya conoce).
func isoTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

// parseISO lee una fecha tal y como la guarda la base. Devuelve cero si la
// cadena está vacía o no es válido; los campos opcionales lo toleran.
func parseISO(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

// nullStr convierte una cadena vacía en NULL para la base: "Una columna
// opcional se deja NULL. No se usan cadenas vacías como sustituto de 'sin
// valor'" (SCHEMA.md §1).
func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// strNull lee una columna nullable que puede venir nula (resolved_at, reason,
// contenidos de change_history).
func strNull(v any) string {
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

// nullInt convierte un entero negativo en NULL: -1 se usa en este paquete para
// "sin valor" en columnas numéricas opcionales (tokens).
func nullInt(n int) any {
	if n < 0 {
		return nil
	}
	return n
}

// intNull lee una columna entera nullable devolviendo -1 cuando es NULL, de
// modo que el llamador distinga "no lo sé" de "cero" (TABLES.md: NULL significa
// "no lo sé", 0 significa "cero").
func intNull(v any) int {
	switch x := v.(type) {
	case nil:
		return -1
	case int64:
		return int(x)
	case int:
		return x
	}
	return -1
}
