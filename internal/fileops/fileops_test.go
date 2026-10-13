package fileops

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"localcli/internal/store"
	"localcli/internal/tools"
)

func storeTemporal(t *testing.T) (*sql.DB, func() error, error) {
	t.Helper()
	return store.CrearDBTemporal()
}

func TestResolverRechazaTildeYResuelveRelativas(t *testing.T) {
	raiz := "/proyecto"
	if abs, code := Resolver(raiz, "main.go"); code != "" || abs != filepath.Join(raiz, "main.go") {
		t.Fatalf("relativa mal resuelta: %q %q", abs, code)
	}
	if abs, _ := Resolver(raiz, "../otro.go"); abs != "/otro.go" {
		t.Fatalf("`..` debería resolver: %q", abs)
	}
	if _, code := Resolver(raiz, "~/secreto.txt"); !strings.HasPrefix(code, tools.CodBadArgs) {
		t.Fatalf("`~` debería rechazarse con E_BAD_ARGS: %q", code)
	}
	if _, code := Resolver(raiz, "$HOME/x"); !strings.HasPrefix(code, tools.CodBadArgs) {
		t.Fatalf("`$HOME` debería rechazarse: %q", code)
	}
}

func TestLaFronteraNoSeEludeConPuntos(t *testing.T) {
	raiz := "/proyecto"
	if !Dentro(raiz, "/proyecto/sub/a.go") {
		t.Fatal("una ruta interna debería estar dentro")
	}
	if Dentro(raiz, "/proyecto/../otro/a.go") {
		t.Fatal("`..` no debería contar como dentro")
	}
}

func TestLeerArchivoPequenoEntero(t *testing.T) {
	raiz := t.TempDir()
	if err := os.WriteFile(filepath.Join(raiz, "a.txt"), []byte("uno\ndos\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	o := &Operaciones{Raiz: raiz}
	res, err := o.LeerHandler()(context.Background(), tools.PeticionLeerArchivo{Ruta: "a.txt"}, tools.Contexto{})
	if err != nil || res.Error != "" {
		t.Fatalf("no debería fallar: %+v %v", res, err)
	}
	if !strings.Contains(res.Salida, "uno") || res.Truncado {
		t.Fatalf("un archivo pequeño se devuelve entero: %q", res.Salida)
	}
	if res.Meta["medida"] != "2 líneas" {
		t.Fatalf("medida = %v", res.Meta["medida"])
	}
}

func TestLeerArchivoGrandeSeTrunca(t *testing.T) {
	raiz := t.TempDir()
	var b strings.Builder
	for i := 0; i < 5000; i++ {
		b.WriteString("linea de relleno\n")
	}
	if err := os.WriteFile(filepath.Join(raiz, "grande.txt"), []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	o := &Operaciones{Raiz: raiz}
	res, _ := o.LeerHandler()(context.Background(), tools.PeticionLeerArchivo{Ruta: "grande.txt"}, tools.Contexto{})
	if !res.Truncado {
		t.Fatal("un archivo grande debería venir truncado")
	}
	if _, ok := res.Meta["rango"]; !ok {
		t.Fatal("un truncado debería publicar su rango para completar la línea")
	}
}

func TestCrearArchivoNoSobrescribe(t *testing.T) {
	raiz := t.TempDir()
	os.WriteFile(filepath.Join(raiz, "a.txt"), []byte("x"), 0o644)
	o := &Operaciones{Raiz: raiz}
	res, _ := o.crearArchivo(context.Background(), tools.PeticionCrearArchivo{Ruta: "a.txt", Contenido: "y"},
		aprobadorFileops())
	if !strings.HasPrefix(res.Error, tools.CodPathExists) {
		t.Fatalf("crear_archivo no debería sobrescribir: %q", res.Error)
	}
}

func TestTodaEscrituraPideAprobacion(t *testing.T) {
	raiz := t.TempDir()
	o := &Operaciones{Raiz: raiz}
	declinador := tools.Contexto{Ask: func(context.Context, tools.Solicitud) (tools.Decision, error) {
		return tools.Declinar, nil
	}}
	res, _ := o.crearArchivo(context.Background(), tools.PeticionCrearArchivo{Ruta: "nuevo.txt", Contenido: "y"}, declinador)
	if !strings.HasPrefix(res.Error, tools.CodApprovalDeclined) {
		t.Fatalf("una escritura declinada debería dar E_APPROVAL_DECLINED: %q", res.Error)
	}
	if _, err := os.Stat(filepath.Join(raiz, "nuevo.txt")); err == nil {
		t.Fatal("no debería haberse creado el archivo")
	}
}

func TestEscrituraAprobadaRegistraEnHistorial(t *testing.T) {
	db, cleanup, err := storeTemporal(t)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	raiz := t.TempDir()
	o := &Operaciones{Raiz: raiz, DB: db}
	if _, err := o.crearArchivo(context.Background(), tools.PeticionCrearArchivo{Ruta: "nuevo.txt", Contenido: "hola"}, aprobadorFileops()); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM change_history").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("debería quedar un cambio registrado: %d", n)
	}
}

func aprobadorFileops() tools.Contexto {
	return tools.Contexto{Ask: func(context.Context, tools.Solicitud) (tools.Decision, error) {
		return tools.Aprobar, nil
	}}
}
