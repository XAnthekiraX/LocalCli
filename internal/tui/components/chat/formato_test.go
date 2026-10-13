package chat

import (
	"strings"
	"testing"
	"time"

	"localcli/internal/tui/render"
)

// La duración se mide igual para una respuesta y para una herramienta:
// milisegundos por debajo del segundo, décimas de segundo por debajo del minuto y
// minutos y segundos a partir de ahí.
func TestFormatearDuracionEligeLaUnidad(t *testing.T) {
	casos := []struct {
		d      time.Duration
		quiero string
	}{
		{0, "0 ms"},
		{400 * time.Millisecond, "400 ms"},
		{999 * time.Millisecond, "999 ms"},
		{time.Second, "1.0 s"},
		{1500 * time.Millisecond, "1.5 s"},
		{59900 * time.Millisecond, "59.9 s"},
		{time.Minute, "1 m 0 s"},
		{90 * time.Second, "1 m 30 s"},
		{-time.Second, "0 ms"},
	}
	for _, c := range casos {
		if got := formatearDuracion(c.d); got != c.quiero {
			t.Errorf("formatearDuracion(%v) = %q, quiero %q", c.d, got, c.quiero)
		}
	}
}

// Una respuesta sin medición no pinta nada: ni un cero ni un hueco reservado.
func TestSufijoDuracionCallaSinMedicion(t *testing.T) {
	if got := sufijoDuracion(0); got != "" {
		t.Errorf("sin medición el sufijo debería estar vacío: %q", got)
	}
	if got := render.SinANSI(sufijoDuracion(1500 * time.Millisecond)); got != "(1.5 s)" {
		t.Errorf("el sufijo = %q, quiero %q", got, "(1.5 s)")
	}
}

// La línea de herramienta nace con su verbo y su tema, y el tema ausente no deja
// corchetes vacíos.
func TestLineaHerramientaAbierta(t *testing.T) {
	if got := LineaHerramientaAbierta("LEER", "internal/tui/app.go"); got != "LEER [internal/tui/app.go]" {
		t.Errorf("la línea abierta = %q", got)
	}
	if got := LineaHerramientaAbierta("EJEC", ""); got != "EJEC" {
		t.Errorf("sin tema la línea = %q, quiero %q", got, "EJEC")
	}
}

// La línea cerrada une marca, medida, recorte, motivo y tiempo en una sola línea,
// y lo que falta no deja hueco.
func TestLineaHerramientaCerrada(t *testing.T) {
	ok := LineaHerramientaCerrada("LEER [x]", true, false, "70 líneas", "", 400*time.Millisecond)
	if ok != "✓ LEER [x] · 70 líneas · 400 ms" {
		t.Errorf("la línea cerrada = %q", ok)
	}
	fallo := LineaHerramientaCerrada("EJEC [go test]", false, true, "", "el comando falló", 0)
	if fallo != "✗ EJEC [go test] · recortado · el comando falló" {
		t.Errorf("la línea fallida = %q", fallo)
	}
	// Sin medida, sin recorte, sin motivo y sin tiempo queda solo la marca.
	if got := LineaHerramientaCerrada("LEER [x]", true, false, "", "", 0); got != "✓ LEER [x]" {
		t.Errorf("la línea sin datos = %q, quiero %q", got, "✓ LEER [x]")
	}
	// El motivo viaja en una sola línea y recortado.
	largo := LineaHerramientaCerrada("CREAR [x]", false, false, "", strings.Repeat("motivo ", 20), 0)
	if strings.Contains(largo, "\n") {
		t.Errorf("el motivo no debería partir la línea: %q", largo)
	}
	if !strings.Contains(largo, "…") {
		t.Errorf("el motivo largo debería recortarse: %q", largo)
	}
}

// La línea de una etapa de flujo se anuncia solo con su nombre.
func TestLineaProceso(t *testing.T) {
	if got := LineaProceso("Diagnosticar"); got != "[Sub Proceso] Diagnosticar" {
		t.Errorf("la línea de proceso = %q", got)
	}
}
