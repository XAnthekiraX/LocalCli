package flow

import (
	"os"
	"path/filepath"
	"testing"

	"localcli/internal/task"
)

// TestGenerarTODOCreaArchivoLegible — el TODO generado existe, y `task` lo
// puede leer: los elementos y sus campos vuelven a salir del archivo.
func TestGenerarTODOCreaArchivoLegible(t *testing.T) {
	raiz := t.TempDir()
	if err := os.MkdirAll(filepath.Join(raiz, "ai", "tasks", "backend"), 0o755); err != nil {
		t.Fatal(err)
	}
	elementos := []ElementoTODO{
		{ID: "T-B100", Accion: task.AccionCrear, Descripcion: "Modelo", Documentos: []string{"backend/01-domain/DOMAIN"}},
		{ID: "T-B101", Accion: task.AccionCrear, Descripcion: "Repositorio", DependeDe: []string{"T-B100"}},
	}
	ruta, err := GenerarTODO(raiz, task.CapaBackend, elementos)
	if err != nil {
		t.Fatalf("GenerarTODO: %v", err)
	}
	if _, err := os.Stat(ruta); err != nil {
		t.Fatalf("el archivo no existe: %v", err)
	}

	b, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	leidos, err := task.ParseTabla(b, task.CapaBackend)
	if err != nil {
		t.Fatalf("task.ParseTabla: %v", err)
	}
	if len(leidos) != 2 {
		t.Fatalf("elementos leídos = %d, quiero 2", len(leidos))
	}
	if leidos[0].ID != "T-B100" || leidos[0].Accion != task.AccionCrear || leidos[0].Estado != task.EstadoPendiente {
		t.Errorf("primer elemento = %+v", leidos[0])
	}
	if len(leidos[0].Documentos) != 1 || leidos[0].Documentos[0] != "backend/01-domain/DOMAIN" {
		t.Errorf("documentos = %v, quiero la ruta referenciada", leidos[0].Documentos)
	}
	if len(leidos[1].DependeDe) != 1 || leidos[1].DependeDe[0] != "T-B100" {
		t.Errorf("dependencias = %v, quiero T-B100", leidos[1].DependeDe)
	}
}
