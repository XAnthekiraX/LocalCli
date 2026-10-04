package store

import (
	"os"
	"path/filepath"
	"testing"
)

// --- T-B0xx: el hilo de procesamiento del chat (migración 004) ---

func TestChatEventoGuardaLeeYCaeEnCascada(t *testing.T) {
	db := abrirBaseTemporal(t)
	s, err := CrearSesion(db, "chat", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := InsertarChatEvento(db, &ChatEvento{
		SessionID: s.ID, Tipo: ChatTipoProceso, Content: "[Sub Proceso] Entender el problema",
	}); err != nil {
		t.Fatalf("registrar proceso: %v", err)
	}
	if err := InsertarChatEvento(db, &ChatEvento{
		SessionID: s.ID, Tipo: ChatTipoHerramienta, Content: "✓ LEER [AGENTS.md] · 93 líneas",
	}); err != nil {
		t.Fatalf("registrar herramienta: %v", err)
	}
	eventos, err := ChatEventosSesion(db, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(eventos) != 2 {
		t.Fatalf("eventos = %d, queremos 2", len(eventos))
	}
	if eventos[0].Tipo != ChatTipoProceso || eventos[1].Tipo != ChatTipoHerramienta {
		t.Errorf("orden de las líneas de procesamiento: %+v", eventos)
	}

	// Borrar la sesión arrastra su hilo de procesamiento (RELATIONSHIPS.md §3).
	if err := BorrarSesion(db, s.ID); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM chat_evento WHERE session_id = ?`, s.ID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("quedaron %d líneas de procesamiento tras borrar la sesión", n)
	}
}

func TestChatEventoSeAislaPorSesion(t *testing.T) {
	db := abrirBaseTemporal(t)
	a, _ := CrearSesion(db, "a", "")
	b, _ := CrearSesion(db, "b", "")
	if err := InsertarChatEvento(db, &ChatEvento{SessionID: a.ID, Tipo: ChatTipoProceso, Content: "de a"}); err != nil {
		t.Fatal(err)
	}
	if err := InsertarChatEvento(db, &ChatEvento{SessionID: b.ID, Tipo: ChatTipoProceso, Content: "de b"}); err != nil {
		t.Fatal(err)
	}
	eventos, err := ChatEventosSesion(db, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(eventos) != 1 || eventos[0].Content != "de a" {
		t.Errorf("el hilo de una sesión no debe mezclarse: %+v", eventos)
	}
}

// La línea de una herramienta se guarda CON su duración; la de un sub-proceso
// no se mide y queda en NULL («no se midió»), no en cero (TABLES.md §3).
func TestChatEventoGuardaLaDuracionDeLaLinea(t *testing.T) {
	db := abrirBaseTemporal(t)
	s, err := CrearSesion(db, "chat", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := InsertarChatEvento(db, &ChatEvento{
		SessionID: s.ID, Tipo: ChatTipoHerramienta,
		Content: "✓ LEER [AGENTS.md] · 93 líneas · 400 ms", DuracionMS: 400,
	}); err != nil {
		t.Fatalf("registrar herramienta: %v", err)
	}
	if err := InsertarChatEvento(db, &ChatEvento{
		SessionID: s.ID, Tipo: ChatTipoProceso, Content: "[Sub Proceso] Entender el problema",
		DuracionMS: -1, // no se midió: la columna queda NULL, no en cero
	}); err != nil {
		t.Fatalf("registrar proceso: %v", err)
	}
	eventos, err := ChatEventosSesion(db, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(eventos) != 2 || eventos[0].DuracionMS != 400 {
		t.Fatalf("la línea de herramienta lleva su duración: %+v", eventos)
	}
	if eventos[1].DuracionMS != -1 {
		t.Errorf("un sub-proceso no se mide: %+v", eventos[1])
	}
	// Y en la base la línea sin medición es NULL de verdad, no un cero.
	var nulos int
	if err := db.QueryRow(`SELECT COUNT(*) FROM chat_evento WHERE duration_ms IS NULL`).Scan(&nulos); err != nil {
		t.Fatal(err)
	}
	if nulos != 1 {
		t.Errorf("filas con duration_ms NULL = %d, queremos 1", nulos)
	}
}

// El procesamiento es parte del hilo que se pinta, pero NUNCA del contexto que
// recibe el modelo: HistorialSesion (de donde sale el historial del chat) solo
// lee messages.
func TestHistorialSesionNoIncluyeElProcesamiento(t *testing.T) {
	db := abrirBaseTemporal(t)
	s, _ := CrearSesion(db, "chat", "")
	if err := InsertarMensaje(db, &Message{SessionID: s.ID, Role: "user", Content: "hola"}); err != nil {
		t.Fatal(err)
	}
	if err := InsertarChatEvento(db, &ChatEvento{SessionID: s.ID, Tipo: ChatTipoHerramienta, Content: "✓ LEER [a] · 1 línea"}); err != nil {
		t.Fatal(err)
	}
	h, err := HistorialSesion(db, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(h) != 1 || h[0].Content != "hola" {
		t.Fatalf("el contexto del modelo no debe incluir procesamiento: %+v", h)
	}
}

// Las migraciones 004 y 005 son aditivas: una base en v3 con conversación y
// razonamiento pasa a v5 sin perder nada, y las columnas nuevas quedan NULL en
// las filas viejas.
func TestMigracion004Y005AditivasConservanLosDatos(t *testing.T) {
	proyecto := proyectoTemporal(t)
	dbPath, err := DBPath(proyecto)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		t.Fatal(err)
	}
	db, err := openPath(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Se fabrica una base en v3: el esquema base y las migraciones 002 y 003, sin
	// la 004.
	for _, ddl := range []string{schemaSQL, todoSQL, flowContextSQL} {
		if _, err := db.Exec(ddl); err != nil {
			t.Fatalf("no se pudo crear el esquema v3: %v", err)
		}
	}
	if _, err := db.Exec("PRAGMA user_version = 3"); err != nil {
		t.Fatal(err)
	}

	// Se inserta con las columnas de la v3 (sin el par motor/modelo de la 006),
	// como una fila anterior a las migraciones.
	s := &Session{ID: newID(), Name: "previa", Status: StatusInactiva}
	if _, err := db.Exec(
		`INSERT INTO sessions (id, name, layer, status, created_at, updated_at)
		 VALUES (?, 'previa', NULL, 'inactiva', ?, ?)`, s.ID, nowISO(), nowISO()); err != nil {
		t.Fatal(err)
	}
	// El mensaje va con sus columnas de la v3 (sin duration_ms).
	msgID := newID()
	if _, err := db.Exec(
		`INSERT INTO messages (id, session_id, role, content, input_tokens, output_tokens, created_at)
		 VALUES (?, ?, 'agent', 'respuesta vieja', 820, 140, ?)`,
		msgID, s.ID, nowISO()); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(
		`INSERT INTO reasoning (id, message_id, content, created_at) VALUES (?, ?, 'razonaba así', ?)`,
		newID(), msgID, nowISO()); err != nil {
		t.Fatal(err)
	}

	if err := migrate(db); err != nil {
		t.Fatalf("migrate v3->v6: %v", err)
	}
	var v int
	if err := db.QueryRow("PRAGMA user_version").Scan(&v); err != nil {
		t.Fatal(err)
	}
	if v != 6 {
		t.Fatalf("user_version = %d, queremos 6", v)
	}

	// La conversación y su razonamiento siguen ahí; la duración vieja es NULL (-1).
	h, err := HistorialSesion(db, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(h) != 1 || h[0].Content != "respuesta vieja" || h[0].Reasoning != "razonaba así" {
		t.Fatalf("la migración perdió datos: %+v", h)
	}
	if h[0].InputTokens != 820 || h[0].OutputTokens != 140 || h[0].DurationMS != -1 {
		t.Errorf("tokens/duración tras migrar: %+v", h[0])
	}

	// Y la tabla nueva ya acepta líneas de procesamiento, con su duración.
	if err := InsertarChatEvento(db, &ChatEvento{SessionID: s.ID, Tipo: ChatTipoProceso, Content: "x"}); err != nil {
		t.Fatalf("tras migrar, chat_evento debe existir: %v", err)
	}
	if err := InsertarChatEvento(db, &ChatEvento{SessionID: s.ID, Tipo: ChatTipoHerramienta, Content: "y", DuracionMS: 400}); err != nil {
		t.Fatalf("tras migrar, chat_evento.duration_ms debe existir: %v", err)
	}
}

func TestHiloSesionFusionaEnOrden(t *testing.T) {
	db := abrirBaseTemporal(t)
	s, _ := CrearSesion(db, "chat", "")
	if err := InsertarMensaje(db, &Message{
		SessionID: s.ID, Role: "user", Content: "pregunta", CreatedAt: "2026-01-15T09:00:00Z",
	}); err != nil {
		t.Fatal(err)
	}
	if err := InsertarChatEvento(db, &ChatEvento{
		SessionID: s.ID, Tipo: ChatTipoHerramienta, Content: "✓ LEER [a] · 1 línea", CreatedAt: "2026-01-15T09:00:01Z",
	}); err != nil {
		t.Fatal(err)
	}
	if err := InsertarMensaje(db, &Message{
		SessionID: s.ID, Role: "agent", Content: "respuesta", DurationMS: 1500, CreatedAt: "2026-01-15T09:00:02Z",
	}); err != nil {
		t.Fatal(err)
	}
	hilo, err := HiloSesion(db, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(hilo) != 3 {
		t.Fatalf("hilo = %d líneas, queremos 3: %+v", len(hilo), hilo)
	}
	wantRoles := []string{"user", "proceso", "agent"}
	for i, w := range wantRoles {
		if hilo[i].Rol != w {
			t.Errorf("hilo[%d].Rol = %q, queremos %q", i, hilo[i].Rol, w)
		}
	}
	if hilo[2].DuracionMS != 1500 {
		t.Errorf("la duración del turno se conserva en el hilo: %+v", hilo[2])
	}
}
