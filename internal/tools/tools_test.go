package tools

import (
	"context"
	"testing"
)

// TestCatalogoTieneQuinceHerramientas — T-B007-01: el registro contiene
// exactamente las quince herramientas documentadas. Una más o una menos
// rompería la garantía: una de más que escriba sería un agujero, y una de
// menos dejaría al agente sin poder trabajar.
func TestCatalogoTieneQuinceHerramientas(t *testing.T) {
	h := Herramientas()
	if len(h) != 15 {
		t.Fatalf("el catálogo tiene %d herramientas, quiero 15: %v", len(h), NombresCatalogo())
	}
}

// TestCatalogoNombresDocumentados — los quince nombres exactos de TOOLS.md §1.
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
		"crear_todo", "actualizar_todo",
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

// TestSoloBuildCorrespondeAEscritura — el reparto por agente sale de los
// permisos `write` y `edit`: las seis que escriben en el proyecto son "solo
// build"; el resto —las de lectura y la lista de pasos de la sesión— son de
// ambos (TOOLS.md §1 y §2, SPEC-TOOLS §El reparto).
func TestSoloBuildCorrespondeAEscritura(t *testing.T) {
	var escritura, otros int
	for _, h := range Herramientas() {
		if h.SoloBuild() {
			escritura++
			if h.Permiso != PermisoWrite && h.Permiso != PermisoEdit {
				t.Errorf("%s: SoloBuild con permiso %s", h.Nombre, h.Permiso)
			}
			continue
		}
		otros++
		if h.Permiso == PermisoWrite || h.Permiso == PermisoEdit {
			t.Errorf("%s: herramienta de escritura que no es SoloBuild", h.Nombre)
		}
	}
	if escritura != 6 {
		t.Errorf("herramientas de escritura = %d, quiero 6 (crear, escribir, editar, eliminar archivo, crear y eliminar carpeta)", escritura)
	}
	if otros != 9 {
		t.Errorf("herramientas no exclusivas de build = %d, quiero 9 (4 de archivo, 1 de terminal, 2 de internet y las dos de la lista de pasos)", otros)
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
		{CatTareas, 2},
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

// TestCadaHerramientaDeclaraSuPermiso — el reparto sale de un dato declarado
// del catálogo (`Herramienta.Permiso`), no de un `switch` que lo deduzca.
func TestCadaHerramientaDeclaraSuPermiso(t *testing.T) {
	casos := []struct {
		nombre  string
		permiso Permiso
	}{
		{"leer_archivo", PermisoRead},
		{"buscar_en_archivos", PermisoRead},
		{"ejecutar_comando", PermisoRead},
		{"buscar_en_internet", PermisoRead},
		{"abrir_pagina", PermisoRead},
		{"actualizar_todo", PermisoRead},
		{"crear_archivo", PermisoWrite},
		{"escribir_archivo", PermisoWrite},
		{"eliminar_archivo", PermisoWrite},
		{"crear_carpeta", PermisoWrite},
		{"eliminar_carpeta", PermisoWrite},
		{"editar_archivo", PermisoEdit},
	}
	for _, c := range casos {
		got, ok := PermisoDe(c.nombre)
		if !ok {
			t.Errorf("%s no está en el catálogo", c.nombre)
			continue
		}
		if got != c.permiso {
			t.Errorf("%s: permiso = %s, quiero %s", c.nombre, got, c.permiso)
		}
	}
}

// TestNombresDePermisoReparteLasHerramientas — los tres permisos cubren las
// quince herramientas sin solaparse: la partición que sostiene el catálogo
// derivado.
func TestNombresDePermisoReparteLasHerramientas(t *testing.T) {
	var total int
	for _, p := range []Permiso{PermisoRead, PermisoWrite, PermisoEdit} {
		total += len(NombresDePermiso(p))
	}
	if total != len(NombresCatalogo()) {
		t.Errorf("los permisos cubren %d herramientas, quiero %d", total, len(NombresCatalogo()))
	}
	if len(NombresDePermiso(PermisoWrite)) != 5 {
		t.Errorf("`write` tiene %d herramientas, quiero 5", len(NombresDePermiso(PermisoWrite)))
	}
	if len(NombresDePermiso(PermisoRead)) != 9 {
		t.Errorf("`read` tiene %d herramientas, quiero 9", len(NombresDePermiso(PermisoRead)))
	}
	if len(NombresDePermiso(PermisoEdit)) != 1 {
		t.Errorf("`edit` tiene %d herramientas, quiero 1 (editar_archivo)", len(NombresDePermiso(PermisoEdit)))
	}
	if Permiso("inventada").Valida() {
		t.Error("un permiso fuera de los tres no puede ser válido")
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

// TestPermisosBase — los permisos de los agentes base salen de SPEC-TOOLS: plan
// solo concede `read`; build concede los tres.
func TestPermisosBase(t *testing.T) {
	if PermisoWrite.Permitida(PermisosDePlan()) || PermisoEdit.Permitida(PermisosDePlan()) {
		t.Error("plan no puede conceder `write` ni `edit`")
	}
	if !PermisoRead.Permitida(PermisosDePlan()) {
		t.Error("plan sí concede `read`")
	}
	for _, p := range []Permiso{PermisoRead, PermisoWrite, PermisoEdit} {
		if !p.Permitida(PermisosDeBuild()) {
			t.Errorf("build debe conceder %s", p)
		}
	}
}
