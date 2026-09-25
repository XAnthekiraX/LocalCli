package store

import (
	"database/sql"
	"errors"
	"strings"
	"testing"
)

// --- T-B002-05: repositorio sessions, borrado en cascada ---

func TestCRUDSesionesYCascada(t *testing.T) {
	db := abrirBaseTemporal(t)

	s, err := CrearSesion(db, "backend-pedidos", LayerBackend)
	if err != nil {
		t.Fatalf("CrearSesion: %v", err)
	}
	if s.Status != StatusInactiva {
		t.Errorf("estado inicial = %q, queremos inactiva", s.Status)
	}
	got, err := ObtenerSesion(db, s.ID)
	if err != nil || got.Name != "backend-pedidos" || got.Layer != LayerBackend {
		t.Fatalf("ObtenerSesion: %+v %v", got, err)
	}

	// hijos de toda clase colgados de la sesión
	m := &Message{SessionID: s.ID, Role: "user", Content: "hola", InputTokens: -1, OutputTokens: -1}
	if err := InsertarMensaje(db, m); err != nil {
		t.Fatal(err)
	}
	ma := &Message{SessionID: s.ID, Role: "agent", Content: "adios"}
	if err := InsertarMensaje(db, ma); err != nil {
		t.Fatal(err)
	}
	if err := UpsertRazonamiento(db, ma.ID, "pensé algo"); err != nil {
		t.Fatal(err)
	}
	if _, err := PedirAprobacion(db, s.ID, "escribir x.go"); err != nil {
		t.Fatal(err)
	}
	if err := RegistrarAuditoria(db, &ContextAudit{SessionID: s.ID, Stage: "etapa1", Document: "ai/docs/x.md", Decision: AuditIncluido, Tokens: 10}); err != nil {
		t.Fatal(err)
	}

	// listar por estado
	lista, err := ListarSesiones(db, StatusInactiva)
	if err != nil || len(lista) != 1 {
		t.Fatalf("ListarSesiones(inactiva) = %v, %v", lista, err)
	}

	// transición legal y actualización de estado
	if err := ActualizarEstadoSesion(db, s.ID, StatusTrabajando); err != nil {
		t.Fatalf("transición legal: %v", err)
	}
	// transición ilegal: trabajando -> terminada es legal; probemos inactiva->esperando_permiso desde otra sesión
	s2, _ := CrearSesion(db, "general", "")
	if err := ActualizarEstadoSesion(db, s2.ID, StatusEsperandoPermiso); !errors.Is(err, ErrEstadoIlegal) {
		t.Fatalf("inactiva->esperando_permiso debía ser ilegal, err=%v", err)
	}

	// borrar en cascada: messages, reasoning, approvals y context_audit desaparecen
	if err := BorrarSesion(db, s.ID); err != nil {
		t.Fatalf("BorrarSesion: %v", err)
	}
	for _, q := range []string{
		"SELECT COUNT(*) FROM messages WHERE session_id = '" + s.ID + "'",
		"SELECT COUNT(*) FROM reasoning",
		"SELECT COUNT(*) FROM approvals WHERE session_id = '" + s.ID + "'",
		"SELECT COUNT(*) FROM context_audit WHERE session_id = '" + s.ID + "'",
	} {
		var n int
		if err := db.QueryRow(q).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Errorf("%s = %d, queremos 0 tras el borrado en cascada", q, n)
		}
	}
}

// --- T-B002-06: messages + reasoning ---

func TestRazonamientoLigadoYMaximoUnoPorMensaje(t *testing.T) {
	db := abrirBaseTemporal(t)
	s, _ := CrearSesion(db, "s", "")
	m := &Message{SessionID: s.ID, Role: "agent", Content: "respuesta"}
	if err := InsertarMensaje(db, m); err != nil {
		t.Fatal(err)
	}
	// primer upsert crea; el segundo actualiza la MISMA fila (streaming acumulado)
	if err := UpsertRazonamiento(db, m.ID, "parcial"); err != nil {
		t.Fatal(err)
	}
	if err := UpsertRazonamiento(db, m.ID, "parcial completo"); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM reasoning WHERE message_id = ?", m.ID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("filas de reasoning = %d, queremos máximo 1 por mensaje", n)
	}
	rz, err := RazonamientoDe(db, m.ID)
	if err != nil || rz != "parcial completo" {
		t.Errorf("RazonamientoDe = %q, %v", rz, err)
	}
	// un INSERT directo de un segundo razonamiento choca con el UNIQUE
	_, err = db.Exec("INSERT INTO reasoning (id, message_id, content, created_at) VALUES ('r2', ?, 'otro', 'x')", m.ID)
	if err == nil || !strings.Contains(err.Error(), "UNIQUE") {
		t.Errorf("segundo razonamiento del mismo mensaje debía violar UNIQUE, err=%v", err)
	}
	// historial con LEFT JOIN: cada mensaje conserva SU razonamiento — el de
	// agente sale con contenido y el de usuario con cadena vacía. El orden
	// verificado es el de inserción (created_at + rowid): aquí se insertó
	// primero el mensaje de agente, así que h[0] es el agente.
	mu := &Message{SessionID: s.ID, Role: "user", Content: "pregunta"}
	if err := InsertarMensaje(db, mu); err != nil {
		t.Fatal(err)
	}
	h, err := HistorialSesion(db, s.ID)
	if err != nil || len(h) != 2 {
		t.Fatalf("HistorialSesion = %v, %v", h, err)
	}
	if h[0].Role != "agent" || h[0].Reasoning != "parcial completo" {
		t.Errorf("primer mensaje del historial: esperaba agent con razonamiento, obtuve %+v", h[0])
	}
	if h[1].Role != "user" || h[1].Reasoning != "" {
		t.Errorf("segundo mensaje del historial: esperaba user sin razonamiento, obtuve %+v", h[1])
	}
}

