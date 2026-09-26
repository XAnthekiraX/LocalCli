package exec

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func aprobadorSiempre(v bool) Aprobador {
	return AprobadorFunc(func(ctx context.Context, descripcion string) (bool, error) { return v, nil })
}

// TestComandoAprobadoDevuelveSalida — un comando fuera de la lista blanca, con
// aprobación, se ejecuta y devuelve salida y código.
func TestComandoAprobadoDevuelveSalida(t *testing.T) {
	e := &Ejecutor{Proyecto: t.TempDir(), Aprobador: aprobadorSiempre(true)}
	res, err := e.Ejecutar(context.Background(), "echo hola", "")
	if err != nil {
		t.Fatalf("Ejecutar: %v", err)
	}
	if strings.TrimSpace(res.Salida) != "hola" {
		t.Errorf("salida = %q, quiero hola", res.Salida)
	}
	if !res.Termino || res.Codigo != 0 {
		t.Errorf("termino=%v codigo=%d, quiero true/0", res.Termino, res.Codigo)
	}
}

// TestComandoFueraDeListaSinAprobador — sin lista blanca ni aprobador, no se
// ejecuta nada.
func TestComandoFueraDeListaSinAprobador(t *testing.T) {
	e := &Ejecutor{Proyecto: t.TempDir()}
	if _, err := e.Ejecutar(context.Background(), "echo hola", ""); !errors.Is(err, ErrComandoNoEnBlanco) {
		t.Fatalf("err = %v, quiero E_CMD_NOT_WHITELISTED", err)
	}
}

// TestComandoDeclinadoNoSeEjecuta — el rechazo llega al agente.
func TestComandoDeclinadoNoSeEjecuta(t *testing.T) {
	e := &Ejecutor{Proyecto: t.TempDir(), Aprobador: aprobadorSiempre(false)}
	if _, err := e.Ejecutar(context.Background(), "echo hola", ""); !errors.Is(err, ErrAprobacionDeclinada) {
		t.Fatalf("err = %v, quiero E_APPROVAL_DECLINED", err)
	}
}

// TestComandoDeListaBlancaSeEjecutaSolo — `gofmt -l` corre sin aprobador.
func TestComandoDeListaBlancaSeEjecutaSolo(t *testing.T) {
	e := &Ejecutor{Proyecto: t.TempDir()}
	res, err := e.Ejecutar(context.Background(), "gofmt -l .", "")
	if err != nil {
		t.Fatalf("Ejecutar: %v", err)
	}
	if !res.Termino {
		t.Error("gofmt -l debería terminar")
	}
}

// TestComandoLentoSeCorta — T-B009-03: el límite de tiempo se aplica con
// E_CMD_TIMEOUT.
func TestComandoLentoSeCorta(t *testing.T) {
	e := &Ejecutor{
		Proyecto:  t.TempDir(),
		Aprobador: aprobadorSiempre(true),
		Limite:    150 * time.Millisecond,
	}
	res, err := e.Ejecutar(context.Background(), "sleep 5", "")
	if !errors.Is(err, ErrComandoAgotado) {
		t.Fatalf("err = %v, quiero E_CMD_TIMEOUT", err)
	}
	if res.Termino {
		t.Error("un comando cortado no terminó")
	}
}

// TestSalidaSeTrunca — T-B009-04: la salida se corta y se marca.
func TestSalidaSeTrunca(t *testing.T) {
	e := &Ejecutor{
		Proyecto:     t.TempDir(),
		Aprobador:    aprobadorSiempre(true),
		LimiteSalida: 10,
	}
	res, err := e.Ejecutar(context.Background(), "echo 1234567890abcdef", "")
	if err != nil {
		t.Fatalf("Ejecutar: %v", err)
	}
	if !res.Truncado {
		t.Error("la salida cortada debe marcarse como truncada")
	}
	if len(res.Salida) > 10 {
		t.Errorf("salida = %d bytes, quiero <= 10", len(res.Salida))
	}
}

// TestCarpetaFueraDelProyectoSeRechaza — la carpeta de trabajo no sale del
// proyecto.
func TestCarpetaFueraDelProyectoSeRechaza(t *testing.T) {
	e := &Ejecutor{Proyecto: t.TempDir(), Aprobador: aprobadorSiempre(true)}
	if _, err := e.Ejecutar(context.Background(), "echo hola", "../fuera"); !errors.Is(err, ErrArgumentosInvalidos) {
		t.Fatalf("err = %v, quiero E_BAD_ARGS", err)
	}
}

// TestHelperEscritura es el hijo del test de aislamiento: intenta escribir en
// el proyecto. Solo corre cuando lo invoca ese test.
func TestHelperEscritura(t *testing.T) {
	destino := os.Getenv("HELPER_ESCRITURA")
	if destino == "" {
		t.Skip("solo se usa como proceso hijo del test de aislamiento")
	}
	if err := os.WriteFile(destino, []byte("x"), 0o644); err == nil {
		fmt.Println("ESCRITO")
		return
	}
	fmt.Println("BLOQUEADO")
}

// TestLandlockImpideEscribirEnElProyecto — T-B009-07: la terminal no puede
// escribir en el proyecto, por más indirecto que sea el comando. La prueba usa
// el mecanismo real: si el sistema no tiene Landlock, se salta, no se da por
// buena (TESTING.md §1).
func TestLandlockImpideEscribirEnElProyecto(t *testing.T) {
	if !GarantiaFuerte() {
		t.Skip("este sistema no tiene Landlock")
	}
	proyecto := t.TempDir()
	permitido := t.TempDir()
	salida := filepath.Join(proyecto, "escrito.txt")

	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	t.Setenv("HELPER_ESCRITURA", salida)

	e := &Ejecutor{
		Proyecto:      proyecto,
		Aprobador:     aprobadorSiempre(true),
		EspacioPropio: []string{permitido},
		Limite:        30 * time.Second,
	}
	res, err := e.Ejecutar(context.Background(), exe+" -test.run=TestHelperEscritura", "")
	if err != nil {
		t.Fatalf("Ejecutar: %v", err)
	}
	if strings.Contains(res.Error, "E_NO_LANDLOCK") {
		t.Skip("Landlock no se pudo aplicar en este entorno")
	}
	if _, statErr := os.Stat(salida); statErr == nil {
		t.Fatal("Landlock no bloqueó la escritura en el proyecto")
	}
	if !strings.Contains(res.Salida, "BLOQUEADO") {
		t.Fatalf("el hijo no reportó bloqueo: salida=%q error=%q", res.Salida, res.Error)
	}
}
