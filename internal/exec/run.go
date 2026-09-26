package exec

// run.go — T-B009-03 y T-B009-07: lanzar el proceso con límite de tiempo y
// garantizar que la terminal no toca los archivos del proyecto.
//
// Fuente de verdad: ai/docs/backend/DECISIONS.md ("Límites de comando fijos en
// la primera versión: 120 s de tiempo y 10 KB de salida") y
// ai/docs/backend/02-interfaces/TOOLS.md §5 (control 2: el bloqueo es
// estructural, con Landlock).
//
// La garantía la da el kernel, no el texto del comando: el proceso hijo se
// re-ejecuta con Landlock aplicado (landlock_linux.go) de modo que no puede
// crear, editar ni borrar archivos salvo en su propio espacio (temporales y
// caché de compilación). Si Landlock no está, la garantía es más débil y se
// avisa, sin bloquear el uso (SECURITY.md §6).

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"localcli/internal/tools"
)

const (
	// LimiteTiempo es el tope de tiempo de un comando.
	LimiteTiempo = 120 * time.Second

	// Marca del trampolín de Landlock: el proceso hijo la ve y aplica el
	// bloqueo antes de exec el comando real.
	envTrampolin = "LOCALCLI_EXEC_TRAMPOLIN"
	// Lista de rutas donde el comando sí puede escribir (separadas por `:`).
	envPermitidos = "LOCALCLI_EXEC_PERMISOS"
)

// Aprobador pide permiso para un comando fuera de la lista blanca.
type Aprobador interface {
	Aprobar(ctx context.Context, descripcion string) (bool, error)
}

// AprobadorFunc adapta una función a Aprobador.
type AprobadorFunc func(ctx context.Context, descripcion string) (bool, error)

func (f AprobadorFunc) Aprobar(ctx context.Context, descripcion string) (bool, error) {
	return f(ctx, descripcion)
}

// Ejecutor lanza comandos en el proyecto.
type Ejecutor struct {
	Proyecto     string
	Aprobador    Aprobador
	Limite       time.Duration
	LimiteSalida int
	// EspacioPropio son las rutas donde el comando sí puede escribir. Vacío
	// usa temporales y caché de compilación.
	EspacioPropio []string
}

func (e *Ejecutor) espacioEscritura() []string {
	if len(e.EspacioPropio) > 0 {
		return e.EspacioPropio
	}
	return rutasPermitidasEscritura()
}

func (e *Ejecutor) limite() time.Duration {
	if e.Limite > 0 {
		return e.Limite
	}
	return LimiteTiempo
}

func (e *Ejecutor) limiteSalida() int {
	if e.LimiteSalida > 0 {
		return e.LimiteSalida
	}
	return LimiteSalidaBytes
}

// GarantiaFuerte informa si el bloqueo estructural está disponible. Si es
// false, la terminal funciona con garantía más débil y hay que avisarlo
// (E_NO_LANDLOCK).
func GarantiaFuerte() bool { return soportaLandlock() }

// AvisoGarantia devuelve el aviso a mostrar cuando no hay bloqueo estructural,
// o cadena vacía si Landlock está disponible.
func AvisoGarantia() string {
	if soportaLandlock() {
		return ""
	}
	return avisoDegradacion()
}

