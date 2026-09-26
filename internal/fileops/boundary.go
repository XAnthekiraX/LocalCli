package fileops

// boundary.go — T-B008-01: la frontera de rutas.
//
// Fuente de verdad: ai/docs/backend/03-security/SECURITY.md §3 ("Toda ruta es
// relativa a la carpeta abierta del proyecto. Dentro de la carpeta: accesible.
// Fuera de la carpeta: hace falta permiso y la explicación del agente") y
// ai/docs/backend/05-quality/VALIDATION.md §1 ("Se normaliza la ruta antes de
// validar, para que `../` o rutas equivalentes no esquiven la frontera").
//
// Aquí vive la frontera de la carpeta del proyecto. Se normaliza primero y se
// comprueba después, y una ruta absoluta se rechaza en vez de reinterpretarse:
// la ruta es siempre relativa a la carpeta abierta. Los enlaces simbólicos se
// resuelven para que un enlace de dentro que apunte fuera no sea una puerta
// trasera.

import (
	"os"
	"path/filepath"
	"strings"
)

// Resolver normaliza una ruta relativa y devuelve su ruta absoluta dentro del
// proyecto. Devuelve E_PATH_OUTSIDE si la ruta es absoluta, si escapa con `..`
// o si un enlace simbólico la lleva fuera.
func Resolver(proyecto, ruta string) (string, error) {
	if strings.TrimSpace(ruta) == "" {
		return "", nuevoError(CodigoArgumentosInvalidos, "falta la ruta")
	}
	if filepath.IsAbs(ruta) {
		return "", nuevoError(CodigoRutaFuera,
			"la ruta "+ruta+" es absoluta; toda ruta es relativa a la carpeta del proyecto")
	}
	proyectoAbs, err := filepath.Abs(proyecto)
	if err != nil {
		return "", nuevoError(CodigoRutaFuera,
			"no se pudo resolver la carpeta del proyecto: "+err.Error())
	}
	proyectoReal := proyectoAbs
	if real, err := filepath.EvalSymlinks(proyectoAbs); err == nil {
		proyectoReal = real
	}

	destino := filepath.Clean(filepath.Join(proyectoAbs, ruta))
	if !DentroDe(proyectoAbs, destino) {
		return "", nuevoError(CodigoRutaFuera,
			"la ruta "+ruta+" sale de la carpeta del proyecto")
	}
	if err := comprobarEnlaces(proyectoReal, destino); err != nil {
		return "", err
	}
	return destino, nil
}

// DentroDe informa si destino está dentro de raíz o es la propia raíz. No
// depende de que la ruta exista.
func DentroDe(raiz, destino string) bool {
	rel, err := filepath.Rel(raiz, destino)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	if rel == ".." || filepath.IsAbs(rel) {
		return false
	}
	return !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// comprobarEnlaces resuelve el ancestro existente más profundo de destino y
// comprueba que su ruta real sigue estando dentro del proyecto real. Así un
// enlace simbólico intermedio o final que apunte fuera se detecta.
func comprobarEnlaces(proyectoReal, destino string) error {
	actual := destino
	for {
		if _, err := os.Lstat(actual); err == nil {
			break
		}
		padre := filepath.Dir(actual)
		if padre == actual {
			return nuevoError(CodigoRutaFuera,
				"no se pudo situar "+destino+" dentro del proyecto")
		}
		actual = padre
	}
	real, err := filepath.EvalSymlinks(actual)
	if err != nil {
		return nuevoError(CodigoRutaFuera,
			"no se pudo resolver "+actual+": "+err.Error())
	}
	if !DentroDe(proyectoReal, real) {
		return nuevoError(CodigoRutaFuera,
			"la ruta apunta fuera de la carpeta del proyecto")
	}
	return nil
}

// Relativa devuelve la ruta tal como se guarda en el historial: relativa a la
// carpeta del proyecto y con separadores normalizados.
func Relativa(proyecto, abs string) (string, error) {
	rel, err := filepath.Rel(proyecto, abs)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(rel), nil
}
