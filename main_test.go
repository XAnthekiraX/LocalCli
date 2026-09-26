package main

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNewRootModel — humo del stack: el modelo raíz se construye y cumple tea.Model.
func TestNewRootModel(t *testing.T) {
	m := newRootModel("LocalCli")
	if m.Init() != nil {
		t.Fatal("Init debe devolver nil en el modelo provisional")
	}
	if got := m.(rootModel).View(); got == "" {
		t.Fatal("View no debe devolver vacío")
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
