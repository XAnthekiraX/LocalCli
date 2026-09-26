package exec

// output.go — T-B009-04: capturar salida con tope y marcar `truncado`.
//
// Fuente de verdad: ai/docs/backend/DECISIONS.md ("Límites de comando fijos en
// la primera versión: 120 s de tiempo y 10 KB de salida. 10 KB basta para
// errores de compilación y resúmenes de pruebas, y lo cortado se marca con
// `truncado`") y ai/docs/specs/SPEC-TOOLS.md §Otras reglas ("La salida está
// limitada. Un comando que no termina o que genera salida sin fin no puede
// colgar el harness ni llenar el contexto. Si se corta, se avisa").
//
// El tope es compartido por stdout y stderr: 10 KB de salida en total. `exec`
// escribe ambos desde goroutines distintas, así que la cuenta lleva mutex.

import (
	"bytes"
	"sync"
)

// LimiteSalidaBytes es el tope de salida de un comando.
const LimiteSalidaBytes = 10 * 1024

// capturaSalida acumula stdout y stderr por separado con un tope compartido.
type capturaSalida struct {
	mu       sync.Mutex
	restante int
	truncado bool
	stdout   bytes.Buffer
	stderr   bytes.Buffer
}

func nuevaCaptura(limite int) *capturaSalida {
	return &capturaSalida{restante: limite}
}

func (c *capturaSalida) Stdout() *escritorCaptura { return &escritorCaptura{c: c, destino: &c.stdout} }
func (c *capturaSalida) Stderr() *escritorCaptura { return &escritorCaptura{c: c, destino: &c.stderr} }

// Truncado informa si algo se cortó.
func (c *capturaSalida) Truncado() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.truncado
}

// Salidas devuelve el texto acumulado de cada flujo.
func (c *capturaSalida) Salidas() (string, string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.stdout.String(), c.stderr.String()
}

// escritorCaptura es un io.Writer que comparte el tope de su captura.
type escritorCaptura struct {
	c       *capturaSalida
	destino *bytes.Buffer
}

func (w *escritorCaptura) Write(p []byte) (int, error) {
	w.c.mu.Lock()
	defer w.c.mu.Unlock()
	total := len(p)
	if w.c.restante <= 0 {
		w.c.truncado = true
		return total, nil // se descarta el resto, pero no se miente con un short write
	}
	if len(p) > w.c.restante {
		p = p[:w.c.restante]
		w.c.truncado = true
	}
	w.destino.Write(p)
	w.c.restante -= len(p)
	return total, nil
}
