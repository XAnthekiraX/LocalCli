package tools

import "testing"

// TestCatalogoTieneTreceHerramientas — T-B007-01: el registro contiene
// exactamente las trece herramientas documentadas. Una más o una menos
// rompería la garantía: una de más que escriba sería un agujero, y una de
// menos dejaría al agente sin poder trabajar.
func TestCatalogoTieneTreceHerramientas(t *testing.T) {
	h := Herramientas()
	if len(h) != 13 {
		t.Fatalf("el catálogo tiene %d herramientas, quiero 13: %v", len(h), NombresCatalogo())
	}
}

// TestCatalogoNombresDocumentados — los trece nombres exactos de TOOLS.md §1.
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

// TestSoloBuildCorrespondeAEscritura — el reparto por agente sale de un solo
// dato: las seis de escritura son "Solo build" y las siete de lectura son de
// ambos agentes (TOOLS.md §1 y §2, SPEC-TOOLS §El reparto).
func TestSoloBuildCorrespondeAEscritura(t *testing.T) {
	var escritura, lectura int
	for _, h := range Herramientas() {
		if h.SoloBuild() {
			escritura++
			if h.Modo != Escribe {
				t.Errorf("%s: SoloBuild con Modo %s", h.Nombre, h.Modo)
			}
			continue
		}
		lectura++
		if h.Modo != Lee {
			t.Errorf("%s: lectura con Modo %s", h.Nombre, h.Modo)
		}
	}
	if escritura != 6 {
		t.Errorf("herramientas de escritura = %d, quiero 6 (crear, escribir, editar, eliminar archivo, crear y eliminar carpeta)", escritura)
	}
	if lectura != 7 {
		t.Errorf("herramientas de lectura = %d, quiero 7 (4 de archivo, 1 de terminal, 2 de internet)", lectura)
	}
}

// TestCategoriasRepartenTrezeHerramientas — la categoría decide el destino del
// enrutado (TOOLS.md §7): 10 de archivo, 1 de terminal, 2 de internet.
func TestCategoriasRepartenTreceHerramientas(t *testing.T) {
	casos := []struct {
		cat  Categoria
		want int
	}{
		{CatArchivos, 10},
		{CatTerminal, 1},
		{CatInternet, 2},
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
