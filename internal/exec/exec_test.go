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

func TestPeligroso(t *testing.T) {
	confirmables := [][]string{
		{"rm", "f.txt"},
		{"rm", "-rf", "build"},
		{"rmdir", "x"},
		{"mv", "a.txt", "b.txt"},
		{"dd", "if=/dev/zero", "of=disk"},
		{"mkfs.ext4", "/dev/sda1"},
		{"truncate", "-s", "0", "f.txt"},
		{"kill", "-9", "1"},
		{"shutdown", "-h", "now"},
		{"git", "reset", "--hard"},
		{"git", "clean", "-fd"},
		{"git", "checkout", "--", "f.go"},
		{"git", "push"},
		{"git", "push", "origin", "main"},
		{"git", "push", "--force"},
		{"git", "branch", "-D", "temp"},
		{"curl", "http://example.com"},
		{"wget", "http://example.com/f"},
		{"ssh", "host"},
		{"scp", "f.txt", "host:/tmp"},
		{"nc", "-l", "8080"},
		{"rsync", "-a", "src/", "host:/tmp"},
	}
	for _, argv := range confirmables {
		if !Peligroso(argv) {
			t.Errorf("`%s` debería pedir confirmación", strings.Join(argv, " "))
		}
	}
	inocuos := [][]string{
		{"go", "build", "./..."},
		{"go", "test", "./..."},
		{"go", "vet", "./..."},
		{"gofmt", "-l", "."},
		{"git", "status"},
		{"git", "diff"},
		{"git", "log", "-5"},
		{"git", "commit", "-m", "msg"},
		{"git", "fetch"},
		{"git", "branch", "-d", "temp"},
		{"echo", "hola"},
		{"ls", "-la"},
		{"cat", "grande.txt"},
	}
	for _, argv := range inocuos {
		if Peligroso(argv) {
			t.Errorf("`%s` no debería pedir confirmación", strings.Join(argv, " "))
		}
	}
}

// TestFueraDelProyectoPideConfirmacion: un argumento absoluto fuera de la
// carpeta del proyecto sí pide confirmación; una ruta interna, no.
func TestFueraDelProyectoPideConfirmacion(t *testing.T) {
	raiz := t.TempDir()
	if FueraDelProyecto(raiz, []string{"cat", "/etc/hostname"}) != true {
		t.Fatal("una ruta absoluta fuera del proyecto debería pedir confirmación")
	}
	if FueraDelProyecto(raiz, []string{"cat", filepath.Join(raiz, "a.txt")}) {
		t.Fatal("una ruta dentro del proyecto no debería pedir confirmación")
	}
	if FueraDelProyecto(raiz, []string{"cat", "relativa.txt"}) {
		t.Fatal("una ruta relativa no debería pedir confirmación")
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

func TestComandoDestructivoDeclinadoNoSeEjecuta(t *testing.T) {
	e := &Ejecutor{Proyecto: t.TempDir()}
	c := tools.Contexto{Ask: func(context.Context, tools.Solicitud) (tools.Decision, error) {
		return tools.Declinar, nil
	}}
	_, err := e.Ejecutar(context.Background(), Solicitud{Comando: "rm nuevo.txt"}, c)
	var te *Error
	if err == nil || !asTE(err, &te) || te.Codigo != tools.CodApprovalDeclined {
		t.Fatalf("un comando destructivo declinado debería dar E_APPROVAL_DECLINED: %v", err)
	}
}

// TestComandoConEgresoDeclinadoNoSeEjecuta: un comando que saca información de
// la máquina pide confirmación; declinado, no llega a correr.
func TestComandoConEgresoDeclinadoNoSeEjecuta(t *testing.T) {
	e := &Ejecutor{Proyecto: t.TempDir()}
	c := tools.Contexto{Ask: func(context.Context, tools.Solicitud) (tools.Decision, error) {
		return tools.Declinar, nil
	}}
	_, err := e.Ejecutar(context.Background(), Solicitud{Comando: "curl http://localhost:1/"}, c)
	var te *Error
	if err == nil || !asTE(err, &te) || te.Codigo != tools.CodApprovalDeclined {
		t.Fatalf("un comando con salida de red declinado debería dar E_APPROVAL_DECLINED: %v", err)
	}
}

func TestComandoInocuoNoPregunta(t *testing.T) {
	e := &Ejecutor{Proyecto: t.TempDir()}
	preguntado := false
	c := tools.Contexto{Ask: func(context.Context, tools.Solicitud) (tools.Decision, error) {
		preguntado = true
		return tools.Declinar, nil
	}}
	salida, err := e.Ejecutar(context.Background(), Solicitud{Comando: "echo hola"}, c)
	if err != nil {
		t.Fatalf("un comando inocuo no debería ni preguntar ni fallar: %v", err)
	}
	if preguntado {
		t.Fatal("un comando que no es destructivo ni toca rutas de fuera no debería preguntar")
	}
	if !salida.Termino {
		t.Fatal("echo debería terminar")
	}
}

// TestComandoConRutaFueraPregunta: tocar una ruta absoluta fuera del proyecto
// sí exige confirmación, aunque el programa no sea destructivo.
func TestComandoConRutaFueraPregunta(t *testing.T) {
	e := &Ejecutor{Proyecto: t.TempDir()}
	preguntado := false
	c := tools.Contexto{Ask: func(context.Context, tools.Solicitud) (tools.Decision, error) {
		preguntado = true
		return tools.Declinar, nil
	}}
	_, err := e.Ejecutar(context.Background(), Solicitud{Comando: "cat /etc/hostname"}, c)
	if !preguntado {
		t.Fatal("una ruta fuera de la carpeta debería pedir confirmación")
	}
	var te *Error
	if err == nil || !asTE(err, &te) || te.Codigo != tools.CodApprovalDeclined {
		t.Fatalf("declinar debería dar E_APPROVAL_DECLINED: %v", err)
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
