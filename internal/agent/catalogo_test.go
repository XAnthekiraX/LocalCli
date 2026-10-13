package agent

import "testing"

func catalogoDePrueba() []Herramienta {
	return []Herramienta{
		{Nombre: "leer_archivo", Permiso: PermisoRead},
		{Nombre: "ejecutar_comando", Permiso: PermisoRead},
		{Nombre: "crear_archivo", Permiso: PermisoWrite},
		{Nombre: "eliminar_archivo", Permiso: PermisoWrite},
		{Nombre: "editar_archivo", Permiso: PermisoEdit},
	}
}

func TestCatalogoEfectivoRespetaLosPermisos(t *testing.T) {
	catalogo := catalogoDePrueba()
	plan := Agente{Nombre: "plan", Permissions: map[string]string{"read": EfectoAllow}}
	build := Agente{Nombre: "build", Permissions: map[string]string{
		"read": EfectoAllow, "write": EfectoAllow, "edit": EfectoAllow,
	}}

	efectivo := CatalogoEfectivo(plan, catalogo)
	if len(efectivo) != 2 {
		t.Fatalf("plan solo debería tener las de lectura: %+v", efectivo)
	}
	for _, h := range efectivo {
		if h.Permiso != PermisoRead {
			t.Fatalf("plan no debería tener herramientas de escritura: %+v", h)
		}
	}

	if len(CatalogoEfectivo(build, catalogo)) != len(catalogo) {
		t.Fatal("build debería tener el catálogo completo")
	}

	mixto := Agente{Nombre: "m", Permissions: map[string]string{"read": EfectoAllow, "write": EfectoAllow}}
	for _, h := range CatalogoEfectivo(mixto, catalogo) {
		if h.Permiso == PermisoEdit {
			t.Fatalf("un permiso denegado no debería aparecer: %+v", h)
		}
	}
}
