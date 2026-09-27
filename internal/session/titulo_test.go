package session

// Tests del título de sesión (T-B020): la primera petición pide al modelo un
// título, se persiste como nombre (sin cambiar el id) y no se vuelve a pedir;
// un fallo conserva el nombre provisional y se reintenta con la siguiente
// petición ([[specs/SPEC-SESIONES]]).

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"localcli/internal/flow"
)

// tituladorStub cuenta llamadas y devuelve un título fijo o un error.
type tituladorStub struct {
	mu     sync.Mutex
	titulo string
	err    error
	veces  int
	ultimo string
}

func (t *tituladorStub) Titulo(ctx context.Context, texto string) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.veces++
	t.ultimo = texto
	if t.err != nil {
		return "", t.err
	}
	return t.titulo, nil
}

func (t *tituladorStub) llamadas() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.veces
}

func (t *tituladorStub) fijar(titulo string, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.titulo, t.err = titulo, err
}

// esperarNombre espera a que la sesión tenga el nombre pedido.
func esperarNombre(t *testing.T, g *Gestor, id, quiero string) {
	t.Helper()
	limite := time.Now().Add(3 * time.Second)
	for time.Now().Before(limite) {
		s, err := g.Estado(id)
		if err != nil {
			t.Fatalf("Estado(%s): %v", id, err)
		}
		if s.Nombre == quiero {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	s, _ := g.Estado(id)
	t.Fatalf("nombre de %s = %q, quiero %q", id, s.Nombre, quiero)
}

func TestLaPrimeraPeticionGeneraElTituloDeLaSesion(t *testing.T) {
	g, _ := gestorDe(t, nuevoMotor(flow.EstadoTerminado))
	tit := &tituladorStub{titulo: `  "Documentar sesiones."  `}
	g.Titulador = tit

	ses, err := g.Crear(NombreProvisional, "")
	if err != nil {
		t.Fatalf("Crear: %v", err)
	}
	if err := g.Conversar(context.Background(), ses.ID, "plan", "documenta las sesiones"); err != nil {
		t.Fatalf("Conversar: %v", err)
	}
	esperarEstado(t, g, ses.ID, EstadoTerminada)
	// El título se limpia: sin comillas ni punto final.
	esperarNombre(t, g, ses.ID, "Documentar sesiones")

	if n := tit.llamadas(); n != 1 {
		t.Errorf("el título se pide una sola vez: %d", n)
	}
	if tit.ultimo != "documenta las sesiones" {
		t.Errorf("el título parte de la petición: %q", tit.ultimo)
	}

	// Segunda petición: ya titulada, no se vuelve a pedir.
	if err := g.Conversar(context.Background(), ses.ID, "plan", "y ahora otra cosa"); err != nil {
		t.Fatalf("Conversar: %v", err)
	}
	esperarEstado(t, g, ses.ID, EstadoTerminada)
	time.Sleep(40 * time.Millisecond)
	if n := tit.llamadas(); n != 1 {
		t.Errorf("una sesión ya titulada no vuelve a pedir título: %d", n)
	}
}

func TestUnFalloDelTituloConservaElProvisionalYReintenta(t *testing.T) {
	g, _ := gestorDe(t, nuevoMotor(flow.EstadoTerminado))
	tit := &tituladorStub{err: errors.New("modelo caído")}
	g.Titulador = tit
	ses, err := g.Crear(NombreProvisional, "")
	if err != nil {
		t.Fatalf("Crear: %v", err)
	}

	if err := g.Conversar(context.Background(), ses.ID, "plan", "hola"); err != nil {
		t.Fatalf("Conversar: %v", err)
	}
	esperarEstado(t, g, ses.ID, EstadoTerminada)
	time.Sleep(40 * time.Millisecond)
	if s, _ := g.Estado(ses.ID); s.Nombre != NombreProvisional {
		t.Errorf("sin título se conserva el provisional: %q", s.Nombre)
	}

	// El modelo ya responde: la siguiente petición reintenta y lo aplica.
	tit.fijar("Conversación de prueba", nil)
	if err := g.Conversar(context.Background(), ses.ID, "plan", "sigue"); err != nil {
		t.Fatalf("Conversar: %v", err)
	}
	esperarNombre(t, g, ses.ID, "Conversación de prueba")
	if n := tit.llamadas(); n != 2 {
		t.Errorf("se reintenta el título: %d", n)
	}
}

func TestSinTituladorLaSesionConservaElProvisional(t *testing.T) {
	g, _ := gestorDe(t, nuevoMotor(flow.EstadoTerminado))
	// g.Titulador queda nil.
	ses, err := g.Crear(NombreProvisional, "")
	if err != nil {
		t.Fatalf("Crear: %v", err)
	}
	if err := g.Conversar(context.Background(), ses.ID, "plan", "hola"); err != nil {
		t.Fatalf("Conversar: %v", err)
	}
	esperarEstado(t, g, ses.ID, EstadoTerminada)
	if s, _ := g.Estado(ses.ID); s.Nombre != NombreProvisional {
		t.Errorf("sin titulador no se inventa un nombre: %q", s.Nombre)
	}
}

func TestEsProvisional(t *testing.T) {
	if !EsProvisional(NombreProvisional) || !EsProvisional("  "+NombreProvisional+" ") {
		t.Error("el nombre provisional se reconoce con espacios")
	}
	if EsProvisional("otra cosa") {
		t.Error("un nombre propio no es provisional")
	}
}

func TestLimpiarTitulo(t *testing.T) {
	casos := []struct{ bruto, quiere string }{
		{`"Documentar sesiones."`, "Documentar sesiones"},
		{"- **Memoria de conversación**\nresto", "Memoria de conversación"},
		{"  varios    espacios ", "varios espacios"},
		{"", ""},
		{"```", ""},
	}
	for _, c := range casos {
		if got := LimpiarTitulo(c.bruto); got != c.quiere {
			t.Errorf("LimpiarTitulo(%q) = %q, quiero %q", c.bruto, got, c.quiere)
		}
	}
	// Se acota la longitud.
	largo := make([]rune, LongitudMaximaTitulo+20)
	for i := range largo {
		largo[i] = 'a'
	}
	if got := LimpiarTitulo(string(largo)); len([]rune(got)) != LongitudMaximaTitulo {
		t.Errorf("el título se acota a %d: %d", LongitudMaximaTitulo, len([]rune(got)))
	}
}