func TestMensajeHijoSinSesionRechazado(t *testing.T) {
	db := abrirBaseTemporal(t)
	err := InsertarMensaje(db, &Message{SessionID: "no-existe", Role: "user", Content: "x"})
	if !errors.Is(err, ErrClaveForanea) {
		t.Fatalf("err = %v, queremos E_BAD_ARGS por FK", err)
	}
}

// --- T-B002-07: approvals ---

func TestAprobacionesPendientesSoloPendientes(t *testing.T) {
	db := abrirBaseTemporal(t)
	s, _ := CrearSesion(db, "s2", LayerFrontend)
	a1, _ := PedirAprobacion(db, s.ID, "operación 1")
	a2, _ := PedirAprobacion(db, s.ID, "operación 2")

	pend, err := AprobacionesPendientes(db)
	if err != nil || len(pend) != 2 {
		t.Fatalf("pendientes = %v, %v", pend, err)
	}
	if pend[0].SessionName != "s2" {
		t.Errorf("falta el nombre de sesión del JOIN")
	}
	n, _ := ContarAprobacionesPendientes(db)
	if n != 2 {
		t.Errorf("contador = %d, queremos 2", n)
	}

	if err := ResolverAprobacion(db, a1.ID, ApprovalAprobada); err != nil {
		t.Fatal(err)
	}
	got, _ := ObtenerAprobacion(db, a1.ID)
	if got.Status != ApprovalAprobada || got.ResolvedAt == "" {
		t.Errorf("resuelta sin resolved_at: %+v", got)
	}
	// resolver dos veces es ilegal (los estados finales no vuelven a pendiente)
	if err := ResolverAprobacion(db, a1.ID, ApprovalDeclinada); err == nil {
		t.Error("resolver una aprobación ya resuelta debía fallar")
	}
	pend, _ = AprobacionesPendientes(db)
	if len(pend) != 1 || pend[0].ID != a2.ID {
		t.Errorf("consulta de pendientes devolvió %+v, solo debe quedar la pendiente", pend)
	}
	// el CHECK resolved_at~status rechaza inconsistencias directamente
	if _, err := db.Exec("UPDATE approvals SET resolved_at = NULL WHERE id = ?", a1.ID); err == nil {
		t.Error("CHECK (status='pendiente')=(resolved_at IS NULL) debía rechazar")
	}
}

// --- T-B002-08: context_audit ---

func TestAuditoriaPorSesionYEtapa(t *testing.T) {
	db := abrirBaseTemporal(t)
	s, _ := CrearSesion(db, "s3", "")
	incl := &ContextAudit{SessionID: s.ID, Stage: "implementar", Document: "ai/docs/A.md", Decision: AuditIncluido, Tokens: 100}
	desc := &ContextAudit{SessionID: s.ID, Stage: "implementar", Document: "ai/docs/B.md", Decision: AuditDescartado, Reason: "no aplica", Tokens: -1}
	if err := RegistrarAuditoria(db, incl); err != nil {
		t.Fatal(err)
	}
	if err := RegistrarAuditoria(db, desc); err != nil {
		t.Fatal(err)
	}
	rows, err := AuditoriaDeEtapa(db, s.ID, "implementar")
	if err != nil || len(rows) != 2 {
		t.Fatalf("AuditoriaDeEtapa = %v, %v", rows, err)
	}
	// ORDER BY decision: descartado antes que incluido
	if rows[0].Decision != AuditDescartado || rows[0].Reason != "no aplica" {
		t.Errorf("descarte sin motivo leído: %+v", rows[0])
	}
	if rows[1].Tokens != 100 || rows[0].Tokens != -1 {
		t.Errorf("tokens: incluidos=%d descartado=%d", rows[1].Tokens, rows[0].Tokens)
	}
	// descarte sin motivo lo rechaza la base
	mal := &ContextAudit{SessionID: s.ID, Stage: "e", Document: "d", Decision: AuditDescartado}
	if err := RegistrarAuditoria(db, mal); !errors.Is(err, ErrRestriccion) {
		t.Errorf("descarte sin reason: err = %v, Queremos ErrRestriccion", err)
	}
}

