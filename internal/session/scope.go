// scope.go — T-B013-03: aislamiento por carpeta.
//
// Fuente de verdad: SPEC-SESIONES §Reglas ("Una sesión pertenece a un solo
// proyecto. El proyecto es la carpeta abierta", "El chat de una carpeta nunca
// aparece en otra carpeta") y §Requisitos no funcionales ("no hay fuga de
// contenido entre proyectos ni entre sesiones del mismo proyecto").
//
// El aislamiento no se implementa filtrando: se implementa por construcción.
// Cada carpeta tiene su propio archivo SQLite en `.localcli/state.db`
// (DECISIONS.md), así que dos proyectos no comparten ni una fila y no hay
// consulta que pueda mezclarlos. Lo que sí hay que comprobar es lo contrario:
// que el almacén que se le pasa al gestor es de verdad el de esa carpeta, y que
// una sesión que llega de fuera pertenece a este alcance.
package session

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"localcli/internal/store"
)

// Alcance es el proyecto abierto: su carpeta y el almacén de esa carpeta.
type Alcance struct {
	Carpeta string
	Almacen Almacen
}

// NuevoAlcance valida la carpeta y ata el almacén a ella.
func NuevoAlcance(carpeta string, almacen Almacen) (*Alcance, error) {
	a := &Alcance{Carpeta: carpeta, Almacen: almacen}
	if err := a.Validar(); err != nil {
		return nil, err
	}
	return a, nil
}

// Validar comprueba que hay carpeta y almacén. La carpeta tiene que existir:
// un alcance sin carpeta de verdad acabaría creando estado en un sitio que no
// toca.
func (a *Alcance) Validar() error {
	if a.Almacen == nil {
		return fmt.Errorf("session: el alcance no tiene almacén")
	}
	if a.Carpeta == "" {
		return fmt.Errorf("session: el alcance no tiene carpeta de proyecto")
	}
	info, err := os.Stat(a.Carpeta)
	if err != nil {
		return fmt.Errorf("session: carpeta %s inaccesible: %w", a.Carpeta, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("session: %s no es una carpeta", a.Carpeta)
	}
	return nil
}

// RutaBase es la ruta del archivo SQLite de este alcance. Existe para poder
// comprobar en tests, y en el wiring, que dos carpetas no comparten base.
func (a *Alcance) RutaBase() (string, error) { return store.DBPath(a.Carpeta) }

// Contiene dice si una sesión pertenece a este alcance. Se comprueba leyéndola
// de su almacén: si está aquí, es de aquí; si no, no se toca.
func (a *Alcance) Contiene(sesionID string) (bool, error) {
	if _, err := a.Almacen.Obtener(sesionID); err != nil {
		if errors.Is(err, store.ErrNoEncontrado) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// MismaCarpeta dice si dos rutas son la misma carpeta del proyecto.
func MismaCarpeta(a, b string) bool {
	ra, errA := filepath.Abs(a)
	rb, errB := filepath.Abs(b)
	if errA != nil || errB != nil {
		return a == b
	}
	ra, errA = filepath.EvalSymlinks(ra)
	rb, errB = filepath.EvalSymlinks(rb)
	if errA != nil || errB != nil {
		return a == b
	}
	return ra == rb
}
