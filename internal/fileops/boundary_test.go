package fileops

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestResolverRechazaRutasFuera — una ruta absoluta o que escapa con `..` es
// E_PATH_OUTSIDE: nada sale de la carpeta del proyecto.
func TestResolverRechazaRutasFuera(t *testing.T) {
	proyecto := t.TempDir()
	casos := []string{
		"/etc/passwd",
		"../secreto.txt",
		"sub/../../secreto.txt",
		"..",
	}
	for _, ruta := range casos {
		if _, err := Resolver(proyecto, ruta); !errors.Is(err, ErrRutaFuera) {
			t.Errorf("Resolver(%q): err = %v, quiero E_PATH_OUTSIDE", ruta, err)
		}
	}
}

// TestResolverAceptaDentro — una ruta relativa que se queda dentro se resuelve
// contra la carpeta del proyecto.
func TestResolverAceptaDentro(t *testing.T) {
	proyecto := t.TempDir()
	abs, err := Resolver(proyecto, "ai/docs/PROJECT.md")
	if err != nil {
		t.Fatalf("Resolver: %v", err)
	}
	if !DentroDe(proyecto, abs) {
		t.Errorf("%s quedó fuera de %s", abs, proyecto)
	}
}

// TestResolverRechazaSymlinkExterno — un enlace simbólico dentro del proyecto
// que apunta fuera no es una puerta trasera.
func TestResolverRechazaSymlinkExterno(t *testing.T) {
	proyecto := t.TempDir()
	fuera := t.TempDir()
	if err := os.WriteFile(filepath.Join(fuera, "secreto.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	enlace := filepath.Join(proyecto, "enlace")
	if err := os.Symlink(fuera, enlace); err != nil {
		t.Skipf("no se pudieron crear enlaces simbólicos: %v", err)
	}
	if _, err := Resolver(proyecto, "enlace/secreto.txt"); !errors.Is(err, ErrRutaFuera) {
		t.Fatalf("err = %v, quiero E_PATH_OUTSIDE para un symlink externo", err)
	}
}