// Ejecutar lanza el comando y devuelve su resultado. Un comando que sale con
// error no es un fallo de la herramienta: se devuelve su salida de error y el
// agente sigue (ERRORS.md §5). Lo que sí es un error es que el comando se pase
// del límite (E_CMD_TIMEOUT) o que se decline su aprobación.
func (e *Ejecutor) Ejecutar(ctx context.Context, comando, carpeta string) (tools.RespuestaEjecutarComando, error) {
	argv, err := Validar(comando)
	if err != nil {
		return tools.RespuestaEjecutarComando{}, err
	}
	dir := e.Proyecto
	if strings.TrimSpace(carpeta) != "" {
		if dir, err = resolverCarpeta(e.Proyecto, carpeta); err != nil {
			return tools.RespuestaEjecutarComando{}, err
		}
	}

	if !EnListaBlanca(argv) {
		if e.Aprobador == nil {
			return tools.RespuestaEjecutarComando{}, nuevoError(CodigoComandoNoEnBlanco,
				"el comando `"+comando+"` no está en la lista blanca y necesita aprobación")
		}
		ok, aErr := e.Aprobador.Aprobar(ctx, "ejecutar `"+comando+"` en "+dir)
		if aErr != nil {
			return tools.RespuestaEjecutarComando{}, aErr
		}
		if !ok {
			return tools.RespuestaEjecutarComando{}, nuevoError(CodigoAprobacionDeclinada,
				"el usuario declinó ejecutar `"+comando+"`")
		}
	}

	programa, err := exec.LookPath(argv[0])
	if err != nil {
		return tools.RespuestaEjecutarComando{}, fmt.Errorf("no se encontró el programa %q: %w", argv[0], err)
	}

	ctxT, cancel := context.WithTimeout(ctx, e.limite())
	defer cancel()

	cmd := e.comando(ctxT, programa, argv)
	cmd.Dir = dir
	captura := nuevaCaptura(e.limiteSalida())
	cmd.Stdout = captura.Stdout()
	cmd.Stderr = captura.Stderr()

	errEjec := cmd.Run()
	salida, errorSalida := captura.Salidas()
	res := tools.RespuestaEjecutarComando{
		Salida:   salida,
		Error:    errorSalida,
		Termino:  true,
		Truncado: captura.Truncado(),
	}
	if errEjec == nil {
		return res, nil
	}

	// El deadline se comprueba ANTES de mirar el código de salida: un proceso
	// matado por el timeout también llega como *exec.ExitError, y no es un
	// comando que falló, es un comando que se cortó.
	if errors.Is(ctxT.Err(), context.DeadlineExceeded) {
		res.Termino = false
		res.Truncado = true
		return res, nuevoError(CodigoComandoAgotado,
			fmt.Sprintf("el comando superó el límite de %s y se cortó", e.limite()))
	}
	var exitErr *exec.ExitError
	if errors.As(errEjec, &exitErr) {
		// Falló el comando, no la herramienta.
		res.Codigo = exitErr.ExitCode()
		return res, nil
	}
	return res, fmt.Errorf("no se pudo ejecutar el comando: %w", errEjec)
}

// comando construye el *exec.Cmd. Con Landlock disponible, el proceso hijo es
// el propio binario en modo trampolín: aplica el bloqueo y luego exec el
// comando real. Sin Landlock, se lanza directamente.
func (e *Ejecutor) comando(ctx context.Context, programa string, argv []string) *exec.Cmd {
	resto := argv[1:]
	if soportaLandlock() {
		if exe, err := os.Executable(); err == nil {
			args := append([]string{programa}, resto...)
			cmd := exec.CommandContext(ctx, exe, args...)
			cmd.Env = append(os.Environ(),
				envTrampolin+"=1",
				envPermitidos+"="+strings.Join(e.espacioEscritura(), string(os.PathListSeparator)))
			return cmd
		}
	}
	return exec.CommandContext(ctx, programa, resto...)
}

// rutasPermitidasEscritura es el propio espacio del comando: temporales y caché
// de compilación (y de pruebas). El proyecto nunca está aquí, así que el
// bloqueo lo cubre.
func rutasPermitidasEscritura() []string {
	vistas := map[string]bool{}
	var out []string
	añadir := func(p string) {
		if p == "" || vistas[p] {
			return
		}
		vistas[p] = true
		out = append(out, p)
	}
	añadir(os.TempDir())
	añadir(os.Getenv("GOTMPDIR"))
	if cache := os.Getenv("GOCACHE"); cache != "" {
		añadir(cache)
	} else if uc, err := os.UserCacheDir(); err == nil {
		añadir(filepath.Join(uc, "go-build"))
	}
	return out
}

// resolverCarpeta comprueba que la carpeta de trabajo se queda dentro del
// proyecto. El comando corre en la carpeta del proyecto (TOOLS.md §5); una
// carpeta fuera se rechaza.
func resolverCarpeta(proyecto, carpeta string) (string, error) {
	if filepath.IsAbs(carpeta) {
		return "", nuevoError(CodigoArgumentosInvalidos,
			"la carpeta de trabajo debe ser relativa a la del proyecto")
	}
	raiz, err := filepath.Abs(proyecto)
	if err != nil {
		return "", nuevoError(CodigoArgumentosInvalidos,
			"no se pudo resolver la carpeta del proyecto: "+err.Error())
	}
	destino := filepath.Clean(filepath.Join(raiz, carpeta))
	rel, err := filepath.Rel(raiz, destino)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", nuevoError(CodigoArgumentosInvalidos,
			"la carpeta de trabajo sale del proyecto")
	}
	return destino, nil
}
