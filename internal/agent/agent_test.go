package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func agenteDir(t *testing.T, raiz, nombre string) string {
	t.Helper()
	dir := filepath.Join(raiz, nombre)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func escribir(t *testing.T, dir, archivo, contenido string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, archivo), []byte(contenido), 0o644); err != nil {
		t.Fatal(err)
	}
}

// agenteValido deja una carpeta de agente válida y devuelve su ruta.
func agenteValido(t *testing.T, raiz, carpeta, nombre, prompt string) string {
	t.Helper()
	dir := agenteDir(t, raiz, carpeta)
	escribir(t, dir, ArchivoConfig, "name: "+nombre+"\ndescription: d\npermissions:\n  default: deny\n  read: allow\n")
	escribir(t, dir, ArchivoPrompt, prompt)
	return dir
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
	agenteValido(t, base, "plan", "plan", "eres plan")
	agenteValido(t, base, "build", "build", "eres build")
	agenteValido(t, proyecto, "experto", "experto", "eres un experto")

	c, avisos := CargarAditivo(base, proyecto)

	if len(avisos) != 0 {
		t.Fatalf("no debería haber avisos: %v", avisos)
	}
	if c.Len() != 3 {
		t.Fatalf("el catálogo = %d agentes, quiero 3: %v", c.Len(), c.Nombres())
	}
	if _, ok := c.Buscar("experto"); !ok {
		t.Fatal("el agente propio no se cargó")
	}
}

func TestSinProyectoSoloCarganLosBase(t *testing.T) {
	base := t.TempDir()
	agenteValido(t, base, "plan", "plan", "eres plan")

	c, avisos := CargarAditivo(base, filepath.Join(t.TempDir(), "no-existe"))

	if len(avisos) != 0 {
		t.Fatalf("una raíz propia ausente no es un error: %v", avisos)
	}
	if c.Len() != 1 {
		t.Fatalf("el catálogo = %d agentes, quiero 1", c.Len())
	}
}

func TestCarpetaSinPromptNoCarga(t *testing.T) {
	base := t.TempDir()
	dir := agenteDir(t, base, "sinprompt")
	escribir(t, dir, ArchivoConfig, "name: sinprompt\ndescription: d\n")
	agenteValido(t, base, "plan", "plan", "eres plan")

	c, avisos := CargarAditivo(base, "")

	if _, ok := c.Buscar("sinprompt"); ok {
		t.Fatal("una carpeta sin prompt.md no debería cargar")
	}
	if c.Len() != 1 {
		t.Fatalf("las demás carpetas deberían seguir cargando: %d", c.Len())
	}
	if !contieneAviso(avisos, "prompt.md") {
		t.Fatalf("el aviso debería nombrar prompt.md: %v", avisos)
	}
}

func TestPromptVacioNoCarga(t *testing.T) {
	base := t.TempDir()
	agenteValido(t, base, "vacio", "vacio", "")

	c, avisos := CargarAditivo(base, "")

	if c.Len() != 0 {
		t.Fatalf("un prompt vacío no debería cargar: %v", c.Nombres())
	}
	if !contieneAviso(avisos, "vacío") {
		t.Fatalf("el aviso debería decir que el prompt está vacío: %v", avisos)
	}
}

func TestCamposDesconocidosSeRechazan(t *testing.T) {
	for _, campo := range []string{"skills", "herramientas", "prompt"} {
		t.Run(campo, func(t *testing.T) {
			base := t.TempDir()
			dir := agenteDir(t, base, "raro")
			escribir(t, dir, ArchivoConfig, "name: raro\ndescription: d\n"+campo+": algo\n")
			escribir(t, dir, ArchivoPrompt, "prompt")

			c, avisos := CargarAditivo(base, "")

			if c.Len() != 0 {
				t.Fatalf("`%s` debería rechazar el agente: %v", campo, c.Nombres())
			}
			if !contieneAviso(avisos, campo) {
				t.Fatalf("el aviso debería nombrar `%s`: %v", campo, avisos)
			}
		})
	}
}

