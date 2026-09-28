package tools

import (
	"context"
	"testing"
)

// TestCatalogoTieneCatorceHerramientas — T-B007-01: el registro contiene
// exactamente las catorce herramientas documentadas. Una más o una menos
// rompería la garantía: una de más que escriba sería un agujero, y una de
// menos dejaría al agente sin poder trabajar.
func TestCatalogoTieneCatorceHerramientas(t *testing.T) {
	h := Herramientas()
	if len(h) != 14 {
		t.Fatalf("el catálogo tiene %d herramientas, quiero 14: %v", len(h), NombresCatalogo())
	}
}

// TestCatalogoNombresDocumentados — los catorce nombres exactos de TOOLS.md §1.
// El nombre es la clave con la que el modelo pide la herramienta y la que el
// JSON del agente declara, así que un nombre distinto rompe el contrato en
// los dos extremos.
func TestCatalogoNombresDocumentados(t *testing.T) {
	want := []string{
		"leer_archivo", "listar_carpeta", "buscar_archivos", "buscar_en_archivos",
		"crear_archivo", "escribir_archivo", "editar_archivo", "eliminar_archivo",
		"crear_carpeta", "eliminar_carpeta",
		"ejecutar_comando",
		"buscar_en_internet", "abrir_pagina",
		"actualizar_todo",
	}
	got := NombresCatalogo()
	if len(got) != len(want) {
		t.Fatalf("nombres=%v, quiero %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("nombre %d = %q, quiero %q", i, got[i], want[i])
		}
	}
}

// TestHerramientaInventadaNoExiste — el catálogo es cerrado: una herramienta
// que el modelo se inventa no existe.
func TestHerramientaInventadaNoExiste(t *testing.T) {
	if Existe("borrar_todo") {
		t.Error("una herramienta inventada no puede estar en el catálogo")
	}
	if _, ok := Buscar("borrar_todo"); ok {
		t.Error("Buscar debe fallar para una herramienta fuera del catálogo")
	}
}

// TestSoloBuildCorrespondeAEscritura — el reparto por agente sale de la acción
// `editar`: las seis que escriben en el proyecto son "solo build"; el resto
// —las de lectura y la lista de pasos de la sesión— son de ambos (TOOLS.md §1 y
// §2, SPEC-TOOLS §El reparto).
func TestSoloBuildCorrespondeAEscritura(t *testing.T) {
	var escritura, otros int
	for _, h := range Herramientas() {
		if h.SoloBuild() {
			escritura++
			if h.Accion() != AccionEditar {
				t.Errorf("%s: SoloBuild con acción %s", h.Nombre, h.Accion())
			}
			continue
		}
		otros++
		if h.Accion() == AccionEditar {
			t.Errorf("%s: herramienta de escritura que no es SoloBuild", h.Nombre)
		}
	}
	if escritura != 6 {
		t.Errorf("herramientas de escritura = %d, quiero 6 (crear, escribir, editar, eliminar archivo, crear y eliminar carpeta)", escritura)
	}
	if otros != 8 {
		t.Errorf("herramientas no exclusivas de build = %d, quiero 8 (4 de archivo, 1 de terminal, 2 de internet y la lista de pasos)", otros)
	}
}

// TestCategoriasRepartenLasHerramientas — la categoría decide el destino del
// enrutado (TOOLS.md §7): 10 de archivo, 1 de terminal, 2 de internet y 1 de
// sesión.
func TestCategoriasRepartenLasHerramientas(t *testing.T) {
	casos := []struct {
		cat  Categoria
		want int
	}{
		{CatArchivos, 10},
		{CatTerminal, 1},
		{CatInternet, 2},
		{CatTareas, 1},
	}
	for _, c := range casos {
		if got := len(NombresDeCategoria(c.cat)); got != c.want {
			t.Errorf("%s: %d herramientas, quiero %d", c.cat, got, c.want)
		}
	}
}

