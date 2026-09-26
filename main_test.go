package main

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNuevaApp — humo del cableado real: la app de producción se construye
// sobre su puerto y es un modelo completo de la TUI (tui/app.go), no el
// modelo provisional que había antes de T-B014.
func TestNuevaApp(t *testing.T) {
	a := &Arranque{puerto: &Adaptador{}}
	app := nuevaApp(a)
	if app.View() == "" {
		t.Fatal("View no debe devolver vacío al arrancar")
	}
}

// TestStoreEsElUnicoEscritorDeSQLite — invariante estructural de DECISIONS.md
// ("store es el único que escribe en SQLite"). El binario llegó a abrir la base
// por su cuenta con sql.Open y sin los PRAGMA de conexión, así que el caso queda
// fijado aquí: solo internal/store puede abrir SQLite o importar database/sql.
func TestStoreEsElUnicoEscritorDeSQLite(t *testing.T) {
	const (
		storeDir = "internal/store"
	)

	err := filepath.WalkDir(".", func(ruta string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			base := d.Name()
			if base == ".git" || base == ".serena" || base == ".localcli" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(ruta, ".go") || strings.HasSuffix(ruta, "_test.go") {
			return nil
		}
		if strings.HasPrefix(filepath.ToSlash(ruta), storeDir+"/") {
			return nil
		}
		f, err := parser.ParseFile(token.NewFileSet(), ruta, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range f.Imports {
			rutaImp := strings.Trim(imp.Path.Value, `"`)
			if rutaImp == "database/sql" || rutaImp == "modernc.org/sqlite" {
				t.Errorf("%s importa %q: solo %s puede abrir la base", ruta, rutaImp, storeDir)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("recorrido del árbol: %v", err)
	}
}
