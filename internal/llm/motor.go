// motor.go — T-B036-02 (renombrada en T-B037-01): la interfaz que cada
// adaptador implementa.
//
// Fuente de verdad: ai/docs/backend/BACKEND.md §3 y
// ai/docs/backend/02-interfaces/INTERFACES-GENERAL.md §5.
package llm

import (
	"context"
	"sync"
)

// Motor es la frontera con el runtime que sirve el modelo. Lo implementa
// cada adaptador (`ollama`, `llamacpp`) y lo consumen `agent` y `context` sin
// saber cuál es.
type Motor interface {
	// Nombre devuelve la clave del runtime: `ollama` o `llamacpp`. Es la
	// etiqueta que se muestra («Ollama», «llama.cpp»).
	Nombre() string
	// BaseURL es la dirección configurada del servidor.
	BaseURL() string
	// Chat abre una generación en streaming con la petición neutra.
	Chat(ctx context.Context, req Peticion) (<-chan Evento, error)
	// ListarModelos devuelve los modelos que declara el servidor.
	ListarModelos(ctx context.Context) ([]Modelo, error)
	// Capacidades devuelve, normalizadas a las tres del harness, lo que el
	// modelo declara saber hacer.
	Capacidades(ctx context.Context, modelo string) ([]string, error)
	// VentanaDeContexto devuelve la ventana que fija el servidor y si el
	// motor la declara por petición (true, Ollama) o la lee de él (false,
	// llama.cpp). Un motor que la declara devuelve 0: quien la calcula es
	// el arranque a partir de la lista de modelos.
	VentanaDeContexto(ctx context.Context, modelo string) (int, bool, error)
}

// ConstructorMotor construye el adaptador de un motor a partir de su URL.
type ConstructorMotor func(url string) Motor

// El registro de constructores POR TIPO CERRADO. Los adaptadores se registran
// aquí al cargarse (`init`), así que `llm` —y solo `llm`— instancia adaptadores
// por tipo, sin que el resto del harness importe `ollama` ni `llamacpp` ni
// pregunte por el tipo (BACKEND.md §3 y DECISIONS: «los dos adaptadores se
// registran por su tipo cerrado»).
var (
	constructoresMu sync.RWMutex
	constructores   = map[string]ConstructorMotor{}
)

// RegistrarTipo registra el constructor del adaptador de un tipo del catálogo.
// Es idempotente: volver a registrarlo sustituye el anterior (los tests
// registran dobles).
func RegistrarTipo(tipo string, ctor ConstructorMotor) {
	constructoresMu.Lock()
	defer constructoresMu.Unlock()
	constructores[tipo] = ctor
}

// AdaptadorDeTipo construye el adaptador de un tipo con esa URL. Un tipo sin
// constructor registrado se rechaza con E_MOTOR_TIPO_DESCONOCIDO: el catálogo
// es cerrado y nada más puede instanciar un adaptador.
func AdaptadorDeTipo(tipo, url string) (Motor, error) {
	constructoresMu.RLock()
	ctor, ok := constructores[tipo]
	constructoresMu.RUnlock()
	if !ok {
		return nil, &ErrorMotor{
			Codigo:   CodigoMotorTipoDesconocido,
			Mensaje:  "el tipo " + tipo + " no tiene adaptador registrado",
			Detalle:  "tipo fuera del catálogo cerrado",
			sentinel: ErrMotorTipoDesconocido,
		}
	}
	return ctor(url), nil
}
