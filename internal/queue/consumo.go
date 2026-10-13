package queue

// consumo.go — T-B018-13: solo el orquestador consume la cola.
//
// Fuente de verdad: DECISIONS.md ("Solo orquestador lanza colas; no hay modo de
// tarea suelta en la primera versión… un segundo punto de entrada duplicaría el
// camino de ejecución y debilitaría la regla de un elemento por iteración") y
// BUSINESS_RULES.md §Detección de trabajo ordenado ("Quien detecta es `flow`, no
// `queue`. `queue` consume lo que hay").
//
// El permiso es una capacidad, no un booleano: `Permiso` tiene un campo no
// exportado, así que el único modo de obtener uno es pedirlo aquí. No se puede
// fabricar desde fuera del módulo, ni por accidente ni por descuido.

import "fmt"

type rol string

const rolOrquestador rol = "orquestador"

// Permiso autoriza a consumir una cola. Existe para que el único camino sea el
// orquestador; no es una política de permisos de usuario.
type Permiso struct{ rol rol }

// PermitirAlOrquestador devuelve el permiso que usa el orquestador de sesión.
func PermitirAlOrquestador() Permiso { return Permiso{rol: rolOrquestador} }

// Consumidor es la vista autorizada de la cola: la que implementa la interfaz
// que espera el orquestador (`Siguiente` y `Marcar`). Sin permiso no se llega a
// ella.
type Consumidor struct{ cola *Cola }

// Consumir devuelve la vista de consumo. Sin el permiso del orquestador la
// operación se deniega: la cola no se lanza por su cuenta.
func (c *Cola) Consumir(p Permiso) (*Consumidor, error) {
	if c == nil {
		return nil, fmt.Errorf("queue: no hay cola que consumir")
	}
	if p.rol != rolOrquestador {
		return nil, errSoloOrquestador("consumir la cola exige el permiso del orquestador")
	}
	return &Consumidor{cola: c}, nil
}

// Cola devuelve la cola sobre la que consume este consumidor.
func (c *Consumidor) Cola() *Cola { return c.cola }

// LanzarElemento existe solo para denegar: no hay tarea suelta en esta versión.
// Pausar y cancelar ya cubren el "quiero solo esto".
func (c *Cola) LanzarElemento(id string) error {
	return errSoloOrquestador("lanzar el elemento " + id + " por su cuenta no existe en esta versión")
}

// errSoloOrquestador protege la invariante "solo el orquestador lanza colas".
func errSoloOrquestador(detalle string) error {
	return fmt.Errorf("queue: solo el orquestador consume la cola; %s "+
		"(no hay modo de tarea suelta, ver DECISIONS.md)", detalle)
}
