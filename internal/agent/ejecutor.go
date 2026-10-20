package agent

import (
	"context"
	"fmt"
	"strings"

	"localcli/internal/llm"
)

// MaxRondasDefecto es el tope de rondas de herramientas de un turno.
const MaxRondasDefecto = 8

// MaxEmpujonesDefecto es el tope de empujones de continuación por turno.
const MaxEmpujonesDefecto = 3

// instruccionRedaccion pide cerrar el turno con lo conseguido, sin herramientas.
const instruccionRedaccion = "Ya no puedes usar herramientas. Escribe ahora tu resultado con lo que hayas conseguido."

// instruccionSinHerramientas insiste cuando el modelo vuelve a pedir herramientas.
const instruccionSinHerramientas = "No hay más herramientas disponibles. Responde solo con texto."

// instruccionContinuar evita que el turno se cierre justo después de un fallo
// corregible: empuja al modelo a corregir o a probar otra herramienta.
const instruccionContinuar = "La última herramienta falló. Corrige los argumentos o usa otra herramienta y continúa; no cierres el turno todavía."

// elEmpujonConPeticion reintroduce la petición original del usuario, con sus
// rutas tal cual, cuando los dos primeros empujones no bastaron.
const elEmpujonConPeticion = 3

// Respuesta es el resultado de una ronda del modelo.
type Respuesta struct {
	Texto    string
	Llamadas []Llamada
	Uso      Uso
}

// Motor ejecuta una ronda contra el modelo y entrega el texto por `alDelta`.
type Motor interface {
	Ronda(ctx context.Context, p Peticion, alDelta func(Delta) error) (Respuesta, error)
}

// Resultado es lo que devuelve la ejecución de una herramienta. `Error` es un
// fallo que el modelo puede corregir; un `error` de Go es un fallo del harness.
type Resultado struct {
	Salida string
	Error  string
}

// EjecutorHerramientas despacha una herramienta contra la capa universal (T-B012).
type EjecutorHerramientas interface {
	Ejecutar(ctx context.Context, agente string, l Llamada) (Resultado, error)
}

// Ejecutor corre el ciclo conversacional del agente: pide al modelo, ejecuta lo
// que pida y vuelve a pedir, hasta que responde en texto o se agotan las rondas.
type Ejecutor struct {
	Agente       Agente
	Cerrado      []Herramienta
	Motor        Motor
	Herramientas EjecutorHerramientas
	Capacidad    llm.Capacidad
	MaxRondas    int
	MaxEmpujones int
	// Ventana es la ventana de contexto del turno, en tokens. 0 la deja sin
	// límite: el recorte por ronda no actúa.
	Ventana int
	Avisar  func(motivo, siguiente string)

	// recorteAvisado evita repetir el aviso de recorte en cada ronda del turno.
	recorteAvisado bool
}

// Turno ejecuta un turno y devuelve la respuesta final del agente. Si el modelo
// intenta cerrar con texto justo después de que sus herramientas fallaran, se le
// empuja a continuar en vez de aceptar una rendición: los primeros empujones
// piden corregir o probar otra herramienta; el último reintroduce la petición
// original del usuario con sus rutas tal cual.
func (e *Ejecutor) Turno(ctx context.Context, historial []Mensaje, alDelta func(Delta) error) (Respuesta, error) {
	herramientas, conversacion := e.herramientasDelTurno()
	mensajes := append([]Mensaje{}, historial...)
	peticionOriginal := ultimaPeticionUsuario(historial)

	falloPendiente := false
	empujones := 0
	for ronda := 0; ronda < e.maxRondas(); ronda++ {
		mensajes = e.ajustarAlContexto(mensajes, herramientas)
		// El texto de la ronda se retiene mientras podría ser una llamada: si
		// acaba siéndolo, su JSON no se pinta como respuesta.
		retenido := nuevoRetenedor(alDelta)
		respuesta, err := e.Motor.Ronda(ctx, ConstruirPeticion(e.Agente, mensajes, herramientas), retenido.Fragmento)
		if err != nil {
			_ = retenido.Volcar()
			return Respuesta{}, err
		}
		if conversacion {
			if err := retenido.Volcar(); err != nil {
				return Respuesta{}, err
			}
			return respuesta, nil
		}
		// Un modelo pequeño puede escribir la llamada en vez de pedirla por el
		// canal nativo: se reconoce como último recurso, nunca en su lugar.
		if len(respuesta.Llamadas) == 0 {
			if l, ok := extraerLlamadaDeTexto(respuesta.Texto, herramientas); ok {
				l.ID = fmt.Sprintf("texto-%d", ronda+1)
				respuesta.Llamadas = []Llamada{l}
				respuesta.Texto = ""
				retenido.Descartar()
			}
		}
		if err := retenido.Volcar(); err != nil {
			return Respuesta{}, err
		}
		if len(respuesta.Llamadas) == 0 {
			if falloPendiente && empujones < e.maxEmpujones() {
				empujones++
				mensajes = append(mensajes, Mensaje{Rol: RolSistema, Contenido: instruccionContinuacion(empujones, peticionOriginal)})
				continue
			}
			return respuesta, nil
		}
		mensajes = append(mensajes, Mensaje{Rol: RolAsistente, Llamadas: respuesta.Llamadas})
		fallo := false
		exito := false
		for _, l := range respuesta.Llamadas {
			texto, f := e.ejecutar(ctx, l)
			if f {
				fallo = true
			} else {
				exito = true
			}
			mensajes = append(mensajes, Mensaje{Rol: RolHerramienta, Contenido: texto, IDLlamada: l.ID})
		}
		// Un acierto limpia el fallo pendiente: el agente volvió a avanzar.
		if exito {
			falloPendiente = false
		} else if fallo {
			falloPendiente = true
		}
	}
	return e.redactar(ctx, mensajes, alDelta)
}

