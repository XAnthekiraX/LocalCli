// contexto.go — la memoria de conversación: reconstruir el historial para el
// modelo y compactarlo cuando no cabe.
//
// Fuente de verdad: [[specs/SPEC-HISTORIAL-CONVERSACION]] (el chat le entrega al
// modelo la conversación anterior, no solo el mensaje nuevo; cuando el historial
// supera el presupuesto de contexto, lo antiguo se resume y lo reciente viaja
// entero) y [[specs/SPEC-SESIONES]] ("el contexto acumulado de una sesión no se
// comparte con las demás").
//
// El historial persistido (`messages`, tabla de `store`) es la fuente de verdad
// de la conversación; esto solo decide qué parte se le envía al modelo y con
// qué forma. El resumen vive en memoria por sesión: se recalcula de forma
// perezosa al reiniciar y se reutiliza mientras el tramo reciente siga cabiendo.
// Un fallo del resumidor no bloquea el turno: se envía el tramo reciente sin
// resumen (aviso no bloqueante).
package session

import (
	stdctx "context"
	"strings"

	lcontext "localcli/internal/context"
	"localcli/internal/flow"
	"localcli/internal/store"
)

// Presupuesto por defecto del historial, en tokens, cuando no lo fija la
// preferencia del usuario ni LOCALCLI_CONTEXT_LIMIT.
const presupuestoHistorialDefecto = 4096

// cabeceraResumen precede al resumen inyectado, para que el modelo sepa que no
// es un turno literal del usuario.
const cabeceraResumen = "Resumen de la conversación anterior:\n"

// LongitudMaximaResumen acota el resumen que se reinyecta.
const LongitudMaximaResumen = 4000

// Resumidor condensa una transcripción en un resumen breve. Lo implementa el
// arranque con el modelo local; puede fallar y el llamador sigue sin él.
type Resumidor interface {
	Resumir(ctx stdctx.Context, transcripcion string) (string, error)
}

// resumenSesion es el resumen compactado de una sesión y el último mensaje que
// abarca. `hastaID` permite reutilizarlo mientras no queden antiguos nuevos.
type resumenSesion struct {
	texto   string
	hastaID string
}

// historialPara arma los mensajes que se le entregan al modelo para el turno en
// curso: todo el historial si cabe en el presupuesto; si no, un resumen de lo
// antiguo seguido del tramo reciente completo. Se llama ANTES de escribir el
// mensaje actual, de modo que el turno en curso no aparece duplicado.
func (g *Gestor) historialPara(ctx stdctx.Context, sesionID string) ([]flow.Mensaje, error) {
	filas, err := g.Alcance.Almacen.Historial(sesionID)
	if err != nil {
		return nil, err
	}
	if len(filas) == 0 {
		return nil, nil
	}
	msgs := mensajesDeFlow(filas)
	presupuesto, recencia := g.presupuestos()
	if tokensDe(msgs) <= presupuesto {
		return msgs, nil
	}

	corte, resumen := g.compactar(ctx, sesionID, filas, msgs, presupuesto, recencia)
	if corte <= 0 {
		return msgs, nil
	}
	recientes := msgs[corte:]
	if resumen == "" {
		return recientes, nil
	}
	// El resumen va como mensaje de sistema, antes del tramo literal.
	out := make([]flow.Mensaje, 0, len(recientes)+1)
	out = append(out, flow.Mensaje{Rol: flow.RolSistema, Texto: cabeceraResumen + resumen})
	out = append(out, recientes...)
	return out, nil
}

// compactar decide el corte y el resumen. Devuelve el índice del primer mensaje
// que viaja literal (>=1) y el resumen de lo anterior ("" si no hay). Reutiliza
// el resumen cacheado cuando el tramo reciente sigue cabiendo; si creció, lo
// extiende con los mensajes que quedaron fuera, sin volver a resumir lo ya
// resumido.
func (g *Gestor) compactar(ctx stdctx.Context, sesionID string, filas []store.MensajeConRazonamiento, msgs []flow.Mensaje, presupuesto, recencia int) (int, string) {
	cache, hay := g.resumenGuardado(sesionID)

	if hay {
		if k, ok := indiceDeID(filas, cache.hastaID); ok {
			recientes := msgs[k+1:]
			if tokensDe(recientes)+lcontext.EstimarTokens(cache.texto) <= presupuesto {
				return k + 1, cache.texto
			}
			// El tramo reciente desbordó: se resume el resumen previo junto con
			// los mensajes que dejan de ser recientes. Si no hay nada nuevo que
			// plegar (el propio resumen ya no cabe), se conserva tal cual: no
			// hay una compactación mejor que hacer.
			desde := k + 1
			hasta := cortePorRecencia(msgs, recencia)
			if hasta <= desde {
				return desde, cache.texto
			}
			transcripcion := strings.TrimSpace(cache.texto + "\n\n" + transcripcionDe(msgs[desde:hasta]))
			if r, ok := g.resumir(ctx, transcripcion); ok {
				g.guardarResumen(sesionID, resumenSesion{texto: r, hastaID: filas[hasta-1].ID})
				return hasta, r
			}
			return hasta, cache.texto
		}
	}

	corte := cortePorRecencia(msgs, recencia)
	if corte <= 0 {
		return 0, ""
	}
	if r, ok := g.resumir(ctx, transcripcionDe(msgs[:corte])); ok {
		g.guardarResumen(sesionID, resumenSesion{texto: r, hastaID: filas[corte-1].ID})
		return corte, r
	}
	// Sin resumidor (o falló): se entrega solo el tramo reciente, sin resumen.
	return corte, ""
}

