// integration_frontera_test.go — T-B015-03: ninguna operación sale de la
// carpeta del proyecto.
//
// Fuente de verdad: SECURITY.md §3 y VALIDATION.md §1 ("Se normaliza la ruta
// antes de validar, para que `../` o rutas equivalentes no esquiven la
// frontera"), TESTING.md §2 (fileops y exec: intentos de evasión fallan).
//
// Cada caso aquí es una táctica de evasión real: traversal, symlink de dentro
// hacia fuera, ruta absoluta, carpeta de trabajo fuera para los comandos. Se
// prueban contra el pipeline real, no contra la función de frontera aislada,
// porque la garantía es del conjunto.
package tests

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"localcli/internal/exec"
	"localcli/internal/fileops"
	"localcli/internal/store"
	"localcli/internal/tools"
)

// storeHistorial construye el adaptador de historial para las pruebas.
func storeHistorial(db *sql.DB) store.Historial { return store.Historial{DB: db} }

// TestSinAprobadorNadaSaleDeLaCarpeta — las formas equivalentes de escribir
// `../` no tienen a quién pedir permiso, así que acaban en E_PATH_OUTSIDE y no
// crean nada fuera (ERRORS.md §3: fuera «sin permiso y sin explicación»).
func TestSinAprobadorNadaSaleDeLaCarpeta(t *testing.T) {
	proyecto, db := proyectoTemp(t)
	// Sin aprobador: el estado por defecto es cerrado.
	ops := &fileops.Ops{Proyecto: proyecto, Historial: storeHistorial(db)}

	// Un archivo hermano del proyecto: fuera de la frontera.
	fuera := filepath.Join(filepath.Dir(proyecto), "colado.txt")

	intentos := []string{
		"../colado.txt",
		"./../colado.txt",
		"ai/../../colado.txt",
		"sub/../../colado.txt",
		"../sub/../colado.txt",
	}
	for _, ruta := range intentos {
		if _, err := ops.CrearArchivo(context.Background(), ruta, "x"); !errors.Is(err, fileops.ErrRutaFuera) {
			t.Errorf("CrearArchivo(%q) = %v, quiero E_PATH_OUTSIDE", ruta, err)
		}
		// Normalizando la ruta del intento, el destino nunca es el archivo
		// hermano: si lo fuera, alguna de las rutas de arriba habría colado.
		if _, err := os.Stat(fuera); err == nil {
			t.Fatalf("GARANTÍA ROTA: %q creó %s", ruta, fuera)
		}
	}
}

// TestFueraDeLaCarpetaSeAplicaConPermiso — la spec no prohíbe salir: exige
// permiso (SPEC-ARCHIVOS §Reglas). Con la aprobación del usuario, la operación
// se aplica.
func TestFueraDeLaCarpetaSeAplicaConPermiso(t *testing.T) {
	proyecto, db := proyectoTemp(t)
	ops := &fileops.Ops{
		Proyecto:  proyecto,
		Historial: storeHistorial(db),
		Aprobador: aprobadorTotal(),
		Motivo:    "el usuario lo pidió",
	}

	fuera := filepath.Join(filepath.Dir(proyecto), "permitido.txt")
	t.Cleanup(func() { _ = os.Remove(fuera) })

	if _, err := ops.CrearArchivo(context.Background(), "../permitido.txt", "x"); err != nil {
		t.Fatalf("con permiso, escribir fuera se aplica: %v", err)
	}
	if _, err := os.Stat(fuera); err != nil {
		t.Errorf("el archivo aprobado debe existir fuera: %v", err)
	}
}

// TestElSymlinkNoEsPuertaTrasera — un enlace dentro del proyecto que apunta a
// una carpeta exterior se marca como fuera: sin aprobador no se toca, y pedir
// permiso es obligatorio para llegar a él.
func TestElSymlinkNoEsPuertaTrasera(t *testing.T) {
	proyecto, db := proyectoTemp(t)
	ops := &fileops.Ops{Proyecto: proyecto, Historial: storeHistorial(db)}

	exterior := t.TempDir()
	if err := os.Symlink(exterior, filepath.Join(proyecto, "puerta")); err != nil {
		t.Skipf("no se pueden crear symlinks aquí: %v", err)
	}
	_, err := ops.CrearArchivo(context.Background(), "puerta/colado.txt", "x")
	if !errors.Is(err, fileops.ErrRutaFuera) {
		t.Fatalf("escribir a través del symlink = %v, quiero E_PATH_OUTSIDE", err)
	}
	if _, statErr := os.Stat(filepath.Join(exterior, "colado.txt")); statErr == nil {
		t.Fatal("GARANTÍA ROTA: el symlink llevó la escritura fuera")
	}
	// Y leer a través de él tampoco, cuando no hay con quién pedir permiso.
	if _, err := fileops.LeerArchivo(proyecto, "puerta/colado.txt"); !errors.Is(err, fileops.ErrRutaFuera) {
		t.Errorf("leer a través del symlink = %v, quiero E_PATH_OUTSIDE", err)
	}
}

