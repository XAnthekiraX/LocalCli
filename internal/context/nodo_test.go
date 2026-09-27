package context

import (
	"context"
	"errors"
	"sort"
	"strings"
	"testing"

	"localcli/internal/store"
)

// --- stubs -----------------------------------------------------------------

type grafoStub struct {
	contenido map[string]string
	deps      map[string][]string
}

func (g grafoStub) Rutas() []string {
	out := make([]string, 0, len(g.contenido))
	for r := range g.contenido {
		out = append(out, r)
	}
	sort.Strings(out)
	return out
}

func (g grafoStub) Contenido(ruta string) (string, error) {
	c, ok := g.contenido[ruta]
	if !ok {
		return "", errors.New("E_DOC_NOT_FOUND: " + ruta)
	}
	return c, nil
}

func (g grafoStub) Cierre(ruta string) ([]string, error) {
	if _, ok := g.contenido[ruta]; !ok {
		return nil, errors.New("E_DOC_NOT_FOUND: " + ruta)
	}
	return g.deps[ruta], nil
}

type auditorStub struct{ filas []store.ContextAudit }

func (a *auditorStub) Registrar(x *store.ContextAudit) error {
	a.filas = append(a.filas, *x)
	return nil
}

// --- tests -----------------------------------------------------------------

// TestCandidatosSigueDependencias — los candidatos son la semilla y el cierre
// de sus dependencias declaradas.
func TestCandidatosSigueDependencias(t *testing.T) {
	g := grafoStub{
		contenido: map[string]string{"a.md": "a", "b.md": "b", "c.md": "c"},
		deps:      map[string][]string{"a.md": {"b.md", "c.md"}},
	}
	got, err := Candidatos(g, []string{"a.md"})
	if err != nil {
		t.Fatal(err)
	}
	quiero := []string{"a.md", "b.md", "c.md"}
	if strings.Join(got, ",") != strings.Join(quiero, ",") {
		t.Fatalf("candidatos = %v, quiero %v", got, quiero)
	}
}

// TestPrepararRegistraIncluidos — lo seleccionado entra al bloque y queda
// auditado.
func TestPrepararRegistraIncluidos(t *testing.T) {
	g := grafoStub{contenido: map[string]string{"a.md": "## A\n\ntexto"}}
	aud := &auditorStub{}
	n := &Nodo{
		Grafo:   g,
		Modelo:  &modeloStub{devuelve: []string{"a.md"}},
		Auditor: aud,
		Limite:  1000,
	}
	armado, err := n.Preparar(context.Background(), SolicitudContexto{
		Objetivo: "objetivo", Etapa: "etapa-1", SessionID: "s1",
	})
	if err != nil {
		t.Fatalf("Preparar: %v", err)
	}
	if len(armado.Documentos) != 1 || !strings.Contains(armado.Bloque, "## A") {
		t.Fatalf("bloque inesperado: %q", armado.Bloque)
	}
	if len(aud.filas) != 1 || aud.filas[0].Decision != store.AuditIncluido || aud.filas[0].Stage != "etapa-1" {
		t.Fatalf("auditoría inesperada: %+v", aud.filas)
	}
}

// TestPrepararDocumentoInexistenteDetiene — si el modelo pide algo que no
// existe, la etapa se detiene.
func TestPrepararDocumentoInexistenteDetiene(t *testing.T) {
	g := grafoStub{contenido: map[string]string{"a.md": "a"}}
	n := &Nodo{Grafo: g, Modelo: &modeloStub{devuelve: []string{"no-existe.md"}}}
	_, err := n.Preparar(context.Background(), SolicitudContexto{Objetivo: "x", Etapa: "e"})
	if err == nil || !strings.Contains(err.Error(), "E_DOC_NOT_FOUND") {
		t.Fatalf("err = %v, quiero E_DOC_NOT_FOUND", err)
	}
}

// TestPrepararRecortaYAuditaDescartes — al recortar, lo que sale queda
// registrado con motivo.
func TestPrepararRecortaYAuditaDescartes(t *testing.T) {
	contenidoA := strings.Repeat("## A\n\ntexto\n", 10)
	contenidoB := strings.Repeat("## B\n\ntexto\n", 10)
	g := grafoStub{contenido: map[string]string{"a.md": contenidoA, "b.md": contenidoB}}
	aud := &auditorStub{}
	// El límite es justo el tamaño del primer documento: entra entero y el
	// segundo ya no tiene cupo, así que queda descartado con su motivo.
	limite := EstimarTokens(contenidoA)
	n := &Nodo{Grafo: g, Modelo: &modeloStub{devuelve: []string{"a.md", "b.md"}}, Auditor: aud, Limite: limite}
	armado, err := n.Preparar(context.Background(), SolicitudContexto{Objetivo: "x", Etapa: "e", SessionID: "s"})
	if err != nil {
		t.Fatalf("Preparar: %v", err)
	}
	if armado.Tokens > limite {
		t.Fatalf("tokens = %d, quiero <= %d", armado.Tokens, limite)
	}
	var descartado bool
	for _, f := range aud.filas {
		if f.Decision == store.AuditDescartado && f.Reason == "" {
			t.Error("un descarte debe llevar motivo")
		}
		if f.Decision == store.AuditDescartado {
			descartado = true
		}
	}
	if !descartado {
		t.Fatalf("esperaba al menos un descarte auditado: %+v", aud.filas)
	}
}

