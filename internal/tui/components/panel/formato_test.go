package panel

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

// Un dato vacío se pinta como ausente: el panel no deja huecos ni inventa valores.
func TestValorODefectoMarcaLoAusente(t *testing.T) {
	if got := valorODefecto(""); got != "—" {
		t.Errorf("un dato vacío = %q, quiero %q", got, "—")
	}
	if got := valorODefecto("   "); got != "—" {
		t.Errorf("un dato en blanco = %q, quiero %q", got, "—")
	}
	if got := valorODefecto("build"); got != "build" {
		t.Errorf("un dato con valor = %q", got)
	}
}

// El estado acompaña al nombre entre paréntesis, y sin estado no deja paréntesis
// colgando.
func TestEstadoEntreParentesis(t *testing.T) {
	if got := estadoEntreParentesis("trabajando"); got != " (trabajando)" {
		t.Errorf("estado = %q", got)
	}
	if got := estadoEntreParentesis(""); got != "" {
		t.Errorf("sin estado = %q, quiero vacío", got)
	}
}

// El porcentaje de contexto se mide sobre el límite, no pasa del 100% y sin
// límite desconocido vale cero.
func TestPorcentajeDeContexto(t *testing.T) {
	casos := []struct {
		tokens, limite, quiero int
	}{
		{1234, 2000, 61},
		{0, 0, 0},
		{500, 0, 0},
		{1000, 1000, 100},
		{3000, 2000, 100},
	}
	for _, c := range casos {
		m := Model{Datos: Datos{ContextoTokens: c.tokens, LimiteTokens: c.limite}}
		if got := m.PorcentajeContexto(); got != c.quiero {
			t.Errorf("%d de %d = %d%%, quiero %d%%", c.tokens, c.limite, got, c.quiero)
		}
	}
}

// El aviso de contexto se enciende al llegar al 80%: antes de que la petición
// falle (SPEC-PANEL-CONTEXTO §Flujos alternativos).
func TestElAvisoDeContextoSeEnciendeAlOchenta(t *testing.T) {
	if (Model{Datos: Datos{ContextoTokens: 790, LimiteTokens: 1000}}).ContextoApretado() {
		t.Error("al 79% no debería avisar")
	}
	if !(Model{Datos: Datos{ContextoTokens: 800, LimiteTokens: 1000}}).ContextoApretado() {
		t.Error("al 80% debería avisar")
	}
}

// El contexto dice si el número es una estimación, y el límite dice cuándo es el
// tope del harness y no la ventana del modelo.
func TestElContextoDiceSiEsEstimacionYDeDondeVieneElLimite(t *testing.T) {
	m := Model{Datos: Datos{ContextoTokens: 1234, TokensEstimados: true, LimiteTokens: 2000, LimiteDelHarness: true}}
	if got := m.textoContexto(); got != "1234 tokens (estimado)" {
		t.Errorf("el contexto = %q", got)
	}
	if got := m.textoPorcentajeContexto(); got != "61% usada · límite del harness" {
		t.Errorf("el porcentaje = %q", got)
	}
	sinEstimacion := Model{Datos: Datos{ContextoTokens: 10}}
	if got := sinEstimacion.textoContexto(); got != "10 tokens" {
		t.Errorf("sin estimación el contexto = %q", got)
	}
}

// Git sin rama dice «sin iniciar»: es un dato, no un «no sé».
func TestGitSinRamaDiceSinIniciar(t *testing.T) {
	if got := (Model{}).textoGit(); got != "sin iniciar" {
		t.Errorf("sin rama = %q, quiero %q", got, "sin iniciar")
	}
	m := Model{Datos: Datos{GitRama: "refact/logic", GitCambios: 2}}
	if got := m.textoGit(); got != "refact/logic · 2" {
		t.Errorf("con rama = %q", got)
	}
}

// La firma del harness pinta lo que haya, sin dejar un separador colgando.
func TestLaFirmaDelHarnessNoDejaSeparadoresColgando(t *testing.T) {
	casos := []struct{ proyecto, version, quiero string }{
		{"LocalCli", "v0.1 alpha", "LocalCli · v0.1 alpha"},
		{"LocalCli", "", "LocalCli"},
		{"", "v0.1 alpha", "v0.1 alpha"},
	}
	for _, c := range casos {
		m := Model{Datos: Datos{Proyecto: c.proyecto, Version: c.version}}
		if got := m.firmaHarness(); got != c.quiero {
			t.Errorf("firma(%q,%q) = %q, quiero %q", c.proyecto, c.version, got, c.quiero)
		}
	}
}

