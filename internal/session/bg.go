// bg.go — T-B013-04: cada sesión activa corre en su propia goroutine.
//
// Fuente de verdad: SPEC-SESIONES §Reglas ("Una sesión en segundo plano sigue
// ejecutándose aunque el usuario cambie de sesión o salga de la vista", "Cambiar
// de sesión no detiene nada"), BUSINESS_RULES.md §Sesiones (ídem) y EVENTS.md §4
// ("una sesión en segundo plano sigue trabajando aunque la pantalla no esté al
// día").
//
// El diseño es el más simple que cumple eso: una goroutine por sesión con
// trabajo en curso, guardada en un mapa por sesión, y ninguna goroutine de
// vigilancia global. Cambiar de sesión en la TUI no toca nada de aquí, porque la
// TUI no participa en la ejecución: solo se suscribe a los eventos. Por eso
// `Gestor` no tiene ni una referencia a la vista.
package session

import (
	"context"
	"fmt"
	"sync"

	"localcli/internal/store"
)

// Gestor es el ciclo de vida de las sesiones de un proyecto: crear, listar,
// retomar, cerrar, y la ejecución en segundo plano de cada una.
type Gestor struct {
	Alcance *Alcance
	Motor   Motor
	Bus     *Bus
	// Motores dice si el motor de una sesión sigue disponible. Opcional: lo
	// inyecta el arranque sobre el registro de `llm`; sin él no se comprueba.
	Motores Motores
	// Titulador produce el título de una sesión a partir de su primera petición.
	// Opcional: sin él, la sesión conserva el nombre provisional.
	Titulador Titulador
	// Resumidor compacta la conversación cuando no cabe en el presupuesto.
	// Opcional: sin él, se entrega solo el tramo reciente.
	Resumidor Resumidor
	// Presupuesto es el tope de tokens del historial que se le envía al modelo.
	// 0 = LOCALCLI_CONTEXT_LIMIT o el valor por defecto.
	Presupuesto int
	// PresupuestoRecencia es cuántos tokens de ese tope se reservan al tramo
	// literal más reciente. 0 = la mitad del presupuesto.
	PresupuestoRecencia int

	mu         sync.Mutex
	enCurso    map[string]*trabajo
	notificado map[string]string
	// resumenes guarda, por sesión, el resumen compactado y hasta qué mensaje
	// abarca. Vive en memoria: se recalcula de forma perezosa al reiniciar.
	resumenes map[string]resumenSesion
}

// trabajo es la ejecución viva de una sesión.
type trabajo struct {
	cancel context.CancelFunc
	hecho  chan struct{}
	// pausada se lee desde la cola envuelta y se escribe desde Pausar/Reanudar.
	pausada bool
	cola    *colaPausable
}

// NuevoGestor arma el gestor de un alcance. Sin motor no hay nada que ejecutar,
// así que se falla al construir en vez de al primer mensaje.
func NuevoGestor(alcance *Alcance, motor Motor, bus *Bus) (*Gestor, error) {
	if alcance == nil {
		return nil, fmt.Errorf("session: el gestor necesita un alcance (carpeta y almacén)")
	}
	if err := alcance.Validar(); err != nil {
		return nil, err
	}
	if motor == nil {
		return nil, fmt.Errorf("session: el gestor necesita el motor de flujos")
	}
	if bus == nil {
		bus = NuevoBus()
	}
	return &Gestor{
		Alcance:    alcance,
		Motor:      motor,
		Bus:        bus,
		enCurso:    map[string]*trabajo{},
		notificado: map[string]string{},
		resumenes:  map[string]resumenSesion{},
	}, nil
}

// Crear crea una sesión nueva y la devuelve inactiva. Nace sin par
// motor/modelo propio: lo fija el llamante con el último usado (SPEC-SESIONES).
func (g *Gestor) Crear(nombre, capa string) (*Sesion, error) {
	return g.CrearConMotor(nombre, capa, "", "")
}

// CrearConMotor crea una sesión nueva con su par motor/modelo ya asignado: una
// sesión nueva parte del último motor y modelo usados, que son una preferencia
// global del usuario (SPEC-SESIONES §Reglas).
func (g *Gestor) CrearConMotor(nombre, capa, motorID, modelo string) (*Sesion, error) {
	if nombre == "" {
		return nil, fmt.Errorf("session: la sesión necesita nombre")
	}
	s, err := g.Alcance.Almacen.CrearConMotor(nombre, capa, motorID, modelo)
	if err != nil {
		return nil, err
	}
	return DeStore(s), nil
}

// ParMotorModelo devuelve el motor y el modelo que usa la sesión. "" en
// cualquiera de los dos significa «sin asignar»: al retomarla se le aplica el
// motor por defecto. Es lo que lee el arranque del turno para saber contra qué
// motor generar.
func (g *Gestor) ParMotorModelo(sesionID string) (motorID, modelo string, err error) {
	ses, err := g.sesion(sesionID)
	if err != nil {
		return "", "", err
	}
	return ses.MotorID, ses.Modelo, nil
}

// CambiarMotor fija el motor de la sesión. El modelo no se toca aquí: si el
// nuevo motor no declara el modelo en curso, quien decide es la interfaz
// (SPEC-INTERFAZ §Modales: «si el modelo en curso no existe en él, el modelo
// queda vacío y hay que elegir otro»), que después llama a CambiarModelo.
// Cambiar de motor no toca el historial ni el nombre: son la identidad de la
// sesión, y el par es un recurso suyo (SPEC-MODELO-MOTOR).
func (g *Gestor) CambiarMotor(sesionID, motorID string) error {
	return g.cambiarPar(sesionID, &motorID, nil)
}

