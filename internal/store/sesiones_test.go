package store

import (
	"database/sql"
	"testing"
)

// nuevaBase abre una base temporal aislada para la prueba.
func nuevaBase(t *testing.T) *sql.DB {
	t.Helper()
	db, limpiar, err := CrearDBTemporal()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = limpiar() })
	return db
}

// proyectoDePrueba resuelve el proyecto de una carpeta temporal nueva.
func proyectoDePrueba(t *testing.T, db *sql.DB) string {
	t.Helper()
	id, err := ResolucionProyecto(db, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestResolverProyectoCreaYReutiliza(t *testing.T) {
	db := nuevaBase(t)
	cwd := t.TempDir()

	primero, err := ResolucionProyecto(db, cwd)
	if err != nil {
		t.Fatal(err)
	}
	segundo, err := ResolucionProyecto(db, cwd)
	if err != nil {
		t.Fatal(err)
	}
	if primero != segundo {
		t.Fatalf("la misma carpeta debería reutilizar la fila: %s != %s", primero, segundo)
	}

	otro, err := ResolucionProyecto(db, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if otro == primero {
		t.Fatal("otra carpeta debería ser otro proyecto")
	}
}

func TestListarSesionesFiltraPorProyecto(t *testing.T) {
	db := nuevaBase(t)
	uno := proyectoDePrueba(t, db)
	dos := proyectoDePrueba(t, db)

	if _, err := CrearSesion(db, uno, "uno", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := CrearSesion(db, dos, "dos", nil); err != nil {
		t.Fatal(err)
	}

	sesiones, err := ListarSesiones(db, uno)
	if err != nil {
		t.Fatal(err)
	}
	if len(sesiones) != 1 || sesiones[0].Name != "uno" {
		t.Fatalf("las sesiones deberían ser solo las del proyecto: %+v", sesiones)
	}
}

func TestActualizarEstadoTocaUpdatedAt(t *testing.T) {
	db := nuevaBase(t)
	proyecto := proyectoDePrueba(t, db)
	s, err := CrearSesion(db, proyecto, "s", nil)
	if err != nil {
		t.Fatal(err)
	}

	const antiguo = "2000-01-01T00:00:00Z"
	if _, err := db.Exec("UPDATE sessions SET updated_at = ? WHERE id = ?", antiguo, s.ID); err != nil {
		t.Fatal(err)
	}
	if err := ActualizarEstadoSesion(db, proyecto, s.ID, "trabajando"); err != nil {
		t.Fatal(err)
	}

	nueva, err := ObtenerSesion(db, proyecto, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if nueva.Status != "trabajando" {
		t.Fatalf("estado = %q", nueva.Status)
	}
	if nueva.UpdatedAt == antiguo {
		t.Fatal("actualizar el estado debería tocar updated_at")
	}
}

func TestEliminarSesionDejaChangeHistory(t *testing.T) {
	db := nuevaBase(t)
	proyecto := proyectoDePrueba(t, db)
	s, err := CrearSesion(db, proyecto, "s", nil)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := db.Exec(
		`INSERT INTO change_history (id, session_id, operation, file_path, before_content, after_content, created_at)
		 VALUES ('c1', ?, 'crear_archivo', 'a.txt', NULL, 'x', '2026-01-01T00:00:00Z')`, s.ID,
	); err != nil {
		t.Fatal(err)
	}

	if err := EliminarSesion(db, proyecto, s.ID); err != nil {
		t.Fatal(err)
	}

	var sessionID sql.NullString
	if err := db.QueryRow("SELECT session_id FROM change_history WHERE id = 'c1'").Scan(&sessionID); err != nil {
		t.Fatalf("change_history debería sobrevivir a la sesión: %v", err)
	}
	if sessionID.Valid {
		t.Fatalf("session_id debería quedar NULL, es %q", sessionID.String)
	}
}
