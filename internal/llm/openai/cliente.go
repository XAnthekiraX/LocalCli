package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Cliente habla el núcleo OpenAI-compatible contra una dirección base.
type Cliente struct {
	baseURL string
	http    *http.Client
}

// NuevoCliente devuelve un cliente para la dirección base dada.
func NuevoCliente(baseURL string) *Cliente {
	return &Cliente{baseURL: strings.TrimRight(baseURL, "/"), http: &http.Client{}}
}

// BaseURL es la dirección base del servidor al que habla el cliente.
func (c *Cliente) BaseURL() string { return c.baseURL }

// Mensaje es un turno de la conversación que se le entrega al modelo. El
// asistente lleva las llamadas pedidas; la herramienta, el id que responde. Con
// imágenes, `content` viaja como partes y no como cadena: de eso se encarga
// `MarshalJSON`.
type Mensaje struct {
	Rol       string    `json:"-"`
	Contenido string    `json:"-"`
	Imagenes  []Imagen  `json:"-"`
	Llamadas  []Llamada `json:"tool_calls,omitempty"`
	IDLlamada string    `json:"tool_call_id,omitempty"`
}

// Imagen es una imagen adjunta a un mensaje. El núcleo común la serializa como
// `image_url` con data URI: es el único sitio que conoce ese formato.
type Imagen struct {
	MIME  string
	Datos []byte
}

// MarshalJSON arma `content` como cadena —sin imágenes— o como partes con
// `image_url` cuando las lleva. Es el punto único que conoce el formato del
// protocolo para las imágenes; el resto del harness no lo ve.
func (m Mensaje) MarshalJSON() ([]byte, error) {
	bruto := struct {
		Rol       string    `json:"role"`
		Contenido any       `json:"content"`
		Llamadas  []Llamada `json:"tool_calls,omitempty"`
		IDLlamada string    `json:"tool_call_id,omitempty"`
	}{Rol: m.Rol, Llamadas: m.Llamadas, IDLlamada: m.IDLlamada}
	if len(m.Imagenes) == 0 {
		bruto.Contenido = m.Contenido
	} else {
		bruto.Contenido = partesDeContenido(m.Contenido, m.Imagenes)
	}
	return json.Marshal(bruto)
}

// partesDeContenido compone las partes de `content`: primero el texto —si lo
// hay— y después una parte `image_url` por imagen, con su data URI en base64.
func partesDeContenido(texto string, imagenes []Imagen) []any {
	partes := make([]any, 0, len(imagenes)+1)
	if texto != "" {
		partes = append(partes, map[string]any{"type": "text", "text": texto})
	}
	for _, im := range imagenes {
		partes = append(partes, map[string]any{
			"type": "image_url",
			"image_url": map[string]any{
				"url": "data:" + im.MIME + ";base64," + base64.StdEncoding.EncodeToString(im.Datos),
			},
		})
	}
	return partes
}

// Llamada es una herramienta pedida por el modelo, ya formada.
type Llamada struct {
	ID      string  `json:"id"`
	Tipo    string  `json:"type"`
	Funcion Funcion `json:"function"`
}

// Funcion es el nombre y los argumentos de una llamada.
type Funcion struct {
	Nombre     string `json:"name"`
	Argumentos string `json:"arguments"`
}

// Herramienta es la definición de una herramienta para el canal nativo.
type Herramienta struct {
	Tipo    string     `json:"type"`
	Funcion Definicion `json:"function"`
}

// Definicion declara nombre, descripción y esquema de los argumentos.
type Definicion struct {
	Nombre      string          `json:"name"`
	Descripcion string          `json:"description"`
	Parametros  json.RawMessage `json:"parameters,omitempty"`
}

// Delta es un fragmento del stream: texto final o razonamiento.
type Delta struct {
	Texto        string
	Razonamiento bool
}

// Uso es el conteo de tokens de un turno, si el servidor lo entrega.
type Uso struct {
	Prompt     int
	Completion int
}

// Respuesta es el resultado de un turno: texto, llamadas pedidas y uso.
type Respuesta struct {
	Texto    string
	Llamadas []Llamada
	Uso      Uso
}

