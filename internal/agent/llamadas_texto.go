package agent

import (
	"encoding/json"
	"strings"
)

// etiquetaLlamada es el envoltorio que usan muchas plantillas de chat (Qwen
// entre ellas) para las llamadas: el motor lo parsea cuando el modelo lo
// respeta; aquí se tolera cuando no lo hace.
const (
	etiquetaLlamadaAbre   = "<tool_call>"
	etiquetaLlamadaCierra = "</tool_call>"
)

// cercaCodigo es la cerca de Markdown que algunos modelos ponen alrededor del
// JSON de la llamada.
const cercaCodigo = "```"

// extraerLlamadasDeTexto reconoce una o varias herramientas pedidas como texto:
// el modelo escribe el JSON en vez de usar el canal nativo, algo frecuente en
// modelos pequeños servidos por llama.cpp. Solo se acepta la respuesta entera
// —una secuencia de bloques con nada más que espacios entre ellos— y solo si
// todos los nombres están en el catálogo efectivo del turno: nunca se inventa
// una herramienta ni se ejecuta el JSON de ejemplo de una explicación.
func extraerLlamadasDeTexto(texto string, permitidas []Herramienta) []Llamada {
	cuerpos, ok := cuerposDeLlamada(texto)
	if !ok {
		return nil
	}
	llamadas := make([]Llamada, 0, len(cuerpos))
	for _, cuerpo := range cuerpos {
		llamada, ok := decodificarLlamada(cuerpo, permitidas)
		if !ok {
			return nil
		}
		llamadas = append(llamadas, llamada)
	}
	return llamadas
}

// decodificarLlamada convierte el JSON de una llamada en la forma que consume
// la capa universal. El nombre tiene que estar entre las herramientas
// concedidas y los argumentos ser un objeto válido.
func decodificarLlamada(cuerpo string, permitidas []Herramienta) (Llamada, bool) {
	var bruto struct {
		Nombre     string          `json:"name"`
		Argumentos json.RawMessage `json:"arguments"`
		Funcion    *struct {
			Nombre     string          `json:"name"`
			Argumentos json.RawMessage `json:"arguments"`
		} `json:"function"`
	}
	if err := json.Unmarshal([]byte(cuerpo), &bruto); err != nil {
		return Llamada{}, false
	}

	nombre, argumentos := strings.TrimSpace(bruto.Nombre), bruto.Argumentos
	// Algunos modelos envuelven la llamada en `function`, como el protocolo
	// de OpenAI; se acepta la misma forma escrita a mano.
	if bruto.Funcion != nil {
		if nombre == "" {
			nombre = strings.TrimSpace(bruto.Funcion.Nombre)
		}
		if len(argumentos) == 0 {
			argumentos = bruto.Funcion.Argumentos
		}
	}
	if !herramientaDelTurno(nombre, permitidas) {
		return Llamada{}, false
	}

	normalizados, ok := normalizarArgumentos(argumentos)
	if !ok {
		return Llamada{}, false
	}
	return Llamada{Nombre: nombre, Argumentos: normalizados}, true
}

// cuerposDeLlamada parte la respuesta en los objetos JSON de cada llamada.
// Devuelve false si hay texto fuera de los bloques: entonces es una
// explicación, no una llamada.
func cuerposDeLlamada(texto string) ([]string, bool) {
	return escanearBloques(strings.TrimSpace(texto))
}

// escanearBloques recorre el texto aceptando bloques de llamada —objetos
// desnudos, cercas de Markdown o etiquetas de la plantilla— separados por
// espacios. Un bloque de llamada puede contener a su vez varios objetos.
func escanearBloques(t string) ([]string, bool) {
	var cuerpos []string
	for {
		t = strings.TrimSpace(t)
		if t == "" {
			break
		}
		switch {
		case strings.HasPrefix(t, cercaCodigo):
			interior, resto, ok := sacarCerca(t)
			if !ok {
				return nil, false
			}
			mas, ok := escanearBloques(interior)
			if !ok {
				return nil, false
			}
			cuerpos = append(cuerpos, mas...)
			t = resto
		case strings.HasPrefix(t, etiquetaLlamadaAbre):
			interior, resto, ok := sacarEtiqueta(t)
			if !ok {
				return nil, false
			}
			mas, ok := escanearBloques(interior)
			if !ok {
				return nil, false
			}
			cuerpos = append(cuerpos, mas...)
			t = resto
		case strings.HasPrefix(t, "{"):
			cuerpo, resto, ok := primerObjeto(t)
			if !ok {
				return nil, false
			}
			cuerpos = append(cuerpos, cuerpo)
			t = resto
		default:
			return nil, false
		}
	}
	if len(cuerpos) == 0 {
		return nil, false
	}
	return cuerpos, true
}

