package session

import (
	"context"
	"testing"

	"localcli/internal/llm/openai"
	"localcli/internal/store"
)

// crearMensajeDePrueba deja un mensaje en la sesión del orquestador.
func crearMensajeDePrueba(t *testing.T, o *Orquestador, sesion, rol, texto string) {
	t.Helper()
	if _, err := store.CrearMensaje(o.db, o.projectID, sesion, rol, texto, nil, nil, nil); err != nil {
		t.Fatal(err)
	}
}

// Deshacer saca el último turno de la conversación y rehacer lo devuelve.
func TestDeshacerYRehacerEnElOrquestador(t *testing.T) {
	o, sesion := orquestadorDeTodo(t)
	crearMensajeDePrueba(t, o, sesion, "user", "uno")
	crearMensajeDePrueba(t, o, sesion, "agent", "resp uno")
	crearMensajeDePrueba(t, o, sesion, "user", "dos")
	crearMensajeDePrueba(t, o, sesion, "agent", "resp dos")

	if n, err := o.Deshacer(sesion); err != nil || n != 2 {
		t.Fatalf("deshacer = (%d, %v), quiero (2, nil)", n, err)
	}
	if visibles, _ := store.ListarMensajes(o.db, o.projectID, sesion); len(visibles) != 2 {
		t.Fatalf("tras deshacer = %d mensajes, quiero 2", len(visibles))
	}
	if n, err := o.Rehacer(sesion); err != nil || n != 2 {
		t.Fatalf("rehacer = (%d, %v), quiero (2, nil)", n, err)
	}
	if visibles, _ := store.ListarMensajes(o.db, o.projectID, sesion); len(visibles) != 4 {
		t.Fatalf("tras rehacer = %d mensajes, quiero 4", len(visibles))
	}
}

// Un turno nuevo invalida el rehacer: lo deshecho ya no encaja detrás de él.
func TestRehacerSePierdeTrasUnTurnoNuevo(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("LOCALCLI_CONTEXT_LIMIT", "100000")
	srv := servidorSSE(t, "respuesta")
	db, limpiar, _ := store.CrearDBTemporal()
	defer limpiar()
	cwd := t.TempDir()
	projectID, _ := store.ResolucionProyecto(db, cwd)
	o := Nuevo(db, projectID, cwd, agentesDePrueba(t), openai.NuevoCliente(srv.URL), "m", 100000, nil, nil)
	o.ConHerramientas(nil, nil)
	sesion, _ := o.Crear()

	if err := o.Enviar(context.Background(), sesion.ID, "plan", "hola"); err != nil {
		t.Fatalf("enviar: %v", err)
	}
	if n, err := o.Deshacer(sesion.ID); err != nil || n == 0 {
		t.Fatalf("debería haber un turno que deshacer: (%d, %v)", n, err)
	}
	if err := o.Enviar(context.Background(), sesion.ID, "plan", "otra"); err != nil {
		t.Fatalf("enviar: %v", err)
	}
	if n, err := o.Rehacer(sesion.ID); err != nil || n != 0 {
		t.Fatalf("rehacer tras un turno nuevo = (%d, %v), quiero (0, nil)", n, err)
	}
}
