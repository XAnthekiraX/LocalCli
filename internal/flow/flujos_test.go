package flow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// flujoJSON construye un flujo mínimo válido con ese comando y nombre.
func flujoJSON(comando, nombre string) string {
	return `{"comando":"` + comando + `","nombre":"` + nombre + `","descripcion":"d",` +
		`"peticion":"p","pregunta":"q","etapas":[` +
		`{"id":"e1","nombre":"n","agente":"plan","pregunta":"¿q?","entrega":true}]}`
}

func escribirFlujo(t *testing.T, dir, archivo, contenido string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, archivo), []byte(contenido), 0o644); err != nil {
		t.Fatal(err)
	}
}

func contieneAviso(avisos []string, busca string) bool {
	for _, a := range avisos {
		if strings.Contains(a, busca) {
			return true
		}
	}
	return false
}

func TestCargaAditivaDeLasDosRaices(t *testing.T) {
	base := t.TempDir()
	proyecto := t.TempDir()
	escribirFlujo(t, base, "crear.json", flujoJSON("/crear", "crear"))
	escribirFlujo(t, proyecto, "propio.json", flujoJSON("/propio", "propio"))

	c, avisos := CargarAditivo(base, proyecto)

	if len(avisos) != 0 {
		t.Fatalf("no debería haber avisos: %v", avisos)
	}
	if c.Len() != 2 {
		t.Fatalf("el catálogo = %d flujos, quiero 2: %v", c.Len(), c.Comandos())
	}
	if _, ok := c.PorComando("/propio"); !ok {
		t.Fatal("el flujo propio no se cargó")
	}
}

func TestSinProyectoSoloOficiales(t *testing.T) {
	base := t.TempDir()
	escribirFlujo(t, base, "crear.json", flujoJSON("/crear", "crear"))

	c, avisos := CargarAditivo(base, filepath.Join(t.TempDir(), "no-existe"))

	if len(avisos) != 0 {
		t.Fatalf("una raíz propia ausente no es un error: %v", avisos)
	}
	if c.Len() != 1 {
		t.Fatalf("el catálogo = %d flujos, quiero 1", c.Len())
	}
}

func TestJSONRotoNoTumbaElArranque(t *testing.T) {
	base := t.TempDir()
	proyecto := t.TempDir()
	escribirFlujo(t, base, "crear.json", flujoJSON("/crear", "crear"))
	escribirFlujo(t, proyecto, "roto.json", "{no es json}")

	c, avisos := CargarAditivo(base, proyecto)

	if _, ok := c.PorComando("/crear"); !ok {
		t.Fatal("el flujo oficial debería seguir cargado")
	}
	if c.Len() != 1 {
		t.Fatalf("solo debería quedar el oficial: %v", c.Comandos())
	}
	if !contieneAviso(avisos, "roto.json") {
		t.Fatalf("el aviso debería nombrar el archivo roto: %v", avisos)
	}
}

func TestColisionConservaElOficial(t *testing.T) {
	base := t.TempDir()
	proyecto := t.TempDir()
	escribirFlujo(t, base, "crear.json", flujoJSON("/crear", "crear"))
	escribirFlujo(t, proyecto, "mio.json", flujoJSON("/crear", "mio"))

	c, avisos := CargarAditivo(base, proyecto)

	f, ok := c.PorComando("/crear")
	if !ok {
		t.Fatal("`/crear` debería seguir en el catálogo")
	}
	if f.Nombre != "crear" {
		t.Fatalf("debería conservarse el oficial, no el propio: %q", f.Nombre)
	}
	if c.Len() != 1 {
		t.Fatalf("el catálogo debería tener solo el oficial: %v", c.Comandos())
	}
	if !contieneAviso(avisos, "se conserva el oficial") {
		t.Fatalf("el aviso debería explicar la colisión: %v", avisos)
	}
}

func TestCamposDesconocidosSeRechazan(t *testing.T) {
	base := t.TempDir()
	escribirFlujo(t, base, "raro.json",
		`{"comando":"/raro","nombre":"raro","pregunta":"q","etapas":[],"inventado":true}`)

	c, avisos := CargarAditivo(base, "")

	if c.Len() != 0 {
		t.Fatalf("un campo desconocido debería rechazar el flujo: %v", c.Comandos())
	}
	if !contieneAviso(avisos, "raro.json") {
		t.Fatalf("el aviso debería nombrar el archivo: %v", avisos)
	}
}

func TestFlujoInvalidoSeIgnora(t *testing.T) {
	base := t.TempDir()
	escribirFlujo(t, base, "sin_entrega.json",
		`{"comando":"/x","nombre":"x","pregunta":"q","etapas":[{"id":"e1","agente":"plan","pregunta":"q"}]}`)

	c, avisos := CargarAditivo(base, "")

	if c.Len() != 0 {
		t.Fatalf("un flujo sin entrega no debería cargar: %v", c.Comandos())
	}
	if !contieneAviso(avisos, "entrega") {
		t.Fatalf("el aviso debería mencionar la entrega: %v", avisos)
	}
}

func TestDecodificarFlujoValido(t *testing.T) {
	f, err := DecodificarFlujo([]byte(flujoJSON("/crear", "crear")))
	if err != nil {
		t.Fatalf("un flujo válido no debería fallar: %v", err)
	}
	if f.Comando != "/crear" || f.Nombre != "crear" || len(f.Etapas) != 1 {
		t.Fatalf("el flujo se decodificó mal: %+v", f)
	}
}
