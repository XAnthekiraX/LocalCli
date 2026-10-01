// integration_garantia_test.go — T-B015-02: la garantía central.
//
// Fuente de verdad: BUSINESS_RULES.md §Invariantes y TESTING.md §4 ("Nada se
// escribe sin pasar por la aprobación"). La prueba E2E: con el pipeline real
// (tools → fileops → store) y un aprobador doble, SIN aprobación no hay cambio
// en disco, y CON aprobación el cambio queda registrado en change_history con su
// antes y después.
//
// Esto es una prueba de la garantía, no de una función: la petición entra por
// la misma puerta que usaría el modelo (la capa universal de `tools`, con las
// acciones que el agente declara) y el efecto sale por la misma puerta que
// aplicaría el motor (fileops.Ops). Si alguien conecta un atajo que se salte la
// aprobación, esta prueba deja de pasar.
package tests

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"localcli/internal/fileops"
	"localcli/internal/store"
	"localcli/internal/tools"
)

// canalDeHerramientas arma el registro real de tools con fileops conectado,
// sobre un proyecto temporal con su base. Es el pipeline de producción.
func canalDeHerramientas(t *testing.T, aprobador fileops.Aprobador, _ []string) (*tools.Registro, *fileops.Ops, *sql.DB) {
	t.Helper()
	proyecto, db := proyectoTemp(t)
	ops := &fileops.Ops{
		Proyecto:  proyecto,
		Historial: store.Historial{DB: db},
		Aprobador: aprobador,
	}
	handlers := handlersDeArchivos(ops)
	var impls []tools.Herramienta
	for _, nombre := range tools.NombresCatalogo() {
		h, _ := tools.NuevaHerramienta(nombre, handlers[nombre])
		impls = append(impls, h)
	}
	registro, err := tools.NuevoRegistro(impls)
	if err != nil {
		t.Fatalf("NuevoRegistro: %v", err)
	}
	return registro, ops, db
}

// handlersDeArchivos es el handler propio de cada herramienta de archivo: cada
// uno llama a su método de fileops, como hace el cableado de producción.
func handlersDeArchivos(ops *fileops.Ops) map[string]tools.Ejecutar {
	fallo := func(err error) (tools.Resultado, error) {
		if err == nil {
			return tools.Resultado{}, nil
		}
		return tools.Resultado{Error: err.Error()}, nil
	}
	return map[string]tools.Ejecutar{
		"leer_archivo": func(ctx context.Context, args any, _ tools.Contexto) (tools.Resultado, error) {
			p, ok := args.(*tools.PeticionLeerArchivo)
			if !ok {
				return tools.Resultado{}, fmt.Errorf("argumentos inesperados %T", args)
			}
			resp, err := fileops.LeerArchivo(ops.Proyecto, p.Ruta)
			if err != nil {
				return fallo(err)
			}
			return tools.Resultado{Salida: resp.Contenido}, nil
		},
		"crear_archivo": func(ctx context.Context, args any, _ tools.Contexto) (tools.Resultado, error) {
			p, ok := args.(*tools.PeticionCrearArchivo)
			if !ok {
				return tools.Resultado{}, fmt.Errorf("argumentos inesperados %T", args)
			}
			resp, err := ops.CrearArchivo(ctx, p.Ruta, p.Contenido)
			if err != nil {
				return fallo(err)
			}
			return tools.Resultado{Salida: resp.Confirmacion}, nil
		},
		"escribir_archivo": func(ctx context.Context, args any, _ tools.Contexto) (tools.Resultado, error) {
			p, ok := args.(*tools.PeticionEscribirArchivo)
			if !ok {
				return tools.Resultado{}, fmt.Errorf("argumentos inesperados %T", args)
			}
			resp, err := ops.EscribirArchivo(ctx, p.Ruta, p.Contenido)
			if err != nil {
				return fallo(err)
			}
			return tools.Resultado{Salida: resp.Confirmacion}, nil
		},
		"editar_archivo": func(ctx context.Context, args any, _ tools.Contexto) (tools.Resultado, error) {
			p, ok := args.(*tools.PeticionEditarArchivo)
			if !ok {
				return tools.Resultado{}, fmt.Errorf("argumentos inesperados %T", args)
			}
			resp, err := ops.EditarArchivo(ctx, p.Ruta, p.Cambio)
			if err != nil {
				return fallo(err)
			}
			return tools.Resultado{Salida: resp.Confirmacion}, nil
		},
		"eliminar_archivo": func(ctx context.Context, args any, _ tools.Contexto) (tools.Resultado, error) {
			p, ok := args.(*tools.PeticionEliminarArchivo)
			if !ok {
				return tools.Resultado{}, fmt.Errorf("argumentos inesperados %T", args)
			}
			resp, err := ops.EliminarArchivo(ctx, p.Ruta)
			if err != nil {
				return fallo(err)
			}
			return tools.Resultado{Salida: resp.Confirmacion}, nil
		},
	}
}

// peticion arma la petición de `build` con los argumentos en JSON crudo, como
// llegarían del modelo.
func peticion(herramienta string, args string) tools.Peticion {
	return tools.Peticion{
		Agente:      tools.AgenteBuild,
		Permisos:    tools.PermisosDeBuild(),
		Herramienta: herramienta,
		Argumentos:  json.RawMessage(args),
	}
}

