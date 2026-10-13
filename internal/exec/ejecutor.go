package exec

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"

	"localcli/internal/tools"
)

// Ejecutor lanza comandos en la carpeta del proyecto, bajo el aislamiento de
// Landlock y con límites de tiempo y salida.
type Ejecutor struct {
	Proyecto string
	Limite   time.Duration
}

// Solicitud es lo que el handler pide ejecutar.
type Solicitud struct {
	Comando string
	Carpeta string
}

// Ejecutar parte el comando, consigue aprobación si no está en la lista blanca y
// lo corre. Un comando que sale con error no es un fallo: devuelve su salida.
func (e *Ejecutor) Ejecutar(ctx context.Context, s Solicitud, c tools.Contexto) (Salida, error) {
	argv, err := PartirComando(s.Comando)
	if err != nil {
		return Salida{}, err
	}
	if !EnListaBlanca(argv) {
		if err := aprobar(ctx, c, tools.Solicitud{Titulo: "Ejecutar comando", Descripcion: s.Comando}); err != nil {
			return Salida{}, err
		}
	}
	carpeta := s.Carpeta
	if carpeta == "" {
		carpeta = e.Proyecto
	}
	prog, err := exec.LookPath(argv[0])
	if err != nil {
		return Salida{}, nuevaError(tools.CodBadArgs, "no se encontró el programa `"+argv[0]+"`")
	}
	return e.correr(ctx, prog, argv[1:], carpeta)
}

// correr lanza el proceso y recoge su salida sin usar un shell.
func (e *Ejecutor) correr(ctx context.Context, prog string, args []string, carpeta string) (Salida, error) {
	limite := e.Limite
	if limite <= 0 {
		limite = TiempoLimite
	}
	ctx, cancel := context.WithTimeout(ctx, limite)
	defer cancel()

	cmd := exec.CommandContext(ctx, prog, args...)
	cmd.Dir = carpeta
	aislar(cmd)

	captura := nuevaCaptura(LimiteSalidaBytes)
	cmd.Stdout = captura.Stdout()
	cmd.Stderr = captura.Stderr()

	err := cmd.Run()
	stdout, stderr := captura.Salidas()
	salida := Salida{Salida: stdout, Error: stderr, Truncado: captura.Truncado(), Termino: true}

	if ctx.Err() == context.DeadlineExceeded {
		salida.Termino = false
		salida.Truncado = true
		salida.Error = tools.CodCmdTimeout + ": el comando no terminó a tiempo y se cortó"
		return salida, nil
	}
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			salida.Codigo = ee.ExitCode()
			return salida, nil
		}
		return Salida{}, err
	}
	return salida, nil
}

// EjecutarEquipo corre una lista de argumentos fija por la entrada estándar del
// subproceso. Lo usan las herramientas del usuario: la entrada no se añade a la
// lista, así que `equipo` queda fijo y verificable.
func (e *Ejecutor) EjecutarEquipo(ctx context.Context, equipo []string, entrada, carpeta string, limite time.Duration, c tools.Contexto) (Salida, error) {
	if len(equipo) == 0 {
		return Salida{}, nuevaError(tools.CodBadArgs, "el `equipo` está vacío")
	}
	if err := aprobar(ctx, c, tools.Solicitud{
		Titulo:      "Ejecutar herramienta del usuario",
		Descripcion: equipo[0],
	}); err != nil {
		return Salida{}, err
	}
	if carpeta == "" {
		carpeta = e.Proyecto
	}
	if limite <= 0 {
		limite = TiempoLimite
	}
	prog, err := exec.LookPath(equipo[0])
	if err != nil {
		return Salida{}, nuevaError(tools.CodBadArgs, "no se encontró el programa `"+equipo[0]+"`")
	}
	return e.correrConEntrada(ctx, prog, equipo[1:], carpeta, limite, entrada)
}

func (e *Ejecutor) correrConEntrada(ctx context.Context, prog string, args []string, carpeta string, limite time.Duration, entrada string) (Salida, error) {
	ctx, cancel := context.WithTimeout(ctx, limite)
	defer cancel()

	cmd := exec.CommandContext(ctx, prog, args...)
	cmd.Dir = carpeta
	cmd.Stdin = strings.NewReader(entrada)
	aislar(cmd)

	captura := nuevaCaptura(LimiteSalidaBytes)
	cmd.Stdout = captura.Stdout()
	cmd.Stderr = captura.Stderr()

	err := cmd.Run()
	stdout, stderr := captura.Salidas()
	salida := Salida{Salida: stdout, Error: stderr, Truncado: captura.Truncado(), Termino: true}
	if ctx.Err() == context.DeadlineExceeded {
		salida.Termino = false
		salida.Truncado = true
		salida.Error = tools.CodCmdTimeout + ": el comando no terminó a tiempo y se cortó"
		return salida, nil
	}
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			salida.Codigo = ee.ExitCode()
			return salida, nil
		}
		return Salida{}, err
	}
	return salida, nil
}
