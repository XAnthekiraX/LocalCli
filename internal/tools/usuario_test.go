package tools

import "testing"

func TestHerramientaDeUsuarioEsDeLectura(t *testing.T) {
	h := HerramientaUsuario{Nombre: "contar_lineas", Descripcion: "cuenta", Equipo: []string{"wc", "-l"}}.ComoHerramienta()
	if h.Permiso != PermisoRead || h.Modo != ModoLee || h.Categoria != CategoriaUsuario {
		t.Fatalf("una herramienta del usuario entra como lectura: %+v", h)
	}
	if h.Esquema().Tipo != "object" {
		t.Fatalf("su esquema es un objeto genérico: %+v", h.Esquema())
	}
}

func TestValidacionDeDeclaracion(t *testing.T) {
	valido := HerramientaUsuario{Nombre: "x", Descripcion: "d", Equipo: []string{"echo"}}
	if msg := valido.ValidaDeclaracion(); msg != "" {
		t.Fatalf("una declaración válida no debería fallar: %s", msg)
	}
	if msg := (HerramientaUsuario{Nombre: "x", Descripcion: "d", Equipo: []string{"e"}, Modo: ModoEscribe}).ValidaDeclaracion(); msg == "" {
		t.Fatal("`modo: escribe` debería rechazarse")
	}
	if msg := (HerramientaUsuario{Descripcion: "d", Equipo: []string{"e"}}).ValidaDeclaracion(); msg == "" {
		t.Fatal("sin nombre debería fallar")
	}
	if msg := (HerramientaUsuario{Nombre: "x", Descripcion: "d"}).ValidaDeclaracion(); msg == "" {
		t.Fatal("sin equipo debería fallar")
	}
}
