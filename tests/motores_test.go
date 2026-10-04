// motores_test.go — T-B037-16: el par motor/modelo como recurso de la sesión,
// sobre la base real y el registro de `llm`.
//
// Fuente de verdad: ai/docs/specs/SPEC-MODELO-MOTOR §Motor y modelo por sesión,
// §Cambiar a mitad de conversación y §Eliminar; ai/docs/specs/SPEC-SESIONES §El
// par motor/modelo por sesión.
package tests

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"localcli/internal/flow"
	"localcli/internal/llm"
	"localcli/internal/session"
	"localcli/internal/store"
)

// motoresDelRegistro adapta el registro de `llm` al contrato `session.Motores`:
// un motor sin adaptador vivo (desactivado o eliminado) no está disponible. Es
// el mismo contrato que inyecta el arranque de producción.
type motoresDelRegistro struct{ registro *llm.Registro }

func (m motoresDelRegistro) MotorDisponible(idMotor string) error {
	if _, ok := m.registro.Adaptador(idMotor); ok {
		return nil
	}
	return errors.New(llm.CodigoMotorNoDisponible + ": el motor " + idMotor + " ya no está disponible")
}

// motorFlujosNulo es un `session.Motor` de prueba: no ejecuta etapas, solo deja
// construir el gestor (los tests miran el par y el historial, no la inferencia).
type motorFlujosNulo struct{}

func (motorFlujosNulo) Conversar(context.Context, string, string, []flow.Mensaje, []string) (flow.Resultado, error) {
	return flow.Resultado{}, nil
}

func (motorFlujosNulo) EjecutarFlujo(context.Context, flow.Flujo, string) (flow.EstadoFlujo, error) {
	return flow.EstadoTerminado, nil
}

func (motorFlujosNulo) ConsumirCola(context.Context, flow.Cola, flow.ResolutorFlujo) error {
	return nil
}

// gestorConRegistro deja un gestor de sesiones sobre la base real y un registro
// de motores con sus dos entradas por defecto.
func gestorConRegistro(t *testing.T) (*session.Gestor, *llm.Registro, store.Sesiones) {
	t.Helper()
	proyecto, db := proyectoTemp(t)
	alm := store.Sesiones{DB: db}
	alcance, err := session.NuevoAlcance(proyecto, alm)
	if err != nil {
		t.Fatalf("NuevoAlcance: %v", err)
	}
	g, err := session.NuevoGestor(alcance, motorFlujosNulo{}, session.NuevoBus())
	if err != nil {
		t.Fatalf("NuevoGestor: %v", err)
	}
	registro := &llm.Registro{}
	// Sin archivo se registran `ollama` y `llamacpp` con sus direcciones por
	// defecto; los constructores los registran los adaptadores al importarse.
	registro.Cargar(filepath.Join(t.TempDir(), "motores.json"))
	g.Motores = motoresDelRegistro{registro: registro}
	return g, registro, alm
}

// TestCambiarDeMotorAMitadDeConversacionMantieneElHilo — cambiar de motor con la
// conversación ya escrita no toca el id, el nombre ni el historial: el par es un
// recurso de la sesión, no su identidad (SPEC-MODELO-MOTOR).
func TestCambiarDeMotorAMitadDeConversacionMantieneElHilo(t *testing.T) {
	g, _, alm := gestorConRegistro(t)

	ses, err := g.CrearConMotor("conversación viva", "", "ollama-local", "qwen3:8b")
	if err != nil {
		t.Fatalf("CrearConMotor: %v", err)
	}
	if err := alm.EscribirMensaje(&store.Message{SessionID: ses.ID, Role: "user", Content: "hola"}); err != nil {
		t.Fatalf("EscribirMensaje: %v", err)
	}

	if err := g.CambiarMotor(ses.ID, "llamacpp-local"); err != nil {
		t.Fatalf("CambiarMotor: %v", err)
	}

	motorID, modelo, err := g.ParMotorModelo(ses.ID)
	if err != nil {
		t.Fatalf("ParMotorModelo: %v", err)
	}
	if motorID != "llamacpp-local" || modelo != "qwen3:8b" {
		t.Errorf("cambiar de motor solo cambia el motor: %q / %q", motorID, modelo)
	}

	actual, err := g.Estado(ses.ID)
	if err != nil {
		t.Fatalf("Estado: %v", err)
	}
	if actual.ID != ses.ID || actual.Nombre != "conversación viva" {
		t.Errorf("el id y el nombre no cambian: %+v", actual)
	}
	hilo, err := g.Historial(ses.ID)
	if err != nil {
		t.Fatalf("Historial: %v", err)
	}
	if len(hilo) != 1 || hilo[0].Content != "hola" {
		t.Errorf("el historial sigue intacto tras cambiar de motor: %+v", hilo)
	}
}

// TestEliminarElMotorNoEliminaLaSesion — eliminar un motor del registro no borra
// ni invalida las sesiones que lo usaban: la sesión conserva su fila y su
// historial, y avisa con E_MOTOR_NO_DISPONIBLE (SPEC-MODELO-MOTOR §Eliminar).
func TestEliminarElMotorNoEliminaLaSesion(t *testing.T) {
	g, registro, alm := gestorConRegistro(t)

	ses, err := g.CrearConMotor("viva", "", "ollama-local", "qwen3:8b")
	if err != nil {
		t.Fatalf("CrearConMotor: %v", err)
	}
	if err := alm.EscribirMensaje(&store.Message{SessionID: ses.ID, Role: "user", Content: "hola"}); err != nil {
		t.Fatalf("EscribirMensaje: %v", err)
	}

	if err := registro.Eliminar("ollama-local"); err != nil {
		t.Fatalf("Eliminar: %v", err)
	}

	// La sesión sigue existiendo y conserva su par y su historial.
	actual, err := g.Estado(ses.ID)
	if err != nil {
		t.Fatalf("la sesión no debe desaparecer con su motor: %v", err)
	}
	if actual.Nombre != "viva" {
		t.Errorf("la fila de la sesión no se invalida: %+v", actual)
	}
	if err := g.MotorDisponible(ses.ID); err == nil || !errContiene(err, llm.CodigoMotorNoDisponible) {
		t.Fatalf("el motor eliminado se avisa con %s, llegó: %v", llm.CodigoMotorNoDisponible, err)
	}
	hilo, err := g.Historial(ses.ID)
	if err != nil {
		t.Fatalf("Historial: %v", err)
	}
	if len(hilo) != 1 || hilo[0].Content != "hola" {
		t.Errorf("el historial queda intacto: %+v", hilo)
	}
}