// La capa y la cola se pintan juntas, con la capa ausente marcada.
func TestCapaYCola(t *testing.T) {
	m := Model{Datos: Datos{Capa: "frontend", TareasGrandes: 2}}
	if got := m.textoCapa(); got != "frontend · 2 tareas grandes" {
		t.Errorf("capa = %q", got)
	}
	if got := (Model{}).textoCapa(); got != "— · 0 tareas grandes" {
		t.Errorf("sin capa = %q", got)
	}
}

// Cada estado de un paso tiene su glifo, y un estado desconocido no inventa marca:
// queda como pendiente.
func TestGlifoDeTarea(t *testing.T) {
	casos := map[string]string{
		EnProgreso:  "[•]",
		Completada:  "[✓]",
		Cancelada:   "[x]",
		Pendiente:   "[ ]",
		"inventado": "[ ]",
	}
	for estado, quiero := range casos {
		if got := glifoDeTarea(estado); got != quiero {
			t.Errorf("glifo(%q) = %q, quiero %q", estado, got, quiero)
		}
	}
}

// La lista de pasos solo se pinta si queda algo accionable.
func TestSoloEsAccionableLoQueQuedaPorHacer(t *testing.T) {
	if accionable(nil) {
		t.Error("sin pasos no hay nada accionable")
	}
	if accionable([]Tarea{{Estado: Completada}, {Estado: Cancelada}}) {
		t.Error("con todo hecho o cancelado no hay nada accionable")
	}
	if !accionable([]Tarea{{Estado: Completada}, {Estado: Pendiente}}) {
		t.Error("un paso pendiente es accionable")
	}
	if !accionable([]Tarea{{Estado: EnProgreso}}) {
		t.Error("un paso en curso es accionable")
	}
}

// El recorte deja el principio y pone «…»; con una sola columna cabe la marca.
// La ruta, en cambio, se envuelve: no se pierde ningún tramo y, cuando puede, el
// corte cae tras un separador.
func TestTruncarYEnvolverRuta(t *testing.T) {
	if got := truncar("abcdefgh", 5); got != "abcd…" {
		t.Errorf("truncar = %q, quiero %q", got, "abcd…")
	}
	if got := truncar("abc", 5); got != "abc" {
		t.Errorf("lo que cabe no se toca: %q", got)
	}
	if got := truncar("abc", 1); got != "…" {
		t.Errorf("con una columna = %q, quiero %q", got, "…")
	}
	// Lo que cabe en una fila no se parte.
	if got := envolverPorRuta("[~/proyecto]", 20); len(got) != 1 || got[0] != "[~/proyecto]" {
		t.Errorf("una ruta que cabe = %q", got)
	}
	larga := "[~/Documentos/01-Proyectos/DESKTOP/LocalCli]"
	got := envolverPorRuta(larga, 20)
	if len(got) < 2 {
		t.Fatalf("una ruta larga debería partirse: %q", got)
	}
	if unido := strings.Join(got, ""); unido != larga {
		t.Errorf("la ruta partida perdió tramos: %q", unido)
	}
	for _, l := range got {
		if utf8.RuneCountInString(l) > 20 {
			t.Errorf("una fila desborda el ancho: %q", l)
		}
	}
	if !strings.HasSuffix(got[0], "/") {
		t.Errorf("el corte debería caer tras un separador: %q", got[0])
	}
}

// La ruta del proyecto se abrevia con el home del usuario; lo que no cuelga de él
// se pinta tal cual.
func TestLaRutaSeAbreviaConElHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		t.Skip("el sistema no declara el home del usuario")
	}
	if got := rutaBreve(filepath.Join(home, "proyecto")); got != "~/proyecto" {
		t.Errorf("ruta dentro del home = %q, quiero %q", got, "~/proyecto")
	}
	if got := rutaBreve(home); got != "~" {
		t.Errorf("el home = %q, quiero %q", got, "~")
	}
	if got := rutaBreve("/tmp/proyecto"); got != "/tmp/proyecto" {
		t.Errorf("ruta fuera del home = %q", got)
	}
}
