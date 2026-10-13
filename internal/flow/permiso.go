package flow

import "sort"

// PermisoNecesario devuelve los permisos que hay que conceder para lanzar el
// flujo: `read` si alguna etapa corre con `plan`, `write` si alguna con `build`.
func PermisoNecesario(f Flujo) []string {
	necesarios := map[string]bool{}
	for _, e := range f.Etapas {
		if e.Agente == "build" {
			necesarios["write"] = true
		} else {
			necesarios["read"] = true
		}
	}
	salida := make([]string, 0, len(necesarios))
	for p := range necesarios {
		salida = append(salida, p)
	}
	sort.Strings(salida)
	return salida
}

// PuedeLanzar dice si un agente con esos permisos puede lanzar el flujo: ha de
// tener todos los permisos que el flujo necesita.
func PuedeLanzar(f Flujo, permisos []string) bool {
	concedidos := map[string]bool{}
	for _, p := range permisos {
		concedidos[p] = true
	}
	for _, p := range PermisoNecesario(f) {
		if !concedidos[p] {
			return false
		}
	}
	return true
}