// --- T-B002-09: change_history sobrevive al borrado ---

func TestChangeHistorySobreviveALaSesion(t *testing.T) {
	db := abrirBaseTemporal(t)
	s, _ := CrearSesion(db, "s4", LayerBackend)
	h := &ChangeHistory{SessionID: s.ID, Operation: OpEditarArchivo, FilePath: "internal/x.go", BeforeContent: "viejo", AfterContent: "nuevo"}
	if err := RegistrarCambio(db, h); err != nil {
		t.Fatal(err)
	}
	if err := BorrarSesion(db, s.ID); err != nil {
		t.Fatal(err)
	}
	hs, err := HistorialDeArchivo(db, "internal/x.go")
	if err != nil || len(hs) != 1 {
		t.Fatalf("HistorialDeArchivo = %v, %v", hs, err)
	}
	if hs[0].SessionID != "" {
		t.Errorf("session_id = %q, quería NULL tras borrar la sesión", hs[0].SessionID)
	}
	if hs[0].BeforeContent != "viejo" || hs[0].AfterContent != "nuevo" {
		t.Errorf("contenido perdido: %+v", hs[0])
	}
	// crear_archivo con before_content viola el CHECK por operación
	if err := RegistrarCambio(db, &ChangeHistory{Operation: OpCrearArchivo, FilePath: "p", BeforeContent: "x", AfterContent: "y"}); !errors.Is(err, ErrRestriccion) {
		t.Errorf("crear con before: err = %v", err)
	}
}

// --- T-B002-10: transacciones con rollback ---

func TestTransaccionRevierteTodoAnteError(t *testing.T) {
	db := abrirBaseTemporal(t)
	s, _ := CrearSesion(db, "s5", "")
	sentinel := errors.New("fallo a mitad")
	err := EjecutarTX(db, func(tx *sql.Tx) error {
		if err := insertarMensaje(tx, &Message{SessionID: s.ID, Role: "user", Content: "dentro de la tx"}); err != nil {
			return err
		}
		if _, err := tx.Exec(`UPDATE sessions SET status = 'trabajando', updated_at = ? WHERE id = ?`, nowISO(), s.ID); err != nil {
			return err
		}
		return sentinel // fallo deliberado tras escribir en dos tablas
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, quería el centinela", err)
	}
	var nm int
	if err := db.QueryRow("SELECT COUNT(*) FROM messages WHERE session_id = ?", s.ID).Scan(&nm); err != nil || nm != 0 {
		t.Errorf("el mensaje sobrevivió al rollback: %d filas (%v)", nm, err)
	}
	st, _ := ObtenerSesion(db, s.ID)
	if st.Status != StatusInactiva {
		t.Errorf("el estado sobrevivió al rollback: %q", st.Status)
	}
}

func TestCerrarTurnoAgenteAtomico(t *testing.T) {
	db := abrirBaseTemporal(t)
	s, _ := CrearSesion(db, "s6", "")
	if err := ActualizarEstadoSesion(db, s.ID, StatusTrabajando); err != nil {
		t.Fatal(err)
	}
	m, err := CerrarTurnoAgente(db, s.ID, "respuesta", 820, 140, "razonaba así", StatusInactiva)
	if err != nil {
		t.Fatalf("CerrarTurnoAgente: %v", err)
	}
	h, err := HistorialSesion(db, s.ID)
	if err != nil || len(h) != 1 || h[0].Reasoning != "razonaba así" {
		t.Fatalf("turno no cerrado como una unidad: %+v %v", h, err)
	}
	if h[0].InputTokens != 820 || h[0].OutputTokens != 140 {
		t.Errorf("tokens: %+v", h[0])
	}
	_ = m
	// transición ilegal dentro del turno: no debe quedar ni el mensaje
	_, err = CerrarTurnoAgente(db, s.ID, "otra", -1, -1, "", StatusTerminada) // inactiva->terminada es ilegal
	if !errors.Is(err, ErrEstadoIlegal) {
		t.Fatalf("err = %v, quería ErrEstadoIlegal", err)
	}
	var n int
	db.QueryRow("SELECT COUNT(*) FROM messages WHERE session_id = ? AND content = 'otra'", s.ID).Scan(&n)
	if n != 0 {
		t.Error("el mensaje quedó pese al rechazo de la transición (rollback ausente)")
	}
}
