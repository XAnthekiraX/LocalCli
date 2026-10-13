package context

import (
	"context"
	"strings"
	"testing"
)

type fuenteFake struct {
	docs      []Documento
	contenido map[string]string
}

func (f *fuenteFake) Documentos() []Documento { return f.docs }
func (f *fuenteFake) Leer(ruta string) (string, error) {
	return f.contenido[ruta], nil
}

type modeloFake struct {
	elegidos []string
}

func (m *modeloFake) Elegir(context.Context, string, []string) ([]string, error) {
	return m.elegidos, nil
}

type auditoriaFake struct {
	filas []string
}

func (a *auditoriaFake) Registrar(sesion, etapa, documento string, d Decision, motivo string, tokens int) error {
	a.filas = append(a.filas, etapa+":"+documento+":"+string(d))
	return nil
}

func nuevaNodo(elegidos []string, contenido map[string]string) (*Nodo, *auditoriaFake) {
	aud := &auditoriaFake{}
	nodo := &Nodo{
		Fuente: &fuenteFake{
			docs: []Documento{
				{Ruta: "api.md", Dependencias: []string{"dto.md"}},
				{Ruta: "dto.md", Dependencias: []string{"entity.md"}},
				{Ruta: "entity.md"},
			},
			contenido: contenido,
		},
		Modelo:      &modeloFake{elegidos: elegidos},
		Auditoria:   aud,
		Presupuesto: 1000,
	}
	return nodo, aud
}

func TestCargaSoloLoElegidoYAudita(t *testing.T) {
	nodo, aud := nuevaNodo([]string{"api.md", "dto.md"}, map[string]string{
		"api.md": "# API\ncontenido de la api",
		"dto.md": "# DTO\ncontenido del dto",
	})
	bloque, err := nodo.Preparar(context.Background(), Peticion{Etapa: "plan", Objetivo: "implementar la api"})
	if err != nil {
		t.Fatal(err)
	}
	if len(bloque.Documentos) != 2 {
		t.Fatalf("debería traer solo lo elegido: %+v", bloque.Documentos)
	}
	if len(aud.filas) != 2 {
		t.Fatalf("debería auditar cada documento: %v", aud.filas)
	}
}

func TestDocumentoInexistenteDetieneLaEtapa(t *testing.T) {
	nodo, _ := nuevaNodo([]string{"api.md", "fantasma.md"}, map[string]string{"api.md": "# API"})
	_, err := nodo.Preparar(context.Background(), Peticion{Etapa: "plan", Objetivo: "x"})
	if err == nil || !strings.Contains(err.Error(), CodDocNotFound) {
		t.Fatalf("un documento pedido que no existe detiene la etapa: %v", err)
	}
}

func TestChatSinSemillaNoConsultaAlModelo(t *testing.T) {
	nodo, aud := nuevaNodo([]string{"api.md"}, map[string]string{"api.md": "# API"})
	bloque, err := nodo.Preparar(context.Background(), Peticion{SinSemilla: true, Objetivo: "hola"})
	if err != nil {
		t.Fatal(err)
	}
	if !bloque.SinSemilla || len(bloque.Documentos) != 0 {
		t.Fatalf("sin semilla el bloque va solo con el objetivo: %+v", bloque)
	}
	if len(aud.filas) != 1 {
		t.Fatalf("debería auditar el bloque sin semilla: %v", aud.filas)
	}
}

func TestExcluyeLoYaEntregado(t *testing.T) {
	nodo, _ := nuevaNodo([]string{"api.md", "dto.md"}, map[string]string{"api.md": "# API", "dto.md": "# DTO"})
	bloque, err := nodo.Preparar(context.Background(), Peticion{Objetivo: "x", YaEntregados: []string{"api.md"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range bloque.Documentos {
		if d.Ruta == "api.md" {
			t.Fatal("un documento ya entregado no debería repetirse")
		}
	}
}

func TestRecortaPorSeccionContraElPresupuesto(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 50; i++ {
		b.WriteString("# Sección ")
		b.WriteString(strings.Repeat("x", 200))
		b.WriteString("\n")
	}
	recortado, truncado, tokens := RecortarPorSeccion(b.String(), 60)
	if !truncado {
		t.Fatal("un documento que desborda debería recortarse")
	}
	if tokens > 60 {
		t.Fatalf("no debería pasarse del presupuesto: %d", tokens)
	}
	if !strings.HasPrefix(recortado, "# Sección") {
		t.Fatalf("debería entregar el inicio: %q", recortado)
	}
}