// Opciones ajusta una petición: tope de tokens, silenciar el razonamiento y la
// elección de herramienta. Las dos primeras se usan en las llamadas auxiliares
// del harness (título y resumen), que no deben gastar el turno pensando; la
// tercera solo cuando hay canal de herramientas.
type Opciones struct {
	MaxTokens       int
	SinRazonamiento bool
	// EleccionHerramienta es el valor de `tool_choice` (`"auto"`, `"none"`,
	// `"required"` o una herramienta concreta). Vacío no lo envía: el servidor
	// decide y usa `auto`.
	EleccionHerramienta string
}

// Modelo es una entrada de `GET /v1/models`.
type Modelo struct {
	ID           string
	Ventana      int  // ventana efectiva declarada (0 si no la da)
	Herramientas bool // si declara capacidad de herramientas
	Vision       bool // si declara capacidad de visión (imágenes)
}

// Chat es una petición sin herramientas ni opciones: devuelve el uso y entrega
// el texto por `alDelta`. Un error de `alDelta` corta la generación.
func (c *Cliente) Chat(ctx context.Context, modelo string, mensajes []Mensaje, alDelta func(Delta) error) (Uso, error) {
	respuesta, err := c.Completar(ctx, modelo, mensajes, nil, Opciones{}, alDelta)
	return respuesta.Uso, err
}

