package exec

import (
	"errors"
	"testing"
)

// TestListaBlancaCerrada — compilar, probar, revisar estilo/tipos y el git de
// lectura se ejecutan sin preguntar; cualquier otra cosa no.
func TestListaBlancaCerrada(t *testing.T) {
	blancos := [][]string{
		{"go", "build", "./..."},
		{"go", "test", "./..."},
		{"go", "vet", "./..."},
		{"gofmt", "-l", "."},
		{"git", "status"},
		{"git", "diff"},
		{"git", "log", "--oneline"},
	}
	for _, argv := range blancos {
		if !EnListaBlanca(argv) {
			t.Errorf("%v debería estar en la lista blanca", argv)
		}
	}
	fuera := [][]string{
		{"rm", "-rf", "."},
		{"go", "run", "main.go"},
		{"go", "fmt", "./..."},       // reescribe archivos
		{"gofmt", "-w", "."},         // reescribe archivos
		{"git", "push"},              // publica
		{"git", "commit", "-m", "x"}, // crea historial
		{"echo", "hola"},
	}
	for _, argv := range fuera {
		if EnListaBlanca(argv) {
			t.Errorf("%v no debería estar en la lista blanca", argv)
		}
	}
}

// TestValidarRechazaShellYComposicion — T-B009-02: `sh -c`, tuberías y
// redirecciones no pasan.
func TestValidarRechazaShellYComposicion(t *testing.T) {
	casos := []string{
		`sh -c "rm -rf ."`,
		`bash -c 'echo x > f'`,
		`go test ./... | tee out.txt`,
		`echo hola > archivo`,
		`git log && rm -rf .`,
		`go build $(echo x)`,
	}
	for _, comando := range casos {
		if _, err := Validar(comando); !errors.Is(err, ErrArgumentosInvalidos) {
			t.Errorf("Validar(%q): err = %v, quiero E_BAD_ARGS", comando, err)
		}
	}
}

// TestValidarParseaArgv — un comando normal se parte en argumentos.
func TestValidarParseaArgv(t *testing.T) {
	argv, err := Validar("go test ./internal/exec/...")
	if err != nil {
		t.Fatalf("Validar: %v", err)
	}
	if len(argv) != 3 || argv[0] != "go" || argv[1] != "test" {
		t.Fatalf("argv = %v", argv)
	}
	if _, err := Validar("   "); !errors.Is(err, ErrArgumentosInvalidos) {
		t.Errorf("comando vacío: err = %v, quiero E_BAD_ARGS", err)
	}
}
