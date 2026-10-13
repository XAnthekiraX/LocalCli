package exec

import (
	"bytes"
	"sync"
	"time"
)

// TiempoLimite es el tope de tiempo de un comando; LimiteSalidaBytes el de su salida.
const (
	TiempoLimite      = 120 * time.Second
	LimiteSalidaBytes = 10 * 1024
)

// captura acumula stdout y stderr por separado con un tope compartido.
type captura struct {
	mu       sync.Mutex
	restante int
	truncado bool
	stdout   bytes.Buffer
	stderr   bytes.Buffer
}

func nuevaCaptura(limite int) *captura { return &captura{restante: limite} }

func (c *captura) Stdout() *escritor { return &escritor{c: c, destino: &c.stdout} }
func (c *captura) Stderr() *escritor { return &escritor{c: c, destino: &c.stderr} }

func (c *captura) Truncado() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.truncado
}

func (c *captura) Salidas() (string, string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.stdout.String(), c.stderr.String()
}

type escritor struct {
	c       *captura
	destino *bytes.Buffer
}

func (w *escritor) Write(p []byte) (int, error) {
	w.c.mu.Lock()
	defer w.c.mu.Unlock()
	total := len(p)
	if w.c.restante <= 0 {
		w.c.truncado = true
		return total, nil
	}
	if len(p) > w.c.restante {
		p = p[:w.c.restante]
		w.c.truncado = true
	}
	w.c.restante -= len(p)
	w.destino.Write(p)
	return total, nil
}
