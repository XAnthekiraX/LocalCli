package session

import (
	"context"
	"os"
	"strconv"
	"strings"

	"localcli/internal/agent"
	"localcli/internal/llm/openai"
	"localcli/internal/store"
	"localcli/internal/tui"
)

// presupuestoHistorialDefecto es el presupuesto de tokens sin preferencia ni entorno.
const presupuestoHistorialDefecto = 4096

const cabeceraResumen = "Resumen de la conversación anterior:\n"

const longitudMaximaResumen = 4000

const promptResumen = "Resume la conversación siguiente en pocas líneas, conservando " +
	"los datos y las decisiones importantes y sin añadir nada que no aparezca en ella. " +
	"Devuelve solo el resumen."

// opcionesResumen acotan el resumen: con tope y sin razonamiento.
var opcionesResumen = openai.Opciones{MaxTokens: 512, SinRazonamiento: true}

// resumenSesion es el resumen del tramo antiguo y el id del último mensaje que abarca.
type resumenSesion struct {
	texto   string
	hastaID string
}

// mensajeConv es un mensaje cerrado con el id de su fila.
type mensajeConv struct {
	msg agent.Mensaje
	id  string
}

// historialPara arma el historial del modelo: entero si cabe, o resumen + recientes.
func (o *Orquestador) historialPara(ctx context.Context, sesionID string) ([]agent.Mensaje, error) {
	filas, err := store.ListarMensajes(o.db, o.projectID, sesionID)
	if err != nil {
		return nil, err
	}
	conv := conversacionDe(filas)
	if len(conv) == 0 {
		return nil, nil
	}
	presupuesto := presupuestoHistorial(o.ventanaActual())
	if tokensDe(conv) <= presupuesto {
		return mensajesDe(conv), nil
	}

	corte, resumen := o.compactar(ctx, sesionID, conv, presupuesto)
	if corte <= 0 {
		return mensajesDe(conv), nil
	}
	recientes := mensajesDe(conv[corte:])
	if resumen == "" {
		return recientes, nil
	}
	salida := make([]agent.Mensaje, 0, len(recientes)+1)
	salida = append(salida, agent.Mensaje{Rol: agent.RolSistema, Contenido: cabeceraResumen + resumen})
	return append(salida, recientes...), nil
}

// compactar decide el corte y el resumen, reutilizando el ya calculado si cubre el tramo.
func (o *Orquestador) compactar(ctx context.Context, sesionID string, conv []mensajeConv, presupuesto int) (int, string) {
	corte := cortePorRecencia(conv, presupuesto/2)
	if corte <= 0 {
		return 0, ""
	}
	if cache, ok := o.resumenGuardado(sesionID); ok && cache.hastaID == conv[corte-1].id {
		return corte, cache.texto
	}
	resumen, ok := o.resumir(ctx, transcripcionDe(conv[:corte]))
	if !ok {
		return corte, ""
	}
	o.guardarResumen(sesionID, resumenSesion{texto: resumen, hastaID: conv[corte-1].id})
	return corte, resumen
}

// cortePorRecencia devuelve el primer índice del tramo reciente que cabe en la recencia.
func cortePorRecencia(conv []mensajeConv, recencia int) int {
	if recencia < 0 {
		recencia = 0
	}
	acumulado := 0
	for i := len(conv) - 1; i >= 0; i-- {
		siguiente := acumulado + tokensDeMensaje(conv[i].msg)
		if siguiente > recencia && i < len(conv)-1 {
			return i + 1
		}
		acumulado = siguiente
	}
	return 0
}

// resumir pide al modelo el resumen; un fallo o una respuesta vacía son «sin resumen».
func (o *Orquestador) resumir(ctx context.Context, transcripcion string) (string, bool) {
	if o.llm == nil || strings.TrimSpace(transcripcion) == "" {
		return "", false
	}
	mensajes := []openai.Mensaje{
		{Rol: "system", Contenido: promptResumen},
		{Rol: "user", Contenido: transcripcion},
	}
	var b strings.Builder
	if _, err := o.llm.Completar(ctx, o.modeloActual(), mensajes, nil, opcionesResumen, func(d openai.Delta) error {
		if !d.Razonamiento {
			b.WriteString(d.Texto)
		}
		return nil
	}); err != nil {
		return "", false
	}
	r := LimpiarResumen(b.String())
	if r == "" {
		return "", false
	}
	return r, true
}

