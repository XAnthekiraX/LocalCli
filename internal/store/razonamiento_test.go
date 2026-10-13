package store

import (
	"testing"
	"time"
)

func TestGuardarRazonamientoMantieneUnaFila(t *testing.T) {
	db := nuevaBase(t)
	proyecto := proyectoDePrueba(t, db)
	s, err := CrearSesion(db, proyecto, "s", nil)
	if err != nil {
		t.Fatal(err)
	}
	m, err := CrearMensaje(db, proyecto, s.ID, "agent", "respuesta", nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := GuardarRazonamiento(db, m.ID, "uno"); err != nil {
		t.Fatal(err)
	}
	if err := GuardarRazonamiento(db, m.ID, "uno y dos"); err != nil {
		t.Fatal(err)
	}

	var filas int
	if err := db.QueryRow("SELECT COUNT(*) FROM reasoning WHERE message_id = ?", m.ID).Scan(&filas); err != nil {
		t.Fatal(err)
	}
	if filas != 1 {
		t.Fatalf("el razonamiento debería ser una sola fila, hay %d", filas)
	}

	var contenido string
	if err := db.QueryRow("SELECT content FROM reasoning WHERE message_id = ?", m.ID).Scan(&contenido); err != nil {
		t.Fatal(err)
	}
	if contenido != "uno y dos" {
		t.Fatalf("contenido = %q", contenido)
	}
}

func TestAcumuladorRespetaElIntervalo(t *testing.T) {
	inicio := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ahora := inicio
	var escrituras []string
	a := &AcumuladorRazonamiento{
		guardar:   func(c string) error { escrituras = append(escrituras, c); return nil },
		intervalo: 200 * time.Millisecond,
		ahora:     func() time.Time { return ahora },
		ultimo:    inicio,
	}

	if err := a.Añadir("a"); err != nil {
		t.Fatal(err)
	}
	ahora = inicio.Add(100 * time.Millisecond)
	if err := a.Añadir("b"); err != nil {
		t.Fatal(err)
	}
	ahora = inicio.Add(200 * time.Millisecond)
	if err := a.Añadir("c"); err != nil {
		t.Fatal(err)
	}
	if len(escrituras) != 1 || escrituras[0] != "abc" {
		t.Fatalf("debería escribirse una vez al cumplir el intervalo: %v", escrituras)
	}

	ahora = inicio.Add(250 * time.Millisecond)
	if err := a.Añadir("d"); err != nil {
		t.Fatal(err)
	}
	if err := a.Cerrar(); err != nil {
		t.Fatal(err)
	}
	if len(escrituras) != 2 || escrituras[1] != "abcd" {
		t.Fatalf("cerrar debería persistir lo pendiente: %v", escrituras)
	}
}
