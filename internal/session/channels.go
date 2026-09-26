// channels.go — T-B013-08: los canales hacia `tui`.
//
// Fuente de verdad: EVENTS.md §1 (la tabla de eventos y quién los emite), §2
// ("`estado_sesion` … consumidor principal `tui`"; `notificacion` "llega aunque
// la sesión no sea la que se está viendo") y §4 ("los eventos son
// notificaciones, no comandos", "un consumidor lento no bloquea al productor",
// "la TUI no pregunta nada: se le notifica").
//
// Por eso el bus es de una sola dirección y sin respuesta: `Emitir` no vuelve
// nada y nunca bloquea. Cada suscriptor tiene su propio canal con cola; si se
// llena, el evento se descarta para ese suscriptor y la producción sigue. Es la
// traducción literal de "un consumidor lento no bloquea al productor": perder un
// token de una pantalla que no da abasto es mejor que parar el trabajo.
//
// El bus implementa `flow.Emisor`, así que el motor de flujos le entrega sus
// eventos (`etapa_iniciada`…) directamente, sin adaptador.
package session

import (
	"sync"

	"localcli/internal/flow"
)

// Evento es el tipo de evento del motor. Se usa el de `flow` para que no haya
// dos formas de evento circulando por el mismo canal.
type Evento = flow.Evento

// Nombres de los eventos que emite `session` (EVENTS.md §1).
const (
	EventoEstadoSesion = "estado_sesion"
	EventoNotificacion = "notificacion"
)

// BufferSuscriptor es la cola de cada suscriptor. Un suscriptor que no lee a
// tiempo pierde eventos, nunca bloquea a quien los emite.
const BufferSuscriptor = 64

// Bus reparte eventos a sus suscriptores. Cumple `flow.Emisor`.
type Bus struct {
	mu      sync.Mutex
	subs    map[int]chan Evento
	siguien int
	cerrado bool
}

// NuevoBus crea un bus sin suscriptores.
func NuevoBus() *Bus { return &Bus{subs: map[int]chan Evento{}} }

// Emitir reparte el evento a todos los suscriptores sin bloquearse.
func (b *Bus) Emitir(e Evento) {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.cerrado {
		return
	}
	for _, ch := range b.subs {
		select {
		case ch <- e:
		default:
			// Suscriptor lento: se descarta para él y se sigue.
		}
	}
}

// Suscribir devuelve un canal de eventos y la función para darse de baja. La
// baja cierra el canal, así que el consumidor no se queda esperando para
// siempre.
func (b *Bus) Suscribir() (<-chan Evento, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()
	id := b.siguien
	b.siguien++
	ch := make(chan Evento, BufferSuscriptor)
	if b.cerrado {
		close(ch)
		return ch, func() {}
	}
	b.subs[id] = ch
	var once sync.Once
	return ch, func() {
		once.Do(func() {
			b.mu.Lock()
			defer b.mu.Unlock()
			if c, ok := b.subs[id]; ok {
				delete(b.subs, id)
				close(c)
			}
		})
	}
}

// Cerrar da de baja a todos los suscriptores. Lo llama el arranque al salir.
func (b *Bus) Cerrar() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.cerrado {
		return
	}
	b.cerrado = true
	for id, ch := range b.subs {
		delete(b.subs, id)
		close(ch)
	}
}

// Compila: el bus es el emisor que espera el motor de flujos.
var _ flow.Emisor = (*Bus)(nil)
