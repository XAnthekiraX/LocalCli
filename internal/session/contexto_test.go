package session

// Tests de la memoria de conversación y su compactación (T-B021): el historial
// se reconstruye para el modelo; si cabe, entero; si no, un resumen de lo
// antiguo seguido del tramo reciente, reutilizando el resumen entre turnos
// ([[specs/SPEC-HISTORIAL-CONVERSACION]]).

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"localcli/internal/flow"
	"localcli/internal/store"
)

type resumidorStub struct {
	mu     sync.Mutex
	veces  int
	ultimo string
}

func (r *resumidorStub) Resumir(ctx context.Context, transcripcion string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.veces++
	r.ultimo = transcripcion
	return "RESUMEN", nil
}

func (r *resumidorStub) llamadas() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.veces
}

// sembrarMensajes añade n mensajes de usuario, cada uno de 40 bytes (≈10 tokens
// con 4 caracteres por token), con ids únicos para poder anclar el resumen.
func sembrarMensajes(alm *almacenMem, sesionID string, n int) {
	for i := 0; i < n; i++ {
		_ = alm.EscribirMensaje(&store.Message{
			ID:        fmt.Sprintf("m%d", i),
			SessionID: sesionID,
			Role:      "user",
			Content:   strings.Repeat(string(rune('a'+i%26)), 40),
		})
	}
}

func TestElHistorialQueCabeSeEnviaEntero(t *testing.T) {
	g, alm := gestorDe(t, nuevoMotor(flow.EstadoTerminado))
	g.Presupuesto = 1000
	ses, err := g.Crear(NombreProvisional, "")
	if err != nil {
		t.Fatalf("Crear: %v", err)
	}
	sembrarMensajes(alm, ses.ID, 3)

	msgs, err := g.historialPara(context.Background(), ses.ID)
	if err != nil {
		t.Fatalf("historialPara: %v", err)
	}
	if len(msgs) != 3 {
		t.Fatalf("mensajes = %d, quiero 3: %+v", len(msgs), msgs)
	}
	for _, m := range msgs {
		if m.Rol != flow.RolUsuario {
			t.Errorf("sin compactar todos son del usuario: %+v", m)
		}
	}
}

func TestElHistorialLargoSeCompactaConResumen(t *testing.T) {
	g, alm := gestorDe(t, nuevoMotor(flow.EstadoTerminado))
	g.Presupuesto = 30         // 10 mensajes de 10 tokens = 100 > 30
	g.PresupuestoRecencia = 20 // caben los 2 últimos
	res := &resumidorStub{}
	g.Resumidor = res
	ses, err := g.Crear(NombreProvisional, "")
	if err != nil {
		t.Fatalf("Crear: %v", err)
	}
	sembrarMensajes(alm, ses.ID, 10)

	msgs, err := g.historialPara(context.Background(), ses.ID)
	if err != nil {
		t.Fatalf("historialPara: %v", err)
	}
	if len(msgs) != 3 {
		t.Fatalf("quiero resumen + 2 recientes, tengo %d: %+v", len(msgs), msgs)
	}
	if msgs[0].Rol != flow.RolSistema || !strings.Contains(msgs[0].Texto, "RESUMEN") {
		t.Errorf("el primer mensaje debe ser el resumen del sistema: %+v", msgs[0])
	}
	if res.llamadas() != 1 {
		t.Errorf("el resumen se pide una vez: %d", res.llamadas())
	}
	if !strings.Contains(res.ultimo, "usuario:") {
		t.Errorf("la transcripción que se resume lleva los turnos: %q", res.ultimo)
	}
}

func TestLaCompactacionReutilizaElResumenEntreTurnos(t *testing.T) {
	g, alm := gestorDe(t, nuevoMotor(flow.EstadoTerminado))
	g.Presupuesto = 30
	g.PresupuestoRecencia = 20
	res := &resumidorStub{}
	g.Resumidor = res
	ses, _ := g.Crear(NombreProvisional, "")
	sembrarMensajes(alm, ses.ID, 10)

	if _, err := g.historialPara(context.Background(), ses.ID); err != nil {
		t.Fatalf("historialPara: %v", err)
	}
	if _, err := g.historialPara(context.Background(), ses.ID); err != nil {
		t.Fatalf("historialPara: %v", err)
	}
	if n := res.llamadas(); n != 1 {
		t.Errorf("el resumen se reutiliza mientras el tramo reciente cabe: %d llamadas", n)
	}
}

// Cuando el tramo reciente desborda el presupuesto, el resumen cacheado se
// extiende con los mensajes que dejan de ser recientes (sin volver a resumir lo
// ya resumido desde cero).
func TestLaCompactacionSeExtiendeCuandoElTramoRecienteDesborda(t *testing.T) {
	g, alm := gestorDe(t, nuevoMotor(flow.EstadoTerminado))
	g.Presupuesto = 30
	g.PresupuestoRecencia = 10
	res := &resumidorStub{}
	g.Resumidor = res
	ses, _ := g.Crear(NombreProvisional, "")
	sembrarMensajes(alm, ses.ID, 10)

	if _, err := g.historialPara(context.Background(), ses.ID); err != nil {
		t.Fatalf("historialPara: %v", err)
	}
	if n := res.llamadas(); n != 1 {
		t.Fatalf("primer resumen: %d llamadas", n)
	}

	// La conversación crece: el tramo reciente ya no cabe y el resumen se
	// amplía con los mensajes que salen de la ventana reciente.
	for i := 10; i < 13; i++ {
		_ = alm.EscribirMensaje(&store.Message{
			ID: fmt.Sprintf("m%d", i), SessionID: ses.ID, Role: "user",
			Content: strings.Repeat("z", 40),
		})
	}
	msgs, err := g.historialPara(context.Background(), ses.ID)
	if err != nil {
		t.Fatalf("historialPara: %v", err)
	}
	if n := res.llamadas(); n != 2 {
		t.Errorf("el resumen se extiende una segunda vez: %d llamadas", n)
	}
	if len(msgs) == 0 || msgs[0].Rol != flow.RolSistema {
		t.Errorf("sigue habiendo resumen al frente: %+v", msgs)
	}
}

func TestSinResumidorSeEntregaSoloElTramoReciente(t *testing.T) {
	g, alm := gestorDe(t, nuevoMotor(flow.EstadoTerminado))
	g.Presupuesto = 30
	g.PresupuestoRecencia = 20
	// g.Resumidor queda nil.
	ses, _ := g.Crear(NombreProvisional, "")
	sembrarMensajes(alm, ses.ID, 10)

	msgs, err := g.historialPara(context.Background(), ses.ID)
	if err != nil {
		t.Fatalf("historialPara: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("sin resumidor quedan solo los recientes: %+v", msgs)
	}
	for _, m := range msgs {
		if m.Rol == flow.RolSistema {
			t.Errorf("sin resumidor no hay mensaje de sistema: %+v", m)
		}
	}
}

// El rol `agent` de la base es el `assistant` de la conversación.
func TestElRolAgentSeTraduceAAsistente(t *testing.T) {
	filas := []store.MensajeConRazonamiento{
		{Message: store.Message{Role: "user", Content: "hola"}},
		{Message: store.Message{Role: "agent", Content: "qué tal"}},
	}
	msgs := mensajesDeFlow(filas)
	if len(msgs) != 2 || msgs[0].Rol != flow.RolUsuario || msgs[1].Rol != flow.RolAsistente {
		t.Fatalf("roles traducidos = %+v", msgs)
	}
}
