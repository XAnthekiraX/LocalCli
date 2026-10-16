package fileops

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"localcli/internal/tools"
)

func editar(t *testing.T, contenido string, cambios ...tools.Cambio) (tools.Resultado, string) {
	t.Helper()
	raiz := t.TempDir()
	ruta := filepath.Join(raiz, "doc.md")
	if err := os.WriteFile(ruta, []byte(contenido), 0o644); err != nil {
		t.Fatal(err)
	}
	o := &Operaciones{Raiz: raiz}
	res, err := o.editarArchivo(context.Background(), tools.PeticionEditarArchivo{Ruta: "doc.md", Reemplazos: cambios},
		aprobadorFileops())
	if err != nil {
		t.Fatalf("error de harness: %v", err)
	}
	datos, _ := os.ReadFile(ruta)
	return res, string(datos)
}

func TestEditarToleraCRLF(t *testing.T) {
	antes := "uno\r\ndos\r\ntres\r\n"
	res, despues := editar(t, antes, tools.Cambio{Buscar: "uno\ndos", Reemplazo: "UNO\nDOS"})
	if res.Error != "" {
		t.Fatalf("no debería fallar con CRLF: %q", res.Error)
	}
	if despues != "UNO\r\nDOS\r\ntres\r\n" {
		t.Fatalf("debería conservar CRLF: %q", despues)
	}
}

func TestEditarToleraUnicodeTipografico(t *testing.T) {
	antes := "dijo “hola” con NBSP adicional y tres puntos…\n"
	res, despues := editar(t, antes,
		tools.Cambio{Buscar: `dijo "hola" con NBSP adicional y tres puntos...`, Reemplazo: "ok"})
	if res.Error != "" {
		t.Fatalf("no debería fallar con Unicode: %q", res.Error)
	}
	if despues != "ok\n" {
		t.Fatalf("sustitución mal aplicada: %q", despues)
	}
}

func TestEditarMultiplesOcurrenciasFallanSinTocar(t *testing.T) {
	antes := "x = 1\ny = x\n"
	res, despues := editar(t, antes, tools.Cambio{Buscar: "x", Reemplazo: "z"})
	if !strings.Contains(res.Error, "2 veces") {
		t.Fatalf("debería quejarse de ambigüedad: %q", res.Error)
	}
	if despues != antes {
		t.Fatalf("no debería modificar el archivo: %q", despues)
	}
}

func TestEditarToleraEspaciosFinales(t *testing.T) {
	antes := "línea con espacio final   \nninguna otra\n"
	res, despues := editar(t, antes, tools.Cambio{Buscar: "línea con espacio final", Reemplazo: "limpia"})
	if res.Error != "" {
		t.Fatalf("no debería fallar por espacios finales: %q", res.Error)
	}
	if despues != "limpia   \nninguna otra\n" {
		t.Fatalf("el fragmento debería sustituir aunque el archivo tenga espacios finales: %q", despues)
	}
}

func TestEditarToleraSangriaFlexible(t *testing.T) {
	antes := "func main() {\n\tfmt.Println(\"hola\")\n}\n"
	res, despues := editar(t, antes,
		tools.Cambio{Buscar: "    fmt.Println(\"hola\")", Reemplazo: "    fmt.Println(\"adios\")"})
	if res.Error != "" {
		t.Fatalf("no debería fallar por tab vs espacios: %q", res.Error)
	}
	if !strings.Contains(despues, "adios") || strings.Contains(despues, "hola") {
		t.Fatalf("sustitución mal aplicada: %q", despues)
	}
}

func TestEditarReemplazosVaciosYInvalidos(t *testing.T) {
	if _, despues := editar(t, "contenido\n"); despues != "contenido\n" {
		t.Fatalf("sin reemplazos no debería tocar nada: %q", despues)
	}
	res, _ := editar(t, "contenido\n", tools.Cambio{Buscar: "", Reemplazo: "x"})
	if !strings.HasPrefix(res.Error, tools.CodBadArgs) {
		t.Fatalf("buscar vacío debería dar E_BAD_ARGS: %q", res.Error)
	}
	res, _ = editar(t, "contenido\n", tools.Cambio{Buscar: "contenido", Reemplazo: "contenido"})
	if !strings.HasPrefix(res.Error, tools.CodBadArgs) {
		t.Fatalf("reemplazo idéntico debería dar E_BAD_ARGS: %q", res.Error)
	}
}

func TestEditarEncadenaVariosReemplazos(t *testing.T) {
	antes := "alpha\nbeta\ngamma\n"
	res, despues := editar(t, antes,
		tools.Cambio{Buscar: "alpha", Reemplazo: "A"},
		tools.Cambio{Buscar: "beta", Reemplazo: "B"})
	if res.Error != "" {
		t.Fatalf("no debería fallar: %q", res.Error)
	}
	if despues != "A\nB\ngamma\n" {
		t.Fatalf("encadenado mal aplicado: %q", despues)
	}
}

func TestEditarNoEncontradoDaErrorAcccionable(t *testing.T) {
	res, _ := editar(t, "hola mundo\n", tools.Cambio{Buscar: "adios", Reemplazo: "x"})
	if !strings.Contains(res.Error, "no se encontró") || !strings.Contains(res.Error, "leer_archivo") {
		t.Fatalf("el error debería orientar al modelo: %q", res.Error)
	}
}

func TestEditarConservaBOM(t *testing.T) {
	antes := bomUTF8 + "primera línea\nsegunda\n"
	res, despues := editar(t, antes, tools.Cambio{Buscar: "segunda", Reemplazo: "2da"})
	if res.Error != "" {
		t.Fatalf("no debería fallar con BOM: %q", res.Error)
	}
	if !strings.HasPrefix(despues, bomUTF8) {
		t.Fatalf("debería conservar el BOM: %q", despues)
	}
	if !strings.Contains(despues, "2da") {
		t.Fatalf("sustitución no aplicada: %q", despues)
	}
}

func TestEditarArchivoNoModificaSiHayAmbigua(t *testing.T) {
	antes := "repetir\nrepetir\n"
	res, despues := editar(t, antes,
		tools.Cambio{Buscar: "nuevo", Reemplazo: "x"}, // no está: corta antes de escribir
	)
	_ = res
	if despues != antes {
		t.Fatalf("un fallo no debería dejar escritura parcial: %q", despues)
	}
}
