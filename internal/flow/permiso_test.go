package flow

import (
	"reflect"
	"testing"
)

func flujo(agentes ...string) Flujo {
	f := Flujo{Nombre: "f", Comando: "/f", Pregunta: "p", Etapas: []Etapa{{ID: "entrega", Nombre: "Fin", Agente: "plan", Pregunta: "p", Entrega: true}}}
	for i, a := range agentes {
		f.Etapas = append([]Etapa{{ID: string(rune('a' + i)), Nombre: "e", Agente: a, Pregunta: "p"}}, f.Etapas...)
	}
	return f
}

func TestPermisoNecesarioDerivaDeLasEtapas(t *testing.T) {
	if got := PermisoNecesario(flujo("plan")); !reflect.DeepEqual(got, []string{"read"}) {
		t.Fatalf("un flujo de solo plan pide read: %v", got)
	}
	if got := PermisoNecesario(flujo("plan", "build")); !reflect.DeepEqual(got, []string{"read", "write"}) {
		t.Fatalf("un flujo con build pide read+write: %v", got)
	}
}

func TestPuedeLanzarSegunLosPermisos(t *testing.T) {
	soloLectura := flujo("plan")
	if !PuedeLanzar(soloLectura, []string{"read"}) {
		t.Fatal("plan con read debería lanzar un flujo de solo plan")
	}
	conBuild := flujo("plan", "build")
	if PuedeLanzar(conBuild, []string{"read"}) {
		t.Fatal("plan con solo read no debería lanzar un flujo que escribe")
	}
	if !PuedeLanzar(conBuild, []string{"read", "write", "edit"}) {
		t.Fatal("build debería lanzar el flujo completo")
	}
}