// TestPrepararFallbackSinModelo — si el modelo no decide, se usa la semilla.
func TestPrepararFallbackSinModelo(t *testing.T) {
	g := grafoStub{contenido: map[string]string{"a.md": "a", "b.md": "b"}}
	n := &Nodo{Grafo: g, Modelo: &modeloStub{devuelve: nil}, Limite: 1000}
	armado, err := n.Preparar(context.Background(), SolicitudContexto{
		Objetivo: "x", Etapa: "e", Semilla: []string{"a.md"},
	})
	if err != nil {
		t.Fatalf("Preparar: %v", err)
	}
	if len(armado.Documentos) != 1 || armado.Documentos[0].Ruta != "a.md" {
		t.Fatalf("documentos = %+v, quiero solo la semilla", armado.Documentos)
	}
}

// TestContextoParaDevuelveBloque — la interfaz que consume flow devuelve el
// bloque ensamblado.
func TestContextoParaDevuelveBloque(t *testing.T) {
	g := grafoStub{contenido: map[string]string{"a.md": "## A\n\ntexto"}}
	n := &Nodo{Grafo: g, Modelo: &modeloStub{devuelve: []string{"a.md"}}, Limite: 1000, Etapa: "e"}
	bloque, err := n.ContextoPara(context.Background(), "e", "objetivo")
	if err != nil {
		t.Fatalf("ContextoPara: %v", err)
	}
	if !strings.Contains(bloque, "objetivo") || !strings.Contains(bloque, "## A") {
		t.Fatalf("bloque = %q", bloque)
	}
}

// TestElChatNoVuelcaElProyecto — sin selección y sin semilla, el bloque del
// chat queda sin documentos: el modelo no puede responder resumiendo una
// estructura que no recibió (el modelo nunca ve el proyecto entero).
func TestElChatNoVuelcaElProyecto(t *testing.T) {
	g := grafoStub{contenido: map[string]string{
		"AGENTS.md": "# agents", "PROJECT.md": "# proyecto",
	}}
	n := &Nodo{Grafo: g, Modelo: &modeloStub{devuelve: nil}, Limite: 1000}
	bloque, err := n.ContextoPara(context.Background(), EtapaChat, "hola")
	if err != nil {
		t.Fatalf("ContextoPara: %v", err)
	}
	if strings.Contains(bloque, "# agents") || strings.Contains(bloque, "# proyecto") {
		t.Fatalf("el chat no debe recibir el proyecto entero: %q", bloque)
	}
	if !strings.Contains(bloque, "hola") {
		t.Fatalf("el bloque sin documentos aún lleva el objetivo: %q", bloque)
	}
}

// TestElChatSinSemillaNoConsultaAlModelo — el chat sin semilla no pregunta al
// modelo qué documentos quiere: su bloque va solo con el objetivo, así que la
// consulta no cambiaría lo entregado y costaría una generación entera por turno.
func TestElChatSinSemillaNoConsultaAlModelo(t *testing.T) {
	g := grafoStub{contenido: map[string]string{"a.md": "## A\n\ntexto"}}
	m := &modeloStub{devuelve: []string{"a.md"}}
	n := &Nodo{Grafo: g, Modelo: m, Limite: 1000}
	bloque, err := n.ContextoPara(context.Background(), EtapaChat, "hola")
	if err != nil {
		t.Fatalf("ContextoPara: %v", err)
	}
	if m.objetivo != "" {
		t.Errorf("el chat sin semilla no debe consultar al modelo, y lo hizo con %q", m.objetivo)
	}
	if strings.Contains(bloque, "## A") {
		t.Errorf("sin consulta el bloque va solo con el objetivo: %q", bloque)
	}
}

// TestElChatConSemillaSiConsultaAlModelo — con semilla, el modelo sigue
// decidiendo qué es relevante: la omisión es solo del chat sin semilla.
func TestElChatConSemillaSiConsultaAlModelo(t *testing.T) {
	g := grafoStub{contenido: map[string]string{"a.md": "## A\n\ntexto"}}
	m := &modeloStub{devuelve: []string{"a.md"}}
	n := &Nodo{Grafo: g, Modelo: m, Limite: 1000, Semilla: []string{"a.md"}}
	if _, err := n.ContextoPara(context.Background(), EtapaChat, "hola"); err != nil {
		t.Fatalf("ContextoPara: %v", err)
	}
	if m.objetivo != "hola" {
		t.Errorf("con semilla el modelo sigue decidiendo qué es relevante: %q", m.objetivo)
	}
}

// TestUnaEtapaDeFlujoConservaElRespaldo — fuera del chat, sin selección ni
// semilla se usan los candidatos: los flujos siguen recibiendo contexto.
func TestUnaEtapaDeFlujoConservaElRespaldo(t *testing.T) {
	g := grafoStub{contenido: map[string]string{"a.md": "## A\n\ntexto"}}
	n := &Nodo{Grafo: g, Modelo: &modeloStub{devuelve: nil}, Limite: 1000}
	bloque, err := n.ContextoPara(context.Background(), "etapa_flujo", "objetivo")
	if err != nil {
		t.Fatalf("ContextoPara: %v", err)
	}
	if !strings.Contains(bloque, "## A") {
		t.Fatalf("la etapa de flujo conserva el respaldo a candidatos: %q", bloque)
	}
}
