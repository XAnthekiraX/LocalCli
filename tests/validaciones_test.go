// validaciones_test.go — T-B015-04: cada caso inválido documentado en
// VALIDATION.md devuelve su error.
//
// Fuente de verdad: ai/docs/backend/05-quality/VALIDATION.md §1 (rutas, comandos,
// peticiones del modelo) y §3 (campos obligatorios: "Una herramienta sin su
// campo obligatorio no se ejecuta; se rechaza con un error claro"). Recorre la
// tabla completa de campos obligatorios: la ausencia de cada uno produce
// E_BAD_ARGS, y el tipo equivocado también.
package tests

import (
	"testing"

	"localcli/internal/task"
	"localcli/internal/tools"
)

// TestCadaCampoObligatorioProduceSuError — la tabla §3 de VALIDATION.md, caso a
// caso: sin su campo obligatorio, la herramienta no se ejecuta.
func TestCadaCampoObligatorioProduceSuError(t *testing.T) {
	casos := []struct {
		nombre     string
		argumentos any
	}{
		{"leer_archivo", &tools.PeticionLeerArchivo{Ruta: ""}},
		{"listar_carpeta", &tools.PeticionListarCarpeta{Ruta: " "}},
		{"buscar_archivos", &tools.PeticionBuscarArchivos{Patron: ""}},
		{"buscar_en_archivos", &tools.PeticionBuscarEnArchivos{Patron: ""}},
		{"crear_archivo", &tools.PeticionCrearArchivo{Ruta: "a", Contenido: ""}},
		{"crear_archivo", &tools.PeticionCrearArchivo{Ruta: "", Contenido: "x"}},
		{"escribir_archivo", &tools.PeticionEscribirArchivo{Ruta: "a", Contenido: ""}},
		{"editar_archivo", &tools.PeticionEditarArchivo{Ruta: "a", Cambio: ""}},
		{"eliminar_archivo", &tools.PeticionEliminarArchivo{Ruta: ""}},
		{"crear_carpeta", &tools.PeticionCrearCarpeta{Ruta: ""}},
		{"eliminar_carpeta", &tools.PeticionEliminarCarpeta{Ruta: ""}},
		{"ejecutar_comando", &tools.PeticionEjecutarComando{Comando: ""}},
		{"buscar_en_internet", &tools.PeticionBuscarInternet{Consulta: " "}},
		{"abrir_pagina", &tools.PeticionAbrirPagina{Direccion: ""}},
	}
	for _, c := range casos {
		err := tools.Validar(c.nombre, c.argumentos)
		if err == nil {
			t.Errorf("%s con campo obligatorio vacío no se rechazó", c.nombre)
			continue
		}
		if !errContiene(err, "E_BAD_ARGS") {
			t.Errorf("%s: err = %v, quiero E_BAD_ARGS", c.nombre, err)
		}
	}
}

// TestUnPayloadConCampoDeMasSeRechaza — el contrato es estricto: un campo que
// no existe en el DTO no pasa (VALIDATION.md §2: los payloads están en
// TOOLS-DTO; lo que no está ahí no es parte del contrato).
func TestUnPayloadConCampoDeMasSeRechaza(t *testing.T) {
	_, err := tools.Decodificar("leer_archivo", []byte(`{"ruta":"a.md","contenido":"sobra"}`))
	if err == nil || !errContiene(err, "E_BAD_ARGS") {
		t.Errorf("err = %v, quiero E_BAD_ARGS por campo de más", err)
	}
}

// TestUnPayloadMalFormadoSeRechaza — JSON roto no es un payload: es E_BAD_ARGS,
// nunca un caso que se intente aplicar a medias.
func TestUnPayloadMalFormadoSeRechaza(t *testing.T) {
	_, err := tools.Decodificar("leer_archivo", []byte(`{"ruta": `))
	if err == nil || !errContiene(err, "E_BAD_ARGS") {
		t.Errorf("err = %v, quiero E_BAD_ARGS por JSON roto", err)
	}
}

// TestElVocabularioDelTODOSigueCerrado — task valida el frontmatter al leerlo
// (VALIDATION.md §2). Un elemento con vocabulario fuera del catálogo no entra
// al TODO.
func TestElVocabularioDelTODOSigueCerrado(t *testing.T) {
	e := task.NuevoElemento("T-B001", task.CapaBackend, task.Accion("inventar"), task.EstadoPendiente, nil, nil, nil)
	if err := task.ValidarElemento(e); err == nil {
		t.Error("una acción fuera del catálogo debe rechazarse")
	}
	e = task.NuevoElemento("T-B001", task.CapaBackend, task.AccionCrear, task.Estado("a medias"), nil, nil, nil)
	if err := task.ValidarElemento(e); err == nil {
		t.Error("un estado fuera del catálogo debe rechazarse")
	}
	// bloqueada_por solo con estado bloqueada (DECISIONS.md [28]).
	e = task.NuevoElemento("T-B001", task.CapaBackend, task.AccionCrear, task.EstadoPendiente, nil, []string{"T-B002"}, nil)
	if err := task.ValidarElemento(e); err == nil {
		t.Error("bloqueada_por sin estado bloqueada debe rechazarse")
	}
	// Y uno bien formado pasa.
	bien := task.NuevoElemento("T-B001", task.CapaBackend, task.AccionCrear, task.EstadoPendiente, nil, nil, nil)
	if err := task.ValidarElemento(bien); err != nil {
		t.Errorf("un elemento bien formado no valida: %v", err)
	}
}
