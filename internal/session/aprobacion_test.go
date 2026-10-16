package session

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"localcli/internal/agent"
	"localcli/internal/fileops"
	"localcli/internal/llm/openai"
	"localcli/internal/store"
	"localcli/internal/tools"
	"localcli/internal/tui"
)

// agentesConBuild deja un agente `plan` de solo lectura y un `build` que puede
// leer, escribir y editar.
func agentesConBuild(t *testing.T) *agent.Catalogo {
	t.Helper()
	base := t.TempDir()
	escribirAgente(t, base, "plan", "  default: deny\n  read: allow\n")
	escribirAgente(t, base, "build", "  default: deny\n  read: allow\n  write: allow\n  edit: allow\n")
	catalogo, avisos := agent.CargarAditivo(base, "")
	if len(avisos) != 0 {
		t.Fatalf("avisos de agentes: %v", avisos)
	}
	return catalogo
}

func escribirAgente(t *testing.T, base, nombre, permisos string) {
	t.Helper()
	dir := filepath.Join(base, nombre)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	config := "name: " + nombre + "\ndescription: d\npermissions:\n" + permisos
	if err := os.WriteFile(filepath.Join(dir, "agent.yaml"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "prompt.md"), []byte("eres "+nombre), 0o644); err != nil {
		t.Fatal(err)
	}
}

// servidorTool pide la herramienta `nombre` con `args` en el primer turno y
// responde en texto tras recibir el resultado. Las peticiones de título reciben
// un título.
func servidorTool(t *testing.T, nombre string, args map[string]string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cuerpo, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		emite := func(v any) {
			b, _ := json.Marshal(v)
			fmt.Fprintf(w, "data: %s\n\n", b)
			if flusher != nil {
				flusher.Flush()
			}
		}
		switch {
		case bytes.Contains(cuerpo, []byte(marcaTitulo)):
			emite(map[string]any{"choices": []any{map[string]any{"delta": map[string]any{"content": "Titulo"}}}})
		case bytes.Contains(cuerpo, []byte(`"role":"tool"`)):
			emite(map[string]any{"choices": []any{map[string]any{"delta": map[string]any{"content": "listo"}}}})
		default:
			argumentos, _ := json.Marshal(args)
			emite(map[string]any{"choices": []any{map[string]any{"delta": map[string]any{"tool_calls": []any{
				map[string]any{"index": 0, "id": "call_1", "type": "function",
					"function": map[string]any{"name": nombre, "arguments": string(argumentos)}},
			}}}}})
		}
		emite(map[string]any{"choices": []any{}, "usage": map[string]any{"prompt_tokens": 4, "completion_tokens": 2}})
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(srv.Close)
	return srv
}

// catalogoDelRegistro adapta las definiciones del registro a la vista del agente,
// como hace el cableado de arranque.
func catalogoDelRegistro(registro *tools.Registro) []agent.Herramienta {
	defs := registro.Definiciones()
	salida := make([]agent.Herramienta, 0, len(defs))
	for _, d := range defs {
		salida = append(salida, agent.Herramienta{
			Nombre: d.Nombre, Descripcion: d.Descripcion, Permiso: string(d.Permiso), Esquema: d.Esquema,
		})
	}
	return salida
}