// presupuestoHistorial resuelve el presupuesto del historial: la preferencia del
// usuario, si no el override de entorno, si no una fracción de la ventana —el
// resto queda para el prompt, las herramientas y el recorte por ronda— y, sin
// ventana conocida, el defecto.
func presupuestoHistorial(ventana int) int {
	if p, err := tui.CargarPreferencias(); err == nil && p.HistorialTokens > 0 {
		return p.HistorialTokens
	}
	if v := limiteDeEntorno(); v > 0 {
		return v
	}
	if ventana > 0 {
		return ventana / 2
	}
	return presupuestoHistorialDefecto
}

// limiteDeEntorno lee LOCALCLI_CONTEXT_LIMIT; 0 si falta o no es válido.
func limiteDeEntorno() int {
	v := strings.TrimSpace(os.Getenv("LOCALCLI_CONTEXT_LIMIT"))
	if v == "" {
		return 0
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

// conversacionDe traduce las filas a los turnos cerrados de usuario y agente.
func conversacionDe(filas []store.Mensaje) []mensajeConv {
	conv := make([]mensajeConv, 0, len(filas))
	for _, f := range filas {
		if strings.TrimSpace(f.Content) == "" || (f.Role != "user" && f.Role != "agent") {
			continue
		}
		rol := agent.RolUsuario
		if f.Role == "agent" {
			rol = agent.RolAsistente
		}
		conv = append(conv, mensajeConv{msg: agent.Mensaje{Rol: rol, Contenido: f.Content}, id: f.ID})
	}
	return conv
}

// mensajesDe proyecta la conversación a los mensajes que viajan al modelo.
func mensajesDe(conv []mensajeConv) []agent.Mensaje {
	salida := make([]agent.Mensaje, len(conv))
	for i, c := range conv {
		salida[i] = c.msg
	}
	return salida
}

// transcripcionDe compone el texto que se resume.
func transcripcionDe(conv []mensajeConv) string {
	var b strings.Builder
	for _, c := range conv {
		b.WriteString(etiquetaRol(c.msg.Rol))
		b.WriteString(": ")
		b.WriteString(c.msg.Contenido)
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

func etiquetaRol(rol string) string {
	if rol == agent.RolAsistente {
		return "Agente"
	}
	return "Usuario"
}

// tokensDe estima los tokens de una conversación.
func tokensDe(conv []mensajeConv) int {
	total := 0
	for _, c := range conv {
		total += tokensDeMensaje(c.msg)
	}
	return total
}

// tokensDeMensaje estima los tokens de un mensaje con su coste fijo.
func tokensDeMensaje(m agent.Mensaje) int { return estimarTokens(m.Contenido) + 4 }

// estimarTokens estima tokens por longitud (≈4 caracteres por token).
func estimarTokens(texto string) int {
	if texto == "" {
		return 0
	}
	n := (len([]rune(texto)) + 3) / 4
	if n < 1 {
		n = 1
	}
	return n
}

// LimpiarResumen recorta espacios y acota el resumen reinyectado.
func LimpiarResumen(s string) string {
	s = strings.TrimSpace(s)
	if r := []rune(s); len(r) > longitudMaximaResumen {
		s = string(r[:longitudMaximaResumen])
	}
	return s
}

// resumenGuardado lee el resumen en memoria de una sesión.
func (o *Orquestador) resumenGuardado(sesionID string) (resumenSesion, bool) {
	o.muResumen.Lock()
	defer o.muResumen.Unlock()
	if o.resumenes == nil {
		return resumenSesion{}, false
	}
	r, ok := o.resumenes[sesionID]
	return r, ok
}

// guardarResumen deja el resumen en memoria para reutilizarlo.
func (o *Orquestador) guardarResumen(sesionID string, r resumenSesion) {
	o.muResumen.Lock()
	defer o.muResumen.Unlock()
	if o.resumenes == nil {
		o.resumenes = map[string]resumenSesion{}
	}
	o.resumenes[sesionID] = r
}
