package git

// Tests del lector de estado del repositorio: lo que el panel muestra al pie
// (SPEC-INTERFAZ §Zonas 3, dato «Git»). Cada caso monta un repositorio real en
// un temporal, porque el lector habla con el binario `git` y un doble no
// probaría nada.

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Si el binario no está, el lector no puede probarse: no es un fallo, es que el
// entorno no tiene git.
func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git no está instalado en este entorno")
	}
}

// repoCrea un repositorio nuevo en un temporal y devuelve su ruta. Se salta la
// prueba si git no está.
func repoCrea(t *testing.T) string {
	t.Helper()
	requireGit(t)
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "--initial-branch=main"},
		{"config", "user.email", "prueba@localcli"},
		{"config", "user.name", "prueba"},
	} {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	return dir
}

func escribe(t *testing.T, dir, nombre, contenido string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, nombre), []byte(contenido), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestEstadoDevuelveLaRamaYElArbolLimpio(t *testing.T) {
	dir := repoCrea(t)
	escribe(t, dir, "main.go", "package main\n")
	commit(t, dir)

	rama, cambios := Estado(dir)
	if rama != "main" {
		t.Errorf("la rama activa es la que sale de git, no un valor inventado: %q", rama)
	}
	if cambios != 0 {
		t.Errorf("un árbol sin cambios pendientes tiene cero cambios: %d", cambios)
	}
}

func TestEstadoDetectaLosCambiosSinConfirmar(t *testing.T) {
	dir := repoCrea(t)
	escribe(t, dir, "main.go", "package main\n")
	commit(t, dir)

	// Un archivo sin confirmar: lo acaba de escribir el agente, no hay commit.
	escribe(t, dir, "main.go", "package main\n\n// cambio\n")

	if _, cambios := Estado(dir); cambios == 0 {
		t.Error("un archivo modificado sin commit cuenta como un cambio")
	}
}

func TestEstadoSinGitIniciadoNoDevuelveRama(t *testing.T) {
	requireGit(t)
	// Un temporal normal: no está dentro de ningún repositorio.
	dir := t.TempDir()

	rama, _ := Estado(dir)
	if rama != "" {
		t.Errorf("sin git inicializado la rama va vacía, que es lo que se pinta como «sin iniciar»: %q", rama)
	}
}

func TestEstadoEnUnRepoSinCommitsDevuelveLaRama(t *testing.T) {
	dir := repoCrea(t) // `git init` y nada más: la rama existe, los commits no.
	escribe(t, dir, "main.go", "package main\n")

	rama, _ := Estado(dir)
	if rama != "main" {
		t.Errorf("un repositorio recién inicializado ya tiene rama, aunque no tenga commits: %q", rama)
	}
}

func TestEstadoEnUnSubdirectorioDaLaRamaDelRepositorio(t *testing.T) {
	dir := repoCrea(t)
	escribe(t, dir, "main.go", "package main\n")
	commit(t, dir)
	sub := filepath.Join(dir, "internal", "tui")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	if rama, _ := Estado(sub); rama != "main" {
		t.Errorf("`-C` sube hasta el repositorio: %q", rama)
	}
}

func TestRamaDeCabecera(t *testing.T) {
	casos := []struct {
		linea, rama string
	}{
		{"## main...origin/main", "main"},
		{"## main", "main"},
		{"## No commits yet on main", "main"},
		{"## feature/nueva...origin/feature/nueva", "feature/nueva"},
		{"## HEAD (no branch)", ""},
		{" M main.go", ""},
		{"?? nuevo.txt", ""},
	}
	for _, c := range casos {
		if got := ramaDeCabecera(c.linea); got != c.rama {
			t.Errorf("ramaDeCabecera(%q) = %q, se esperaba %q", c.linea, got, c.rama)
		}
	}
}

func commit(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("git", "-C", dir, "add", ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
	cmd = exec.Command("git", "-C", dir, "commit", "-m", "prueba")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
}