// TestElTurnoPideAprobacionYElBorradoSeAplica: lo destructivo pausa el turno
// hasta que el usuario decide; aprobado, el archivo queda borrado.
func TestElTurnoPideAprobacionYElBorradoSeAplica(t *testing.T) {
	srv := servidorTool(t, "eliminar_archivo", map[string]string{"ruta": "nota.txt"})
	db, limpiar, _ := store.CrearDBTemporal()
	defer limpiar()
	cwd := t.TempDir()
	projectID, _ := store.ResolucionProyecto(db, cwd)
	nota := filepath.Join(cwd, "nota.txt")
	if err := os.WriteFile(nota, []byte("hola"), 0o644); err != nil {
		t.Fatal(err)
	}

	o := Nuevo(db, projectID, cwd, agentesConBuild(t), openai.NuevoCliente(srv.URL), "m", 8192, nil, nil)
	registro := tools.NuevoRegistro(tools.Catalogo())
	ops := &fileops.Operaciones{Raiz: cwd, DB: db, ProjectID: projectID}
	registro.Instalar("eliminar_archivo", ops.EscribirHandlers()["eliminar_archivo"])
	o.ConHerramientas(registro, catalogoDelRegistro(registro))

	canal, cerrar := o.Suscribir()
	defer cerrar()
	sesion, _ := o.Crear()

	resultado := make(chan error, 1)
	go func() { resultado <- o.Enviar(context.Background(), sesion.ID, "build", "borra nota.txt") }()

	id := esperarPeticionAprobacion(t, canal)
	if estadoDeSesion(t, db, projectID, sesion.ID) != EstadoEsperandoPermiso {
		t.Fatalf("la sesión debería quedar esperando permiso")
	}
	if err := o.ResponderAprobacion(sesion.ID, id, true); err != nil {
		t.Fatalf("responder: %v", err)
	}

	select {
	case err := <-resultado:
		if err != nil {
			t.Fatalf("turno: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("el turno no terminó tras aprobar")
	}

	if _, err := os.Stat(nota); !os.IsNotExist(err) {
		t.Fatalf("el archivo no se borró: %v", err)
	}
}

// TestDeclinarLaAprobacionImpideElBorrado: declinada, el archivo sigue ahí y el
// turno sigue su curso (el error vuelve al modelo como resultado corregible).
func TestDeclinarLaAprobacionImpideElBorrado(t *testing.T) {
	srv := servidorTool(t, "eliminar_archivo", map[string]string{"ruta": "nota.txt"})
	db, limpiar, _ := store.CrearDBTemporal()
	defer limpiar()
	cwd := t.TempDir()
	projectID, _ := store.ResolucionProyecto(db, cwd)
	nota := filepath.Join(cwd, "nota.txt")
	if err := os.WriteFile(nota, []byte("hola"), 0o644); err != nil {
		t.Fatal(err)
	}

	o := Nuevo(db, projectID, cwd, agentesConBuild(t), openai.NuevoCliente(srv.URL), "m", 8192, nil, nil)
	registro := tools.NuevoRegistro(tools.Catalogo())
	ops := &fileops.Operaciones{Raiz: cwd, DB: db, ProjectID: projectID}
	registro.Instalar("eliminar_archivo", ops.EscribirHandlers()["eliminar_archivo"])
	o.ConHerramientas(registro, catalogoDelRegistro(registro))

	canal, cerrar := o.Suscribir()
	defer cerrar()
	sesion, _ := o.Crear()

	resultado := make(chan error, 1)
	go func() { resultado <- o.Enviar(context.Background(), sesion.ID, "build", "borra nota.txt") }()

	id := esperarPeticionAprobacion(t, canal)
	if err := o.ResponderAprobacion(sesion.ID, id, false); err != nil {
		t.Fatalf("responder: %v", err)
	}

	select {
	case <-resultado:
	case <-time.After(3 * time.Second):
		t.Fatal("el turno no terminó tras declinar")
	}

	if _, err := os.Stat(nota); err != nil {
		t.Fatalf("declinada, el archivo no debería haberse borrado: %v", err)
	}
}

// TestUnaEscrituraDentroNoPideAprobacion: crear dentro de la carpeta corre sin
// ninguna pausa; no se emite ninguna petición de aprobación.
func TestUnaEscrituraDentroNoPideAprobacion(t *testing.T) {
	srv := servidorTool(t, "escribir_archivo", map[string]string{"ruta": "nota.txt", "contenido": "hola"})
	db, limpiar, _ := store.CrearDBTemporal()
	defer limpiar()
	cwd := t.TempDir()
	projectID, _ := store.ResolucionProyecto(db, cwd)

	o := Nuevo(db, projectID, cwd, agentesConBuild(t), openai.NuevoCliente(srv.URL), "m", 8192, nil, nil)
	registro := tools.NuevoRegistro(tools.Catalogo())
	ops := &fileops.Operaciones{Raiz: cwd, DB: db, ProjectID: projectID}
	registro.Instalar("escribir_archivo", ops.EscribirHandlers()["escribir_archivo"])
	o.ConHerramientas(registro, catalogoDelRegistro(registro))

	canal, cerrar := o.Suscribir()
	defer cerrar()
	sesion, _ := o.Crear()

	resultado := make(chan error, 1)
	go func() { resultado <- o.Enviar(context.Background(), sesion.ID, "build", "crea nota.txt") }()

	vistos, err := eventosHastaElFinal(t, canal, resultado)
	if err != nil {
		t.Fatalf("turno: %v", err)
	}
	for _, e := range vistos {
		if e.Tipo == tui.EventoPeticionAprobacion {
			t.Fatal("una escritura dentro de la carpeta no debería pedir aprobación")
		}
	}
	datos, err := os.ReadFile(filepath.Join(cwd, "nota.txt"))
	if err != nil || string(datos) != "hola" {
		t.Fatalf("el archivo no se escribió: %q (%v)", datos, err)
	}
}

// eventosHastaElFinal devuelve los eventos vistos mientras el turno no termina.
func eventosHastaElFinal(t *testing.T, canal <-chan tui.Evento, resultado chan error) ([]tui.Evento, error) {
	t.Helper()
	var vistos []tui.Evento
	for {
		select {
		case e := <-canal:
			vistos = append(vistos, e)
		case err := <-resultado:
			for {
				select {
				case e := <-canal:
					vistos = append(vistos, e)
				default:
					return vistos, err
				}
			}
		case <-time.After(5 * time.Second):
			t.Fatal("el turno no terminó")
			return vistos, nil
		}
	}
}

// esperarPeticionAprobacion lee el bus hasta que llega la petición y devuelve su id.
func esperarPeticionAprobacion(t *testing.T, canal <-chan tui.Evento) string {
	t.Helper()
	for {
		select {
		case e := <-canal:
			if e.Tipo == tui.EventoPeticionAprobacion {
				if id := e.Datos["aprobacion"]; id != "" {
					return id
				}
			}
		case <-time.After(3 * time.Second):
			t.Fatal("no llegó la petición de aprobación")
			return ""
		}
	}
}

func estadoDeSesion(t *testing.T, db *sql.DB, projectID, sesionID string) string {
	t.Helper()
	s, err := store.ObtenerSesion(db, projectID, sesionID)
	if err != nil {
		t.Fatal(err)
	}
	return s.Status
}
