// proveedor.go — T-B036-02: la interfaz que cada adaptador implementa.
//
// Fuente de verdad: ai/docs/backend/BACKEND.md §3 y
// ai/docs/backend/02-interfaces/INTERFACES-GENERAL.md §5.
package llm

import "context"

// Proveedor es la frontera con el runtime que sirve el modelo. Lo implementa
// cada adaptador (`ollama`, `openai`) y lo consumen `agent` y `context` sin
// saber cuál es.
type Proveedor interface {
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
	// proveedor la declara por petición (true, Ollama) o la lee de él (false,
	// llama.cpp). Un proveedor que la declara devuelve 0: quien la calcula es
	// el arranque a partir de la lista de modelos.
	VentanaDeContexto(ctx context.Context, modelo string) (int, bool, error)
}
