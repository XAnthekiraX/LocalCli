package tools

// def_test.go — T-B026: el verbo, el tema y la medida que la TUI pinta en la
// línea de herramienta. El tema expone SOLO el campo declarado por el catálogo
// (la ruta, el patrón, el comando); nunca el resto de argumentos.

import (
	"strings"
	"testing"
)

func TestVerboDeCaeEnElNombre(t *testing.T) {
	h, ok := Buscar("leer_archivo")
	if !ok {
		t.Fatal("falta leer_archivo en el catálogo")
	}
	if got := VerboDe(h); got != "LEER" {
		t.Errorf("verbo de leer_archivo = %q, quiero LEER", got)
	}
	// Una herramienta sin verbo declarado —una del usuario— cae en su nombre.
	if got := VerboDe(Herramienta{Nombre: "propia"}); got != "propia" {
		t.Errorf("sin verbo, cae en el nombre: %q", got)
	}
}

func TestTemaDeSoloExponeElCampoDeclarado(t *testing.T) {
	h, _ := Buscar("leer_archivo")
	if got := TemaDe(h, &PeticionLeerArchivo{Ruta: "internal/tui/chat.go"}); got != "internal/tui/chat.go" {
		t.Errorf("tema = %q, quiero la ruta", got)
	}

	// Colapsa espacios y saltos de línea.
	hCmd, _ := Buscar("ejecutar_comando")
	if got := TemaDe(hCmd, &PeticionEjecutarComando{Comando: "go\n  test   ./..."}); got != "go test ./..." {
		t.Errorf("el tema colapsa el espacio: %q", got)
	}

	// Sin tema declarado no hay objetivo.
	hTodo, _ := Buscar("actualizar_todo")
	if got := TemaDe(hTodo, nil); got != "" {
		t.Errorf("actualizar_todo no tiene tema: %q", got)
	}

	// Args nil o de otro tipo no revientan: devuelven vacío.
	if got := TemaDe(h, nil); got != "" {
		t.Errorf("sin args no hay tema: %q", got)
	}
	if got := TemaDe(h, "no soy una petición"); got != "" {
		t.Errorf("args que no son struct no tienen tema: %q", got)
	}
}

func TestTemaDeRecortaLoEnorme(t *testing.T) {
	h, _ := Buscar("buscar_archivos")
	got := TemaDe(h, &PeticionBuscarArchivos{Patron: strings.Repeat("x", 200)})
	if !strings.HasSuffix(got, "…") {
		t.Errorf("un tema enorme se recorta con «…»: %q", got)
	}
	if len([]rune(got)) != 61 {
		t.Errorf("el tema recortado ocupa 60 runas + «…»: %d", len([]rune(got)))
	}
}

func TestMedidaDeCuentaEnSuUnidad(t *testing.T) {
	lectura, _ := Buscar("leer_archivo")
	busca, _ := Buscar("buscar_archivos")
	escritura, _ := Buscar("escribir_archivo")

	casos := []struct {
		nombre string
		h      Herramienta
		salida string
		quiero string
	}{
		{"tres líneas", lectura, "a\nb\nc", "3 líneas"},
		{"una línea", lectura, "solo", "1 línea"},
		{"líneas vacías no cuentan", lectura, "a\n\n\nb", "2 líneas"},
		{"plural", busca, "uno\ndos", "2 coincidencias"},
		{"sin unidad no hay medida", escritura, "creado", ""},
		{"sin salida no hay medida", lectura, "", ""},
	}
	for _, c := range casos {
		if got := MedidaDe(c.h, Resultado{Salida: c.salida}); got != c.quiero {
			t.Errorf("%s: medida = %q, quiero %q", c.nombre, got, c.quiero)
		}
	}
}
