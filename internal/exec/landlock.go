package exec

import (
	"os"
	"strings"
)

const (
	envTrampolin  = "LOCALCLI_LANDLOCK"
	envPermitidos = "LOCALCLI_LANDLOCK_RUTAS"
)

// rutasPermitidas son las rutas con escritura concedida: el espacio propio del
// comando (temporales y caché) y los dispositivos nulos. El resto del sistema,
// incluido el proyecto, queda denegado.
func rutasPermitidas() []string {
	rutas := []string{os.TempDir(), "/dev/null", "/dev/zero", "/dev/full"}
	if cache, err := os.UserCacheDir(); err == nil {
		rutas = append(rutas, cache)
	}
	return rutas
}

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
