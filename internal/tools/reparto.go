package tools

// Repartir deriva el catálogo efectivo de un agente desde sus permisos: una
// herramienta cae en el reparto si su permiso está concedido. Las de permiso
// derivado (ejecutar_flujo) se resuelven por turno y no entran aquí.
func Repartir(permisos []string, catalogo []Herramienta) []Herramienta {
	concedidos := make(map[Permiso]bool, len(permisos))
	for _, p := range permisos {
		concedidos[Permiso(p)] = true
	}
	var efectivo []Herramienta
	for _, h := range catalogo {
		if h.Permiso == PermisoDerivado {
			continue
		}
		if concedidos[h.Permiso] {
			efectivo = append(efectivo, h)
		}
	}
	return efectivo
}
