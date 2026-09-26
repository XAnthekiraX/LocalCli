package tools

import (
	"errors"
	"testing"
)

// TestValidarAceptaPayloadsCompletos — cada herramienta con sus campos
// obligatorios presentes valida sin error.
func TestValidarAceptaPayloadsCompletos(t *testing.T) {
	casos := []struct {
		nombre string
		args   any
	}{
		{"leer_archivo", &PeticionLeerArchivo{Ruta: "a.md"}},
		{"listar_carpeta", &PeticionListarCarpeta{Ruta: "."}},
		{"buscar_archivos", &PeticionBuscarArchivos{Patron: "*.go"}},
		{"buscar_en_archivos", &PeticionBuscarEnArchivos{Patron: "x"}},
		{"crear_archivo", &PeticionCrearArchivo{Ruta: "a.md", Contenido: "x"}},
		{"escribir_archivo", &PeticionEscribirArchivo{Ruta: "a.md", Contenido: "x"}},
		{"editar_archivo", &PeticionEditarArchivo{Ruta: "a.md", Cambio: "x"}},
		{"eliminar_archivo", &PeticionEliminarArchivo{Ruta: "a.md"}},
		{"crear_carpeta", &PeticionCrearCarpeta{Ruta: "d"}},
		{"eliminar_carpeta", &PeticionEliminarCarpeta{Ruta: "d"}},
		{"ejecutar_comando", &PeticionEjecutarComando{Comando: "go build"}},
		{"buscar_en_internet", &PeticionBuscarInternet{Consulta: "x"}},
		{"abrir_pagina", &PeticionAbrirPagina{Direccion: "https://x"}},
	}
	for _, c := range casos {
		if err := Validar(c.nombre, c.args); err != nil {
			t.Errorf("%s: %v", c.nombre, err)
		}
	}
}

// TestValidarRechazaTipoDeOtraHerramienta — validar una petición con el
// contrato de otra herramienta no puede pasar en silencio.
func TestValidarRechazaTipoDeOtraHerramienta(t *testing.T) {
	err := Validar("leer_archivo", &PeticionListarCarpeta{Ruta: "a"})
	if !errors.Is(err, ErrArgumentosInvalidos) {
		t.Errorf("err = %v, quiero E_BAD_ARGS", err)
	}
}

// TestValidarRechazaHerramientaDesconocida — validar algo fuera del catálogo
// es E_TOOL_UNKNOWN, aunque el payload sea válido para otra cosa.
func TestValidarRechazaHerramientaDesconocida(t *testing.T) {
	err := Validar("inventada", &PeticionLeerArchivo{Ruta: "a"})
	if !errors.Is(err, ErrHerramientaDesconocida) {
		t.Errorf("err = %v, quiero E_TOOL_UNKNOWN", err)
	}
}

// TestDecodificarRechazaDatosSobrantes — dos objetos JSON seguidos no son un
// payload válido.
func TestDecodificarRechazaDatosSobrantes(t *testing.T) {
	_, err := Decodificar("leer_archivo", []byte(`{"ruta":"a.md"} {"ruta":"b.md"}`))
	if !errors.Is(err, ErrArgumentosInvalidos) {
		t.Errorf("err = %v, quiero E_BAD_ARGS", err)
	}
}
