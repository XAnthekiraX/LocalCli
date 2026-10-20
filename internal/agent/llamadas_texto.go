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

// extraerLlamadaDeTexto reconoce una herramienta pedida como texto: el modelo
// escribe el JSON en vez de usar el canal nativo, algo frecuente en modelos
// pequeños servidos por llama.cpp. Solo se acepta la respuesta entera —un JSON
// de ejemplo dentro de una explicación no se ejecuta— y solo si el nombre está
// en el catálogo efectivo del turno: nunca se inventa una herramienta.
func extraerLlamadaDeTexto(texto string, permitidas []Herramienta) (Llamada, bool) {
	cuerpo, ok := cuerpoDeLlamada(texto)
	if !ok {
		return Llamada{}, false
	}

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

// cuerpoDeLlamada deja el JSON desnudo. Acepta la respuesta entera, con cerca
// de Markdown, con la etiqueta de la plantilla o con las dos cosas. Devuelve
// false si hay texto fuera del envoltorio: entonces es una explicación, no una
// llamada.
func cuerpoDeLlamada(texto string) (string, bool) {
	t := strings.TrimSpace(texto)
	if t == "" {
		return "", false
	}
	if interior, ok := desenvolverEtiqueta(t); ok {
		t = interior
	}
	if interior, ok := desenvolverCerca(t); ok {
		t = interior
	}
	if interior, ok := desenvolverEtiqueta(t); ok {
		t = interior
	}
	t = desdoblarLlaves(t)
	if !strings.HasPrefix(t, "{") || !strings.HasSuffix(t, "}") {
		return "", false
	}
	return t, true
}

// desenvolverEtiqueta quita `<tool_call>…</tool_call>` cuando envuelve todo el
// texto. Un prefijo o un sufijo de prosa invalida la llamada.
func desenvolverEtiqueta(t string) (string, bool) {
	if !strings.HasPrefix(t, etiquetaLlamadaAbre) || !strings.HasSuffix(t, etiquetaLlamadaCierra) {
		return "", false
	}
	interior := strings.TrimSpace(t[len(etiquetaLlamadaAbre) : len(t)-len(etiquetaLlamadaCierra)])
	if interior == "" {
		return "", false
	}
	return interior, true
}

// desenvolverCerca quita una cerca de Markdown que ocupe todo el texto. La
// primera línea puede ser el lenguaje (`json`); se descarta si no es ya el
// propio JSON.
func desenvolverCerca(t string) (string, bool) {
	const cerca = "```"
	if !strings.HasPrefix(t, cerca) || !strings.HasSuffix(t, cerca) || len(t) < 2*len(cerca) {
		return "", false
	}
	interior := strings.TrimSpace(t[len(cerca) : len(t)-len(cerca)])
	if interior == "" || strings.Contains(interior, cerca) {
		return "", false
	}
	if i := strings.IndexByte(interior, '\n'); i >= 0 {
		primera := strings.TrimSpace(interior[:i])
		if primera != "" && !strings.HasPrefix(primera, "{") && !strings.HasPrefix(primera, etiquetaLlamadaAbre) {
			interior = strings.TrimSpace(interior[i+1:])
		}
	}
	return interior, true
}

// desdoblarLlaves corrige el `{{…}}` que algunos modelos copian de la plantilla
// al escribir la llamada. Solo quita una capa y solo si el interior sigue siendo
// un objeto JSON.
func desdoblarLlaves(t string) string {
	if !strings.HasPrefix(t, "{{") || !strings.HasSuffix(t, "}}") {
		return t
	}
	interior := strings.TrimSpace(t[1 : len(t)-1])
	if strings.HasPrefix(interior, "{") && strings.HasSuffix(interior, "}") {
		return interior
	}
	return t
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
