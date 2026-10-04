// ventana.go — T-B036-02: la regla de la ventana de contexto y las capacidades
// normalizadas. La caché de ventana y ficha pasa a ser POR MOTOR desde
// T-B037-06.
//
// Fuente de verdad: ai/docs/specs/SPEC-MODELO-MOTOR §La ventana de contexto y
// §Lo que cambia por motor («dos motores del mismo tipo no comparten ficha de
// capacidades ni ventana, aunque sirvan un modelo del mismo nombre») y
// ai/docs/backend/DECISIONS.md («la ventana se declara por petición en Ollama
// y se lee del servidor en llama.cpp; la efectiva es la menor entre lo que
// declara el modelo, lo que fija el servidor y el tope»).
//
// Ninguna lógica del harness lee el formato del motor: las capacidades se
// normalizan a las tres que el harness entiende —herramientas, visión y
// razonamiento— venga de donde vengan.
package llm

import "context"

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

// VentanaDeContexto devuelve la ventana EFECTIVA de `modelo` en el motor
// `idMotor`: la menor entre la que declara el modelo, la que fija el servidor
// y el tope (LOCALCLI_CONTEXT_LIMIT, que aporta el llamante; 0 = el tope por
// defecto). Todo lo que ya se sabe viene de la caché POR MOTOR del registro;
// lo que falta se le pregunta al motor y se cachea, para no repetir la
// consulta en cada turno.
//
// Un fallo al preguntar no fija el dato: sin la ventana del modelo se trata
// como «no la declara» (el tope manda) y sin la del servidor se vuelve a
// preguntar en el próximo turno. Un motor que no responde no tumba el turno:
// el llamante avisa aparte si hace falta.
func (r *Registro) VentanaDeContexto(ctx context.Context, m Motor, idMotor, modelo string, tope int) int {
	v, _ := r.VentanaDeContextoYOrigen(ctx, m, idMotor, modelo, tope)
	return v
}

// VentanaDeContextoYOrigen es como VentanaDeContexto y, además, dice si la
// ventana efectiva la fijó el HARNESS: verdadero cuando ni el modelo ni el
// servidor declararon una ventana menor, así que manda el tope. La vista usa ese
// segundo valor para marcar «límite del harness» en vez de hacerlo pasar por la
// ventana del modelo (SPEC-MODELO-MOTOR §La ventana de contexto).
func (r *Registro) VentanaDeContextoYOrigen(ctx context.Context, m Motor, idMotor, modelo string, tope int) (int, bool) {
	delModelo, cacheado := r.LargoDeContexto(idMotor, modelo)
	if !cacheado {
		if modelos, err := m.ListarModelos(ctx); err == nil {
			// El listado trae la ventana de todos los modelos: se cachea entero.
			r.CachearModelos(idMotor, modelos)
			delModelo, _ = r.LargoDeContexto(idMotor, modelo)
		}
	}

	delServidor, leida := r.VentanaDelServidor(idMotor)
	if !leida {
		if v, declarada, err := m.VentanaDeContexto(ctx, modelo); err == nil {
			if declarada {
				// El motor la declara por petición: no la fija él (Ollama).
				v = 0
			}
			r.CachearVentanaDelServidor(idMotor, v)
			delServidor = v
		}
	}

	// Ni el modelo ni el servidor declararon ventana: el tope manda y el límite
	// es del harness, no del modelo.
	delHarness := delModelo <= 0 && delServidor <= 0
	return VentanaEfectiva(delModelo, delServidor, tope), delHarness
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
