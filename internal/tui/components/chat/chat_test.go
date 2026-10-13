package chat

import (
	"strings"
	"testing"
	"time"

	"localcli/internal/tui/render"
)

// Una línea del sistema se interpone en su sitio cronológico: el texto anterior
// queda cerrado arriba y lo que llega después abre un globo nuevo.
func TestUnaLineaDelSistemaCierraElGloboAnterior(t *testing.T) {
	m := Nuevo()
	m.Fragmento("antes de la herramienta")
	m.AnotarInvocacion("leer", "LEER", "internal/tui/app.go")
	m.Fragmento("después de la herramienta")
	m.CerrarTurno()

	piezas := m.Piezas()
	if len(piezas) != 3 {
		t.Fatalf("el hilo tiene %d piezas, quiero 3 (globo, línea, globo)", len(piezas))
	}
	primero, ok := piezas[0].(*Segmento)
	if !ok || !strings.Contains(primero.Texto, "antes de la herramienta") {
		t.Fatalf("la primera pieza no es el texto previo: %+v", piezas[0])
	}
	if _, ok := piezas[1].(*Linea); !ok {
		t.Fatalf("la segunda pieza no es la línea de herramienta: %+v", piezas[1])
	}
	ultimo, ok := piezas[2].(*Segmento)
	if !ok || !strings.Contains(ultimo.Texto, "después de la herramienta") {
		t.Fatalf("la tercera pieza no es el texto posterior: %+v", piezas[2])
	}
}

// Un turno que termina justo después de una herramienta, sin texto nuevo, no deja
// un globo vacío: su tiempo se cuelga del último segmento del agente.
func TestTurnoSinTextoNoDejaGloboVacio(t *testing.T) {
	m := Nuevo()
	m.AñadirUsuario("arregla el login")
	m.Fragmento("voy a mirarlo")
	m.AnotarInvocacion("leer", "LEER", "internal/tui/app.go")
	m.inicio = time.Now().Add(-2 * time.Second)

	m.CerrarTurno()

	if len(m.Piezas()) != 3 {
		t.Fatalf("el hilo tiene %d piezas, quiero 3 sin globo vacío", len(m.Piezas()))
	}
	agente, ok := m.Piezas()[1].(*Segmento)
	if !ok || agente.Hablante != Agente {
		t.Fatalf("la pieza del agente no está donde se esperaba: %+v", m.Piezas()[1])
	}
	if agente.Duracion < 2*time.Second {
		t.Fatalf("el tiempo del turno no se colgó del último segmento: %v", agente.Duracion)
	}
	if m.HayTurno() {
		t.Fatal("el turno debería haberse cerrado")
	}
}

// Cerrar un turno con texto deja el globo con su duración y su razonamiento.
func TestCerrarTurnoGuardaTextoRazonamientoYTiempo(t *testing.T) {
	m := Nuevo()
	m.AñadirUsuario("resúmelo")
	m.FragmentoRazonamiento("pienso el resumen")
	m.Fragmento("aquí va el resumen")
	m.inicio = time.Now().Add(-1500 * time.Millisecond)

	m.CerrarTurno()

	agente, ok := m.Piezas()[1].(*Segmento)
	if !ok || agente.Texto != "aquí va el resumen" {
		t.Fatalf("el segmento del agente no se cerró bien: %+v", m.Piezas()[1])
	}
	if agente.Razonamiento != "pienso el resumen" || agente.SinRazonamiento {
		t.Fatalf("el razonamiento no se guardó con su segmento: %+v", agente)
	}
	if agente.Duracion < 1500*time.Millisecond {
		t.Fatalf("la duración guardada = %v", agente.Duracion)
	}
}

// El indicador en vivo dice qué está pasando y solo existe mientras hay turno.
func TestLaActividadDiceQueEstaPasando(t *testing.T) {
	m := Nuevo()
	if m.Actividad(0) != "" {
		t.Fatal("sin turno no debería haber indicador")
	}
	m.AñadirUsuario("hola")
	if act := m.Actividad(0); !strings.Contains(act, "Pensando") {
		t.Fatalf("sin respuesta debería decir «Pensando»: %q", act)
	}
	m.Fragmento("ya escribo")
	if act := m.Actividad(0); !strings.Contains(act, "Generando") {
		t.Fatalf("con respuesta debería decir «Generando»: %q", act)
	}
	m.AnotarInvocacion("leer", "LEER", "x")
	if act := m.Actividad(0); !strings.Contains(act, "Usando herramienta: leer") {
		t.Fatalf("con herramienta debería decir cuál: %q", act)
	}
	// El glifo gira con el frame y nunca sale vacío.
	if glifoActividad(0) == glifoActividad(1) {
		t.Fatal("el glifo debería cambiar de frame")
	}
	if glifoActividad(-1) == "" || glifoActividad(len(framesActividad)) != glifoActividad(0) {
		t.Fatal("el glifo debería envolver su índice y no salir vacío")
	}
}

// El razonamiento no se vuelca por defecto, y revelarlo lo pinta sin perderlo.
func TestElRazonamientoSeRevelaConCtrlR(t *testing.T) {
	m := Nuevo()
	m.AñadirSegmento(Segmento{Hablante: Agente, Texto: "la respuesta", Razonamiento: "el razonamiento crudo"})

	if hilo := render.SinANSI(m.Hilo(80)); strings.Contains(hilo, "el razonamiento crudo") {
		t.Fatalf("el razonamiento no debería volcarse por defecto:\n%s", hilo)
	}
	m.AlternarRazonamiento()
	if hilo := render.SinANSI(m.Hilo(80)); !strings.Contains(hilo, "el razonamiento crudo") {
		t.Fatalf("revelado, el razonamiento debería verse:\n%s", hilo)
	}
	m.AlternarRazonamiento()
	if hilo := render.SinANSI(m.Hilo(80)); strings.Contains(hilo, "el razonamiento crudo") {
		t.Fatal("volver a pulsar `ctrl+r` debería ocultarlo")
	}
}