// ultimaPeticionUsuario devuelve el último mensaje del usuario con contenido.
func ultimaPeticionUsuario(historial []Mensaje) string {
	for i := len(historial) - 1; i >= 0; i-- {
		if historial[i].Rol == RolUsuario && strings.TrimSpace(historial[i].Contenido) != "" {
			return historial[i].Contenido
		}
	}
	return ""
}

// instruccionContinuacion arma el empujón: genérico al principio y, en el
// último, con la petición original para que el modelo reencuentre las rutas.
func instruccionContinuacion(n int, peticionOriginal string) string {
	if n >= elEmpujonConPeticion && strings.TrimSpace(peticionOriginal) != "" {
		return instruccionContinuar + " Tu petición original fue: «" + peticionOriginal + "». Vuelve a intentarlo usando exactamente las rutas que aparecen ahí."
	}
	return instruccionContinuar
}

// maxEmpujones acota los empujones de continuación de un turno.
func (e *Ejecutor) maxEmpujones() int {
	if e.MaxEmpujones > 0 {
		return e.MaxEmpujones
	}
	return MaxEmpujonesDefecto
}

// herramientasDelTurno devuelve las herramientas concedidas. Sin capacidad
// `soportada` cae a modo conversación y lo avisa, en vez de fingir que trabaja.
func (e *Ejecutor) herramientasDelTurno() ([]Herramienta, bool) {
	if e.Capacidad != llm.Soportada {
		e.avisar("modo conversación: el modelo no admite herramientas", "cambia de modelo si quieres que las use")
		return nil, true
	}
	return CatalogoEfectivo(e.Agente, e.Cerrado), false
}

// ejecutar despacha una herramienta y devuelve el texto que va al modelo más si
// hubo un fallo que el modelo puede intentar corregir. Un fallo —corregible o
// del harness— vuelve como resultado y no corta el turno.
func (e *Ejecutor) ejecutar(ctx context.Context, l Llamada) (string, bool) {
	if e.Herramientas == nil {
		return "la herramienta " + l.Nombre + " no está disponible", false
	}
	resultado, err := e.Herramientas.Ejecutar(ctx, e.Agente.Nombre, l)
	if err != nil {
		return "la herramienta " + l.Nombre + " falló: " + err.Error(), true
	}
	if resultado.Error != "" {
		return resultado.Error, true
	}
	return resultado.Salida, false
}

// redactar cierra el turno con una pasada sin herramientas. Si el modelo vuelve a
// pedirlas, se reintenta una sola vez; si tampoco entrega texto, el turno falla.
func (e *Ejecutor) redactar(ctx context.Context, mensajes []Mensaje, alDelta func(Delta) error) (Respuesta, error) {
	intento := append([]Mensaje{}, mensajes...)
	intento = append(intento, Mensaje{Rol: RolSistema, Contenido: instruccionRedaccion})
	for reintento := 0; reintento < 2; reintento++ {
		intento = e.ajustarAlContexto(intento, nil)
		respuesta, err := e.Motor.Ronda(ctx, ConstruirPeticion(e.Agente, intento, nil), alDelta)
		if err != nil {
			return Respuesta{}, err
		}
		if len(respuesta.Llamadas) == 0 && strings.TrimSpace(respuesta.Texto) != "" {
			return respuesta, nil
		}
		intento = append(intento, Mensaje{Rol: RolSistema, Contenido: instruccionSinHerramientas})
	}
	e.avisar("el turno agotó las rondas y el modelo no entregó texto", "reintenta o cambia de modelo")
	return Respuesta{}, fmt.Errorf("agent: se agotaron las rondas y el modelo no entregó texto")
}

func (e *Ejecutor) maxRondas() int {
	if e.MaxRondas > 0 {
		return e.MaxRondas
	}
	return MaxRondasDefecto
}

func (e *Ejecutor) avisar(motivo, siguiente string) {
	if e.Avisar != nil {
		e.Avisar(motivo, siguiente)
	}
}

