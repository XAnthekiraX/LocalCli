//go:build !linux

package exec

import "os/exec"

// aislar no hace nada fuera de Linux: no hay Landlock, así que el comando corre
// directamente y la garantía es más débil (aviso E_NO_LANDLOCK). No se finge un
// aislamiento que no existe.
func aislar(*exec.Cmd) {}

func soportaLandlock() bool { return false }
