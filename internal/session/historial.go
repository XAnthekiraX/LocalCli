package session

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"localcli/internal/agent"
	"localcli/internal/compact"
	"localcli/internal/llm/openai"
	"localcli/internal/store"
	"localcli/internal/tui"
)

// presupuestoHistorialDefecto es el presupuesto de tokens sin preferencia ni entorno.
const presupuestoHistorialDefecto = 4096

const cabeceraResumen = "Resumen de la conversación anterior:\n"

const longitudMaximaResumen = 8000

// opcionesResumen acotan el resumen: con tope y sin razonamiento. El tope da
// margen al formato estructurado, que ocupa más que un resumen libre.
var opcionesResumen = openai.Opciones{MaxTokens: 1024, SinRazonamiento: true}

// mensajeConv es un mensaje cerrado con el id de su fila.
type mensajeConv struct {
	msg agent.Mensaje
	id  string
}

// historialPara arma el historial del modelo: entero si cabe, o resumen + recientes.
// Un resumen ya persistido —automático o de `/compactar`— manda: se entrega él y
// lo posterior literal, y solo se amplía si lo reciente no cabe en el presupuesto.
func (o *Orquestador) historialPara(ctx context.Context, sesionID string) ([]agent.Mensaje, error) {
	filas, err := store.ListarMensajes(o.db, o.projectID, sesionID)
	if err != nil {
		return nil, err
	}
	conv := conversacionDe(filas)
	if len(conv) == 0 {
		return nil, nil
	}
	prefs := preferenciasSeguras()
	presupuesto := presupuestoHistorial(o.ventanaActual(), prefs)

	sum, hay := o.resumenPersistido(sesionID)
	cubierto := indiceCubierto(conv, sum.HastaID)
	if hay && cubierto == 0 {
		// El historial ya no contiene el tramo resumido (p. ej. se deshizo): el
		// resumen quedó obsoleto.
		_ = store.LimpiarResumenHistorial(o.db, o.projectID, sesionID)
		hay = false
	}
	if hay && tokensDe(conv[cubierto:]) <= presupuesto {
		return armarHistorial(conv, cubierto, sum.Texto), nil
	}
	if !hay && tokensDe(conv) <= presupuesto {
		return mensajesDe(conv), nil
	}

	corte, resumen := o.compactar(ctx, sesionID, conv, recenciaDeCompactacion(prefs, presupuesto), prefs)
	if corte <= 0 {
		if hay {
			return armarHistorial(conv, cubierto, sum.Texto), nil
		}
		return mensajesDe(conv), nil
	}
	return armarHistorial(conv, corte, resumen), nil
}

// armarHistorial compone la entrega al modelo: el resumen como mensaje de sistema
// seguido del tramo reciente literal. Sin resumen, solo el tramo reciente.
func armarHistorial(conv []mensajeConv, corte int, resumen string) []agent.Mensaje {
	recientes := mensajesDe(conv[corte:])
	if strings.TrimSpace(resumen) == "" {
		return recientes
	}
	salida := make([]agent.Mensaje, 0, len(recientes)+1)
	salida = append(salida, agent.Mensaje{Rol: agent.RolSistema, Contenido: cabeceraResumen + resumen})
	return append(salida, recientes...)
}

// compactar decide el corte y el resumen. Reutiliza el resumen persistido si ya
// cubre el tramo; lo amplía de forma incremental si quedó corto; y lo pide entero
// si no hay ninguno. Devuelve el corte (0 = no se compactó) y el resumen a usar.
func (o *Orquestador) compactar(ctx context.Context, sesionID string, conv []mensajeConv, recencia int, prefs tui.Preferencias) (int, string) {
	corte := cortePorRecencia(conv, recencia)
	if corte <= 0 {
		return 0, ""
	}

	sum, hay := o.resumenPersistido(sesionID)
	cubierto := indiceCubierto(conv, sum.HastaID)
	if hay && cubierto >= corte {
		// El resumen ya abarca el tramo reciente: se reutiliza sin llamar al modelo.
		return cubierto, sum.Texto
	}
	if hay && cubierto > 0 {
		resumen, ok := o.resumir(ctx, prefs, sum.Texto, transcripcionDe(conv[cubierto:corte]))
		if !ok {
			// No se pudo ampliar: se conserva el resumen previo y, literales, los
			// mensajes que aún no abarcaba —nada se pierde.
			return cubierto, sum.Texto
		}
		o.persistirResumen(sesionID, resumen, conv[corte-1].id)
		return corte, resumen
	}
	resumen, ok := o.resumir(ctx, prefs, "", transcripcionDe(conv[:corte]))
	if !ok {
		return corte, ""
	}
	o.persistirResumen(sesionID, resumen, conv[corte-1].id)
	return corte, resumen
}

