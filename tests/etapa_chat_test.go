package tests

import (
	"testing"

	tcontext "localcli/internal/context"
	"localcli/internal/flow"
)

// TestEtapaChatCoincide — `flow.Motor.Conversar` pide su contexto bajo
// flow.EtapaChat y el nodo de contexto decide su fallback con
// context.EtapaChat. Son dos literales por capa (flow no importa context), y
// si se separan el chat deja de reconocerse de un lado: este test los ata.
func TestEtapaChatCoincide(t *testing.T) {
	if flow.EtapaChat != tcontext.EtapaChat {
		t.Fatalf("flow.EtapaChat = %q pero context.EtapaChat = %q: deben coincidir",
			flow.EtapaChat, tcontext.EtapaChat)
	}
}