func TestDefaultAllowSeRechaza(t *testing.T) {
	base := t.TempDir()
	dir := agenteDir(t, base, "permisivo")
	escribir(t, dir, ArchivoConfig, "name: permisivo\ndescription: d\npermissions:\n  default: allow\n")
	escribir(t, dir, ArchivoPrompt, "prompt")

	c, avisos := CargarAditivo(base, "")

	if c.Len() != 0 {
		t.Fatalf("`default: allow` debería rechazar el agente: %v", c.Nombres())
	}
	if !contieneAviso(avisos, "default") {
		t.Fatalf("el aviso debería explicar el rechazo de `default`: %v", avisos)
	}
}

func TestPermisoDesconocidoSeRechaza(t *testing.T) {
	base := t.TempDir()
	dir := agenteDir(t, base, "raro")
	escribir(t, dir, ArchivoConfig, "name: raro\ndescription: d\npermissions:\n  volar: allow\n")
	escribir(t, dir, ArchivoPrompt, "prompt")

	c, avisos := CargarAditivo(base, "")

	if c.Len() != 0 {
		t.Fatalf("un permiso desconocido debería rechazar el agente: %v", c.Nombres())
	}
	if !contieneAviso(avisos, "volar") {
		t.Fatalf("el aviso debería nombrar el permiso: %v", avisos)
	}
}

func TestSinPermisosSoloConversacion(t *testing.T) {
	base := t.TempDir()
	dir := agenteDir(t, base, "charla")
	escribir(t, dir, ArchivoConfig, "name: charla\ndescription: d\n")
	escribir(t, dir, ArchivoPrompt, "prompt")

	c, _ := CargarAditivo(base, "")
	a, ok := c.Buscar("charla")

	if !ok {
		t.Fatal("un agente sin permisos debería cargar")
	}
	if !a.SoloConversacion() {
		t.Fatal("un agente sin permisos debería ser de solo conversación")
	}
}

// El nombre lo decide `name`, no la carpeta.
func TestElNombreLoDecideName(t *testing.T) {
	base := t.TempDir()
	agenteValido(t, base, "investigador", "research", "prompt")

	c, _ := CargarAditivo(base, "")

	if _, ok := c.Buscar("research"); !ok {
		t.Fatalf("el agente debería llamarse `research`: %v", c.Nombres())
	}
	if _, ok := c.Buscar("investigador"); ok {
		t.Fatal("el nombre de la carpeta no decide el nombre del agente")
	}
}

func TestElPromptSeLeeIntegro(t *testing.T) {
	base := t.TempDir()
	contenido := "línea uno\n\n  línea con sangría  \n"
	agenteValido(t, base, "plan", "plan", contenido)

	c, _ := CargarAditivo(base, "")
	a, _ := c.Buscar("plan")

	if a.Prompt != contenido {
		t.Fatalf("el prompt no se leyó íntegro: %q", a.Prompt)
	}
}

func TestAvisoDelFormatoAntiguo(t *testing.T) {
	base := t.TempDir()
	escribir(t, base, "viejo.json", "{}")
	agenteValido(t, base, "plan", "plan", "eres plan")

	c, avisos := CargarAditivo(base, "")

	if c.Len() != 1 {
		t.Fatalf("solo el agente de carpeta debería cargar: %v", c.Nombres())
	}
	if !contieneAviso(avisos, "viejo.json") {
		t.Fatalf("el aviso debería nombrar el archivo antiguo: %v", avisos)
	}
	if !contieneAviso(avisos, "agent.yaml") {
		t.Fatalf("el aviso debería decir cuál es el formato nuevo: %v", avisos)
	}
}

func TestColisionConservaElBase(t *testing.T) {
	base := t.TempDir()
	proyecto := t.TempDir()
	baseBuild := agenteValido(t, base, "build", "build", "eres build base")
	agenteValido(t, proyecto, "build", "build", "eres build propio")

	c, avisos := CargarAditivo(base, proyecto)

	a, ok := c.Buscar("build")
	if !ok {
		t.Fatal("`build` debería seguir en el catálogo")
	}
	if a.Origen != baseBuild {
		t.Fatalf("debería conservarse el base (%s), no el propio (%s)", baseBuild, a.Origen)
	}
	if c.Len() != 1 {
		t.Fatalf("el catálogo debería tener solo el base: %v", c.Nombres())
	}
	if !contieneAviso(avisos, "se conserva el base") {
		t.Fatalf("el aviso debería explicar la colisión: %v", avisos)
	}
}