// Compactar fuerza la compactación del historial de una sesión: resume todo lo
// anterior al tramo reciente y lo deja persistido para los próximos turnos. Es lo
// que dispara `/compactar` ([[specs/SPEC-HISTORIAL-CONVERSACION]]). No se salta
// porque el historial quepa en el presupuesto: se pide a mano para reducir el
// contexto ahora. Devuelve `true` si dejó un resumen persistido y `false` si no
// había tramo antiguo que resumir (todo cabe en el tramo reciente).
func (o *Orquestador) Compactar(ctx context.Context, sesionID string) (bool, error) {
	filas, err := store.ListarMensajes(o.db, o.projectID, sesionID)
	if err != nil {
		return false, err
	}
	conv := conversacionDe(filas)
	if len(conv) == 0 {
		return false, nil
	}
	prefs := preferenciasSeguras()
	recencia := recenciaDeCompactacion(prefs, presupuestoHistorial(o.ventanaActual(), prefs))
	corte, resumen := o.compactar(ctx, sesionID, conv, recencia, prefs)
	if corte <= 0 {
		return false, nil
	}
	if strings.TrimSpace(resumen) == "" {
		return false, fmt.Errorf("session: no se pudo resumir el historial")
	}
	return true, nil
}

// indiceCubierto devuelve cuántos mensajes ya abarca un resumen cuyo último
// mensaje es `hastaID`; 0 si no lo encuentra (historial deshecho, por ejemplo).
func indiceCubierto(conv []mensajeConv, hastaID string) int {
	if hastaID == "" {
		return 0
	}
	for i, c := range conv {
		if c.id == hastaID {
			return i + 1
		}
	}
	return 0
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

// resumir pide al modelo el resumen de un tramo; si hay un resumen anterior, pide
// su actualización incremental. Un fallo o una respuesta vacía son «sin resumen».
func (o *Orquestador) resumir(ctx context.Context, prefs tui.Preferencias, anterior, tramo string) (string, bool) {
	if o.llm == nil || strings.TrimSpace(tramo) == "" {
		return "", false
	}
	op := compact.Opciones{
		Formato:   prefs.Compactacion.FormatoEfectivo(),
		Plantilla: prefs.Compactacion.PlantillaDeCompactacion(),
	}
	prompt := op.PromptInicial()
	if strings.TrimSpace(anterior) != "" {
		prompt = op.PromptIncremental(anterior)
	}
	mensajes := []openai.Mensaje{
		{Rol: "system", Contenido: prompt},
		{Rol: "user", Contenido: tramo},
	}
	var b strings.Builder
	if _, err := o.llm.Completar(ctx, o.modeloDeCompactacion(prefs), mensajes, nil, opcionesResumen, func(d openai.Delta) error {
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

// modeloDeCompactacion elige el modelo que produce el resumen: el configurado
// para compactar o, sin él, el activo.
func (o *Orquestador) modeloDeCompactacion(prefs tui.Preferencias) string {
	if m := prefs.Compactacion.ModeloDeCompactacion(); m != "" {
		return m
	}
	return o.modeloActual()
}

// resumenPersistido lee el resumen guardado de una sesión.
func (o *Orquestador) resumenPersistido(sesionID string) (store.ResumenHistorial, bool) {
	sum, ok, err := store.ObtenerResumenHistorial(o.db, o.projectID, sesionID)
	if err != nil {
		return store.ResumenHistorial{}, false
	}
	return sum, ok
}

// persistirResumen guarda el resumen para reutilizarlo en los próximos turnos.
func (o *Orquestador) persistirResumen(sesionID, texto, hastaID string) {
	if err := store.GuardarResumenHistorial(o.db, o.projectID, sesionID, texto, hastaID); err != nil {
		o.avisar(sesionID, "no se pudo guardar el resumen de contexto: "+err.Error(), "")
	}
}

// preferenciasSeguras lee las preferencias del usuario; un fallo devuelve las
// vacías, que resuelven a los valores por defecto.
func preferenciasSeguras() tui.Preferencias {
	prefs, err := tui.CargarPreferencias()
	if err != nil {
		return tui.Preferencias{}
	}
	return prefs
}

// reservaResumenTokens es el margen que se deja al presupuesto para el resumen
// inyectado: el tramo reciente literal no lo ocupa todo, de modo que resumen y
// recientes juntos no superan el presupuesto ([[specs/SPEC-HISTORIAL-CONVERSACION]]).
const reservaResumenTokens = 2048

// recenciaDeCompactacion acota el tramo reciente literal al presupuesto: la
// preferencia de compactación, sin superar lo que cabe una vez reservado el hueco
// del resumen.
func recenciaDeCompactacion(prefs tui.Preferencias, presupuesto int) int {
	recencia := prefs.Compactacion.ConservarEfectivo()
	if presupuesto > 0 {
		tope := presupuesto - reservaResumenTokens
		if tope < 1 {
			tope = presupuesto
		}
		if recencia > tope {
			recencia = tope
		}
	}
	if recencia < 0 {
		recencia = 0
	}
	return recencia
}

// presupuestoHistorial resuelve el presupuesto del historial: la preferencia del
// usuario, si no el override de entorno y, si no, la fracción de la ventana que
// fija el umbral de compactación —el resto queda para el prompt, las herramientas
// y el recorte por ronda—; sin ventana conocida, el defecto.
func presupuestoHistorial(ventana int, prefs tui.Preferencias) int {
	if prefs.HistorialTokens > 0 {
		return prefs.HistorialTokens
	}
	if v := limiteDeEntorno(); v > 0 {
		return v
	}
	if ventana > 0 {
		return int(float64(ventana) * prefs.Compactacion.UmbralEfectivo())
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
