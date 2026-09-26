package tools

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestPeticionesFixtureDecodifican — T-B007-02: hay un fixture JSON por cada
// una de las trece herramientas y todos decodifican en su tipo de contrato sin
// rechazo. Es el contrato de request de TOOLS-DTO.md §2 ejercitado de verdad.
func TestPeticionesFixtureDecodifican(t *testing.T) {
	bruto, err := os.ReadFile(filepath.Join("testdata", "peticiones.json"))
	if err != nil {
		t.Fatalf("no se pudo leer el fixture: %v", err)
	}
	var fixtures map[string]json.RawMessage
	if err := json.Unmarshal(bruto, &fixtures); err != nil {
		t.Fatalf("fixture ilegible: %v", err)
	}
	for _, nombre := range NombresCatalogo() {
		cuerpo, ok := fixtures[nombre]
		if !ok {
			t.Errorf("falta el fixture de %s", nombre)
			continue
		}
		if _, err := Decodificar(nombre, cuerpo); err != nil {
			t.Errorf("%s: el fixture no decodifica: %v", nombre, err)
		}
	}
	if len(fixtures) != 13 {
		t.Errorf("el fixture tiene %d entradas, quiero 13", len(fixtures))
	}
}

// TestArgumentosConCampoDesconocidoSeRechazan — un campo que no existe en el
// contrato es E_BAD_ARGS: el modelo no puede colar parámetros que la
// herramienta no entiende (VALIDATION.md §1).
func TestArgumentosConCampoDesconocidoSeRechazan(t *testing.T) {
	_, err := Decodificar("leer_archivo", []byte(`{"ruta":"a.md","inventado":1}`))
	if !errors.Is(err, ErrArgumentosInvalidos) {
		t.Fatalf("err = %v, quiero E_BAD_ARGS", err)
	}
}

// TestArgumentosIncompletosSeRechazan — una herramienta sin su campo
// obligatorio no se ejecuta (VALIDATION.md §3).
func TestArgumentosIncompletosSeRechazan(t *testing.T) {
	casos := []struct {
		herramienta string
		cuerpo      string
	}{
		{"leer_archivo", `{"ruta":"   "}`},
		{"crear_archivo", `{"ruta":"a.md"}`},
		{"editar_archivo", `{"ruta":"a.md"}`},
		{"buscar_en_archivos", `{"patron":""}`},
		{"ejecutar_comando", `{"comando":""}`},
		{"buscar_en_internet", `{}`},
		{"abrir_pagina", `{"direccion":""}`},
	}
	for _, c := range casos {
		if _, err := Decodificar(c.herramienta, []byte(c.cuerpo)); !errors.Is(err, ErrArgumentosInvalidos) {
			t.Errorf("%s %s: err = %v, quiero E_BAD_ARGS", c.herramienta, c.cuerpo, err)
		}
	}
}

// TestResponseSchemasCodifican — T-B007-03: lo que devuelve cada categoría
// lleva los campos de TOOLS-DTO.md §3, incluido `truncado` en la terminal.
func TestResponseSchemasCodifican(t *testing.T) {
	casos := []struct {
		nombre string
		resp   any
		claves []string
	}{
		{"leer_archivo", RespuestaLeerArchivo{Contenido: "x"}, []string{"contenido"}},
		{"listar_carpeta", RespuestaListarCarpeta{Entradas: []string{"a"}}, []string{"entradas"}},
		{"buscar_archivos", RespuestaBuscarArchivos{Rutas: []string{"a"}}, []string{"rutas"}},
		{"buscar_en_archivos", RespuestaBuscarEnArchivos{Coincidencias: []Coincidencia{{Archivo: "a", Linea: 1, Fragmento: "x"}}}, []string{"coincidencias"}},
		{"crear_archivo", RespuestaEscritura{Confirmacion: "ok", Ruta: "a"}, []string{"confirmacion", "ruta"}},
		{"escribir_archivo", RespuestaEscritura{Confirmacion: "ok", Ruta: "a"}, []string{"confirmacion", "ruta"}},
		{"editar_archivo", RespuestaEscritura{Confirmacion: "ok", Ruta: "a"}, []string{"confirmacion", "ruta"}},
		{"eliminar_archivo", RespuestaEliminar{Confirmacion: "ok"}, []string{"confirmacion"}},
		{"crear_carpeta", RespuestaEscritura{Confirmacion: "ok", Ruta: "a"}, []string{"confirmacion", "ruta"}},
		{"eliminar_carpeta", RespuestaEliminar{Confirmacion: "ok"}, []string{"confirmacion"}},
		{"ejecutar_comando", RespuestaEjecutarComando{Salida: "x", Error: "", Codigo: 0, Truncado: true, Termino: true}, []string{"salida", "error", "codigo", "truncado", "termino"}},
		{"buscar_en_internet", RespuestaBuscarInternet{Resultados: []ResultadoInternet{{Titulo: "t", Direccion: "d", Fragmento: "f"}}}, []string{"resultados"}},
		{"abrir_pagina", RespuestaAbrirPagina{Contenido: "x"}, []string{"contenido"}},
	}
	for _, c := range casos {
		b, err := json.Marshal(c.resp)
		if err != nil {
			t.Errorf("%s: no codifica: %v", c.nombre, err)
			continue
		}
		var m map[string]json.RawMessage
		if err := json.Unmarshal(b, &m); err != nil {
			t.Errorf("%s: JSON inválido: %v", c.nombre, err)
			continue
		}
		for _, k := range c.claves {
			if _, ok := m[k]; !ok {
				t.Errorf("%s: falta el campo %q en %s", c.nombre, k, b)
			}
		}
	}
}
