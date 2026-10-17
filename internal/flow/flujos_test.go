package flow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// flujoJSON construye un flujo mínimo válido con ese comando.
func flujoJSON(comando string) string {
	return flujoJSONEtiquetado(comando, "n")
}

// flujoJSONEtiquetado es el mismo flujo, con el nombre de su única etapa a
// elección: sirve para distinguir dos definiciones del mismo comando.
func flujoJSONEtiquetado(comando, etiqueta string) string {
	return `{"comando":"` + comando + `","etapas":[` +
		`{"id":"e1","nombre":"` + etiqueta + `","orden":"o","respuesta":"r"}]}`
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
	escribirFlujo(t, base, "crear.json", flujoJSON("/crear"))
	escribirFlujo(t, proyecto, "propio.json", flujoJSON("/propio"))

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
	escribirFlujo(t, base, "crear.json", flujoJSON("/crear"))

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
	escribirFlujo(t, base, "crear.json", flujoJSON("/crear"))
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
	escribirFlujo(t, base, "crear.json", flujoJSONEtiquetado("/crear", "oficial"))
	escribirFlujo(t, proyecto, "mio.json", flujoJSONEtiquetado("/crear", "propio"))

	c, avisos := CargarAditivo(base, proyecto)

	f, ok := c.PorComando("/crear")
	if !ok {
		t.Fatal("`/crear` debería seguir en el catálogo")
	}
	if len(f.Etapas) != 1 || f.Etapas[0].Nombre != "oficial" {
		t.Fatalf("debería conservarse el oficial, no el propio: %+v", f.Etapas)
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
		`{"comando":"/raro","etapas":[],"inventado":true}`)

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
	escribirFlujo(t, base, "sin_orden.json",
		`{"comando":"/x","etapas":[{"id":"e1","nombre":"n"}]}`)

	c, avisos := CargarAditivo(base, "")

	if c.Len() != 0 {
		t.Fatalf("una etapa sin `orden` no debería cargar: %v", c.Comandos())
	}
	if !contieneAviso(avisos, "orden") {
		t.Fatalf("el aviso debería mencionar el `orden` que falta: %v", avisos)
	}
}

// El error de un flujo con campos del esquema antiguo sugiere regenerarlo.
func TestEsquemaViejoSugiereRegenerar(t *testing.T) {
	base := t.TempDir()
	escribirFlujo(t, base, "viejo.json",
		`{"comando":"/viejo","nombre":"viejo","peticion":"p","etapas":[`+
			`{"id":"e1","nombre":"n","agente":"plan","pregunta":"q","entrega":true}]}`)

	_, avisos := CargarAditivo(base, "")

	if !contieneAviso(avisos, "esquema anterior") {
		t.Fatalf("el aviso debería orientar a regenerar el flujo: %v", avisos)
	}
}

func TestDecodificarFlujoValido(t *testing.T) {
	f, err := DecodificarFlujo([]byte(flujoJSON("/crear")))
	if err != nil {
		t.Fatalf("un flujo válido no debería fallar: %v", err)
	}
	if f.Comando != "/crear" || len(f.Etapas) != 1 {
		t.Fatalf("el flujo se decodificó mal: %+v", f)
	}
	if f.Etapas[0].Type != TipoLineal {
		t.Fatalf("una etapa sin `type` es `linear`: %q", f.Etapas[0].Type)
	}
}

func TestEtapaLoopTomaLaCotaPorDefecto(t *testing.T) {
	f, err := DecodificarFlujo([]byte(
		`{"comando":"/loop","etapas":[{"id":"e1","nombre":"n","orden":"o","type":"loop"}]}`))
	if err != nil {
		t.Fatalf("una etapa `loop` sin cota es válida: %v", err)
	}
	if !f.Etapas[0].EsLoop() {
		t.Fatal("la etapa debería ser `loop`")
	}
	if f.Etapas[0].MaxIteraciones != IteracionesLoopPorDefecto {
		t.Fatalf("cota por defecto = %d, quiero %d", f.Etapas[0].MaxIteraciones, IteracionesLoopPorDefecto)
	}
}

func TestEtapaLinearNoAdmiteCota(t *testing.T) {
	_, err := DecodificarFlujo([]byte(
		`{"comando":"/x","etapas":[{"id":"e1","nombre":"n","orden":"o","respuesta":"r","max_iteraciones":3}]}`))
	if err == nil {
		t.Fatal("`max_iteraciones` fuera de un `loop` debería fallar")
	}
}

func TestTypeDesconocidoSeRechaza(t *testing.T) {
	_, err := DecodificarFlujo([]byte(
		`{"comando":"/x","etapas":[{"id":"e1","nombre":"n","orden":"o","type":"raro"}]}`))
	if err == nil {
		t.Fatal("un `type` desconocido debería fallar")
	}
}

func TestEtapaLinearExigeRespuesta(t *testing.T) {
	_, err := DecodificarFlujo([]byte(
		`{"comando":"/x","etapas":[{"id":"e1","nombre":"n","orden":"o"}]}`))
	if err == nil {
		t.Fatal("una etapa `linear` sin `respuesta` debería fallar")
	}
}
