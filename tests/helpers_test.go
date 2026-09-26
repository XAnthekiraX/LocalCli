// helpers_test.go — piezas compartidas de las pruebas de integración (T-B015).
//
// TESTING.md §3: base temporal por prueba creada desde el esquema y destruida
// al terminar; nunca la base de un proyecto real. El proyecto de prueba lleva
// ai/docs/PROJECT.md para que `store.Open` lo reconozca como proyecto.
package tests

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"localcli/internal/fileops"
	"localcli/internal/store"
)

// proyectoTemp crea un proyecto mínimo con su base abierta.
func proyectoTemp(t *testing.T) (string, *sql.DB) {
	t.Helper()
	proyecto := t.TempDir()
	if err := os.MkdirAll(filepath.Join(proyecto, "ai", "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(proyecto, "ai", "docs", "PROJECT.md"), []byte("# prueba\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(proyecto)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return proyecto, db
}

// aprobadorTotal concede todo, con confirmación explícita incluida: es la
// persona que dice que sí a lo que se le propone.
func aprobadorTotal() fileops.Aprobador {
	return fileops.AprobadorFunc(func(ctx context.Context, s fileops.SolicitudAprobacion) (fileops.Decision, error) {
		return fileops.Decision{Aprobada: true, Explicita: true}, nil
	})
}

// negador declina todo: la persona que dice que no.
func negador() fileops.Aprobador {
	return fileops.AprobadorFunc(func(ctx context.Context, s fileops.SolicitudAprobacion) (fileops.Decision, error) {
		return fileops.Decision{Aprobada: false}, nil
	})
}

// historialDe lee las filas de change_history de un archivo.
func historialDe(t *testing.T, db *sql.DB, ruta string) []store.ChangeHistory {
	t.Helper()
	filas, err := store.HistorialDeArchivo(db, ruta)
	if err != nil {
		t.Fatalf("historial de %s: %v", ruta, err)
	}
	return filas
}

// contarCambios cuenta las filas de change_history de un archivo.
func contarCambios(t *testing.T, db *sql.DB, ruta string) int {
	return len(historialDe(t, db, ruta))
}

// errContiene informa si el error menciona un código interno documentado.
// (ERRORS.md §1: el código viaja en el mensaje, y errors.Is sobre los
// centinelas de cada módulo es la comparación fina; aquí basta el texto.)
func errContiene(err error, codigo string) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), codigo)
}