// TestUnaRutaAbsolutaFueraPidePermiso — la ruta absoluta que no cuelga del
// proyecto sale fuera: sin aprobador no se toca; con aprobación, se aplica.
func TestUnaRutaAbsolutaFueraPidePermiso(t *testing.T) {
	proyecto, db := proyectoTemp(t)
	destino := filepath.Join(t.TempDir(), "absoluto.txt")

	cerrado := &fileops.Ops{Proyecto: proyecto, Historial: storeHistorial(db)}
	if _, err := cerrado.CrearArchivo(context.Background(), destino, "x"); !errors.Is(err, fileops.ErrRutaFuera) {
		t.Fatalf("CrearArchivo(absoluta) sin aprobador = %v, quiero E_PATH_OUTSIDE", err)
	}
	if _, err := os.Stat(destino); err == nil {
		t.Fatal("GARANTÍA ROTA: la ruta absoluta escribió sin permiso")
	}

	abierto := &fileops.Ops{
		Proyecto:  proyecto,
		Historial: storeHistorial(db),
		Aprobador: aprobadorTotal(),
		Motivo:    "el usuario lo pidió",
	}
	if _, err := abierto.CrearArchivo(context.Background(), destino, "x"); err != nil {
		t.Fatalf("con permiso, la ruta absoluta de fuera se aplica: %v", err)
	}
	if _, err := os.Stat(destino); err != nil {
		t.Errorf("el archivo aprobado debe existir: %v", err)
	}
}

// TestLosComandosNoCorrenFueraDelProyecto — la carpeta de trabajo de
// ejecutar_comando es relativa al proyecto; una que escape se rechaza
// (VALIDATION.md §1: "El comando corre en la carpeta del proyecto").
func TestLosComandosNoCorrenFueraDelProyecto(t *testing.T) {
	proyecto, _ := proyectoTemp(t)
	e := &exec.Ejecutor{Proyecto: proyecto, Limite: 5}

	for _, carpeta := range []string{"..", "../..", "sub/../../"} {
		_, err := e.Ejecutar(context.Background(), "go version", carpeta)
		if !errors.Is(err, exec.ErrArgumentosInvalidos) {
			t.Errorf("Ejecutar con carpeta %q = %v, quiero E_BAD_ARGS", carpeta, err)
		}
	}
}

// TestLaListaBlancaYElBloqueoSostienenLaTerminal — sin aprobador, un comando
// fuera de la lista blanca no se ejecuta. El bloqueo de escritura real lo
// prueba exec con Landlock (run_test.go); aquí se comprueba el rechazo de la
// puerta.
func TestLaListaBlancaYElBloqueoSostienenLaTerminal(t *testing.T) {
	proyecto, _ := proyectoTemp(t)
	e := &exec.Ejecutor{Proyecto: proyecto, Limite: 5}

	if _, err := e.Ejecutar(context.Background(), "rm -rf /", ""); !errors.Is(err, exec.ErrComandoNoEnBlanco) {
		t.Errorf("comando fuera de lista sin aprobador = %v, quiero E_CMD_NOT_WHITELISTED", err)
	}
	// Con la salida desactivada, las herramientas de internet tampoco pasan.
	if tools.InternetPermitida() {
		t.Skip("LOCALCLI_ALLOW_INTERNET está activa en el entorno; no se puede probar el cierre")
	}
}

// TestElProyectoSigueIntactoTrasLosIntentos — después de todas las evasiones,
// el contenido del proyecto es el que era.
func TestElProyectoSigueIntactoTrasLosIntentos(t *testing.T) {
	proyecto, db := proyectoTemp(t)
	marcador := filepath.Join(proyecto, "ai", "docs", "PROJECT.md")
	antes, err := os.ReadFile(marcador)
	if err != nil {
		t.Fatal(err)
	}

	ops := &fileops.Ops{Proyecto: proyecto, Historial: storeHistorial(db), Aprobador: negador()}
	_, _ = ops.CrearArchivo(context.Background(), "../x.txt", "x")
	_, _ = ops.CrearArchivo(context.Background(), "cualquiera.txt", "x")
	_, _ = ops.EliminarArchivo(context.Background(), "ai/docs/PROJECT.md")

	después, err := os.ReadFile(marcador)
	if err != nil {
		t.Fatalf("el marcador debía sobrevivir: %v", err)
	}
	if string(antes) != string(después) {
		t.Error("el proyecto cambió con el aprobador cerrado")
	}
}