// Completar hace una petición con el canal de herramientas opcional y devuelve el
// texto, las llamadas pedidas y el uso.
func (c *Cliente) Completar(ctx context.Context, modelo string, mensajes []Mensaje, herramientas []Herramienta, opciones Opciones, alDelta func(Delta) error) (Respuesta, error) {
	p := peticion{
		Modelo:       modelo,
		Mensajes:     normalizarMensajes(mensajes),
		Herramientas: herramientas,
		Stream:       true,
		Uso:          &opcionesUso{Incluir: true},
	}
	if opciones.MaxTokens > 0 {
		p.MaxTokens = opciones.MaxTokens
	}
	if opciones.EleccionHerramienta != "" {
		p.EleccionHerramienta = opciones.EleccionHerramienta
	}
	if opciones.SinRazonamiento {
		p.ChatTemplateKwargs = map[string]any{"enable_thinking": false}
	}
	datos, err := json.Marshal(p)
	if err != nil {
		return Respuesta{}, errorf("no se pudo preparar la petición: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/chat/completions", bytes.NewReader(datos))
	if err != nil {
		return Respuesta{}, errorf("no se pudo preparar la petición: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.http.Do(req)
	if err != nil {
		return Respuesta{}, errorf("no se pudo conectar con el servidor en %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body := trozo(resp.Body)
		return Respuesta{}, nuevoErrorServidor(resp.StatusCode, body)
	}
	return leerStream(resp.Body, alDelta)
}

// Roles del protocolo que el harness maneja. El resto viaja como cadena tal cual.
const (
	rolSistema = "system"
	rolUsuario = "user"
)

// normalizarMensajes ajusta los turnos al contrato del chat de llama.cpp: como
// mucho un mensaje de sistema y siempre el primero. El bloque inicial de
// mensajes de sistema se funde en uno solo —el prompt del agente y el resumen del
// historial comparten ese sitio—; un mensaje de sistema posterior, que el harness
// inyecta a mitad del turno como empujón o instrucción de redacción, pasa a rol de
// usuario para conservar su posición. Sin esto, una plantilla de chat estricta
// —la de Qwen, por ejemplo— rechaza la petición con «System message must be at the
// beginning».
func normalizarMensajes(mensajes []Mensaje) []Mensaje {
	if len(mensajes) == 0 {
		return mensajes
	}
	sistema := make([]string, 0, 1)
	i := 0
	for i < len(mensajes) && mensajes[i].Rol == rolSistema {
		if c := strings.TrimSpace(mensajes[i].Contenido); c != "" {
			sistema = append(sistema, c)
		}
		i++
	}
	salida := make([]Mensaje, 0, len(mensajes))
	if len(sistema) > 0 {
		salida = append(salida, Mensaje{Rol: rolSistema, Contenido: strings.Join(sistema, "\n\n")})
	}
	for ; i < len(mensajes); i++ {
		m := mensajes[i]
		if m.Rol == rolSistema {
			m.Rol = rolUsuario
		}
		salida = append(salida, m)
	}
	return salida
}

// Modelos lista los modelos que sirve el servidor.
func (c *Cliente) Modelos(ctx context.Context) ([]Modelo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/models", nil)
	if err != nil {
		return nil, errorf("no se pudo preparar la consulta: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, errorf("no se pudo conectar con el servidor en %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body := trozo(resp.Body)
		return nil, nuevoErrorServidor(resp.StatusCode, body)
	}

	var bruto respuestaModelos
	if err := json.NewDecoder(resp.Body).Decode(&bruto); err != nil {
		return nil, errorf("la lista de modelos no se pudo leer: %w", err)
	}

	// `data` trae el id y la ventana (`meta.n_ctx`); `models` trae las
	// capacidades. Van alineadas por índice.
	if len(bruto.Data) > 0 {
		salida := make([]Modelo, 0, len(bruto.Data))
		for i, m := range bruto.Data {
			var capacidades []string
			if i < len(bruto.Models) {
				capacidades = bruto.Models[i].Capacidades
			}
			salida = append(salida, Modelo{
				ID:           m.ID,
				Ventana:      m.Meta.NCtx,
				Herramientas: contiene(capacidades, "tools"),
				Vision:       contiene(capacidades, "vision"),
			})
		}
		return salida, nil
	}
	salida := make([]Modelo, 0, len(bruto.Models))
	for _, m := range bruto.Models {
		salida = append(salida, Modelo{
			ID:           m.Nombre,
			Herramientas: contiene(m.Capacidades, "tools"),
			Vision:       contiene(m.Capacidades, "vision"),
		})
	}
	return salida, nil
}

// peticion es el cuerpo de `POST /v1/chat/completions`.
type peticion struct {
	Modelo              string         `json:"model"`
	Mensajes            []Mensaje      `json:"messages"`
	Herramientas        []Herramienta  `json:"tools,omitempty"`
	EleccionHerramienta string         `json:"tool_choice,omitempty"`
	Stream              bool           `json:"stream"`
	Uso                 *opcionesUso   `json:"stream_options,omitempty"`
	MaxTokens           int            `json:"max_tokens,omitempty"`
	ChatTemplateKwargs  map[string]any `json:"chat_template_kwargs,omitempty"`
}

type opcionesUso struct {
	Incluir bool `json:"include_usage"`
}

// fragmento es un `data:` del stream.
type fragmento struct {
	Elecciones []struct {
		Delta struct {
			Contenido    *string            `json:"content"`
			Razonamiento *string            `json:"reasoning_content"`
			Llamadas     []fragmentoLlamada `json:"tool_calls"`
		} `json:"delta"`
	} `json:"choices"`
	Uso *struct {
		Prompt     int `json:"prompt_tokens"`
		Completion int `json:"completion_tokens"`
	} `json:"usage"`
}

// fragmentoLlamada es un trozo de una llamada a herramienta dentro del stream.
type fragmentoLlamada struct {
	Indice  int    `json:"index"`
	ID      string `json:"id"`
	Tipo    string `json:"type"`
	Funcion struct {
		Nombre     string `json:"name"`
		Argumentos string `json:"arguments"`
	} `json:"function"`
}

// respuestaModelos es la respuesta de `GET /v1/models`.
type respuestaModelos struct {
	Data []struct {
		ID   string `json:"id"`
		Meta struct {
			NCtx int `json:"n_ctx"`
		} `json:"meta"`
	} `json:"data"`
	Models []struct {
		Nombre      string   `json:"model"`
		Capacidades []string `json:"capabilities"`
	} `json:"models"`
}

// leerStream recorre el SSE, entrega cada fragmento y acumula el texto, las
// llamadas a herramienta (que llegan troceadas por índice) y el uso. Una línea
// ilegible se ignora.
func leerStream(r io.Reader, alDelta func(Delta) error) (Respuesta, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var respuesta Respuesta
	llamadas := nuevoAcumuladorLlamadas()
	cerrado := false

	for sc.Scan() {
		linea := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(linea, "data:") {
			continue
		}
		carga := strings.TrimSpace(strings.TrimPrefix(linea, "data:"))
		if carga == "[DONE]" {
			cerrado = true
			break
		}
		var f fragmento
		if err := json.Unmarshal([]byte(carga), &f); err != nil {
			continue
		}
		for _, e := range f.Elecciones {
			for _, tc := range e.Delta.Llamadas {
				llamadas.añadir(tc)
			}
			if e.Delta.Razonamiento != nil && *e.Delta.Razonamiento != "" {
				if err := entregar(alDelta, Delta{Texto: *e.Delta.Razonamiento, Razonamiento: true}); err != nil {
					return respuesta, err
				}
			}
			if e.Delta.Contenido != nil && *e.Delta.Contenido != "" {
				respuesta.Texto += *e.Delta.Contenido
				if err := entregar(alDelta, Delta{Texto: *e.Delta.Contenido}); err != nil {
					return respuesta, err
				}
			}
		}
		if f.Uso != nil {
			respuesta.Uso = Uso{Prompt: f.Uso.Prompt, Completion: f.Uso.Completion}
		}
	}
	if err := sc.Err(); err != nil {
		return respuesta, errorf("la lectura del stream se interrumpió: %w", err)
	}
	if !cerrado {
		return respuesta, errorf("el stream terminó sin [DONE]: la conexión se cortó antes de tiempo")
	}
	respuesta.Llamadas = llamadas.llamadas()
	return respuesta, nil
}

// acumuladorLlamadas recompone las llamadas que llegan troceadas por índice.
type acumuladorLlamadas struct {
	porIndice map[int]*Llamada
	orden     []int
}

func nuevoAcumuladorLlamadas() *acumuladorLlamadas {
	return &acumuladorLlamadas{porIndice: map[int]*Llamada{}}
}

func (a *acumuladorLlamadas) añadir(f fragmentoLlamada) {
	llamada, ok := a.porIndice[f.Indice]
	if !ok {
		llamada = &Llamada{Tipo: "function"}
		a.porIndice[f.Indice] = llamada
		a.orden = append(a.orden, f.Indice)
	}
	if f.ID != "" {
		llamada.ID = f.ID
	}
	if f.Tipo != "" {
		llamada.Tipo = f.Tipo
	}
	if f.Funcion.Nombre != "" {
		llamada.Funcion.Nombre = f.Funcion.Nombre
	}
	llamada.Funcion.Argumentos += f.Funcion.Argumentos
}

func (a *acumuladorLlamadas) llamadas() []Llamada {
	salida := make([]Llamada, 0, len(a.orden))
	for _, indice := range a.orden {
		salida = append(salida, *a.porIndice[indice])
	}
	return salida
}

func entregar(alDelta func(Delta) error, d Delta) error {
	if alDelta == nil {
		return nil
	}
	return alDelta(d)
}

func contiene(xs []string, busca string) bool {
	for _, x := range xs {
		if x == busca {
			return true
		}
	}
	return false
}

// trozo lee un fragmento corto del cuerpo para el mensaje de un error.
func trozo(r io.Reader) string {
	const tope = 300
	datos, _ := io.ReadAll(io.LimitReader(r, tope))
	return strings.TrimSpace(string(datos))
}

// ErrorServidor representa un error devuelto por el servidor LLM.
type ErrorServidor struct {
	Codigo    int
	Mensaje   string
	Tipo      string
	EsTransitorio bool
}

func (e ErrorServidor) Error() string {
	return fmt.Sprintf("llm/openai: servidor %d: %s", e.Codigo, e.Mensaje)
}

// EsErrorTransitorio determina si un error del servidor es transitorio (reintentar).
func EsErrorTransitorio(err error) bool {
	var eErr ErrorServidor
	if errors.As(err, &eErr) {
		return eErr.EsTransitorio
	}
	return false
}

func errorf(format string, args ...any) error {
	return fmt.Errorf("llm/openai: "+format, args...)
}

func nuevoErrorServidor(statusCode int, body string) error {
	msg := strings.TrimSpace(body)
	tipo := "desconocido"
	transitorio := false

	if statusCode >= 500 && statusCode < 600 {
		tipo = "servidor"
		transitorio = true
	} else if statusCode == 429 {
		tipo = "rate_limit"
		transitorio = true
	} else if statusCode >= 400 && statusCode < 500 {
		tipo = "cliente"
		transitorio = false
	}

	return ErrorServidor{
		Codigo:        statusCode,
		Mensaje:       msg,
		Tipo:          tipo,
		EsTransitorio: transitorio,
	}
}