// TestSinAprobacionNoHayCambioEnDisco — la garantía: una escritura pedida por
// `build` con el aprobador cerrado no toca el disco ni deja rastro.
func TestSinAprobacionNoHayCambioEnDisco(t *testing.T) {
	registro, ops, db := canalDeHerramientas(t, nil, tools.HerramientasDeBuild())
	res, err := registro.Ejecutar(context.Background(), peticion("crear_archivo", `{"ruta":"nuevo.txt","contenido":"hola"}`))
	if err != nil {
		t.Fatalf("sin aprobador, un fallo cerrado es un resultado corregible: %v", err)
	}
	if res.Error == "" {
		t.Fatal("sin aprobador la escritura debe rechazarse")
	}
	if _, err := os.Stat(filepath.Join(ops.Proyecto, "nuevo.txt")); err == nil {
		t.Fatal("GARANTÍA ROTA: hay un archivo en disco sin aprobación")
	}
	if n := contarCambios(t, db, "nuevo.txt"); n != 0 {
		t.Fatalf("GARANTÍA ROTA: %d filas de change_history sin aprobación", n)
	}
}

// TestEscrituraDeclinadaNoDejaRastro — declinar tampoco toca nada.
func TestEscrituraDeclinadaNoDejaRastro(t *testing.T) {
	registro, ops, db := canalDeHerramientas(t, negador(), tools.HerramientasDeBuild())
	res, err := registro.Ejecutar(context.Background(), peticion("escribir_archivo", `{"ruta":"doc.md","contenido":"v1"}`))
	if err != nil {
		t.Fatalf("una declaración es un fallo de negocio, no un error duro: %v", err)
	}
	if res.Error == "" {
		t.Fatal("una escritura declinada debe rechazarse")
	}
	if _, err := os.Stat(filepath.Join(ops.Proyecto, "doc.md")); err == nil {
		t.Fatal("GARANTÍA ROTA: hay archivo tras una declinación")
	}
	if n := contarCambios(t, db, "doc.md"); n != 0 {
		t.Fatalf("GARANTÍA ROTA: %d filas tras una declinación", n)
	}
}

// TestConAprobacionElCambioQuedaRegistrado — el camino feliz: aprobado, el
// archivo existe y change_history tiene el antes y el después.
func TestConAprobacionElCambioQuedaRegistrado(t *testing.T) {
	registro, ops, db := canalDeHerramientas(t, aprobadorTotal(), tools.HerramientasDeBuild())
	res, err := registro.Ejecutar(context.Background(), peticion("crear_archivo", `{"ruta":"doc.md","contenido":"v1"}`))
	if err != nil || res.Error != "" {
		t.Fatalf("crear aprobado: %v / %s", err, res.Error)
	}
	datos, err := os.ReadFile(filepath.Join(ops.Proyecto, "doc.md"))
	if err != nil || string(datos) != "v1" {
		t.Fatalf("el archivo aprobado no está como se aprobó: %q, %v", datos, err)
	}
	res, err = registro.Ejecutar(context.Background(), peticion("editar_archivo", `{"ruta":"doc.md","cambio":"v1\n---\nv2"}`))
	if err != nil || res.Error != "" {
		t.Fatalf("editar aprobado: %v / %s", err, res.Error)
	}
	filas := historialDe(t, db, "doc.md")
	if len(filas) != 2 {
		t.Fatalf("historial = %d filas, quiero 2 (crear + editar)", len(filas))
	}
	if filas[1].BeforeContent != "v1" || filas[1].AfterContent != "v2" {
		t.Errorf("la edición no registró antes/después: %+v", filas[1])
	}
}

// TestPlanNoPuedeEscribirNiConAprobadorAbierto — la garantía por catálogo: a
// `plan` no le pasa ni con el aprobador concediendo todo, porque el permiso se
// comprueba ANTES de llegar a fileops.
func TestPlanNoPuedeEscribirNiConAprobadorAbierto(t *testing.T) {
	registro, ops, _ := canalDeHerramientas(t, aprobadorTotal(), tools.HerramientasDePlan())
	_, err := registro.Ejecutar(context.Background(), tools.Peticion{
		Agente:      tools.AgentePlan,
		Permisos:    tools.PermisosDePlan(),
		Herramienta: "crear_archivo",
		Argumentos:  json.RawMessage(`{"ruta":"colado.txt","contenido":"x"}`),
	})
	if err == nil {
		t.Fatal("GARANTÍA ROTA: plan escribió")
	}
	if _, err := os.Stat(filepath.Join(ops.Proyecto, "colado.txt")); err == nil {
		t.Fatal("GARANTÍA ROTA: hay archivo escrito por plan")
	}
}

// TestNingunaEscrituraSinSuFilaDeHistorial — cada escritura aprobada deja su
// registro; si el registro falla, el archivo se revierte (DATA_FLOW.md).
func TestNingunaEscrituraSinSuFilaDeHistorial(t *testing.T) {
	_, ops, db := canalDeHerramientas(t, aprobadorTotal(), tools.HerramientasDeBuild())
	if _, err := ops.CrearArchivo(context.Background(), "con-huella.txt", "v1"); err != nil {
		t.Fatalf("CrearArchivo: %v", err)
	}
	if n := contarCambios(t, db, "con-huella.txt"); n != 1 {
		t.Fatalf("cada escritura aprobada deja su fila: hay %d", n)
	}
}