// sacarCerca quita una cerca de Markdown. La primera palabra puede ser el
// lenguaje (`json`); se descarta si no es ya el propio JSON.
func sacarCerca(t string) (interior, resto string, ok bool) {
	fin := strings.Index(t[len(cercaCodigo):], cercaCodigo)
	if fin < 0 {
		return "", "", false
	}
	interior = strings.TrimSpace(t[len(cercaCodigo) : len(cercaCodigo)+fin])
	resto = t[len(cercaCodigo)+fin+len(cercaCodigo):]
	interior = quitarLenguaje(interior)
	if interior == "" {
		return "", "", false
	}
	return interior, resto, true
}

// quitarLenguaje descarta la etiqueta de lenguaje que abre una cerca cuando no
// forma parte del JSON: `json {…}` o `json` en su propia línea.
func quitarLenguaje(t string) string {
	t = strings.TrimSpace(t)
	if t == "" || strings.HasPrefix(t, "{") || strings.HasPrefix(t, etiquetaLlamadaAbre) {
		return t
	}
	if i := strings.IndexAny(t, " \t\r\n"); i >= 0 {
		return strings.TrimSpace(t[i+1:])
	}
	return ""
}

// sacarEtiqueta quita `<tool_call>…</tool_call>`. Un prefijo o un sufijo de
// prosa invalida la llamada.
func sacarEtiqueta(t string) (interior, resto string, ok bool) {
	fin := strings.Index(t[len(etiquetaLlamadaAbre):], etiquetaLlamadaCierra)
	if fin < 0 {
		return "", "", false
	}
	interior = strings.TrimSpace(t[len(etiquetaLlamadaAbre) : len(etiquetaLlamadaAbre)+fin])
	resto = t[len(etiquetaLlamadaAbre)+fin+len(etiquetaLlamadaCierra):]
	if interior == "" {
		return "", "", false
	}
	return interior, resto, true
}

// primerObjeto extrae el objeto JSON que abre el texto y devuelve lo que queda.
// Corrige el `{{…}}` que algunos modelos copian de la plantilla: solo quita una
// capa y solo si el interior sigue siendo un objeto JSON.
func primerObjeto(t string) (cuerpo, resto string, ok bool) {
	doblado := strings.HasPrefix(t, "{{")
	if doblado {
		t = t[1:]
	}
	cuerpo, resto, ok = objetoInicial(t)
	if !ok {
		return "", "", false
	}
	if doblado {
		resto = strings.TrimLeft(resto, " \t\r\n")
		if !strings.HasPrefix(resto, "}") {
			return "", "", false
		}
		resto = resto[1:]
	}
	return cuerpo, resto, true
}

// objetoInicial decodifica el primer objeto JSON del texto y deja el resto sin
// consumir, para poder encadenar varios objetos.
func objetoInicial(t string) (cuerpo, resto string, ok bool) {
	dec := json.NewDecoder(strings.NewReader(t))
	var bruto json.RawMessage
	if err := dec.Decode(&bruto); err != nil {
		return "", "", false
	}
	fin := dec.InputOffset()
	if fin <= 0 || fin > int64(len(t)) {
		return "", "", false
	}
	var objeto map[string]json.RawMessage
	if err := json.Unmarshal(bruto, &objeto); err != nil || objeto == nil {
		return "", "", false
	}
	return string(bruto), t[fin:], true
}

// herramientaDelTurno comprueba que el nombre pedido está entre las herramientas
// concedidas: una llamada a algo que el agente no tiene no se ejecuta.
func herramientaDelTurno(nombre string, permitidas []Herramienta) bool {
	if nombre == "" {
		return false
	}
	for _, h := range permitidas {
		if h.Nombre == nombre {
			return true
		}
	}
	return false
}

// normalizarArgumentos valida que los argumentos sean un objeto JSON y los deja
// en la forma que consume la capa universal. Acepta el objeto y también la
// cadena JSON que usan algunos modelos (`"{\"ruta\":\"x\"}"`).
func normalizarArgumentos(bruto json.RawMessage) (string, bool) {
	if len(bruto) == 0 || string(bruto) == "null" {
		return "{}", true
	}
	texto := strings.TrimSpace(string(bruto))
	if strings.HasPrefix(texto, `"`) {
		var cadena string
		if err := json.Unmarshal(bruto, &cadena); err != nil {
			return "", false
		}
		texto = strings.TrimSpace(cadena)
	}
	var objeto map[string]json.RawMessage
	if err := json.Unmarshal([]byte(texto), &objeto); err != nil {
		return "", false
	}
	return texto, true
}