// Un turno que no entregó razonamiento lo dice al revelarlo, en vez de dejar la
// pantalla muda.
func TestSinRazonamientoLoAvisaAlRevelar(t *testing.T) {
	m := Nuevo()
	m.MostrarRazonamiento = true
	m.AñadirSegmento(Segmento{Hablante: Agente, Texto: "solo respuesta", SinRazonamiento: true})

	hilo := render.SinANSI(m.Hilo(80))
	if !strings.Contains(hilo, "no entregó razonamiento") {
		t.Fatalf("debería avisar de que no hay razonamiento:\n%s", hilo)
	}
}

// Sin subir, la vista sigue el final; al subir respeta la posición y lo avisa.
func TestElHistorialSigueElFinalHastaQueSeSube(t *testing.T) {
	m := Nuevo()
	for i := 0; i < 20; i++ {
		m.AñadirSegmento(Segmento{Hablante: Agente, Texto: "linea " + string(rune('a'+i))})
	}

	abajo := render.SinANSI(m.Vista(40, 6))
	if !strings.Contains(abajo, "↑") || !strings.Contains(abajo, "líneas arriba") {
		t.Fatalf("con historial fuera de la ventana debería avisar arriba:\n%s", abajo)
	}
	if m.scrolleado {
		t.Fatal("sin subir no debería quedar scrolleado")
	}
	m.Subir(2)
	if !m.scrolleado || m.offset == 0 {
		t.Fatalf("subir debería mover el offset y dejar de seguir el final: %d", m.offset)
	}
	m.IrAlFinal()
	if m.scrolleado || m.offset != m.maxOffset {
		t.Fatalf("ir al final debería volver a pegar la vista: %d/%d", m.offset, m.maxOffset)
	}
	// Un historial que cabe entero no se recorta ni avisa.
	corto := Nuevo()
	corto.AñadirSegmento(Segmento{Hablante: Agente, Texto: "una sola línea"})
	if got := render.SinANSI(corto.Vista(40, 10)); strings.Contains(got, "líneas arriba") {
		t.Fatalf("con el historial entero no debería haber aviso:\n%s", got)
	}
}

// Al vaciar el hilo —cambio de sesión— se sigue revelando el razonamiento, porque
// la preferencia es del usuario y no de la sesión.
func TestVaciarConservaElRazonamientoRevelado(t *testing.T) {
	m := Nuevo()
	m.AlternarRazonamiento()
	m.AñadirUsuario("de la sesión que se deja")
	m.Vaciar()

	if !m.MostrarRazonamiento {
		t.Fatal("vaciar debería conservar la revelación del razonamiento")
	}
	if !m.Vacio() || len(m.Piezas()) != 0 {
		t.Fatal("vaciar debería dejar el hilo sin piezas")
	}
}

// La línea de herramienta se completa en la misma línea que se abrió (FIFO), no
// en una nueva, y lo que no se midió no deja rastro de medida.
func TestLaLineaDeHerramientaSeCompletaEnSuSitio(t *testing.T) {
	m := Nuevo()
	m.AnotarInvocacion("leer", "LEER", "a.go")
	m.AnotarInvocacion("leer", "LEER", "b.go")
	m.CerrarHerramienta("leer", true, false, "10 líneas", "", 400*time.Millisecond)

	piezas := m.Piezas()
	if len(piezas) != 2 {
		t.Fatalf("el cierre de una herramienta no debería añadir líneas: %d", len(piezas))
	}
	primera := piezas[0].(*Linea).Texto
	if !strings.HasPrefix(primera, "✓ LEER [a.go]") {
		t.Fatalf("la línea cerrada no es la primera abierta: %q", primera)
	}
	if segunda := piezas[1].(*Linea).Texto; strings.HasPrefix(segunda, "✓") {
		t.Fatalf("la segunda línea sigue abierta y sin marca: %q", segunda)
	}
}

// Cancelar un turno conserva lo que el modelo ya entregó: lo parcial se cierra
// como un segmento con su tiempo, en vez de borrarse de la pantalla.
func TestCancelarTurnoConservaLoEntregado(t *testing.T) {
	m := Nuevo()
	m.AñadirUsuario("cuéntame")
	m.Fragmento("esto ya lo dije")
	m.inicio = time.Now().Add(-1 * time.Second)

	m.CancelarTurno()

	if len(m.Piezas()) != 2 {
		t.Fatalf("lo parcial debería quedar como un segmento: %d piezas", len(m.Piezas()))
	}
	agente, ok := m.Piezas()[1].(*Segmento)
	if !ok || agente.Hablante != Agente || agente.Texto != "esto ya lo dije" {
		t.Fatalf("lo parcial no se conservó como segmento del agente: %+v", m.Piezas()[1])
	}
	if agente.Duracion <= 0 {
		t.Fatalf("el segmento parcial debería guardar su tiempo: %v", agente.Duracion)
	}
	if m.HayTurno() {
		t.Fatal("el turno debería quedar cerrado tras cancelar")
	}
}
