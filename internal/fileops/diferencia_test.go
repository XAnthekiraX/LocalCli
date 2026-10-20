package fileops

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"localcli/internal/tools"
)

func TestDiferenciaMuestraElAntesYElDespues(t *testing.T) {
	antes := "uno\ndos\ntres\ncuatro\ncinco\n"
	despues := "uno\ndos\nTRES\ncuatro\ncinco\n"

	diff := Diferencia(antes, despues)
	if !strings.Contains(diff, "@@ -1,5 +1,5 @@") {
		t.Fatalf("falta la cabecera del tramo:\n%s", diff)
	}
	for _, esperada := range []string{" uno", " dos", "-tres", "+TRES", " cuatro", " cinco"} {
		if !strings.Contains(diff, esperada) {
			t.Fatalf("falta la línea %q:\n%s", esperada, diff)
		}
	}
	if strings.Contains(diff, "-uno") || strings.Contains(diff, "+dos") {
		t.Fatalf("el contexto no debería marcarse como cambio:\n%s", diff)
	}
}

func TestDiferenciaDeUnArchivoNuevoEsTodoAñadido(t *testing.T) {
	diff := Diferencia("", "linea uno\nlinea dos\n")
	if !strings.Contains(diff, "@@ -0,0 +1,2 @@") {
		t.Fatalf("cabecera inesperada:\n%s", diff)
	}
	if !strings.Contains(diff, "+linea uno") || !strings.Contains(diff, "+linea dos") {
		t.Fatalf("todo el contenido debería ir añadido:\n%s", diff)
	}
	if strings.Contains(diff, "\n-") {
		t.Fatalf("un archivo nuevo no tiene líneas quitadas:\n%s", diff)
	}
}

func TestDiferenciaSinCambiosNoDejaBloque(t *testing.T) {
	if diff := Diferencia("igual\n", "igual\n"); diff != "" {
		t.Fatalf("sin cambios no debería haber diff: %q", diff)
	}
}

func TestDiferenciaRecortaYCuentaLoQueQueda(t *testing.T) {
	var antes, despues strings.Builder
	for i := 0; i < 60; i++ {
		antes.WriteString("vieja\n")
		despues.WriteString("nueva\n")
	}

	diff := Diferencia(antes.String(), despues.String())
	lineas := strings.Split(diff, "\n")
	if len(lineas) != topeBloque+1 {
		t.Fatalf("el bloque debería quedar en %d líneas más el aviso: %d", topeBloque, len(lineas))
	}
	if !strings.HasPrefix(lineas[len(lineas)-1], "… ") || !strings.HasSuffix(lineas[len(lineas)-1], " líneas más") {
		t.Fatalf("falta el aviso de recorte: %q", lineas[len(lineas)-1])
	}
}

func TestEditarArchivoDevuelveElBloqueDeDiferencia(t *testing.T) {
	res, _ := editar(t, "uno\ndos\ntres\n", tools.Cambio{Buscar: "dos", Reemplazo: "DOS"})
	if res.Error != "" {
		t.Fatalf("no debería fallar: %q", res.Error)
	}
	if res.Meta["bloque_tipo"] != "diferencia" {
		t.Fatalf("tipo de bloque = %v", res.Meta["bloque_tipo"])
	}
	bloque, _ := res.Meta["bloque"].(string)
	if !strings.Contains(bloque, "-dos") || !strings.Contains(bloque, "+DOS") {
		t.Fatalf("el bloque no muestra el antes y el después:\n%s", bloque)
	}
}

func TestUnCambioFallidoNoDevuelveBloque(t *testing.T) {
	res, _ := editar(t, "uno\n", tools.Cambio{Buscar: "no está", Reemplazo: "x"})
	if res.Error == "" {
		t.Fatal("el reemplazo debería fallar")
	}
	if res.Meta["bloque"] != nil || res.Meta["bloque_tipo"] != nil {
		t.Fatalf("un fallo no deja bloque: %v", res.Meta)
	}
}

func TestCrearArchivoDevuelveElContenidoComoAñadido(t *testing.T) {
	raiz := t.TempDir()
	o := &Operaciones{Raiz: raiz}
	res, err := o.crearArchivo(context.Background(),
		tools.PeticionCrearArchivo{Ruta: "nuevo.txt", Contenido: "hola\nmundo\n"}, aprobadorFileops())
	if err != nil || res.Error != "" {
		t.Fatalf("no debería fallar: %+v %v", res, err)
	}
	bloque, _ := res.Meta["bloque"].(string)
	if !strings.Contains(bloque, "+hola") || !strings.Contains(bloque, "+mundo") {
		t.Fatalf("el archivo nuevo debería verse como líneas añadidas:\n%s", bloque)
	}
	if _, err := os.Stat(filepath.Join(raiz, "nuevo.txt")); err != nil {
		t.Fatalf("el archivo no se creó: %v", err)
	}
}
