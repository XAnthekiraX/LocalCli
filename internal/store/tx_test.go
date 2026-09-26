package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// Estos tests cubren lo que DATA_FLOW.md y TESTING.md exigen y que antes no
// estaba comprobado: que las escrituras compuestas sean atómicas, que WAL
// permita leer mientras se escribe, y que el esquema vivo sea el declarado.

// --- DATA_FLOW.md: pedir permiso es un solo paso lógico ---

func TestPedirPermisoInsertaAprobacionYEsperaPermiso(t *testing.T) {
	db := abrirBaseTemporal(t)
	s, err := CrearSesion(db, "s1", LayerBackend)
	if err != nil {
		t.Fatal(err)
	}
	if err := ActualizarEstadoSesion(db, s.ID, StatusTrabajando); err != nil {
		t.Fatal(err)
	}

	a, err := PedirPermiso(db, s.ID, "escribir internal/x.go")
	if err != nil {
		t.Fatalf("PedirPermiso: %v", err)
	}
	if a.Status != ApprovalPendiente {
		t.Errorf("estado de la aprobación = %q, queremos %q", a.Status, ApprovalPendiente)
	}
	got, err := ObtenerAprobacion(db, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.SessionID != s.ID {
		t.Errorf("la aprobación quedó en otra sesión: %s", got.SessionID)
	}
	ses, err := ObtenerSesion(db, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if ses.Status != StatusEsperandoPermiso {
		t.Errorf("estado de la sesión = %q, queremos %q", ses.Status, StatusEsperandoPermiso)
	}
}

// Si la transición de sesión es ilegal, no debe quedar la aprobación: el
// "flujo que puede quedar a medias" es justo lo que la transacción evita.
func TestPedirPermisoRevierteSiLaTransicionDeSesionEsIllegal(t *testing.T) {
	db := abrirBaseTemporal(t)
	s, err := CrearSesion(db, "s1", LayerBackend)
	if err != nil {
		t.Fatal(err)
	}
	// La sesión está inactiva; inactiva → esperando_permiso no es legal.
	_, err = PedirPermiso(db, s.ID, "no debería quedar")
	if !errors.Is(err, ErrEstadoIlegal) {
		t.Fatalf("err = %v, queremos ErrEstadoIlegal", err)
	}
	n, err := ContarAprobacionesPendientes(db)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("quedaron %d aprobaciones; la transacción debió revertirlas", n)
	}
}

// --- DATA_FLOW.md: resolver permiso actualiza ambas filas juntas ---

func TestResolverPermisoResuelveYAbreLaEspera(t *testing.T) {
	db := abrirBaseTemporal(t)
	s, err := CrearSesion(db, "s1", LayerBackend)
	if err != nil {
		t.Fatal(err)
	}
	if err := ActualizarEstadoSesion(db, s.ID, StatusTrabajando); err != nil {
		t.Fatal(err)
	}
	a, err := PedirPermiso(db, s.ID, "borrar x")
	if err != nil {
		t.Fatal(err)
	}
	if err := ResolverPermiso(db, a.ID, ApprovalAprobada, StatusTrabajando); err != nil {
		t.Fatalf("ResolverPermiso: %v", err)
	}
	got, err := ObtenerAprobacion(db, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != ApprovalAprobada {
		t.Errorf("estado = %q, queremos %q", got.Status, ApprovalAprobada)
	}
	if got.ResolvedAt == "" {
		t.Error("resolved_at quedó vacío tras resolver")
	}
	ses, err := ObtenerSesion(db, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if ses.Status != StatusTrabajando {
		t.Errorf("la sesión sigue en %q; no volvió a trabajar", ses.Status)
	}
	// Resolver dos veces no puede colarse: el segundo intento no cambia nada.
	if err := ResolverPermiso(db, a.ID, ApprovalDeclinada, StatusTrabajando); !errors.Is(err, ErrNoEncontrado) {
		t.Errorf("segunda resolución: err = %v, queremos ErrNoEncontrado", err)
	}
}

func TestObsoletarPendientesDeSesion(t *testing.T) {
	db := abrirBaseTemporal(t)
	s1, _ := CrearSesion(db, "s1", LayerBackend)
	s2, _ := CrearSesion(db, "s2", LayerBackend)
	a1, err := PedirAprobacion(db, s1.ID, "a")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := PedirAprobacion(db, s1.ID, "b"); err != nil {
		t.Fatal(err)
	}
	if _, err := PedirAprobacion(db, s2.ID, "c"); err != nil {
		t.Fatal(err)
	}
	n, err := ObsoletarPendientesDeSesion(db, s1.ID)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("obsoletadas = %d, queremos 2", n)
	}
	if _, err := ObtenerAprobacion(db, a1.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := ObtenerAprobacion(db, a1.ID)
	if got.Status != ApprovalObsoleta {
		t.Errorf("estado = %q, queremos %q", got.Status, ApprovalObsoleta)
	}
	// La otra sesión no se toca.
	resto, err := ListarAprobacionesDeSesion(db, s2.ID, ApprovalPendiente)
	if err != nil {
		t.Fatal(err)
	}
	if len(resto) != 1 {
		t.Errorf("la sesión ajena quedó con %d pendientes, queremos 1", len(resto))
	}
}

// --- T-B002-10: RegistrarCambioEnTX enlistа el registro del cambio ---

func TestRegistrarCambioEnTXRevierteSiElPasoAnteriorFalla(t *testing.T) {
	db := abrirBaseTemporal(t)
	s, _ := CrearSesion(db, "s1", LayerBackend)
	h := &ChangeHistory{
		SessionID: s.ID, Operation: OpCrearArchivo, FilePath: "a.go",
		AfterContent: "package a\n",
	}
	fallo := errors.New("la escritura del archivo falló")
	err := RegistrarCambioEnTX(db, h, func(tx *sql.Tx) error { return fallo })
	if !errors.Is(err, fallo) {
		t.Fatalf("err = %v, queremos el error del paso anterior", err)
	}
	hist, err := HistorialDeArchivo(db, "a.go")
	if err != nil {
		t.Fatal(err)
	}
	if len(hist) != 0 {
		t.Errorf("quedó historial pese al fallo: %d filas", len(hist))
	}
}

func TestRegistrarCambioEnTXRegistraCuandoElPasoAnteriorVaBien(t *testing.T) {
	db := abrirBaseTemporal(t)
	s, _ := CrearSesion(db, "s1", LayerBackend)
	h := &ChangeHistory{
		SessionID: s.ID, Operation: OpEditarArchivo, FilePath: "b.go",
		BeforeContent: "v1\n", AfterContent: "v2\n",
	}
	err := RegistrarCambioEnTX(db, h, func(tx *sql.Tx) error {
		_, e := tx.Exec(`UPDATE sessions SET updated_at = updated_at WHERE id = ?`, s.ID)
		return e
	})
	if err != nil {
		t.Fatal(err)
	}
	hist, err := HistorialDeArchivo(db, "b.go")
	if err != nil {
		t.Fatal(err)
	}
	if len(hist) != 1 {
		t.Fatalf("filas = %d, queremos 1", len(hist))
	}
	if hist[0].BeforeContent != "v1\n" || hist[0].AfterContent != "v2\n" {
		t.Errorf("antes/después = %q / %q", hist[0].BeforeContent, hist[0].AfterContent)
	}
}

// --- TESTING.md: WAL permite leer mientras se escribe ---

// Con el pool limitado a una conexión, una transacción abierta retiene el único
// connection y esta lectura se quedaría esperando: WAL no aportaría nada.
func TestWALPermiteLeerDuranteUnaEscrituraAbierta(t *testing.T) {
	db := abrirBaseTemporal(t)
	s, err := CrearSesion(db, "s1", LayerBackend)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(
		`INSERT INTO sessions (id, name, layer, status, created_at, updated_at)
		 VALUES (?, 'dentro', NULL, 'inactiva', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`,
		"sorpresa-"+s.ID); err != nil {
		tx.Rollback()
		t.Fatal(err)
	}

	// La escritura sigue abierta y sin confirmar. Lo que WAL garantiza es que
	// esta lectura NO se bloquee hasta el commit; que no vea la fila todavia
	// sin confirmar es la aislamiento correcto, no un fallo. Sin WAL (o con una
	// sola conexion en el pool) la lectura esperaria al commit.
	hecho := make(chan error, 1)
	go func() {
		var n int
		hecho <- db.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&n)
	}()

	select {
	case err := <-hecho:
		if err != nil {
			tx.Rollback()
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		tx.Rollback()
		t.Fatal("la lectura se bloqueó 3s con una escritura abierta: WAL no está sirviendo para nada")
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
}

// Varias escrituras concurrentes deben poder coexistir thanks a busy_timeout,
// no fallar con la base bloqueada.
func TestEscriturasConcurrentesNoSePierden(t *testing.T) {
	db := abrirBaseTemporal(t)
	base, err := CrearSesion(db, "base", LayerBackend)
	if err != nil {
		t.Fatal(err)
	}
	const n = 8
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = PedirAprobacion(db, base.ID, fmt.Sprintf("p%d", i))
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Errorf("goroutine %d: %v", i, err)
		}
	}
	var total int
	if err := db.QueryRow(`SELECT COUNT(*) FROM approvals WHERE session_id = ?`, base.ID).Scan(&total); err != nil {
		t.Fatal(err)
	}
	if total != n {
		t.Errorf("aprobaciones = %d, queremos %d", total, n)
	}
}

// --- verifyEsquema: el esquema vivo es el declarado ---

// `id` es TEXT PRIMARY KEY; en SQLite eso NO implica NOT NULL. Sin la
// declaración explícita, una fila con id nulo entra sin que nada lo note.
func TestIdNoAdmiteNuloEnNingunaTabla(t *testing.T) {
	db := abrirBaseTemporal(t)
	for _, tabla := range tablasEsperadas {
		notnull, err := idEsNotNull(db, tabla)
		if err != nil {
			t.Fatalf("%s: %v", tabla, err)
		}
		if !notnull {
			t.Errorf("%s.id admite NULL; el esquema no declara NOT NULL", tabla)
		}
		// Y la base debe rechazarlo de verdad, no solo declararlo.
		_, err = db.Exec(`INSERT INTO ` + tabla + ` DEFAULT VALUES`)
		if err == nil {
			t.Errorf("%s aceptó una fila con id NULL", tabla)
		}
	}
}

func TestContextAuditRechazaDuplicadoDeDocumentoYEtapa(t *testing.T) {
	db := abrirBaseTemporal(t)
	s, _ := CrearSesion(db, "s1", LayerBackend)
	a := &ContextAudit{SessionID: s.ID, Stage: "plan", Document: "specs/X.md", Decision: AuditIncluido, Tokens: 10}
	if err := RegistrarAuditoria(db, a); err != nil {
		t.Fatal(err)
	}
	dup := &ContextAudit{SessionID: s.ID, Stage: "plan", Document: "specs/X.md", Decision: AuditIncluido, Tokens: 10}
	err := RegistrarAuditoria(db, dup)
	if !errors.Is(err, ErrConflictivo) {
		t.Fatalf("err = %v, queremos ErrConflictivo (una fila por documento y etapa)", err)
	}
}

// Un descarte sin motivo lo rechaza el CHECK de la base (CONSTRAINTS.md §2).
func TestContextAuditRechazaDescarteSinMotivo(t *testing.T) {
	db := abrirBaseTemporal(t)
	s, _ := CrearSesion(db, "s1", LayerBackend)
	err := RegistrarAuditoria(db, &ContextAudit{
		SessionID: s.ID, Stage: "plan", Document: "specs/Y.md", Decision: AuditDescartado,
	})
	if !errors.Is(err, ErrRestriccion) {
		t.Fatalf("err = %v, queremos ErrRestriccion", err)
	}
}

// La traduccion de errores usa codigos propios de la base, no E_BAD_ARGS: ese
// codigo describe el contrato de una herramienta, no una restriccion de SQLite.
func TestErroresDeBaseUsanCodigoPropio(t *testing.T) {
	db := abrirBaseTemporal(t)
	s, err := CrearSesion(db, "s1", LayerBackend)
	if err != nil {
		t.Fatal(err)
	}

	casos := []struct {
		nombre string
		sql    string
		quiere error
	}{
		{
			"FK huerfana",
			`INSERT INTO messages (id, session_id, role, content, created_at)
			 VALUES ('m1','no-existe','user','x','2026-01-01T00:00:00Z')`,
			ErrClaveForanea,
		},
		{
			"rol fuera de catalogo",
			`INSERT INTO messages (id, session_id, role, content, created_at)
			 VALUES ('m2','` + s.ID + `','inventado','x','2026-01-01T00:00:00Z')`,
			ErrRestriccion,
		},
		{
			"id duplicado",
			`INSERT INTO sessions (id, name, status, created_at, updated_at)
			 VALUES ('dup','x','inactiva','2026-01-01T00:00:00Z','2026-01-01T00:00:00Z')`,
			ErrConflictivo,
		},
	}
	// El caso de unicidad necesita que la fila previa exista: se inserta una vez
	// y el segundo intento es el que debe rebotar.
	if _, err := db.Exec(
		`INSERT INTO sessions (id, name, status, created_at, updated_at)
		 VALUES ('dup','x','inactiva','2026-01-01T00:00:00Z','2026-01-01T00:00:00Z')`); err != nil {
		t.Fatalf("no se pudo crear la fila previa: %v", err)
	}

	for _, c := range casos {
		_, execErr := db.Exec(c.sql)
		if execErr == nil {
			t.Errorf("%s: la base acepto la escritura que deberia rechazar", c.nombre)
			continue
		}
		got := traducirError(execErr)
		if !errors.Is(got, c.quiere) {
			t.Errorf("%s: err = %v, queremos %v", c.nombre, got, c.quiere)
		}
		if strings.Contains(got.Error(), "E_BAD_ARGS") {
			t.Errorf("%s: un fallo de la base no debe llevar E_BAD_ARGS: %v", c.nombre, got)
		}
		if !strings.Contains(got.Error(), "E_DB_") {
			t.Errorf("%s: el error no lleva un codigo de base: %v", c.nombre, got)
		}
	}
}

// Los tres fallos de base deben ser distinguibles entre si: comparten antes un
// mismo E_BAD_ARGS, lo que hacia que fueran el mismo error para el llamador.
func TestFallosDeBaseSonDistinguibles(t *testing.T) {
	vistos := map[string]string{}
	for nombre, err := range map[string]error{
		"restriccion":   ErrRestriccion,
		"clave foranea": ErrClaveForanea,
		"conflicto":     ErrConflictivo,
		"bloqueado":     ErrBloqueado,
	} {
		if err == nil {
			t.Fatalf("%s es nil", nombre)
		}
		if otro, ok := vistos[err.Error()]; ok {
			t.Errorf("%s y %s comparten mensaje: %q", nombre, otro, err)
		}
		vistos[err.Error()] = nombre
	}
}
