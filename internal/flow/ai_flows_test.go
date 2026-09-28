package flow

import (
	"path/filepath"
	"reflect"
	"testing"
)

// TestElFlujoResolverDelProyectoCoincideConElRespaldo — el proyecto define
// /resolver en `ai/flows/resolver.json`. Debe coincidir con el respaldo Go
// (FlujoResolver), para que borrar el JSON no cambie el flujo y para que la
// definición del proyecto no derive en silencio de la oficial.
func TestElFlujoResolverDelProyectoCoincideConElRespaldo(t *testing.T) {
	raiz := filepath.Join("..", "..")
	c, err := CargarFlujos(raiz)
	if err != nil {
		t.Fatalf("CargarFlujos(%s): %v", raiz, err)
	}
	got, ok := c.PorNombre("resolver")
	if !ok {
		t.Fatal("el catálogo del proyecto no tiene el flujo resolver")
	}
	quiero := FlujoResolver()
	if !reflect.DeepEqual(got, quiero) {
		t.Fatalf("ai/flows/resolver.json y FlujoResolver() difieren:\njson = %+v\ngo   = %+v", got, quiero)
	}
}
