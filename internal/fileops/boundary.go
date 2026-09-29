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

// Resolver normaliza la ruta contra la carpeta del proyecto y dice si queda
// fuera de ella. Fuera NO es un error: es una operación que necesita aprobación
// —y la explicación del agente, si la trae— (SPEC-ARCHIVOS §Reglas); quien
// decide es quien puede pedir permiso. Solo la ruta vacía o un proyecto
// irresoluble son error.
//
// Una ruta absoluta se respeta tal cual: si no cuelga del proyecto, sale fuera.
// Una relativa cuelga de la carpeta abierta. Los enlaces simbólicos se resuelven
// para que un enlace de dentro que apunte fuera exija aprobación y no sea una
// puerta trasera.
func Resolver(proyecto, ruta string) (abs string, fuera bool, err error) {
	if strings.TrimSpace(ruta) == "" {
		return "", false, nuevoError(CodigoArgumentosInvalidos, "falta la ruta")
	}
	proyectoAbs, err := filepath.Abs(proyecto)
	if err != nil {
		return "", false, nuevoError(CodigoArgumentosInvalidos,
			"no se pudo resolver la carpeta del proyecto: "+err.Error())
	}
	proyectoReal := proyectoAbs
	if real, err := filepath.EvalSymlinks(proyectoAbs); err == nil {
		proyectoReal = real
	}

	destino := filepath.Clean(ruta)
	if !filepath.IsAbs(ruta) {
		destino = filepath.Clean(filepath.Join(proyectoAbs, ruta))
	}
	if !DentroDe(proyectoAbs, destino) {
		return destino, true, nil
	}
	if comprobarEnlaces(proyectoReal, destino) {
		return destino, true, nil
	}
	return destino, false, nil
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

// comprobarEnlaces resuelve el ancestro existente más profundo de destino y dice
// si su ruta real cae fuera del proyecto real. Así un enlace simbólico
// intermedio o final que apunte fuera se detecta y exige aprobación. Si no se
// puede resolver, se trata como fuera: mejor pedir permiso que colar una ruta.
func comprobarEnlaces(proyectoReal, destino string) bool {
	actual := destino
	for {
		if _, err := os.Lstat(actual); err == nil {
			break
		}
		padre := filepath.Dir(actual)
		if padre == actual {
			return true
		}
		actual = padre
	}
	real, err := filepath.EvalSymlinks(actual)
	if err != nil {
		return true
	}
	return !DentroDe(proyectoReal, real)
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
