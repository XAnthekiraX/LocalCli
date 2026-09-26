// integration_garantia_test.go — T-B015-02: la garantía central.
//
// Fuente de verdad: BUSINESS_RULES.md §Invariantes y TESTING.md §4 ("Nada se
// escribe sin pasar por la aprobación"). La prueba E2E: con el pipeline real
// (tools → fileops → store) y un aprobador doble, SIN aprobación no hay cambio
// en disco, y CON aprobación el cambio queda registrado en change_history con su
// antes y después.
//
// Esto es una prueba de la garantía, no de una función: la petición entra por
// la misma puerta que usaría el modelo (tools.Enrutar, con el permiso que el
// agente declara en su catálogo) y el efecto sale por la misma puerta que
// aplicaría el motor (fileops.Ops). Si alguien conecta un atajo que se salte la
// aprobación, esta prueba deja de pasar.
package tests

import (
	"context"
	"database/sql"
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
	registro, err := tools.NuevoRegistro(tools.Destinos{
		Archivos: func(ctx context.Context, p tools.Peticion) (any, error) {
			return enrutadorArchivos(ops, p)
		},
		Terminal: func(ctx context.Context, p tools.Peticion) (any, error) {
			return nil, nil
		},
		Internet: func(ctx context.Context, p tools.Peticion) (any, error) {
			return nil, nil
		},
	})
	if err != nil {
		t.Fatalf("NuevoRegistro: %v", err)
	}
	return registro, ops, db
}

// enrutadorArchivos es el handler que produce el wiring real: despacha la
// operación concreta al método de fileops que corresponde.
func enrutadorArchivos(ops *fileops.Ops, p tools.Peticion) (any, error) {
	ctx := context.Background()
	switch args := p.Argumentos.(type) {
	case *tools.PeticionCrearArchivo:
		return ops.CrearArchivo(ctx, args.Ruta, args.Contenido)
	case *tools.PeticionEscribirArchivo:
		return ops.EscribirArchivo(ctx, args.Ruta, args.Contenido)
	case *tools.PeticionEditarArchivo:
		return ops.EditarArchivo(ctx, args.Ruta, args.Cambio)
	case *tools.PeticionEliminarArchivo:
		return ops.EliminarArchivo(ctx, args.Ruta)
	case *tools.PeticionLeerArchivo:
		return fileops.LeerArchivo(ops.Proyecto, args.Ruta)
	default:
		return nil, nil
	}
}

// TestSinAprobacionNoHayCambioEnDisco — la garantía: una escritura pedida por
// `build` con el aprobador cerrado no toca el disco ni deja rastro.
func TestSinAprobacionNoHayCambioEnDisco(t *testing.T) {
	registro, ops, db := canalDeHerramientas(t, nil, tools.HerramientasDeBuild())
	peticion := tools.Peticion{
		Agente:      tools.AgenteBuild,
		Permitidas:  tools.HerramientasDeBuild(),
		Herramienta: "crear_archivo",
		Argumentos:  &tools.PeticionCrearArchivo{Ruta: "nuevo.txt", Contenido: "hola"},
	}
	if _, err := registro.Enrutar(context.Background(), peticion); err == nil {
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
	peticion := tools.Peticion{
		Agente:      tools.AgenteBuild,
		Permitidas:  tools.HerramientasDeBuild(),
		Herramienta: "escribir_archivo",
		Argumentos:  &tools.PeticionEscribirArchivo{Ruta: "doc.md", Contenido: "v1"},
	}
	if _, err := registro.Enrutar(context.Background(), peticion); err == nil {
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
	crear := tools.Peticion{
		Agente:      tools.AgenteBuild,
		Permitidas:  tools.HerramientasDeBuild(),
		Herramienta: "crear_archivo",
		Argumentos:  &tools.PeticionCrearArchivo{Ruta: "doc.md", Contenido: "v1"},
	}
	if _, err := registro.Enrutar(context.Background(), crear); err != nil {
		t.Fatalf("crear aprobado: %v", err)
	}
	datos, err := os.ReadFile(filepath.Join(ops.Proyecto, "doc.md"))
	if err != nil || string(datos) != "v1" {
		t.Fatalf("el archivo aprobado no está como se aprobó: %q, %v", datos, err)
	}
	editar := tools.Peticion{
		Agente:      tools.AgenteBuild,
		Permitidas:  tools.HerramientasDeBuild(),
		Herramienta: "editar_archivo",
		Argumentos:  &tools.PeticionEditarArchivo{Ruta: "doc.md", Cambio: "v1" + "\n---\n" + "v2"},
	}
	if _, err := registro.Enrutar(context.Background(), editar); err != nil {
		t.Fatalf("editar aprobado: %v", err)
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
// `plan` no le pasa ni con el aprobador concediendo todo, porque la petición se
// rechaza ANTES de llegar a fileops.
func TestPlanNoPuedeEscribirNiConAprobadorAbierto(t *testing.T) {
	registro, ops, _ := canalDeHerramientas(t, aprobadorTotal(), tools.HerramientasDePlan())
	peticion := tools.Peticion{
		Agente:      tools.AgentePlan,
		Permitidas:  tools.HerramientasDePlan(),
		Herramienta: "crear_archivo",
		Argumentos:  &tools.PeticionCrearArchivo{Ruta: "colado.txt", Contenido: "x"},
	}
	if _, err := registro.Enrutar(context.Background(), peticion); err == nil {
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
