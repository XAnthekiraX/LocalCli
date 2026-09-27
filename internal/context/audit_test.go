package context

import (
	"errors"
	"testing"

	"localcli/internal/store"
)

// TestAuditarUnaFilaPorDocumento — una fila por documento y etapa: los
// incluidos sin motivo y los descartados con el suyo.
func TestAuditarUnaFilaPorDocumento(t *testing.T) {
	aud := &auditorStub{}
	incluidos := []DocumentoSeleccionado{{Ruta: "a.md", Tokens: 5}}
	descartados := []Descarte{{Ruta: "b.md", Motivo: "no cabe en el límite de contexto"}}
	if err := Auditar(aud, "s1", "etapa", incluidos, descartados); err != nil {
		t.Fatalf("Auditar: %v", err)
	}
	if len(aud.filas) != 2 {
		t.Fatalf("filas = %d, quiero 2", len(aud.filas))
	}
	if aud.filas[0].Decision != store.AuditIncluido || aud.filas[0].Reason != "" || aud.filas[0].Tokens != 5 {
		t.Errorf("fila incluida = %+v", aud.filas[0])
	}
	if aud.filas[1].Decision != store.AuditDescartado || aud.filas[1].Reason == "" {
		t.Errorf("fila descartada = %+v", aud.filas[1])
	}
}

// TestAuditarSinAuditorNoFalla — sin auditor, el contexto se arma igual.
func TestAuditarSinAuditorNoFalla(t *testing.T) {
	if err := Auditar(nil, "s", "e", nil, nil); err != nil {
		t.Fatalf("Auditar(nil): %v", err)
	}
}

// auditorErroneo falla siempre con el error dado.
type auditorErroneo struct{ err error }

func (a *auditorErroneo) Registrar(x *store.ContextAudit) error { return a.err }

// TestAuditarDuplicadoNoTumbaElTurno — la terna (sesión, etapa, documento) ya
// registrada no es error: repetir una etapa —reinteto, segundo turno del
// chat— no debe tumbar el turno. El UNIQUE de la base sigue vivo: lo que se
// tolera aquí es que ya exista la fila, no que entren dos.
func TestAuditarDuplicadoNoTumbaElTurno(t *testing.T) {
	aud := &auditorErroneo{err: store.ErrConflictivo}
	incluidos := []DocumentoSeleccionado{{Ruta: "a.md", Tokens: 5}}
	descartados := []Descarte{{Ruta: "b.md", Motivo: "no cabe en el límite de contexto"}}
	if err := Auditar(aud, "s1", "chat", incluidos, descartados); err != nil {
		t.Fatalf("Auditar con terna ya registrada: %v", err)
	}
}

// TestAuditarOtroErrorSePropaga — solo el duplicado se tolera: cualquier otro
// fallo de la base sigue frenando la etapa con su error.
func TestAuditarOtroErrorSePropaga(t *testing.T) {
	aud := &auditorErroneo{err: errors.New("E_DB_UNAVAILABLE: la base no responde")}
	if err := Auditar(aud, "s1", "chat", []DocumentoSeleccionado{{Ruta: "a.md"}}, nil); err == nil {
		t.Fatal("un error de base distinto del duplicado debe propagarse")
	}
}
