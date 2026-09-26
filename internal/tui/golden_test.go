package tui

// Salidas doradas de las vistas (T-F011-04, TESTING.md §3): "un cambio de
// estilo se revisa a la vista, no a ciegas". Las doradas se generan la primera
// vez y a partir de ahí cualquier cambio de formato falla la prueba hasta que
// se regeneran a conciencia con `go test ./internal/tui/ -update`.
//
// Se guardan sin códigos de color para que se puedan revisar a simple vista;
// la comparación va contra el render con el perfil de color fijo en TestMain.

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"localcli/internal/session"
)

var actualizarDoradas = flag.Bool("update", false, "regenera las salidas doradas de testdata/")

func compararDorada(t *testing.T, nombre, got string) {
	t.Helper()
	ruta := filepath.Join("testdata", nombre)
	if _, err := os.Stat(ruta); os.IsNotExist(err) || *actualizarDoradas {
		if err := os.WriteFile(ruta, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("dorada %s escrita: revísala a la vista antes de aprobar cambios", nombre)
	}
	dorado, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	if got != string(dorado) {
		t.Errorf("la vista %s cambió frente a su dorada (regenera con -update si el cambio es intencionado):\n--- render ---\n%s\n--- dorada ---\n%s", nombre, got, string(dorado))
	}
}

func TestLasDoradasDeVistaSeMantienen(t *testing.T) {
	// Panel de datos con el fixture completo de los nueve datos.
	p := Panel{
		Abierto:            true,
		Sesion:             "api de pedidos",
		Estado:             session.EstadoTrabajando,
		Tokens:             1234,
		TokensEstimados:    true,
		LimiteTokens:       2000,
		ElementoActual:     "T-B014",
		ElementosRestantes: 3,
		Ruta:               "/tmp/proyecto",
		GitRama:            "master",
		Capa:               "backend",
		TareasGrandes:      2,
		Aprobaciones:       1,
		Agente:             "build",
		Proyecto:           Nombre,
		Version:            Version,
	}
	compararDorada(t, "panel.golden", sinEstilo(p.Render(AnchoPanel-4))+"\n")

	// Ayuda con el mapa por defecto.
	a := Nuevo(&puertoStub{})
	compararDorada(t, "ayuda.golden", sinEstilo(a.viewAyuda())+"\n")

	// Selector con las tres sesiones y sus estados.
	s := Selector{}
	s.Abrir([]session.Sesion{
		{ID: "s1", Nombre: "primera", Estado: session.EstadoInactiva},
		{ID: "s2", Nombre: "segunda", Estado: session.EstadoTrabajando},
		{ID: "s3", Nombre: "tercera", Estado: session.EstadoEsperandoPermiso},
	}, "s1")
	compararDorada(t, "selector.golden", sinEstilo(s.Render())+"\n")

	// Panel de aprobaciones con dos líneas, la primera seleccionada.
	ap := Aprobaciones{}
	ap.Fijar([]Aprobacion{
		{ID: "a1", Sesion: "api", Descripcion: "crear archivo"},
		{ID: "a2", Sesion: "cli", Descripcion: "borrar carpeta"},
	})
	compararDorada(t, "aprobaciones.golden", sinEstilo(ap.Render())+"\n")
}