// reservaSalida es lo que se deja libre en la ventana para la respuesta del
// modelo al presupuestar el prompt del turno.
const reservaSalida = 1024

// marcaRecorte contextualiza un resultado recortado por no caber en la ventana.
const marcaRecorte = "\n… [recortado: no cabe en la ventana de contexto]"

// ajustarAlContexto recorta los mensajes del turno para que quepan en la ventana.
// Conserva el más nuevo y va añadiendo hacia atrás mientras el prompt —el prompt
// del agente, los esquemas de las herramientas y los mensajes— no supere la
// ventana menos la reserva de salida. Un resultado de herramienta que por sí solo
// no quepa se trunca en vez de descartarse. Sin ventana conocida no actúa.
func (e *Ejecutor) ajustarAlContexto(mensajes []Mensaje, herramientas []Herramienta) []Mensaje {
	if e.Ventana <= 0 || len(mensajes) == 0 {
		return mensajes
	}
	presupuesto := e.Ventana - reservaSalida
	fijo := llm.EstimarTokens(e.Agente.Prompt) + tokensDeHerramientas(herramientas)
	if fijo >= presupuesto {
		// El prompt del sistema y las herramientas ya llenan la ventana: recortar
		// más rompería el turno, así que se avisa y se envía tal cual.
		e.avisar("el prompt del agente y las herramientas llenan la ventana de contexto", "usa un modelo con más contexto")
		return mensajes
	}
	disponible := presupuesto - fijo

	conservados := make([]Mensaje, 0, len(mensajes))
	usados := 0
	recortado := false
	for i := len(mensajes) - 1; i >= 0; i-- {
		costo := tokensDeMensaje(mensajes[i])
		if usados+costo > disponible {
			recortado = true
			break
		}
		usados += costo
		conservados = append(conservados, mensajes[i])
	}
	if !recortado {
		return mensajes
	}

	// El mensaje más nuevo —la petición del usuario o su resultado— siempre se
	// conserva, truncado si él solo supera el presupuesto.
	if len(conservados) == 0 {
		conservados = append(conservados, truncarMensaje(mensajes[len(mensajes)-1], disponible))
	}

	// Reordenar al orden original y quitar un `tool` huérfano al frente: si su
	// llamada quedó fuera, no se puede enviar sin el `assistant` que la pidió.
	invertir(conservados)
	for len(conservados) > 0 && conservados[0].Rol == RolHerramienta {
		conservados = conservados[1:]
	}
	if len(conservados) == 0 {
		conservados = []Mensaje{truncarMensaje(ultimoMensajeNoHerramienta(mensajes), disponible)}
	}

	if !e.recorteAvisado {
		e.recorteAvisado = true
		e.avisar("contexto recortado: el turno superaba la ventana del modelo", "se descartaron mensajes antiguos")
	}
	return conservados
}

// tokensDeHerramientas estima el coste de los esquemas de las herramientas
// concedidas, que viajan en la petición.
func tokensDeHerramientas(herramientas []Herramienta) int {
	total := 0
	for _, h := range herramientas {
		total += llm.EstimarTokens(h.Nombre) + llm.EstimarTokens(h.Descripcion) + llm.EstimarTokens(string(h.Esquema)) + 8
	}
	return total
}

// tokensDeMensaje estima el coste de un mensaje con su sobre de protocolo.
func tokensDeMensaje(m Mensaje) int {
	total := llm.EstimarTokens(m.Contenido) + 4
	for _, l := range m.Llamadas {
		total += llm.EstimarTokens(l.Nombre) + llm.EstimarTokens(l.Argumentos)
	}
	return total
}

// truncarMensaje recorta el contenido de un mensaje grande a lo que quepa.
func truncarMensaje(m Mensaje, tope int) Mensaje {
	if tope < 0 {
		tope = 0
	}
	maxRunas := tope * 4
	if r := []rune(m.Contenido); len(r) > maxRunas {
		m.Contenido = string(r[:maxRunas]) + marcaRecorte
	}
	return m
}

// invertir da la vuelta a una lista de mensajes en el sitio.
func invertir(mensajes []Mensaje) {
	for i, j := 0, len(mensajes)-1; i < j; i, j = i+1, j-1 {
		mensajes[i], mensajes[j] = mensajes[j], mensajes[i]
	}
}

// ultimoMensajeNoHerramienta devuelve el mensaje más nuevo que no sea un
// resultado de herramienta; si no lo hay, el más nuevo. Es el último recurso
// para no dejar el turno sin ningún mensaje válido.
func ultimoMensajeNoHerramienta(mensajes []Mensaje) Mensaje {
	for i := len(mensajes) - 1; i >= 0; i-- {
		if mensajes[i].Rol != RolHerramienta {
			return mensajes[i]
		}
	}
	return mensajes[len(mensajes)-1]
}
