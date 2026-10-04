// errors.go — T-B005-08 / T-B036-04: errores del adaptador sobre el error
// tipado neutro de `llm`.
//
// Fuente de verdad: ai/docs/backend/05-quality/ERRORS.md. El tipo y los códigos
// viven en `llm` (T-B036-03); aquí solo queda lo que es específico de Ollama:
// su instrucción de arranque, la clasificación de fallos de red y el parseo del
// cuerpo de error del servidor.
package ollama

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"localcli/internal/llm"
)

// mensajeLevantarOllama explica cómo actuar: Ollama no responde y hay que
// levantarlo.
const mensajeLevantarOllama = "no se puede generar: Ollama no responde en la dirección configurada; levántalo con `ollama serve`"

// clasificarFalloRed mapea un fallo de conexión contra el servidor local al
// error tipado documentado.
func clasificarFalloRed(err error) (*llm.ErrorMotor, bool) {
	return llm.ClasificarFalloRed(err, mensajeLevantarOllama)
}

// errorDeStatus construye el error tipado de una respuesta HTTP no-200. Lee el
// cuerpo de Ollama —que trae el motivo real en `{"error": …}`— y lo deja en el
// Detalle: sin él, un fallo como «no user query found in messages» llegaba a la
// pantalla como un «(500)» pelado.
func errorDeStatus(resp *http.Response) error {
	detalle := cuerpoDeError(resp)
	if detalle == "" {
		detalle = http.StatusText(resp.StatusCode)
	}
	return llm.NuevoErrorNoDisponible(
		"Ollama respondió con un error ("+itoa(resp.StatusCode)+")",
		detalle,
	)
}

// cuerpoDeError lee el motivo de una respuesta de error: si el cuerpo es el
// `{"error": "…"}` de Ollama, devuelve ese texto; si no, el cuerpo crudo
// colapsado. Vacío si no hay nada legible.
func cuerpoDeError(resp *http.Response) string {
	defer resp.Body.Close()
	datos, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	texto := strings.TrimSpace(string(datos))
	var envoltura struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal([]byte(texto), &envoltura); err == nil && envoltura.Error != "" {
		texto = envoltura.Error
	}
	texto = strings.Join(strings.Fields(texto), " ")
	const maximo = 300
	if r := []rune(texto); len(r) > maximo {
		texto = string(r[:maximo]) + "…"
	}
	return texto
}

// nuevoModeloNoCabe construye el aviso tipado (E_MODEL_TOO_BIG) por modelo que
// no cabe. Es un aviso: el flujo de carga continúa.
func nuevoModeloNoCabe(modelo string, requiereMB, vramMB int64) *llm.ErrorMotor {
	return llm.NuevoModeloNoCabe(modelo, requiereMB, vramMB)
}

// itoa evita arrastrar strconv para un entero en un mensaje.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
