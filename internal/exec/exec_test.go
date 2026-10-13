package exec

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"localcli/internal/tools"
)

func TestListaBlanca(t *testing.T) {
	blancos := [][]string{
		{"go", "build", "./..."},
		{"go", "test", "./..."},
		{"go", "vet", "./..."},
		{"git", "status"},
		{"git", "diff"},
		{"git", "log", "-5"},
		{"gofmt", "-l", "."},
	}
	for _, argv := range blancos {
		if !EnListaBlanca(argv) {
			t.Errorf("`%s` debería correr sin preguntar", strings.Join(argv, " "))
		}
	}
	noBlancos := [][]string{
		{"go", "run", "./..."},
		{"git", "push"},
		{"git", "commit", "-m", "x"},
		{"gofmt", "-w", "."},
		{"echo", "hola"},
	}
	for _, argv := range noBlancos {
		if EnListaBlanca(argv) {
			t.Errorf("`%s` no debería estar en la lista blanca", strings.Join(argv, " "))
		}
	}
}

func TestPartirComandoRechazaShell(t *testing.T) {
	if _, err := PartirComando("ls -la"); err != nil {
		t.Fatalf("un comando simple debería partirse: %v", err)
	}
	for _, comando := range []string{"", "a | b", "echo x > f", "sh -c 'algo'", "echo $(date)"} {
		if _, err := PartirComando(comando); err == nil {
			t.Errorf("`%s` debería rechazarse", comando)
		}
	}
}

func TestComandoDeclinadoNoSeEjecuta(t *testing.T) {
	e := &Ejecutor{Proyecto: t.TempDir()}
	c := tools.Contexto{Ask: func(context.Context, tools.Solicitud) (tools.Decision, error) {
		return tools.Declinar, nil
	}}
	_, err := e.Ejecutar(context.Background(), Solicitud{Comando: "echo hola"}, c)
	var te *Error
	if err == nil || !asTE(err, &te) || te.Codigo != tools.CodApprovalDeclined {
		t.Fatalf("un comando no blanco declinado debería dar E_APPROVAL_DECLINED: %v", err)
	}
}

func TestLaListaBlancaNoPregunta(t *testing.T) {
	e := &Ejecutor{Proyecto: t.TempDir()}
	preguntado := false
	c := tools.Contexto{Ask: func(context.Context, tools.Solicitud) (tools.Decision, error) {
		preguntado = true
		return tools.Declinar, nil
	}}
	salida, err := e.Ejecutar(context.Background(), Solicitud{Comando: "git status"}, c)
	if err != nil {
		t.Fatalf("git status no debería pedir aprobación ni fallar: %v", err)
	}
	if preguntado {
		t.Fatal("la lista blanca no debería preguntar")
	}
	if !salida.Termino {
		t.Fatal("git status debería terminar")
	}
}

func TestUnComandoQueFallaNoEsErrorDeGo(t *testing.T) {
	e := &Ejecutor{Proyecto: t.TempDir()}
	salida, err := e.Ejecutar(context.Background(), Solicitud{Comando: "git status"}, aprobador())
	if err != nil {
		t.Fatalf("un comando con código ≠ 0 no debería dar error de Go: %v", err)
	}
	if !salida.Termino {
		t.Fatal("debería marcar termino=true")
	}
}

func TestTimeoutSeCorta(t *testing.T) {
	e := &Ejecutor{Proyecto: t.TempDir(), Limite: 200 * time.Millisecond}
	salida, err := e.Ejecutar(context.Background(), Solicitud{Comando: "sleep 5"}, aprobador())
	if err != nil {
		t.Fatalf("el timeout no debería ser error de Go: %v", err)
	}
	if !salida.Truncado || !strings.Contains(salida.Error, tools.CodCmdTimeout) {
		t.Fatalf("debería cortarse con E_CMD_TIMEOUT: %+v", salida)
	}
}

func TestSalidaTruncada(t *testing.T) {
	dir := t.TempDir()
	grande := filepath.Join(dir, "grande.txt")
	if err := os.WriteFile(grande, []byte(strings.Repeat("x\n", 20000)), 0o644); err != nil {
		t.Fatal(err)
	}
	e := &Ejecutor{Proyecto: dir}
	salida, err := e.Ejecutar(context.Background(), Solicitud{Comando: "cat grande.txt"}, aprobador())
	if err != nil {
		t.Fatalf("cat no debería dar error de Go: %v", err)
	}
	if !salida.Truncado {
		t.Fatal("una salida enorme debería marcarse truncada")
	}
}

// TestLandlockImpideEscribir comprueba que la terminal no puede escribir en el
// proyecto ni con un comando indirecto.
func TestLandlockImpideEscribir(t *testing.T) {
	if !soportaLandlock() {
		t.Skip("sin Landlock: la garantía es más débil y no se puede comprobar aquí")
	}
	// El proyecto se crea fuera del árbol de temporales: si no, caería dentro de
	// una de las rutas con escritura concedida y la prueba no mediría nada.
	dir, err := os.MkdirTemp(".", "llk-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	archivo := filepath.Join(dir, "mal.go")
	const original = "package x\nfunc  x( ){}\n"
	if err := os.WriteFile(archivo, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	e := &Ejecutor{Proyecto: dir}
	if _, err := e.Ejecutar(context.Background(), Solicitud{Comando: "gofmt -w mal.go", Carpeta: dir}, aprobador()); err != nil {
		t.Fatalf("gofmt no debería dar error de Go: %v", err)
	}
	datos, err := os.ReadFile(archivo)
	if err != nil {
		t.Fatal(err)
	}
	if string(datos) != original {
		t.Fatal("Landlock debería haber impedido reescribir el archivo del proyecto")
	}
}

func aprobador() tools.Contexto {
	return tools.Contexto{Ask: func(context.Context, tools.Solicitud) (tools.Decision, error) {
		return tools.Aprobar, nil
	}}
}

func asTE(err error, destino **Error) bool {
	te, ok := err.(*Error)
	if ok {
		*destino = te
	}
	return ok
}
