package context

import (
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
