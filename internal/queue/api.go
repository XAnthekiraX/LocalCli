// api.go — T-B012-07: solo el motor consume la cola.
//
// Fuente de verdad: DECISIONS.md ("Solo el motor lanza colas; no hay modo de
// tarea suelta en la primera versión… un segundo punto de entrada duplicaría el
// camino de ejecución y debilitaría la regla de un elemento por iteración") y
// BUSINESS_RULES.md §Detección de trabajo ordenado ("Quien detecta es `flow`, no
// `queue`. `queue` consume lo que hay").
//
// El permiso es una capacidad, no un booleano: `Permiso` tiene un campo no
// exportado, así que el único modo de obtener uno es pedirlo aquí. No se puede
// fabricar desde fuera del módulo, ni por accidente ni por descuido.
package queue

import "fmt"

type rol string

const rolMotor rol = "motor"

// Permiso autoriza a consumir una cola. Existe para que el único camino sea el
// motor; no es una política de permisos de usuario.
type Permiso struct{ rol rol }

// PermitirAlMotor devuelve el permiso que usa `flow.Motor.ConsumirCola`.
func PermitirAlMotor() Permiso { return Permiso{rol: rolMotor} }

// Consumidor es la vista autorizada de la cola: la que implementa la interfaz
// que espera el motor (`Siguiente` y `Marcar`). Sin permiso no se llega a ella.
type Consumidor struct{ cola *Cola }

// Consumir devuelve la vista de consumo. Sin el permiso del motor la operación
// se deniega: la cola no se lanza por su cuenta.
func (c *Cola) Consumir(p Permiso) (*Consumidor, error) {
	if c == nil {
		return nil, fmt.Errorf("queue: no hay cola que consumir")
	}
	if p.rol != rolMotor {
		return nil, errSoloMotor("consumir la cola exige el permiso del motor")
	}
	return &Consumidor{cola: c}, nil
}

// Cola devuelve la cola sobre la que consume este consumidor.
func (c *Consumidor) Cola() *Cola { return c.cola }

// LanzarElemento existe solo para denegar: no hay tarea suelta en la primera
// versión. Pausar y cancelar ya cubren el "quiero solo esto".
func (c *Cola) LanzarElemento(id string) error {
	return errSoloMotor("lanzar el elemento " + id + " por su cuenta no existe en esta versión")
}