// TestHerramientasDevuelveCopia — quien recibe el catálogo no puede alterarlo
// y cambiar los permisos de todo el proceso.
func TestHerramientasDevuelveCopia(t *testing.T) {
	h := Herramientas()
	h[0].Nombre = "inventada"
	h[0].Modo = Escribe
	if Existe("inventada") {
		t.Error("el catálogo no puede alterarse desde fuera")
	}
	if Existe("leer_archivo") == false {
		t.Error("alterar la copia no puede borrar entradas reales")
	}
}

// TestAccionSeDerivaDeCategoriaYModo — el reparto por acción sale de un solo
// dato (categoría + modo), sin un campo aparte que pueda contradecirlo.
func TestAccionSeDerivaDeCategoriaYModo(t *testing.T) {
	casos := []struct {
		nombre string
		accion Accion
	}{
		{"leer_archivo", AccionLeer},
		{"buscar_en_archivos", AccionLeer},
		{"crear_archivo", AccionEditar},
		{"eliminar_carpeta", AccionEditar},
		{"ejecutar_comando", AccionEjecutar},
		{"buscar_en_internet", AccionInternet},
		{"abrir_pagina", AccionInternet},
	}
	for _, c := range casos {
		got, ok := AccionDe(c.nombre)
		if !ok {
			t.Errorf("%s no está en el catálogo", c.nombre)
			continue
		}
		if got != c.accion {
			t.Errorf("%s: acción = %s, quiero %s", c.nombre, got, c.accion)
		}
	}
}

// TestNombresDeAccionReparteLasHerramientas — las cinco acciones cubren las
// catorce herramientas sin solaparse: la partición que sostiene el catálogo
// derivado.
func TestNombresDeAccionReparteLasHerramientas(t *testing.T) {
	var total int
	for _, a := range []Accion{AccionLeer, AccionEditar, AccionEjecutar, AccionInternet, AccionTareas} {
		total += len(NombresDeAccion(a))
	}
	if total != len(NombresCatalogo()) {
		t.Errorf("las acciones cubren %d herramientas, quiero %d", total, len(NombresCatalogo()))
	}
	if len(NombresDeAccion(AccionEditar)) != 6 {
		t.Errorf("`editar` tiene %d herramientas, quiero 6", len(NombresDeAccion(AccionEditar)))
	}
	if len(NombresDeAccion(AccionTareas)) != 1 {
		t.Errorf("`tareas` tiene %d herramientas, quiero 1", len(NombresDeAccion(AccionTareas)))
	}
	if Accion("inventada").Valida() {
		t.Error("una acción fuera de las cinco no puede ser válida")
	}
}

// TestNuevaHerramientaConectaElHandler — el registro se construye desde el
// catálogo con su handler propio; un nombre fuera del catálogo no se conecta.
func TestNuevaHerramientaConectaElHandler(t *testing.T) {
	h, ok := NuevaHerramienta("leer_archivo", func(ctx context.Context, args any, c Contexto) (Resultado, error) {
		return Resultado{Salida: "x"}, nil
	})
	if !ok || h.Ejecutar == nil {
		t.Fatalf("NuevaHerramienta no conectó el handler: %+v", h)
	}
	if h.Categoria != CatArchivos || h.Modo != Lee {
		t.Errorf("la entrada del catálogo se copia tal cual: %+v", h)
	}
	if _, ok := NuevaHerramienta("inventada", nil); ok {
		t.Error("un nombre fuera del catálogo no se conecta")
	}
}

// TestAccionesBase — las acciones de los agentes base salen de SPEC-TOOLS: plan
// no concede `editar` pero sí `tareas`; build concede las cinco.
func TestAccionesBase(t *testing.T) {
	if AccionEditar.Permitida(AccionesDePlan()) {
		t.Error("plan no puede conceder `editar`")
	}
	if !AccionTareas.Permitida(AccionesDePlan()) {
		t.Error("plan sí concede `tareas` (la lista de pasos de la sesión)")
	}
	for _, a := range []Accion{AccionLeer, AccionEditar, AccionEjecutar, AccionInternet, AccionTareas} {
		if !a.Permitida(AccionesDeBuild()) {
			t.Errorf("build debe conceder %s", a)
		}
	}
}
