package task

// deps.go — T-B004-06: resolución de dependencias y orden del TODO.
//
// El orden de trabajo lo dan las dependencias (`depende_de`); el prefijo NNN
// del ID es el desempate documentado (SPEC-CICLO-TRABAJO: "orden por prefijo
// como desempate"). Implementamos un orden topológico estable tipo Kahn: en
// cada paso se toma el elegible con NNN menor (y, dentro del mismo archivo,
// el orden de aparición), de modo que la salida es determinista para un
// mismo TODO.
//
// Un elemento queda "bloqueado" cuando alguna de sus dependencias no está
// completada; eso lo consume la cola (T-B012), pero la clasificación vive
// aquí junto al ordenamiento.

import (
	"fmt"
	"sort"
)

// OrdenarTopologico devuelve los elementos en orden de ejecución: dependencias
// antes que dependientes, desempates por prefijo NNN y posición original.
// Error si hay ciclos o dependencias inexistentes (los rangos "A..B" se
// ignoran a efectos de aristas, igual que en validate.go).
func OrdenarTopologico(elems []Elemento) ([]Elemento, error) {
	byID := make(map[string]Elemento, len(elems))
	pos := make(map[string]int, len(elems))
	for i, e := range elems {
		if _, dup := byID[e.ID]; dup {
			return nil, fmt.Errorf("task: id duplicado %q", e.ID)
		}
		byID[e.ID] = e
		pos[e.ID] = i
	}

	type nodo struct {
		elem      Elemento
		prefijo   string
		pos       int
		salientes []string // IDs que dependen de este
		grados    int      // dependencias pendientes de resolver
	}
	nodos := make([]*nodo, 0, len(elems))
	idx := map[string]*nodo{}
	for _, e := range elems {
		p, _ := PrefijoDeID(e.ID)
		n := &nodo{elem: e, prefijo: p, pos: pos[e.ID]}
		nodos = append(nodos, n)
		idx[e.ID] = n
	}
	for _, n := range nodos {
		for _, d := range n.elem.DependeDe {
			if esRangoDep(d) {
				// Un rango es una abreviatura por aristas reales. Sin
				// expandirlas, el nodo con un rango tiene grado 0 y puede salir
				// en el orden ANTES que las tareas que lo bloquean, que es justo
				// lo que el orden topológico debe impedir.
				for _, otro := range nodos {
					if !esTareaGrande(otro.elem.ID) || !dentroDeRango(d, otro.elem.ID) {
						continue
					}
					if otro.elem.ID == n.elem.ID {
						continue
					}
					otro.salientes = append(otro.salientes, n.elem.ID)
					n.grados++
				}
				continue
			}
			dn, ok := idx[d]
			if !ok {
				return nil, fmt.Errorf("task: %s depende de %q, inexistente", n.elem.ID, d)
			}
			dn.salientes = append(dn.salientes, n.elem.ID)
			n.grados++
		}
	}

	menor := func(a, b *nodo) bool {
		if a.prefijo != b.prefijo {
			return a.prefijo < b.prefijo
		}
		return a.pos < b.pos
	}

	var disponibles []*nodo
	for _, n := range nodos {
		if n.grados == 0 {
			disponibles = append(disponibles, n)
		}
	}
	out := make([]Elemento, 0, len(elems))
	for len(disponibles) > 0 {
		sort.Slice(disponibles, func(i, j int) bool { return menor(disponibles[i], disponibles[j]) })
		n := disponibles[0]
		disponibles = disponibles[1:]
		out = append(out, n.elem)
		for _, s := range n.salientes {
			sn := idx[s]
			sn.grados--
			if sn.grados == 0 {
				disponibles = append(disponibles, sn)
			}
		}
	}
	if len(out) != len(elems) {
		var restantes []string
		for _, n := range nodos {
			if n.grados > 0 {
				restantes = append(restantes, n.elem.ID)
			}
		}
		return nil, fmt.Errorf("task: ciclo de dependencias entre %v", restantes)
	}
	return out, nil
}

// Completado informa si el estado cuenta como término de la dependencia.
func Completado(e Elemento) bool { return e.Estado == EstadoCompletada }

// Elegibles devuelve, del conjunto dado, los elementos pendientes/en_progreso
// cuyas dependencias están todas completadas, en orden topológico. Es la
// vista que consumirá la cola (T-B012): bloqueados quedan fuera.
func Elegibles(elems []Elemento) ([]Elemento, error) {
	orden, err := OrdenarTopologico(elems)
	if err != nil {
		return nil, err
	}
	estado := map[string]Estado{}
	for _, e := range elems {
		estado[e.ID] = e.Estado
	}
	var out []Elemento
	for _, e := range orden {
		if e.Estado != EstadoPendiente && e.Estado != EstadoEnProgreso {
			continue
		}
		listo := true
		for _, d := range e.DependeDe {
			if esRangoDep(d) {
				// Un rango exige que TODAS las tareas grandes del rango estén
				// completadas: se comprueban contra los IDs presentes.
				for _, x := range elems {
					if !Completado(x) && dentroDeRango(d, x.ID) {
						listo = false
						break
					}
				}
			} else if estado[d] != EstadoCompletada {
				listo = false
			}
			if !listo {
				break
			}
		}
		if listo {
			out = append(out, e)
		}
	}
	return out, nil
}

// dentroDeRango informa si `id` cae dentro del rango "T-Bxxx..T-BYYY".
//
// Se comparan NNN y letra de capa. Comparar solo los dígitos hacía que un
// rango de backend cumpliera el requisito con una tarea de frontend: T-F003
// contaba como "dentro de T-B002..T-B007" y una dependencia nunca se daba por
// rota. El rango pertenece a una capa, no solo a un intervalo de números.
func dentroDeRango(rango, id string) bool {
	partes := splitOnce(rango, "..")
	if partes == nil {
		return false
	}
	a, okA := PrefijoDeID(partes[0])
	b, okB := PrefijoDeID(partes[1])
	x, okX := PrefijoDeID(id)
	if !okA || !okB || !okX {
		return false
	}
	ca, okCA := CapaDeID(partes[0])
	cb, okCB := CapaDeID(partes[1])
	cx, okCX := CapaDeID(id)
	if !okCA || !okCB || !okCX {
		return false
	}
	// Un rango no puede cruzar de capa: T-B002..T-F007 no significa nada.
	if ca != cb {
		return false
	}
	return ca == cx && a <= x && x <= b
}

// splitOnce divide s por la primera aparición de sep.
func splitOnce(s, sep string) []string {
	for i := 0; i+len(sep) <= len(s); i++ {
		if s[i:i+len(sep)] == sep {
			return []string{s[:i], s[i+len(sep):]}
		}
	}
	return nil
}