// CambiarModelo fija el modelo de la sesión sin tocar su motor.
func (g *Gestor) CambiarModelo(sesionID, modelo string) error {
	return g.cambiarPar(sesionID, nil, &modelo)
}

// cambiarPar aplica el cambio de los campos que se pasen. Nunca elige motor ni
// modelo por su cuenta: solo guarda lo que se le dice.
func (g *Gestor) cambiarPar(sesionID string, motorID, modelo *string) error {
	actual, err := g.Alcance.Almacen.Obtener(sesionID)
	if err != nil {
		return err
	}
	nuevoMotor, nuevoModelo := actual.MotorID, actual.Modelo
	if motorID != nil {
		nuevoMotor = *motorID
	}
	if modelo != nil {
		nuevoModelo = *modelo
	}
	return g.Alcance.Almacen.CambiarMotorModelo(sesionID, nuevoMotor, nuevoModelo)
}

// Listar lista las sesiones del proyecto. Es el listado que se ve al reabrir la
// carpeta, con el historial disponible para retomar cualquiera.
func (g *Gestor) Listar() ([]Sesion, error) {
	filas, err := g.Alcance.Almacen.Listar("")
	if err != nil {
		return nil, err
	}
	out := make([]Sesion, 0, len(filas))
	for i := range filas {
		out = append(out, *DeStore(&filas[i]))
	}
	return out, nil
}

// Estado devuelve la sesión con su estado actual, que es siempre visible
// (SPEC-SESIONES: "El estado de cada sesión es visible en todo momento").
func (g *Gestor) Estado(sesionID string) (*Sesion, error) { return g.sesion(sesionID) }

// Suscribir da el canal de eventos del motor y la función para darse de baja.
// Es el punto único por donde `tui` mira al motor (EVENTS.md §4: "la TUI no
// pregunta nada: se le notifica"): el gestor solo delega en su bus y no añade
// estado propio. El consumidor es responsable de llamar a la baja al terminar,
// para que el canal cierre y nadie quede esperando para siempre.
func (g *Gestor) Suscribir() (<-chan Evento, func()) { return g.Bus.Suscribir() }

// Historial devuelve la conversación de una sesión, para retomarla o revisarla.
func (g *Gestor) Historial(sesionID string) ([]store.MensajeConRazonamiento, error) {
	if _, err := g.sesion(sesionID); err != nil {
		return nil, err
	}
	return g.Alcance.Almacen.Historial(sesionID)
}

// sesion lee la sesión del almacén y la convierte a la vista del módulo.
func (g *Gestor) sesion(id string) (*Sesion, error) {
	if id == "" {
		return nil, fmt.Errorf("session: falta el id de la sesión")
	}
	s, err := g.Alcance.Almacen.Obtener(id)
	if err != nil {
		return nil, err
	}
	return DeStore(s), nil
}

// Cerrar cierra una sesión. Si tiene trabajo en curso no se cierra en silencio:
// se pide decidir qué hacer con el flujo (SPEC-SESIONES §Flujos alternativos:
// "El usuario cierra la sesión mientras corre un flujo: se pregunta qué hacer
// con él"). Con `forzar` se cancela el flujo y se cierra.
func (g *Gestor) Cerrar(sesionID string, forzar bool) error {
	ses, err := g.sesion(sesionID)
	if err != nil {
		return err
	}
	if ses.EnCurso() && !forzar {
		return fmt.Errorf("session: la sesión %s está %s; cancela su flujo o ciérrala con forzar",
			sesionID, ses.Estado)
	}
	if forzar {
		g.Cancelar(sesionID)
	}
	return g.Alcance.Almacen.Borrar(sesionID)
}

// EnCurso dice si la sesión tiene un flujo corriendo ahora mismo.
func (g *Gestor) EnCurso(sesionID string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.enCurso[sesionID] != nil
}

// Esperar bloquea hasta que el trabajo en curso de la sesión termina. Lo usan
// las pruebas y el apagado ordenado; la TUI no espera: recibe eventos.
func (g *Gestor) Esperar(sesionID string) {
	g.mu.Lock()
	t := g.enCurso[sesionID]
	g.mu.Unlock()
	if t != nil {
		<-t.hecho
	}
}

// lanzar arranca `correr` en la goroutine de esa sesión. Si la sesión ya tenía
// trabajo, el anterior se cancela: una sesión no corre dos flujos a la vez.
func (g *Gestor) lanzar(ctx context.Context, sesionID string, t *trabajo, correr func(context.Context) error) {
	cctx, cancel := context.WithCancel(ctx)
	t.cancel = cancel
	t.hecho = make(chan struct{})

	g.mu.Lock()
	if previo := g.enCurso[sesionID]; previo != nil && previo.cancel != nil {
		previo.cancel()
	}
	g.enCurso[sesionID] = t
	g.mu.Unlock()

	go func() {
		defer close(t.hecho)
		defer func() {
			g.mu.Lock()
			if g.enCurso[sesionID] == t {
				delete(g.enCurso, sesionID)
			}
			g.mu.Unlock()
		}()
		_ = correr(cctx)
	}()
}
