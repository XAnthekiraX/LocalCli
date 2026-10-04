// cola_test.go — T-B037-05: una cola por motor.
//
// Fuente de verdad: ai/docs/backend/DECISIONS.md («Una cola de inferencia por
// motor, no una global»), ai/docs/backend/04-infrastructure/INTEGRATIONS.md
// §Concurrencia y ai/docs/backend/05-quality/TESTING.md («dos motores
// distintos no se esperan»).
package llm

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestDosMotoresGeneranSimultaneamente — dos motores son dos servidores
// distintos: uno puede estar generando mientras el otro genera; las colas no
// se esperan entre sí.
func TestDosMotoresGeneranSimultaneamente(t *testing.T) {
	colas := NewColasInferencia()

	// El motor A toma su testigo y se queda generado hasta que le digamos.
	liberarA := make(chan struct{})
	haEntradoA := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = colas.Para("motor-a").Encolar(context.Background(), OpcionesEncolar{}, func(context.Context) error {
			close(haEntradoA)
			<-liberarA
			return nil
		})
	}()
	<-haEntradoA

	// Con A generando, B entra a la primera: no se espera al otro motor.
	conclusoB := make(chan struct{})
	go func() {
		defer close(conclusoB)
		_ = colas.Para("motor-b").Encolar(context.Background(), OpcionesEncolar{}, func(context.Context) error {
			return nil
		})
	}()

	select {
	case <-conclusoB:
		// B terminó sin esperar a A: correcto.
	case <-time.After(2 * time.Second):
		t.Fatal("el motor B no debía esperar al motor A: las colas son por motor")
	}

	close(liberarA)
	wg.Wait()
}

// TestElMismoMotorSerializaPorOrdenDeLlegada — dentro de un motor, las
// sesiones se atienden por orden de llegada: una genera mientras la otra
// espera, nunca las dos a la vez.
func TestElMismoMotorSerializaPorOrdenDeLlegada(t *testing.T) {
	colas := NewColasInferencia()
	cola := colas.Para("motor-a")

	liberarPrimera := make(chan struct{})
	haEntradoPrimera := make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = cola.Encolar(context.Background(), OpcionesEncolar{IdSesion: "A"}, func(context.Context) error {
			close(haEntradoPrimera)
			<-liberarPrimera
			return nil
		})
	}()
	<-haEntradoPrimera

	// La segunda petición del MISMO motor no puede entrar mientras la
	// primera genera.
	segundoEntrado := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = cola.Encolar(context.Background(), OpcionesEncolar{IdSesion: "B"}, func(context.Context) error {
			close(segundoEntrado)
			return nil
		})
	}()

	select {
	case <-segundoEntrado:
		t.Fatal("dos sesiones del mismo motor no pueden generar a la vez")
	case <-time.After(100 * time.Millisecond):
		// Esperando: correcto.
	}

	close(liberarPrimera)
	select {
	case <-segundoEntrado:
		// La segunda pasa al soltarse la primera.
	case <-time.After(2 * time.Second):
		t.Fatal("la segunda sesión no llegó a generar tras liberarse la primera")
	}
	wg.Wait()

	if cola.Esperando() || cola.Ocupada() {
		t.Error("al terminar, la cola del motor queda libre")
	}
}

// TestLasColasCompartenElNotificador — el notificador global de espera se
// suscribe una vez y alcanza a todas las colas, incluidas las creadas después.
func TestLasColasCompartenElNotificador(t *testing.T) {
	colas := NewColasInferencia()
	notificaciones := make(chan bool, 8)
	colas.SetNotificadorGlobal(func(esperando bool) { notificaciones <- esperando })

	// Cola creada DESPUÉS de suscribir: también debe notificar.
	liberar := make(chan struct{})
	haEntrado := make(chan struct{})
	go func() {
		_ = colas.Para("tardio").Encolar(context.Background(), OpcionesEncolar{}, func(context.Context) error {
			close(haEntrado)
			<-liberar
			return nil
		})
	}()
	<-haEntrado

	otro := make(chan struct{})
	go func() {
		defer close(otro)
		_ = colas.Para("tardio").Encolar(context.Background(), OpcionesEncolar{}, func(context.Context) error { return nil })
	}()

	select {
	case esp := <-notificaciones:
		if !esp {
			t.Error("la segunda petición entra en espera y debe notificarlo")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("la cola creada después de suscribir no notificó la espera")
	}

	close(liberar)
	<-otro
}
