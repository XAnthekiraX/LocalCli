package agent

import "encoding/json"

// Herramienta es lo que el agente sabe de una herramienta: su nombre, la
// descripción que lee el modelo, el permiso que la abre y su esquema de argumentos.
type Herramienta struct {
	Nombre      string
	Descripcion string
	Permiso     string
	Esquema     json.RawMessage
}

// CatalogoEfectivo deriva las herramientas del agente desde sus permisos contra
// el catálogo cerrado: no hay una segunda lista que pueda contradecirlos.
func CatalogoEfectivo(a Agente, catalogo []Herramienta) []Herramienta {
	concedidos := make(map[string]bool)
	for _, p := range a.Permisos() {
		concedidos[p] = true
	}
	var efectivo []Herramienta
	for _, h := range catalogo {
		if concedidos[h.Permiso] {
			efectivo = append(efectivo, h)
		}
	}
	return efectivo
}
