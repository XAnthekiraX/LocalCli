package panel

import (
	"strings"
	"testing"

	"localcli/internal/tui/render"
)

// El panel pinta sus bloques y su pie de tres filas con los nueve datos.
func TestElPanelPintaSusBloquesYSuPie(t *testing.T) {
	m := Nuevo()
	m.Datos = Datos{
		Sesion:             "arreglar el login",
		Estado:             "trabajando",
		ContextoTokens:     1234,
		TokensEstimados:    true,
		LimiteTokens:       2000,
		ElementoActual:     "T-F009",
		ElementosRestantes: 3,
		Capa:               "frontend",
		TareasGrandes:      2,
		Aprobaciones:       1,
		Agente:             "build",
		Ruta:               "/tmp/proyecto",
		GitRama:            "refact/logic",
		GitCambios:         2,
		Proyecto:           render.Nombre,
		Version:            render.Version,
		Tareas:             []Tarea{{Contenido: "escribir las doradas", Estado: EnProgreso}},
	}

	pantalla := render.SinANSI(m.Render(Ancho, 30))
	for _, quiero := range []string{
		"arreglar el login (trabajando)",
		"CONTEXTO", "1234 tokens (estimado)", "61% usada",
		"▾ TODO", "T-F009 · quedan 3",
		"LISTA DE TAREAS", "[•] escribir las doradas",
		"ESTADO", "Capa y cola", "Aprobaciones 1 esperando decisión", "Agente build",
		"Git refact/logic · 2", "[/tmp/proyecto]", "LocalCli · v0.1 alpha",
	} {
		if !strings.Contains(pantalla, quiero) {
			t.Errorf("el panel no pinta %q:\n%s", quiero, pantalla)
		}
	}
}

// Con todo hecho o cancelado la lista estorba y se oculta; el resto del panel
// sigue ahí.
func TestLaListaDeTareasSeOcultaSinNadaAccionable(t *testing.T) {
	m := Nuevo()
	m.Datos = Datos{Sesion: "primera", Tareas: []Tarea{
		{Contenido: "hecha", Estado: Completada},
		{Contenido: "cancelada", Estado: Cancelada},
	}}

	pantalla := render.SinANSI(m.Render(Ancho, 30))
	if strings.Contains(pantalla, "LISTA DE TAREAS") {
		t.Fatalf("un checklist terminado no debería pintarse:\n%s", pantalla)
	}
	if !strings.Contains(pantalla, "primera") {
		t.Fatalf("el resto del panel debería seguir:\n%s", pantalla)
	}
}

// Una lista larga no se come el pie: lo que no cabe se resume.
func TestUnaListaLargaSeResumeYNoSeComeElPie(t *testing.T) {
	m := Nuevo()
	m.Sesion, m.GitRama = "primera", "main"
	for i := 0; i < 30; i++ {
		m.Datos.Tareas = append(m.Datos.Tareas, Tarea{Contenido: "paso pendiente", Estado: Pendiente})
	}

	pantalla := render.SinANSI(m.Render(Ancho, 24))
	if !strings.Contains(pantalla, "más") {
		t.Fatalf("la lista recortada debería resumirse:\n%s", pantalla)
	}
	for _, pie := range []string{"[—]", "LocalCli"} {
		if !strings.Contains(pantalla, pie) {
			t.Fatalf("el pie perdió %q:\n%s", pie, pantalla)
		}
	}
}

// Sin git inicializado la fila del pie lo dice, y una columna más baja que el pie
// se recorta por arriba para no perder la ruta ni la firma.
func TestSinGitYColumnaBaja(t *testing.T) {
	m := Nuevo()
	m.Ruta = "/tmp/proyecto"

	if got := render.SinANSI(m.Render(Ancho, 24)); !strings.Contains(got, "sin iniciar") {
		t.Fatalf("sin git debería decirlo:\n%s", got)
	}
	bajo := render.SinANSI(m.Render(Ancho, 2))
	if !strings.Contains(bajo, "LocalCli") {
		t.Fatalf("en una columna baja la firma debería sobrevivir:\n%s", bajo)
	}
	if strings.Contains(bajo, "CONTEXTO") {
		t.Fatalf("en una columna baja el principio es lo que se cae:\n%s", bajo)
	}
}
