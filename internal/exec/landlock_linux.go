//go:build linux

package exec

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"unsafe"

	"golang.org/x/sys/unix"

	"localcli/internal/tools"
)

// init es el trampolín: solo actúa en el proceso hijo marcado. Aplica el bloqueo
// y después reemplaza el proceso por el comando real. La restricción de Landlock
// es irreversible y heredada por los hijos, por eso no se aplica en el padre.
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
		fmt.Fprintln(os.Stderr, err.Error())
	}
	if err := unix.Exec(argv[0], argv, entornoSinMarca(os.Environ())); err != nil {
		fmt.Fprintln(os.Stderr, "localcli: no se pudo ejecutar "+argv[0]+": "+err.Error())
		os.Exit(127)
	}
}

// aislar reescribe el comando para que corra bajo el trampolín de Landlock.
func aislar(cmd *exec.Cmd) {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	cmd.Path = exe
	cmd.Args = append([]string{exe}, cmd.Args...)
	cmd.Env = append(os.Environ(),
		envTrampolin+"=1",
		envPermitidos+"="+strings.Join(rutasPermitidas(), string(os.PathListSeparator)))
}

func soportaLandlock() bool {
	abi, err := abiLandlock()
	return err == nil && abi >= 1
}

func abiLandlock() (int, error) {
	r, _, e := unix.Syscall(unix.SYS_LANDLOCK_CREATE_RULESET, 0, 0,
		uintptr(unix.LANDLOCK_CREATE_RULESET_VERSION))
	if e != 0 {
		return 0, e
	}
	return int(r), nil
}

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
// se restringe. Devuelve E_NO_LANDLOCK si el kernel no lo tiene.
func aplicarLandlock(permitidos []string) error {
	abi, err := abiLandlock()
	if err != nil || abi < 1 {
		return nuevaError(tools.CodNoLandlock,
			"Landlock no está disponible en este sistema; la garantía de la terminal es más débil")
	}
	acceso := derechosEscritura(abi)
	attr := unix.LandlockRulesetAttr{Access_fs: acceso}
	fd, _, e := unix.Syscall(unix.SYS_LANDLOCK_CREATE_RULESET,
		uintptr(unsafe.Pointer(&attr)), unsafe.Sizeof(attr), 0)
	if e != 0 {
		return nuevaError(tools.CodNoLandlock, "no se pudo crear el ruleset de Landlock: "+e.Error())
	}
	defer unix.Close(int(fd))

	for _, ruta := range permitidos {
		concederRuta(fd, ruta, acceso)
	}

	if err := unix.Prctl(unix.PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0); err != nil {
		return nuevaError(tools.CodNoLandlock, "no se pudo activar no_new_privs: "+err.Error())
	}
	if _, _, e := unix.Syscall(unix.SYS_LANDLOCK_RESTRICT_SELF, fd, 0, 0); e != 0 {
		return nuevaError(tools.CodNoLandlock, "no se pudo aplicar el bloqueo de Landlock: "+e.Error())
	}
	return nil
}

func concederRuta(ruleset uintptr, ruta string, acceso uint64) {
	fdRuta, err := unix.Open(ruta, unix.O_PATH|unix.O_CLOEXEC, 0)
	if err != nil {
		return
	}
	defer unix.Close(fdRuta)
	// Sobre un archivo (p. ej. /dev/null) solo aplican los derechos de archivo;
	// los de directorio hacen que el kernel rechace la regla con EINVAL.
	regla := acceso
	var st unix.Stat_t
	if err := unix.Fstat(fdRuta, &st); err == nil && st.Mode&unix.S_IFMT != unix.S_IFDIR {
		regla &= unix.LANDLOCK_ACCESS_FS_WRITE_FILE | unix.LANDLOCK_ACCESS_FS_TRUNCATE
	}
	rule := unix.LandlockPathBeneathAttr{Parent_fd: int32(fdRuta), Allowed_access: regla}
	unix.Syscall6(unix.SYS_LANDLOCK_ADD_RULE, ruleset,
		uintptr(unix.LANDLOCK_RULE_PATH_BENEATH), uintptr(unsafe.Pointer(&rule)), 0, 0, 0)
}
