// ventana_test.go — T-B037-06: la ventana y la ficha se cachean POR MOTOR.
//
// Fuente de verdad: ai/docs/specs/SPEC-MODELO-MOTOR («dos motores del mismo
// tipo no comparten ficha de capacidades ni ventana, aunque sirvan un modelo
// del mismo nombre») y ai/docs/backend/05-quality/TESTING.md §llm.
package llm

import (
	"context"
	"path/filepath"
	"testing"
)

// motorFalso es un doble de `llm.Motor` que declara lo que se le diga: sirve
// para probar la caché por motor sin red.
type motorFalso struct {
	nombre           string
	url              string
	modelos          []Modelo
	ventanaServidor  int
	ventanaDeclaraDo bool // true = la declara por petición (Ollama)
	ficha            map[string][]string
}

func (m *motorFalso) Nombre() string  { return m.nombre }
func (m *motorFalso) BaseURL() string { return m.url }

func (m *motorFalso) Chat(context.Context, Peticion) (<-chan Evento, error) {
	ch := make(chan Evento)
	close(ch)
	return ch, nil
}

func (m *motorFalso) ListarModelos(context.Context) ([]Modelo, error) {
	return m.modelos, nil
}

func (m *motorFalso) Capacidades(_ context.Context, modelo string) ([]string, error) {
	return m.ficha[modelo], nil
}

func (m *motorFalso) VentanaDeContexto(context.Context, string) (int, bool, error) {
	return m.ventanaServidor, m.ventanaDeclaraDo, nil
}

// TestDosMotoresDelMismoTipoNoCompartenVentana — dos motores `ollama` con el
// mismo nombre de modelo declaran ventanas distintas: cada uno cachea la suya
// y la ventana efectiva de uno no se cuela en la del otro.
func TestDosMotoresDelMismoTipoNoCompartenVentana(t *testing.T) {
	var reg Registro
	reg.ruta = filepath.Join(t.TempDir(), "motores.json")

	// Los dos, mismo tipo y mismo modelo, ventanas distintas del servidor.
	pequeño := &motorFalso{
		nombre: "ollama", url: "http://localhost:11434",
		modelos:         []Modelo{{Nombre: "qwen3:8b", ContextLength: 4096}},
		ventanaServidor: 8192,
	}
	grande := &motorFalso{
		nombre: "ollama", url: "http://portatil.local:11434",
		modelos:         []Modelo{{Nombre: "qwen3:8b", ContextLength: 32768}},
		ventanaServidor: 16384,
	}

	const tope = 65536
	vPequeña := reg.VentanaDeContexto(context.Background(), pequeño, "motor-a", "qwen3:8b", tope)
	vGrande := reg.VentanaDeContexto(context.Background(), grande, "motor-b", "qwen3:8b", tope)

	// motor-a: min(modelo 4096, servidor 8192, tope 65536) = 4096.
	if vPequeña != 4096 {
		t.Errorf("motor-a: ventana efectiva 4096, salió %d", vPequeña)
	}
	// motor-b: min(modelo 32768, servidor 16384, tope 65536) = 16384.
	if vGrande != 16384 {
		t.Errorf("motor-b: ventana efectiva 16384, salió %d", vGrande)
	}

	// Preguntar de nuevo no vuelve a consultar: sale de la caché por motor.
	vPequeña2 := reg.VentanaDeContexto(context.Background(), grande, "motor-a", "qwen3:8b", tope)
	if vPequeña2 != vPequeña {
		t.Errorf("la caché de motor-a no debe contaminarse con motor-b: %d != %d", vPequeña2, vPequeña)
	}
}

// TestElTopeMandaSobreLoQueDeclaraElMotor — la efectiva es la MENOR entre
// modelo, servidor y tope: un motor que declara más contexto que el tope usa
// el tope.
func TestElTopeMandaSobreLoQueDeclaraElMotor(t *testing.T) {
	var reg Registro
	reg.ruta = filepath.Join(t.TempDir(), "motores.json")

	m := &motorFalso{
		nombre: "ollama", url: "http://localhost:11434",
		modelos:         []Modelo{{Nombre: "m", ContextLength: 200000}},
		ventanaServidor: 131072,
	}
	if got := reg.VentanaDeContexto(context.Background(), m, "id", "m", 16384); got != 16384 {
		t.Errorf("con declaraciones mayores que el tope, manda el tope: salió %d", got)
	}
}

// TestLaFichaDeCapacidadesEsPorMotor — la ficha cacheada de un motor no la
// ve el otro, aunque se llame igual el modelo; y una ficha vacía no se
// cachea: un fallo no debe fijar «no puede».
func TestLaFichaDeCapacidadesEsPorMotor(t *testing.T) {
	var reg Registro
	reg.ruta = filepath.Join(t.TempDir(), "motores.json")

	reg.CachearFicha("motor-a", "qwen3:8b", []string{CapacidadHerramientas, CapacidadVision})
	reg.CachearFicha("motor-b", "qwen3:8b", []string{CapacidadPensar})
	reg.CachearFicha("motor-c", "qwen3:8b", nil)

	if caps, ok := reg.Ficha("motor-a", "qwen3:8b"); !ok || len(caps) != 2 {
		t.Errorf("motor-a cachea su propia ficha: %v, %v", caps, ok)
	}
	if caps, ok := reg.Ficha("motor-b", "qwen3:8b"); !ok || len(caps) != 1 || caps[0] != CapacidadPensar {
		t.Errorf("motor-b no ve la ficha de motor-a: %v, %v", caps, ok)
	}
	if _, ok := reg.Ficha("motor-c", "qwen3:8b"); ok {
		t.Error("una ficha vacía no se cachea: un fallo no fija «no puede»")
	}
}
