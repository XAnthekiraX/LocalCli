// ventana.go — T-B036-02: la regla de la ventana de contexto y las capacidades
// normalizadas.
//
// Fuente de verdad: ai/docs/specs/SPEC-MODELO-PROVEEDOR §La ventana de contexto
// y ai/docs/backend/DECISIONS.md («la ventana se declara por petición en Ollama
// y se lee del servidor en llama.cpp; la efectiva es la menor entre lo que
// declara el modelo, lo que fija el servidor y el tope»).
//
// Ninguna lógica del harness lee el formato del proveedor: las capacidades se
// normalizan a las tres que el harness entiende —herramientas, visión y
// razonamiento— venga de donde vengan.
package llm

// TopeVentanaPorDefecto es el techo de la ventana cuando el modelo declara más
// contexto del que conviene cargar de una vez (los modelos locales declaran
// cientos de miles de tokens, que no caben en el hardware objetivo).
const TopeVentanaPorDefecto = 16384

// VentanaDeModelo devuelve el num_ctx a pedir para un modelo: el menor entre su
// propia ventana declarada y el tope. Un modelo sin ventana conocida
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

// VentanaEfectiva combina las tres fuentes de la ventana: lo que declara el
// modelo (`delModelo`, 0 = no lo declara), lo que fija el servidor (`delServidor`,
// 0 = no aplica porque el proveedor la declara por petición) y el tope. Devuelve
// la menor de las que existan; si ninguna existe, el tope.
func VentanaEfectiva(delModelo, delServidor, tope int) int {
	if tope <= 0 {
		tope = TopeVentanaPorDefecto
	}
	efectiva := tope
	if delModelo > 0 && delModelo < efectiva {
		efectiva = delModelo
	}
	if delServidor > 0 && delServidor < efectiva {
		efectiva = delServidor
	}
	return efectiva
}

// Nombres de las capacidades normalizadas que el harness entiende. Los
// adaptadores traducen las de su runtime a estos tres nombres.
const (
	CapacidadHerramientas = "tools"
	CapacidadVision       = "vision"
	CapacidadPensar       = "thinking"
)

// PuedeUsarHerramientas dice si el modelo declara la capacidad "tools". Un
// modelo sin ficha clara (lista vacía) no la declara: quien decide avisar es el
// llamante, que ante un error prefiere no alarmar.
func PuedeUsarHerramientas(capacidades []string) bool {
	return tieneCapacidad(capacidades, CapacidadHerramientas)
}

// PuedeVer dice si el modelo declara la capacidad "vision". Igual que con las
// herramientas, una ficha vacía no la declara.
func PuedeVer(capacidades []string) bool {
	return tieneCapacidad(capacidades, CapacidadVision)
}

// PuedePensar dice si el modelo razona antes de responder. Un modelo que no la
// declara no admite que se le pida razonar.
func PuedePensar(capacidades []string) bool {
	return tieneCapacidad(capacidades, CapacidadPensar)
}

func tieneCapacidad(capacidades []string, buscada string) bool {
	for _, c := range capacidades {
		if c == buscada {
			return true
		}
	}
	return false
}
