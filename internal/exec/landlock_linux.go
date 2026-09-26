//go:build linux

package exec

// landlock_linux.go — T-B009-05: bloqueo de escritura con Landlock.
//
// Fuente de verdad: ai/docs/backend/DECISIONS.md ("Landlock para bloquear
// escritura en la terminal: garantía del kernel, sin privilegios ni binarios
// externos, y no se esquiva con `find -delete` ni redirecciones") y
// ai/docs/backend/03-security/SECURITY.md §3.
//
// Landlock restringe el proceso que lo aplica (restrict_self) y esa restricción
// es irreversible y heredada por sus hijos. Por eso no se aplica en el proceso
// padre: el comando se lanza re-ejecutando este mismo binario en modo trampolín
// (esta init), que aplica el bloqueo y después `exec` el comando real. Así el
// proyecto no es accesible para escritura por más indirecto que sea el comando.
//
// Se manejan los derechos de escritura (crear, borrar, escribir, truncar) y se
// conceden solo en las rutas permitidas (temporales y caché). Lo que no está
// concedido, queda denegado.

import (
	"fmt"
	"os"
	"strings"
	"unsafe"

	"golang.org/x/sys/unix"
)

// init es el trampolín: solo actúa en el proceso hijo marcado.
func init() {
	if os.Getenv(envTrampolin) != "1" {
		return
	}
	argv := os.Args[1:]
	if len(argv) == 0 {
		fmt.Fprintln(os.Stderr, "localcli: trampolín sin comando")
		os.Exit(126)
	}
	if err := aplicarLandlock(leerPermitidos(os.Getenv(envPermitidos))); err != nil {
		// Degradación avisada: se ejecuta igual, con garantía más débil
		// (ERRORS.md §2: "Falta Landlock → aviso de que la garantía de la
		// terminal es más débil; sigue funcionando").
		fmt.Fprintln(os.Stderr, err.Error())
	}
	if err := unix.Exec(argv[0], argv, entornoSinMarca(os.Environ())); err != nil {
		fmt.Fprintln(os.Stderr, "localcli: no se pudo ejecutar "+argv[0]+": "+err.Error())
		os.Exit(127)
	}
}

// soportaLandlock informa si el kernel tiene Landlock (ABI ≥ 1).
func soportaLandlock() bool {
	abi, err := abiLandlock()
	return err == nil && abi >= 1
}

// avisoDegradacion explica la garantía reducida cuando Landlock no está.
func avisoDegradacion() string {
	return "la terminal no está bloqueada estructuralmente: este kernel no tiene Landlock (E_NO_LANDLOCK)"
}

// abiLandlock consulta la versión del ABI.
func abiLandlock() (int, error) {
	r, _, e := unix.Syscall(unix.SYS_LANDLOCK_CREATE_RULESET, 0, 0,
		uintptr(unix.LANDLOCK_CREATE_RULESET_VERSION))
	if e != 0 {
		return 0, e
	}
	return int(r), nil
}

// derechosEscritura son los accesos que se manejan: todo lo que crea, modifica
// o borra. `TRUNCATE` (ABI 3) cubre dar de cero un archivo existente.
// `REFER` (ABI 2) se omite a propósito: sin él no se relajan las reglas, y los
// derechos de creación y borrado ya impiden mover archivos con efecto.
func derechosEscritura(abi int) uint64 {
	d := uint64(
		unix.LANDLOCK_ACCESS_FS_WRITE_FILE |
			unix.LANDLOCK_ACCESS_FS_REMOVE_FILE |
			unix.LANDLOCK_ACCESS_FS_REMOVE_DIR |
			unix.LANDLOCK_ACCESS_FS_MAKE_CHAR |
			unix.LANDLOCK_ACCESS_FS_MAKE_DIR |
			unix.LANDLOCK_ACCESS_FS_MAKE_REG |
			unix.LANDLOCK_ACCESS_FS_MAKE_SOCK |
			unix.LANDLOCK_ACCESS_FS_MAKE_FIFO |
			unix.LANDLOCK_ACCESS_FS_MAKE_BLOCK |
			unix.LANDLOCK_ACCESS_FS_MAKE_SYM)
	if abi >= 3 {
		d |= unix.LANDLOCK_ACCESS_FS_TRUNCATE
	}
	return d
}

// aplicarLandlock monta el ruleset, concede escritura en las rutas permitidas y
// se restringe. Devuelve un error tipado E_NO_LANDLOCK si no se puede.
func aplicarLandlock(permitidos []string) error {
	abi, err := abiLandlock()
	if err != nil || abi < 1 {
		return nuevoError(CodigoSinLandlock,
			"Landlock no está disponible en este sistema; la garantía de la terminal es más débil")
	}
	acceso := derechosEscritura(abi)
	attr := unix.LandlockRulesetAttr{Access_fs: acceso}
	fd, _, e := unix.Syscall(unix.SYS_LANDLOCK_CREATE_RULESET,
		uintptr(unsafe.Pointer(&attr)), unsafe.Sizeof(attr), 0)
	if e != 0 {
		return nuevoError(CodigoSinLandlock,
			"no se pudo crear el ruleset de Landlock: "+e.Error())
	}
	defer unix.Close(int(fd))

	for _, ruta := range permitidos {
		rule := unix.LandlockPathBeneathAttr{Allowed_access: acceso}
		fdRuta, err := unix.Open(ruta, unix.O_PATH|unix.O_CLOEXEC, 0)
		if err != nil {
			continue // una ruta opcional que no existe no bloquea el resto
		}
		rule.Parent_fd = int32(fdRuta)
		_, _, e := unix.Syscall6(unix.SYS_LANDLOCK_ADD_RULE, fd,
			uintptr(unix.LANDLOCK_RULE_PATH_BENEATH), uintptr(unsafe.Pointer(&rule)), 0, 0, 0)
		unix.Close(fdRuta)
		if e != 0 {
			return nuevoError(CodigoSinLandlock,
				"no se pudo conceder escritura en "+ruta+": "+e.Error())
		}
	}

	if err := unix.Prctl(unix.PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0); err != nil {
		return nuevoError(CodigoSinLandlock,
			"no se pudo activar no_new_privs: "+err.Error())
	}
	if _, _, e := unix.Syscall(unix.SYS_LANDLOCK_RESTRICT_SELF, fd, 0, 0); e != 0 {
		return nuevoError(CodigoSinLandlock,
			"no se pudo aplicar el bloqueo de Landlock: "+e.Error())
	}
	return nil
}

// leerPermitidos parte la lista que dejó el proceso padre.
func leerPermitidos(valor string) []string {
	if valor == "" {
		return nil
	}
	return strings.Split(valor, string(os.PathListSeparator))
}

// entornoSinMarca quita las variables del trampolín para que un comando que
// invoque a LocalCli no herede el modo trampolín.
func entornoSinMarca(env []string) []string {
	out := make([]string, 0, len(env))
	for _, e := range env {
		if strings.HasPrefix(e, envTrampolin+"=") || strings.HasPrefix(e, envPermitidos+"=") {
			continue
		}
		out = append(out, e)
	}
	return out
}
