package fileops

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestResolverMarcaLasRutasDeFuera — una ruta absoluta que no cuelga del
// proyecto o que escapa con `..` se marca como «fuera»: no es un error, es una
// operación que necesita permiso (SPEC-ARCHIVOS §Reglas).
func TestResolverMarcaLasRutasDeFuera(t *testing.T) {
	proyecto := t.TempDir()
	casos := []string{
		"/etc/passwd",
		"../secreto.txt",
		"sub/../../secreto.txt",
		"..",
	}
	for _, ruta := range casos {
		_, fuera, err := Resolver(proyecto, ruta)
		if err != nil {
			t.Errorf("Resolver(%q): err = %v; una ruta de fuera no es error", ruta, err)
			continue
		}
		if !fuera {
			t.Errorf("Resolver(%q): fuera = false, quiero fuera = true", ruta)
		}
	}
}

// TestResolverAceptaDentro — una ruta relativa que se queda dentro se resuelve
// contra la carpeta del proyecto.
func TestResolverAceptaDentro(t *testing.T) {
	proyecto := t.TempDir()
	abs, fuera, err := Resolver(proyecto, "ai/docs/PROJECT.md")
	if err != nil {
		t.Fatalf("Resolver: %v", err)
	}
	if fuera {
		t.Errorf("%s no debería salir del proyecto", abs)
	}
	if !DentroDe(proyecto, abs) {
		t.Errorf("%s quedó fuera de %s", abs, proyecto)
	}
}

// TestLeerFueraPidePermiso — leer fuera de la carpeta no se rechaza en duro:
// pide permiso (marcado como fuera) y, aprobado, lee. Sin aprobador o
// declinado, no se toca.
func TestLeerFueraPidePermiso(t *testing.T) {
	proyecto := t.TempDir()
	exterior := t.TempDir()
	ruta := filepath.Join(exterior, "foto.txt")
	if err := os.WriteFile(ruta, []byte("contenido"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Sin aprobador: fuera es E_PATH_OUTSIDE (fuera sin permiso ni explicación).
	if _, err := LeerArchivo(proyecto, ruta); !errors.Is(err, ErrRutaFuera) {
		t.Fatalf("sin aprobador = %v, quiero E_PATH_OUTSIDE", err)
	}

	// Con aprobador: se pide permiso, la solicitud va marcada como fuera y se lee.
	var pedida SolicitudAprobacion
	ops := &Ops{Proyecto: proyecto, Aprobador: AprobadorFunc(func(_ context.Context, s SolicitudAprobacion) (Decision, error) {
		pedida = s
		return Decision{Aprobada: true}, nil
	})}
	resp, err := ops.LeerArchivo(context.Background(), ruta)
	if err != nil {
		t.Fatalf("con permiso se lee: %v", err)
	}
	if !pedida.Fuera {
		t.Error("la solicitud debe ir marcada como fuera de la carpeta")
	}
	if resp.Contenido != "contenido" {
		t.Errorf("contenido = %q", resp.Contenido)
	}

	// Declinado: no se lee.
	negado := &Ops{Proyecto: proyecto, Aprobador: AprobadorFunc(func(context.Context, SolicitudAprobacion) (Decision, error) {
		return Decision{}, nil
	})}
	if _, err := negado.LeerArchivo(context.Background(), ruta); !errors.Is(err, ErrAprobacionDeclinada) {
		t.Errorf("declinado = %v, quiero E_APPROVAL_DECLINED", err)
	}
}

// TestLeerUnaImagenNoPidePermiso — la imagen viaja adjunta al turno, así que
// `leer_archivo` sobre una imagen no la lee ni pide permiso: devuelve la
// corrección y el turno sigue (el modelo no se queda esperando una decisión).
func TestLeerUnaImagenNoPidePermiso(t *testing.T) {
	proyecto := t.TempDir()
	exterior := t.TempDir()
	ruta := filepath.Join(exterior, "foto.png")
	if err := os.WriteFile(ruta, []byte("\x89PNG"), 0o644); err != nil {
		t.Fatal(err)
	}
	ops := &Ops{Proyecto: proyecto, Aprobador: AprobadorFunc(func(context.Context, SolicitudAprobacion) (Decision, error) {
		t.Error("una imagen no debe pedir permiso: ya viaja adjunta al turno")
		return Decision{Aprobada: true}, nil
	})}
	_, err := ops.LeerArchivo(context.Background(), ruta)
	if err == nil || !errors.Is(err, ErrArgumentosInvalidos) {
		t.Fatalf("leer una imagen = %v, quiero una corrección", err)
	}
	if !strings.Contains(err.Error(), "adjunta") {
		t.Errorf("la corrección debe decir que la imagen va adjunta: %v", err)
	}
}

// Dentro de la carpeta no se pide nada: es el caso normal.
func TestLeerDentroNoPidePermiso(t *testing.T) {
	proyecto := t.TempDir()
	if err := os.WriteFile(filepath.Join(proyecto, "nota.txt"), []byte("hola"), 0o644); err != nil {
		t.Fatal(err)
	}
	ops := &Ops{Proyecto: proyecto, Aprobador: AprobadorFunc(func(context.Context, SolicitudAprobacion) (Decision, error) {
		t.Error("leer dentro de la carpeta no debe pedir permiso")
		return Decision{Aprobada: true}, nil
	})}
	if _, err := ops.LeerArchivo(context.Background(), "nota.txt"); err != nil {
		t.Fatalf("LeerArchivo: %v", err)
	}
}

// TestResolverMarcaElSymlinkExterno — un enlace simbólico dentro del proyecto
// que apunta fuera no es una puerta trasera: se marca como fuera y exige
// aprobación.
func TestResolverMarcaElSymlinkExterno(t *testing.T) {
	proyecto := t.TempDir()
	fuera := t.TempDir()
	if err := os.WriteFile(filepath.Join(fuera, "secreto.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	enlace := filepath.Join(proyecto, "enlace")
	if err := os.Symlink(fuera, enlace); err != nil {
		t.Skipf("no se pudieron crear enlaces simbólicos: %v", err)
	}
	_, salio, err := Resolver(proyecto, "enlace/secreto.txt")
	if err != nil {
		t.Fatalf("err = %v; un symlink externo no es error, es «fuera»", err)
	}
	if !salio {
		t.Error("un symlink externo debe marcarse como fuera del proyecto")
	}
}
