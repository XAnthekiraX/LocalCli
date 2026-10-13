package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// IntervaloRazonamiento es el ritmo con el que se persiste el razonamiento en streaming.
const IntervaloRazonamiento = 200 * time.Millisecond

// GuardarRazonamiento inserta o actualiza el razonamiento de un mensaje.
func GuardarRazonamiento(db *sql.DB, messageID, contenido string) error {
	_, err := db.Exec(
		`INSERT INTO reasoning (id, message_id, content, created_at) VALUES (?, ?, ?, ?)
		 ON CONFLICT(message_id) DO UPDATE SET content = excluded.content`,
		uuid.New().String(), messageID, contenido, time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("store: no se pudo guardar el razonamiento: %w", err)
	}
	return nil
}

// AcumuladorRazonamiento acumula el razonamiento de un mensaje y lo persiste con límite de frecuencia.
type AcumuladorRazonamiento struct {
	guardar    func(contenido string) error
	intervalo  time.Duration
	ahora      func() time.Time
	texto      strings.Builder
	persistido int
	ultimo     time.Time
}

// NuevoAcumuladorRazonamiento crea un acumulador que persiste en la base.
func NuevoAcumuladorRazonamiento(db *sql.DB, messageID string) *AcumuladorRazonamiento {
	a := &AcumuladorRazonamiento{
		guardar:   func(contenido string) error { return GuardarRazonamiento(db, messageID, contenido) },
		intervalo: IntervaloRazonamiento,
		ahora:     time.Now,
	}
	a.ultimo = a.ahora()
	return a
}

// Añadir acumula un fragmento y lo persiste si pasó el intervalo.
func (a *AcumuladorRazonamiento) Añadir(fragmento string) error {
	a.texto.WriteString(fragmento)
	if a.ahora().Sub(a.ultimo) < a.intervalo {
		return nil
	}
	return a.flush()
}

// Cerrar persiste lo que quede sin guardar.
func (a *AcumuladorRazonamiento) Cerrar() error { return a.flush() }

// flush escribe el texto acumulado si cambió desde la última vez.
func (a *AcumuladorRazonamiento) flush() error {
	contenido := a.texto.String()
	if a.guardar == nil || len(contenido) == a.persistido {
		return nil
	}
	if err := a.guardar(contenido); err != nil {
		return err
	}
	a.persistido = len(contenido)
	a.ultimo = a.ahora()
	return nil
}