// resumir llama al resumidor con el texto dado y limpia la respuesta. Un fallo
// o una respuesta vacía se tratan como "sin resumen".
func (g *Gestor) resumir(ctx stdctx.Context, transcripcion string) (string, bool) {
	if g.Resumidor == nil || strings.TrimSpace(transcripcion) == "" {
		return "", false
	}
	r, err := g.Resumidor.Resumir(ctx, transcripcion)
	if err != nil {
		return "", false
	}
	r = LimpiarResumen(r)
	if r == "" {
		return "", false
	}
	return r, true
}

// presupuestos devuelve el presupuesto total de tokens y el reservado al tramo
// reciente. La preferencia del usuario manda; si no, LOCALCLI_CONTEXT_LIMIT; si
// no, el valor por defecto.
func (g *Gestor) presupuestos() (int, int) {
	p := g.Presupuesto
	if p <= 0 {
		p = lcontext.LimiteDeEntorno()
	}
	if p <= 0 {
		p = presupuestoHistorialDefecto
	}
	r := g.PresupuestoRecencia
	if r <= 0 {
		r = p / 2
	}
	if r > p {
		r = p
	}
	return p, r
}

// resumenGuardado lee el resumen en memoria de una sesión.
func (g *Gestor) resumenGuardado(sesionID string) (resumenSesion, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.resumenes == nil {
		return resumenSesion{}, false
	}
	r, ok := g.resumenes[sesionID]
	return r, ok
}

// guardarResumen deja el resumen en memoria para reutilizarlo.
func (g *Gestor) guardarResumen(sesionID string, r resumenSesion) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.resumenes == nil {
		g.resumenes = map[string]resumenSesion{}
	}
	g.resumenes[sesionID] = r
}

// mensajesDeFlow traduce las filas de `store` a los mensajes del motor. El rol
// `agent` de la base es el `assistant` de la conversación.
func mensajesDeFlow(filas []store.MensajeConRazonamiento) []flow.Mensaje {
	out := make([]flow.Mensaje, 0, len(filas))
	for _, f := range filas {
		if strings.TrimSpace(f.Content) == "" {
			continue
		}
		rol := flow.RolUsuario
		if f.Role == "agent" {
			rol = flow.RolAsistente
		}
		out = append(out, flow.Mensaje{Rol: rol, Texto: f.Content})
	}
	return out
}

// transcripcionDe compone el texto que se le pasa al resumidor.
func transcripcionDe(msgs []flow.Mensaje) string {
	var b strings.Builder
	for _, m := range msgs {
		quien := "usuario"
		if m.Rol == flow.RolAsistente {
			quien = "asistente"
		}
		b.WriteString(quien)
		b.WriteString(": ")
		b.WriteString(m.Texto)
		b.WriteString("\n")
	}
	return b.String()
}

// tokensDe suma la estimación de tokens de una lista de mensajes.
func tokensDe(msgs []flow.Mensaje) int {
	total := 0
	for _, m := range msgs {
		total += lcontext.EstimarTokens(m.Texto)
	}
	return total
}

// cortePorRecencia devuelve el índice del primer mensaje del tramo reciente:
// recorre desde el final acumulando tokens hasta agotar el presupuesto de
// recencia, pero conserva siempre al menos el último mensaje. Nunca devuelve 0
// cuando hay mensajes, salvo que no quepa ni uno.
func cortePorRecencia(msgs []flow.Mensaje, recencia int) int {
	if len(msgs) == 0 {
		return 0
	}
	acumulado := 0
	i := len(msgs) - 1
	for ; i >= 0; i-- {
		t := lcontext.EstimarTokens(msgs[i].Texto)
		if i < len(msgs)-1 && acumulado+t > recencia {
			return i + 1
		}
		acumulado += t
	}
	return 0
}

// indiceDeID busca la posición de una fila por su id.
func indiceDeID(filas []store.MensajeConRazonamiento, id string) (int, bool) {
	if id == "" {
		return 0, false
	}
	for i := range filas {
		if filas[i].ID == id {
			return i, true
		}
	}
	return 0, false
}

// PromptResumen redacta la consulta que condensa una conversación. Es el único
// texto que sale hacia el modelo desde aquí.
func PromptResumen(transcripcion string) string {
	return "Resume la siguiente conversación en pocas líneas, conservando lo que se " +
		"decidió, lo que quedó pendiente y los datos concretos (nombres, rutas, " +
		"números). Es para que el asistente recuerde el contexto en turnos " +
		"posteriores. Responde solo con el resumen.\n\n---\n\n" + transcripcion
}

// LimpiarResumen normaliza la respuesta del resumidor: recorta y acota. Devuelve
// "" si no queda nada; el llamador lo trata como fallo.
func LimpiarResumen(s string) string {
	s = strings.TrimSpace(s)
	r := []rune(s)
	if len(r) > LongitudMaximaResumen {
		s = strings.TrimSpace(string(r[:LongitudMaximaResumen]))
	}
	return s
}
