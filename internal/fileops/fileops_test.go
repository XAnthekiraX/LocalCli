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

// `escribir_archivo` no pisa un archivo con contenido: lo rechaza y el archivo
// queda intacto. Reemplazar lo que ya hay exige `editar_archivo` o borrar y crear.
func TestEscribirArchivoNoPisaContenidoExistente(t *testing.T) {
	raiz := t.TempDir()
	archivo := filepath.Join(raiz, "a.go")
	if err := os.WriteFile(archivo, []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	o := &Operaciones{Raiz: raiz}
	res, _ := o.escribirArchivo(context.Background(),
		tools.PeticionEscribirArchivo{Ruta: "a.go", Contenido: "package b\n"}, aprobadorFileops())
	if !strings.HasPrefix(res.Error, tools.CodFileNotEmpty) {
		t.Fatalf("escribir_archivo no debería pisar un archivo con contenido: %q", res.Error)
	}
	if !strings.Contains(res.Error, "editar_archivo") {
		t.Fatalf("el error debería nombrar la vía correcta: %q", res.Error)
	}
	datos, err := os.ReadFile(archivo)
	if err != nil {
		t.Fatal(err)
	}
	if string(datos) != "package a\n" {
		t.Fatalf("el contenido existente cambió: %q", string(datos))
	}
}

// Un archivo vacío sí se puede rellenar con `escribir_archivo`.
func TestEscribirArchivoRellenaUnoVacio(t *testing.T) {
	raiz := t.TempDir()
	archivo := filepath.Join(raiz, "vacio.txt")
	if err := os.WriteFile(archivo, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	o := &Operaciones{Raiz: raiz}
	res, _ := o.escribirArchivo(context.Background(),
		tools.PeticionEscribirArchivo{Ruta: "vacio.txt", Contenido: "hola"}, aprobadorFileops())
	if res.Error != "" {
		t.Fatalf("un archivo vacío debería aceptarse: %q", res.Error)
	}
	datos, _ := os.ReadFile(archivo)
	if string(datos) != "hola" {
		t.Fatalf("no se escribió el contenido: %q", string(datos))
	}
}

// Si el archivo no existe, `escribir_archivo` lo crea.
func TestEscribirArchivoCreaElQueFalta(t *testing.T) {
	raiz := t.TempDir()
	o := &Operaciones{Raiz: raiz}
	res, _ := o.escribirArchivo(context.Background(),
		tools.PeticionEscribirArchivo{Ruta: "nuevo.txt", Contenido: "hola"}, aprobadorFileops())
	if res.Error != "" {
		t.Fatalf("un archivo que falta debería crearse: %q", res.Error)
	}
	datos, _ := os.ReadFile(filepath.Join(raiz, "nuevo.txt"))
	if string(datos) != "hola" {
		t.Fatalf("no se creó el archivo: %q", string(datos))
	}
}

// Repetir el mismo contenido no es un reemplazo: se acepta sin tocar el archivo.
func TestEscribirArchivoConElMismoContenidoNoFalla(t *testing.T) {
	raiz := t.TempDir()
	archivo := filepath.Join(raiz, "a.txt")
	if err := os.WriteFile(archivo, []byte("igual"), 0o644); err != nil {
		t.Fatal(err)
	}
	o := &Operaciones{Raiz: raiz}
	res, _ := o.escribirArchivo(context.Background(),
		tools.PeticionEscribirArchivo{Ruta: "a.txt", Contenido: "igual"}, aprobadorFileops())
	if res.Error != "" {
		t.Fatalf("el mismo contenido no debería rechazarse: %q", res.Error)
	}
}

// Escribir dentro de la carpeta ya no pide confirmación: solo la piden el
// borrado y todo lo que queda fuera de la carpeta.
func TestEscrituraDentroNoPideConfirmacion(t *testing.T) {
	raiz := t.TempDir()
	o := &Operaciones{Raiz: raiz}
	preguntas := 0
	quePregunta := tools.Contexto{Ask: func(context.Context, tools.Solicitud) (tools.Decision, error) {
		preguntas++
		return tools.Declinar, nil
	}}
	res, _ := o.crearArchivo(context.Background(), tools.PeticionCrearArchivo{Ruta: "nuevo.txt", Contenido: "y"}, quePregunta)
	if res.Error != "" {
		t.Fatalf("crear dentro de la carpeta no debería fallar: %q", res.Error)
	}
	if preguntas != 0 {
		t.Fatalf("una escritura dentro no debería preguntar; preguntó %d veces", preguntas)
	}
	if _, err := os.Stat(filepath.Join(raiz, "nuevo.txt")); err != nil {
		t.Fatalf("el archivo no se creó: %v", err)
	}
}

func TestBorradoPideConfirmacion(t *testing.T) {
	raiz := t.TempDir()
	archivo := filepath.Join(raiz, "a.txt")
	if err := os.WriteFile(archivo, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	o := &Operaciones{Raiz: raiz}
	res, _ := o.eliminarArchivo(context.Background(), tools.PeticionEliminarArchivo{Ruta: "a.txt"}, declinador())
	if !strings.HasPrefix(res.Error, tools.CodApprovalDeclined) {
		t.Fatalf("un borrado declinado debería dar E_APPROVAL_DECLINED: %q", res.Error)
	}
	if _, err := os.Stat(archivo); err != nil {
		t.Fatal("no debería haberse borrado el archivo")
	}
}

// Toda operación fuera de la carpeta sigue pidiendo permiso, también las que no
// son de borrado.
func TestBorradoFueraDeLaCarpetaPideConfirmacion(t *testing.T) {
	raiz := t.TempDir()
	fuera := filepath.Join(t.TempDir(), "x.txt")
	if err := os.WriteFile(fuera, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	o := &Operaciones{Raiz: raiz}
	res, _ := o.eliminarArchivo(context.Background(),
		tools.PeticionEliminarArchivo{Ruta: fuera, Motivo: "lo necesito fuera del proyecto"}, declinador())
	if !strings.HasPrefix(res.Error, tools.CodApprovalDeclined) {
		t.Fatalf("un borrado fuera declinado debería dar E_APPROVAL_DECLINED: %q", res.Error)
	}
	if _, err := os.Stat(fuera); err != nil {
		t.Fatal("no debería haberse borrado el archivo")
	}
}

// Una ruta fuera sin `motivo` se rechaza sin llegar a pedir nada.
func TestFueraSinMotivoSeRechaza(t *testing.T) {
	raiz := t.TempDir()
	fuera := filepath.Join(t.TempDir(), "x.txt")
	o := &Operaciones{Raiz: raiz}
	res, _ := o.eliminarArchivo(context.Background(), tools.PeticionEliminarArchivo{Ruta: fuera}, declinador())
	if !strings.HasPrefix(res.Error, tools.CodPathOutside) {
		t.Fatalf("fuera sin motivo debería dar E_PATH_OUTSIDE: %q", res.Error)
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

func declinador() tools.Contexto {
	return tools.Contexto{Ask: func(context.Context, tools.Solicitud) (tools.Decision, error) {
		return tools.Declinar, nil
	}}
}
