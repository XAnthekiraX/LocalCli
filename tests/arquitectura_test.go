// Package tests — T-B015: verificación integral del backend.
//
// Los invariantes de arquitectura se prueban como invariantes, no como casos
// sueltos (TESTING.md §4). Aquí viven los que cruzan módulos: qué puede
// importar quién, qué tablas existen y quién abre la base.
package tests

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var regexpCreateTable = regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?[\"\x60]?(\w+)[\"\x60]?`)

// limiteImportes codifica lo que cada módulo NO puede importar. La dependencia
// permitida no se lista: se lista la prohibida, que es lo que la regla defiende.
var limiteImportes = map[string][]string{
	// La vista pinta; no lleva lógica de negocio (DOMAIN.md, límite del módulo tui).
	"tui": {"localcli/internal/store", "localcli/internal/flow", "localcli/internal/agent",
		"localcli/internal/tools", "localcli/internal/fileops", "localcli/internal/exec",
		"database/sql", "modernc.org/sqlite"},
	// Solo el motor lanza colas: `queue` no almacena estado ni toca la base
	// (DECISIONS.md, DOMAIN.md §3: "queue no tiene tabla propia").
	"queue": {"database/sql", "modernc.org/sqlite", "localcli/internal/store"},
	// El nodo de contexto registra a través de interfaces… salvo su adaptador
	// con `store`, que es el que decide qué tablas consulta.
	// `session` y `flow` dependen de módulos por interfaz, pero nada les impide
	// conocer `store` (el primero a través de `store.Sesiones`); lo que no
	// pueden es abrir la base, y eso lo coge la regla global de abajo.
}

// TestLimitesDeImporteEntreModulos — cada módulo respeta sus prohibiciones de
// importe (límites de DOMAIN.md §2).
func TestLimitesDeImporteEntreModulos(t *testing.T) {
	for modulo, prohibidos := range limiteImportes {
		dir := filepath.Join("..", "internal", modulo)
		entradas, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("módulo %s: %v", modulo, err)
		}
		for _, e := range entradas {
			n := e.Name()
			if e.IsDir() || !strings.HasSuffix(n, ".go") || strings.HasSuffix(n, "_test.go") {
				continue
			}
			f, err := parser.ParseFile(token.NewFileSet(), filepath.Join(dir, n), nil, parser.ImportsOnly)
			if err != nil {
				t.Fatalf("parsear %s/%s: %v", modulo, n, err)
			}
			for _, imp := range f.Imports {
				r := strings.Trim(imp.Path.Value, `"`)
				for _, p := range prohibidos {
					if r == p {
						t.Errorf("internal/%s/%s importa %q, que le está prohibido", modulo, n, r)
					}
				}
			}
		}
	}
}

// TestStoreEsElUnicoPuntoDeEscrituraSQLite — invariante de DECISIONS.md. Solo
// internal/store importa database/sql o el driver.
func TestStoreEsElUnicoPuntoDeEscrituraSQLite(t *testing.T) {
	err := filepath.WalkDir(filepath.Join("..", "internal"), func(ruta string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(ruta, ".go") || strings.HasSuffix(ruta, "_test.go") {
			return nil
		}
		if strings.Contains(filepath.ToSlash(ruta), "internal/store/") {
			return nil
		}
		f, err := parser.ParseFile(token.NewFileSet(), ruta, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range f.Imports {
			r := strings.Trim(imp.Path.Value, `"`)
			if r == "database/sql" || r == "modernc.org/sqlite" {
				t.Errorf("%s importa %q: solo internal/store puede abrir la base", ruta, r)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("recorrido: %v", err)
	}
}

// TestSinTablasProhibidas — el esquema tiene exactamente las seis tablas de
// TABLES.md. No hay tabla de cola (es una proyección del TODO, DOMAIN.md §3),
// ni tabla de grafo (vive en el frontmatter), ni tabla de usuarios (no hay
// cuentas). Si aparece una séptima, es un error de arquitectura.
func TestSinTablasProhibidas(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "internal", "store", "schema.sql"))
	if err != nil {
		t.Fatalf("leer schema.sql: %v", err)
	}
	permitidas := map[string]bool{
		"sessions": true, "messages": true, "reasoning": true,
		"approvals": true, "context_audit": true, "change_history": true,
	}
	var encontradas []string
	for _, m := range regexpCreateTable.FindAllStringSubmatch(string(b), -1) {
		nombre := m[1]
		if !permitidas[nombre] {
			t.Errorf("la tabla %q no existe en el esquema aprobado", nombre)
		}
		encontradas = append(encontradas, nombre)
	}
	if len(encontradas) != 6 {
		t.Errorf("el esquema tiene %d tablas, quiero 6: %v", len(encontradas), encontradas)
	}
	for _, prohibida := range []string{"cola", "queue", "grafo", "graph", "usuarios", "users", "todo"} {
		if permitidas[prohibida] {
			continue
		}
		for _, e := range encontradas {
			if strings.EqualFold(e, prohibida) {
				t.Errorf("apareció la tabla prohibida %q", e)
			}
		}
	}
}

// TestSinORM — decisión confirmada: "Sin ORM, SQL directo en store". Ninguna
// dependencia del go.mod puede ser un ORM.
func TestSinORM(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	orms := []string{"gorm.io/", "entgo.io/ent", "upper.io", "xorm.io", "go-gorm"}
	for _, o := range orms {
		if strings.Contains(string(b), o) {
			t.Errorf("go.mod contiene %q: la decisión es SQL directo sin ORM", o)
		}
	}
}
