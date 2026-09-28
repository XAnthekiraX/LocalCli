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
	"bytes"
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
	// usa temporales y caché de compilación. Los dispositivos nulos se añaden
	// siempre, tengan o no valor propio.
	EspacioPropio []string
}

func (e *Ejecutor) espacioEscritura() []string {
	base := e.EspacioPropio
	if len(base) == 0 {
		base = rutasPermitidasEscritura()
	}
	return unirRutas(base, dispositivosNulos)
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

// EjecutarEquipo lanza un equipo declarado por el usuario como subproceso argv,
// sin shell y sin lista blanca. Es la misma maquinaria que la terminal: si el
// bloqueo estructural está disponible (Landlock), el proceso corre con él.
//
// La aprobación NO se pide aquí: quien ejecuta una herramienta del usuario ya
// la pidió por `Contexto.Ask`, porque el comando lo escribió el usuario y nunca
// puede estar en la lista blanca (SPEC-TOOLS §Herramientas del usuario). Los
// argumentos del modelo llegan por la entrada estándar como un único JSON, de
// modo que no pueden inyectar un programa ni un argumento nuevo.
func (e *Ejecutor) EjecutarEquipo(ctx context.Context, equipo []string, entrada []byte, timeout time.Duration) (tools.RespuestaEjecutarComando, error) {
	if len(equipo) == 0 {
		return tools.RespuestaEjecutarComando{}, fmt.Errorf("exec: el equipo está vacío")
	}
	programa, err := exec.LookPath(equipo[0])
	if err != nil {
		return tools.RespuestaEjecutarComando{}, fmt.Errorf("no se encontró el programa %q: %w", equipo[0], err)
	}
	limite := e.limite()
	if timeout > 0 {
		limite = timeout
	}
	ctxT, cancel := context.WithTimeout(ctx, limite)
	defer cancel()

	cmd := e.comando(ctxT, programa, equipo)
	cmd.Dir = e.Proyecto
	if len(entrada) > 0 {
		cmd.Stdin = bytes.NewReader(entrada)
	}
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
	if errors.Is(ctxT.Err(), context.DeadlineExceeded) {
		res.Termino = false
		res.Truncado = true
		return res, nil
	}
	var exitErr *exec.ExitError
	if errors.As(errEjec, &exitErr) {
		res.Codigo = exitErr.ExitCode()
		return res, nil
	}
	return res, fmt.Errorf("no se pudo ejecutar el equipo: %w", errEjec)
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

// dispositivosNulos son archivos de dispositivo sin almacenamiento: escribir en
// ellos no toca el proyecto ni crea archivos. El sandbox los deja abiertos para
// escritura porque programas de la lista blanca (git) abren /dev/null con O_RDWR
// aunque no escriban nada. La regla se ancla en el propio archivo, así que no
// concede escritura en el resto de /dev. Ver SECURITY.md §3.
var dispositivosNulos = []string{"/dev/null", "/dev/zero", "/dev/full"}

// unirRutas concatena listas de rutas sin repetir y sin vacíos, preservando el
// orden.
func unirRutas(listas ...[]string) []string {
	vistas := map[string]bool{}
	var out []string
	for _, lista := range listas {
		for _, p := range lista {
			if p == "" || vistas[p] {
				continue
			}
			vistas[p] = true
			out = append(out, p)
		}
	}
	return out
}

// rutasPermitidasEscritura es el propio espacio del comando: temporales y caché
// de compilación (y de pruebas). El proyecto nunca está aquí, así que el
// bloqueo lo cubre.
func rutasPermitidasEscritura() []string {
	var propias []string
	propias = append(propias, os.TempDir())
	propias = append(propias, os.Getenv("GOTMPDIR"))
	if cache := os.Getenv("GOCACHE"); cache != "" {
		propias = append(propias, cache)
	} else if uc, err := os.UserCacheDir(); err == nil {
		propias = append(propias, filepath.Join(uc, "go-build"))
	}
	return unirRutas(propias)
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
