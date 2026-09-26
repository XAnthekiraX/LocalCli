// block.go — T-B012-04: qué elementos no pueden arrancar y qué falta.
//
// Fuente de verdad: SPEC-COLA-TAREAS §Flujos alternativos ("Un elemento queda
// bloqueado: se marca y se informa qué falta. Si nada depende de él, la cola
// sigue con el siguiente"), §Reglas ("Un elemento bloqueado no detiene la cola
// si nada depende de él") y ERRORS.md §5 ("Cola: E_ELEMENTO_BLOQUEADO; la cola
// no es un error, sigue con otra").
//
// Por eso esto devuelve información, no un error: la cola informa del bloqueo y
// continúa con el siguiente elemento elegible. El bloqueo se propaga: un
// elemento que espera a otro bloqueado también queda bloqueado, y su motivo lo
// dice, para que el aviso no sea "falta T-B004" cuando el problema real está más
// atrás.
package queue

import (
	"strings"

	"localcli/internal/task"
)

// Bloqueo es un elemento que no puede arrancar, con lo que le falta.
type Bloqueo struct {
	// ID del elemento bloqueado.
	ID string
	// Falta son los IDs de las dependencias sin completar.
	Falta []string
	// Motivo explica para la persona qué falta y por qué.
	Motivo string
}

// Codigo devuelve el código documentado de este caso (ERRORS.md §3).
func (b Bloqueo) Codigo() string { return CodigoElementoBloqueado }

// Bloqueados calcula, en orden de ejecución, los elementos que no pueden
// arrancar por dependencias. Los completados y los que ya están en progreso
// quedan fuera: un elemento empezado se retoma sin repetir lo hecho, que es lo
// contrario de bloquearse.
func Bloqueados(elems []task.Elemento) []Bloqueo {
	orden, err := Ordenar(elems)
	if err != nil {
		// Un TODO con ciclo no tiene orden: se informa en el orden de lectura
		// en vez de perder el aviso de bloqueo.
		orden = elems
	}

	bloqueoDe := map[string]Bloqueo{}
	var out []Bloqueo
	for _, e := range orden {
		if e.Estado == task.EstadoCompletada || e.Estado == task.EstadoEnProgreso {
			continue
		}
		if e.Estado == task.EstadoBloqueada {
			// El archivo ya lo declara bloqueado y dice por qué.
			b := Bloqueo{
				ID:     e.ID,
				Falta:  e.BloqueadaPor,
				Motivo: "declarado bloqueado en el TODO por " + strings.Join(e.BloqueadaPor, ", "),
			}
			bloqueoDe[e.ID] = b
			out = append(out, b)
			continue
		}
		faltan := task.Faltantes(elems, e)
		if len(faltan) == 0 {
			continue
		}
		motivo := "falta completar " + strings.Join(faltan, ", ")
		if previo, ok := bloqueoDe[faltan[0]]; ok {
			motivo = "depende de " + faltan[0] + ", que está bloqueado: " + previo.Motivo
		}
		b := Bloqueo{ID: e.ID, Falta: faltan, Motivo: motivo}
		bloqueoDe[e.ID] = b
		out = append(out, b)
	}
	return out
}

// Vista construye el estado de la cola: el orden, los bloqueos con su motivo y
// cuánto queda. Es lo que se muestra desde cualquier sesión.
func (c *Cola) Vista() Vista {
	elems := c.Elementos()
	v := Vista{Capa: c.capa, Orden: elems, Bloqueados: Bloqueados(elems)}
	for _, e := range elems {
		switch e.Estado {
		case task.EstadoCompletada:
			v.Completados++
		case task.EstadoEnProgreso:
			v.Pendientes++
		case task.EstadoPendiente, task.EstadoBloqueada:
			v.Pendientes++
		}
	}
	if elegibles, err := task.Elegibles(elems); err == nil && len(elegibles) > 0 {
		v.Elegible = elegibles[0].ID
	}
	return v
}
