package task

// validate.go — T-B004-05: reglas de validación del TODO.
//
// Regla dura de [[backend/DECISIONS]] [28]: `bloqueada_por` solo puede estar
// presente cuando `estado = bloqueada`. Además verificamos las invariantes
// estructurales mínimas para que un elemento entre al TODO:
//
//   - vocabulario cerrado (id, capa, accion, estado) — reutiliza validarVocabulario
//   - IDs de elementos sin duplicar dentro del mismo conjunto
//   - depende_de referencia IDs existentes (o rangos "T-B002..T-B014")
//   - bloqueada_por ⇒ estado == bloqueada; y al revés, bloqueada ⇒ bloqueada_por no vacío

import (
	"fmt"
	"strings"
)

// ValidarElemento comprueba el vocabulario y la regla de bloqueada_por de un
// elemento aislado.
func ValidarElemento(e Elemento) error {
	if err := e.validarVocabulario(); err != nil {
		return err
	}
	return ValidarReglaBloqueo(e)
}

// ValidarReglaBloqueo aplica la regla [28]: bloqueada_por solo con estado
// bloqueada (y un elemento bloqueado debe decir por qué).
func ValidarReglaBloqueo(e Elemento) error {
	if len(e.BloqueadaPor) > 0 && e.Estado != EstadoBloqueada {
		return fmt.Errorf("task: %s: bloqueada_por exige estado bloqueada (tiene %q)", e.ID, e.Estado)
	}
	if e.Estado == EstadoBloqueada && len(e.BloqueadaPor) == 0 {
		return fmt.Errorf("task: %s: estado bloqueada exige bloqueada_por no vacío", e.ID)
	}
	return nil
}

// esRangoDep reconoce una dependencia en forma de rango ("T-B002..T-B014"),
// usada en MAIN-TASKS.md por la tarea de validaciones finales.
func esRangoDep(s string) bool {
	partes := strings.Split(s, "..")
	if len(partes) != 2 {
		return false
	}
	for _, p := range partes {
		if _, ok := PrefijoDeID(p); !ok {
			return false
		}
	}
	return true
}

// ValidarConjunto valida un conjunto de elementos del TODO en bloque:
// vocabulario + regla de bloqueo en cada uno, IDs únicos y dependencias
// resolubles contra los IDs presentes (los rangos se expanden implícitamente
// y no se comprueban celda a celda).
func ValidarConjunto(elems []Elemento) error {
	vistos := map[string]bool{}
	ids := map[string]bool{}
	for _, e := range elems {
		if err := ValidarElemento(e); err != nil {
			return err
		}
		if vistos[e.ID] {
			return fmt.Errorf("task: id duplicado %q en el TODO", e.ID)
		}
		vistos[e.ID] = true
		ids[e.ID] = true
	}
	for _, e := range elems {
		for _, d := range e.DependeDe {
			if esRangoDep(d) {
				continue
			}
			if !ids[d] {
				return fmt.Errorf("task: %s depende de %q, que no existe en el TODO", e.ID, d)
			}
		}
		for _, b := range e.BloqueadaPor {
			if !ids[b] {
				return fmt.Errorf("task: %s bloqueada por %q, que no existe en el TODO", e.ID, b)
			}
		}
	}
	return nil
}
