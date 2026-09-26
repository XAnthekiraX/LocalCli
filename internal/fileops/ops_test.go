package fileops

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"localcli/internal/store"
)

// aprobadorQue responde siempre con la decisión dada.
func aprobadorQue(d Decision) Aprobador {
	return AprobadorFunc(func(ctx context.Context, s SolicitudAprobacion) (Decision, error) {
		return d, nil
	})
}

// proyectoDePrueba crea un proyecto mínimo, escribe un fixture y abre su base.
// Modo Pruebas de CONFIGURATION.md: temporal y aislado.
func proyectoDePrueba(t *testing.T) (string, *sql.DB) {
	t.Helper()
	proyecto := t.TempDir()
	if err := os.MkdirAll(filepath.Join(proyecto, "ai", "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(proyecto, "ai", "docs", "PROJECT.md"), []byte("# p\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(proyecto)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return proyecto, db
}

func opsDePrueba(t *testing.T, aprobador Aprobador) (*Ops, *sql.DB) {
	t.Helper()
	proyecto, db := proyectoDePrueba(t)
	return &Ops{Proyecto: proyecto, Historial: store.Historial{DB: db}, Aprobador: aprobador}, db
}

func aprobado() Aprobador {
	return aprobadorQue(Decision{Aprobada: true, Explicita: true})
}

// TestCrearArchivoAprobadaRegistra — crear con aprobación deja el archivo y su
// fila en change_history con lo anterior (nada) y lo nuevo.
func TestCrearArchivoAprobadaRegistra(t *testing.T) {
	o, db := opsDePrueba(t, aprobado())
	if _, err := o.CrearArchivo(context.Background(), "nota.txt", "hola\n"); err != nil {
		t.Fatalf("CrearArchivo: %v", err)
	}
	datos, err := os.ReadFile(filepath.Join(o.Proyecto, "nota.txt"))
	if err != nil || string(datos) != "hola\n" {
		t.Fatalf("el archivo no quedó como lo aprobado: %q, %v", datos, err)
	}
	hist, err := store.HistorialDeArchivo(db, "nota.txt")
	if err != nil {
		t.Fatalf("HistorialDeArchivo: %v", err)
	}
	if len(hist) != 1 || hist[0].Operation != store.OpCrearArchivo || hist[0].BeforeContent != "" || hist[0].AfterContent != "hola\n" {
		t.Fatalf("historial inesperado: %+v", hist)
	}
}

// TestEscrituraSinAprobadorNoAplica — sin aprobador, el estado por defecto es
// cerrado: ninguna escritura toca el disco.
func TestEscrituraSinAprobadorNoAplica(t *testing.T) {
	o, _ := opsDePrueba(t, nil)
	_, err := o.CrearArchivo(context.Background(), "nota.txt", "x")
	if !errors.Is(err, ErrNecesitaAprobacion) {
		t.Fatalf("err = %v, quiero E_NEEDS_APPROVAL", err)
	}
	if _, statErr := os.Stat(filepath.Join(o.Proyecto, "nota.txt")); statErr == nil {
		t.Error("no se aplica una escritura sin aprobación")
	}
}

// TestEscrituraDeclinadaNoAplica — declinar deja el proyecto intacto.
func TestEscrituraDeclinadaNoAplica(t *testing.T) {
	o, _ := opsDePrueba(t, aprobadorQue(Decision{Aprobada: false}))
	_, err := o.CrearArchivo(context.Background(), "nota.txt", "x")
	if !errors.Is(err, ErrAprobacionDeclinada) {
		t.Fatalf("err = %v, quiero E_APPROVAL_DECLINED", err)
	}
	if _, statErr := os.Stat(filepath.Join(o.Proyecto, "nota.txt")); statErr == nil {
		t.Error("una operación declinada no puede tocar el disco")
	}
}

// TestCrearArchivoExistenteFalla — crear_archivo no sobrescribe.
func TestCrearArchivoExistenteFalla(t *testing.T) {
	o, _ := opsDePrueba(t, aprobado())
	if _, err := o.CrearArchivo(context.Background(), "nota.txt", "uno"); err != nil {
		t.Fatalf("primera creación: %v", err)
	}
	if _, err := o.CrearArchivo(context.Background(), "nota.txt", "dos"); !errors.Is(err, ErrRutaExiste) {
		t.Fatalf("err = %v, quiero E_PATH_EXISTS", err)
	}
}

// TestEscribirRegistraAntesDespues — sobrescribir guarda el antes y el después.
func TestEscribirRegistraAntesDespues(t *testing.T) {
	o, db := opsDePrueba(t, aprobado())
	if _, err := o.CrearArchivo(context.Background(), "nota.txt", "viejo"); err != nil {
		t.Fatal(err)
	}
	if _, err := o.EscribirArchivo(context.Background(), "nota.txt", "nuevo"); err != nil {
		t.Fatalf("EscribirArchivo: %v", err)
	}
	hist, _ := store.HistorialDeArchivo(db, "nota.txt")
	if len(hist) != 2 {
		t.Fatalf("historial = %d filas, quiero 2", len(hist))
	}
	ultimo := hist[len(hist)-1]
	if ultimo.BeforeContent != "viejo" || ultimo.AfterContent != "nuevo" {
		t.Errorf("antes/después = %q/%q, quiero viejo/nuevo", ultimo.BeforeContent, ultimo.AfterContent)
	}
}

// TestEditarReemplazoExacto — el campo `cambio` aplica un reemplazo exacto.
func TestEditarReemplazoExacto(t *testing.T) {
	o, _ := opsDePrueba(t, aprobado())
	if _, err := o.CrearArchivo(context.Background(), "nota.txt", "a\nb\nc\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := o.EditarArchivo(context.Background(), "nota.txt", "b"+SeparadorEdicion+"B"); err != nil {
		t.Fatalf("EditarArchivo: %v", err)
	}
	datos, _ := os.ReadFile(filepath.Join(o.Proyecto, "nota.txt"))
	if string(datos) != "a\nB\nc\n" {
		t.Fatalf("contenido = %q, quiero %q", datos, "a\nB\nc\n")
	}
}

// TestEditarTextoAmbiguoFalla — si el texto a buscar no aparece exactamente una
// vez, la edición no se aplica.
func TestEditarTextoAmbiguoFalla(t *testing.T) {
	o, _ := opsDePrueba(t, aprobado())
	if _, err := o.CrearArchivo(context.Background(), "nota.txt", "b b b"); err != nil {
		t.Fatal(err)
	}
	if _, err := o.EditarArchivo(context.Background(), "nota.txt", "b"+SeparadorEdicion+"B"); !errors.Is(err, ErrArgumentosInvalidos) {
		t.Fatalf("err = %v, quiero E_BAD_ARGS", err)
	}
}

// TestEliminarArchivoPideConfirmacionExplicita — una aprobación genérica no
// basta para borrar.
func TestEliminarArchivoPideConfirmacionExplicita(t *testing.T) {
	o, _ := opsDePrueba(t, aprobadorQue(Decision{Aprobada: true, Explicita: false}))
	if _, err := os.Stat(filepath.Join(o.Proyecto, "ai", "docs", "PROJECT.md")); err != nil {
		t.Fatal(err)
	}
	_, err := o.EliminarArchivo(context.Background(), "ai/docs/PROJECT.md")
	if !errors.Is(err, ErrNecesitaConfirmacion) {
		t.Fatalf("err = %v, quiero E_NEEDS_CONFIRM", err)
	}
	if _, statErr := os.Stat(filepath.Join(o.Proyecto, "ai", "docs", "PROJECT.md")); statErr != nil {
		t.Error("un borrado sin confirmación explícita no puede tocar el disco")
	}
}

// TestEliminarCarpetaRaizFalla — borrar la carpeta del proyecto se rechaza.
func TestEliminarCarpetaRaizFalla(t *testing.T) {
	o, _ := opsDePrueba(t, aprobado())
	if _, err := o.EliminarCarpeta(context.Background(), "."); !errors.Is(err, ErrArgumentosInvalidos) {
		t.Fatalf("err = %v, quiero E_BAD_ARGS al borrar la raíz", err)
	}
}

// TestOperacionesDeCarpeta — crear y borrar una carpeta deja el sistema como lo
// aprobado.
func TestOperacionesDeCarpeta(t *testing.T) {
	o, _ := opsDePrueba(t, aprobado())
	if _, err := o.CrearCarpeta(context.Background(), "nueva"); err != nil {
		t.Fatalf("CrearCarpeta: %v", err)
	}
	info, err := os.Stat(filepath.Join(o.Proyecto, "nueva"))
	if err != nil || !info.IsDir() {
		t.Fatalf("la carpeta no se creó: %v", err)
	}
	if _, err := o.EliminarCarpeta(context.Background(), "nueva"); err != nil {
		t.Fatalf("EliminarCarpeta: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(o.Proyecto, "nueva")); statErr == nil {
		t.Error("la carpeta debería estar borrada")
	}
}

// TestLecturaListadoYBusqueda — las lecturas no escriben y devuelven lo
// documentado.
func TestLecturaListadoYBusqueda(t *testing.T) {
	o, _ := opsDePrueba(t, aprobado())
	res, err := LeerArchivo(o.Proyecto, "ai/docs/PROJECT.md")
	if err != nil || res.Contenido != "# p\n" {
		t.Fatalf("LeerArchivo = %+v, %v", res, err)
	}
	lista, err := ListarCarpeta(o.Proyecto, "ai/docs")
	if err != nil || len(lista.Entradas) != 1 || lista.Entradas[0] != "PROJECT.md" {
		t.Fatalf("ListarCarpeta = %+v, %v", lista, err)
	}
	archivos, err := BuscarArchivos(o.Proyecto, "*.md")
	if err != nil || len(archivos.Rutas) != 1 || archivos.Rutas[0] != "ai/docs/PROJECT.md" {
		t.Fatalf("BuscarArchivos = %+v, %v", archivos, err)
	}
	coincidencias, err := BuscarEnArchivos(o.Proyecto, "PROJECT", "ai/docs")
	if err != nil || len(coincidencias.Coincidencias) != 0 {
		// "PROJECT" es el nombre del archivo, no su contenido; la búsqueda es
		// por contenido, así que no debe encontrar nada.
		t.Fatalf("BuscarEnArchivos = %+v, %v", coincidencias, err)
	}
}
