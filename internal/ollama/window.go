// window.go — la ventana de contexto (num_ctx) que LocalCli pide a Ollama.
//
// Fuente de verdad: [[specs/SPEC-OLLAMA-PERFIL]] ("El tamaño de contexto se
// limita para que quepa junto con el modelo cargado") y [[backend/DECISIONS]].
//
// Ollama usa un num_ctx por defecto pequeño (≈4096 en 0.33). Un turno con
// herramientas —prompt del agente + esquema de las 14 + historial + salidas de
// herramienta— lo supera enseguida; entonces Ollama recorta la lista de
// mensajes y responde 500 `no user query found in messages`, y el turno muere.
// La ventana se declara por petición, derivada del modelo y acotada por un
// tope, para que el servidor no decida por su cuenta.
package ollama

// TopeVentanaPorDefecto es el techo de la ventana cuando el modelo declara más
// contexto del que conviene cargar de una vez (los modelos locales declaran
// cientos de miles de tokens, que no caben en el hardware objetivo).
const TopeVentanaPorDefecto = 16384

// VentanaDeModelo devuelve el num_ctx a pedir para un modelo: el menor entre
// su propia ventana declarada y el tope. Un modelo sin ventana conocida
// (ContextLength == 0) usa el tope tal cual.
func VentanaDeModelo(m Modelo, tope int) int {
	if tope <= 0 {
		tope = TopeVentanaPorDefecto
	}
	if m.ContextLength > 0 && m.ContextLength < tope {
		return m.ContextLength
	}
	return tope
}
