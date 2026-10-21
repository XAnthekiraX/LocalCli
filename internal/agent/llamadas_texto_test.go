package agent

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestExtraerLlamadaDeTexto(t *testing.T) {
	permitidas := []Herramienta{
		{Nombre: "leer_archivo"},
		{Nombre: "ejecutar_comando"},
	}

	casos := []struct {
		nombre   string
		texto    string
		esperado string
		ok       bool
	}{
		{
			nombre:   "objeto desnudo",
			texto:    `{"name":"leer_archivo","arguments":{"ruta":"AGENTS.md"}}`,
			esperado: `{"ruta":"AGENTS.md"}`,
			ok:       true,
		},
		{
			nombre:   "cerca con lenguaje",
			texto:    "```json\n{\n  \"name\": \"ejecutar_comando\",\n  \"arguments\": {\n    \"comando\": \"go build ./...\"\n  }\n}\n```",
			esperado: `{"comando":"go build ./..."}`,
			ok:       true,
		},
		{
			nombre:   "cerca sin lenguaje",
			texto:    "```\n{\"name\":\"leer_archivo\",\"arguments\":{\"ruta\":\"x\"}}\n```",
			esperado: `{"ruta":"x"}`,
			ok:       true,
		},
		{
			nombre:   "etiqueta de la plantilla",
			texto:    "<tool_call>\n{\"name\":\"leer_archivo\",\"arguments\":{\"ruta\":\"x\"}}\n</tool_call>",
			esperado: `{"ruta":"x"}`,
			ok:       true,
		},
		{
			nombre:   "etiqueta dentro de la cerca",
			texto:    "```json\n<tool_call>\n{\"name\":\"leer_archivo\",\"arguments\":{\"ruta\":\"x\"}}\n</tool_call>\n```",
			esperado: `{"ruta":"x"}`,
			ok:       true,
		},
		{
			nombre:   "llaves dobladas",
			texto:    `{{"name":"leer_archivo","arguments":{"ruta":"x"}}}`,
			esperado: `{"ruta":"x"}`,
			ok:       true,
		},
		{
			nombre:   "envuelta en function",
			texto:    `{"function":{"name":"leer_archivo","arguments":{"ruta":"x"}}}`,
			esperado: `{"ruta":"x"}`,
			ok:       true,
		},
		{
			nombre:   "argumentos como cadena",
			texto:    `{"name":"leer_archivo","arguments":"{\"ruta\":\"x\"}"}`,
			esperado: `{"ruta":"x"}`,
			ok:       true,
		},
		{
			nombre:   "sin argumentos",
			texto:    `{"name":"ejecutar_comando"}`,
			esperado: `{}`,
			ok:       true,
		},
		{
			nombre:   "nombre con espacios",
			texto:    `{"name":"  leer_archivo  ","arguments":{"ruta":"x"}}`,
			esperado: `{"ruta":"x"}`,
			ok:       true,
		},
		{
			nombre: "herramienta no concedida",
			texto:  `{"name":"crear_archivo","arguments":{"ruta":"x"}}`,
		},
		{
			nombre: "prosa antes del json",
			texto:  "Voy a leer el archivo:\n```json\n{\"name\":\"leer_archivo\",\"arguments\":{\"ruta\":\"x\"}}\n```",
		},
		{
			nombre: "json de ejemplo dentro de una explicación",
			texto:  "El formato es:\n```json\n{\"name\":\"leer_archivo\",\"arguments\":{\"ruta\":\"x\"}}\n```\nY se ejecuta así.",
		},
		{
			nombre: "dos objetos con prosa en medio",
			texto:  "Primero listo:\n```json\n{\"name\":\"leer_archivo\",\"arguments\":{}}\n```\nDespués leo:\n```json\n{\"name\":\"leer_archivo\",\"arguments\":{}}\n```",
		},
		{
			nombre: "json roto",
			texto:  `{"name":"leer_archivo","arguments":`,
		},
		{
			nombre: "argumentos que no son objeto",
			texto:  `{"name":"leer_archivo","arguments":["ruta"]}`,
		},
		{
			nombre: "cadena de argumentos que no es json",
			texto:  `{"name":"leer_archivo","arguments":"ruta"}`,
		},
		{
			nombre: "vacío",
			texto:  "   ",
		},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			llamadas := extraerLlamadasDeTexto(c.texto, permitidas)
			if ok := len(llamadas) > 0; ok != c.ok {
				t.Fatalf("ok = %v, se esperaba %v (llamadas %+v)", ok, c.ok, llamadas)
			}
			if !c.ok {
				return
			}
			if len(llamadas) != 1 {
				t.Fatalf("se esperaba una llamada, llegaron %d: %+v", len(llamadas), llamadas)
			}
			llamada := llamadas[0]
			if llamada.Nombre == "" {
				t.Fatal("la llamada debería llevar nombre")
			}
			if compacto(t, llamada.Argumentos) != compacto(t, c.esperado) {
				t.Fatalf("argumentos = %s, se esperaba %s", llamada.Argumentos, c.esperado)
			}
		})
	}
}

func TestExtraerVariasLlamadasDeTexto(t *testing.T) {
	permitidas := []Herramienta{
		{Nombre: "listar_carpeta"},
		{Nombre: "leer_archivo"},
	}

	casos := []struct {
		nombre string
		texto  string
	}{
		{
			nombre: "dos cercas consecutivas",
			texto:  "```json\n{\n  \"name\": \"listar_carpeta\",\n  \"arguments\": {\"ruta\": \".\"}\n}\n```\n```json\n{\n  \"name\": \"leer_archivo\",\n  \"arguments\": {\"ruta\": \"main.go\"}\n}\n```",
		},
		{
			nombre: "dos objetos desnudos",
			texto:  "{\"name\":\"listar_carpeta\",\"arguments\":{\"ruta\":\".\"}}\n{\"name\":\"leer_archivo\",\"arguments\":{\"ruta\":\"main.go\"}}",
		},
		{
			nombre: "dos objetos en una sola cerca",
			texto:  "```json\n{\"name\":\"listar_carpeta\",\"arguments\":{\"ruta\":\".\"}}\n{\"name\":\"leer_archivo\",\"arguments\":{\"ruta\":\"main.go\"}}\n```",
		},
		{
			nombre: "dos etiquetas de la plantilla",
			texto:  "<tool_call>{\"name\":\"listar_carpeta\",\"arguments\":{\"ruta\":\".\"}}</tool_call>\n<tool_call>{\"name\":\"leer_archivo\",\"arguments\":{\"ruta\":\"main.go\"}}</tool_call>",
		},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			llamadas := extraerLlamadasDeTexto(c.texto, permitidas)
			if len(llamadas) != 2 {
				t.Fatalf("se esperaban dos llamadas, llegaron %d: %+v", len(llamadas), llamadas)
			}
			if llamadas[0].Nombre != "listar_carpeta" || compacto(t, llamadas[0].Argumentos) != `{"ruta":"."}` {
				t.Fatalf("primera llamada = %+v", llamadas[0])
			}
			if llamadas[1].Nombre != "leer_archivo" || compacto(t, llamadas[1].Argumentos) != `{"ruta":"main.go"}` {
				t.Fatalf("segunda llamada = %+v", llamadas[1])
			}
		})
	}
}

func TestExtraerVariasLlamadasRechazaLaSecuenciaEntera(t *testing.T) {
	permitidas := []Herramienta{
		{Nombre: "listar_carpeta"},
		{Nombre: "leer_archivo"},
	}
	texto := "```json\n{\"name\":\"listar_carpeta\",\"arguments\":{\"ruta\":\".\"}}\n```\n```json\n{\"name\":\"crear_archivo\",\"arguments\":{\"ruta\":\"x\"}}\n```"
	if llamadas := extraerLlamadasDeTexto(texto, permitidas); llamadas != nil {
		t.Fatalf("si una llamada no está concedida no se ejecuta ninguna: %+v", llamadas)
	}
}

func TestExtraerLlamadaDeTextoIgnoraElCatalogoVacio(t *testing.T) {
	if llamadas := extraerLlamadasDeTexto(`{"name":"leer_archivo","arguments":{}}`, nil); llamadas != nil {
		t.Fatal("sin catálogo no se puede ejecutar ninguna llamada")
	}
}

func compacto(t *testing.T, texto string) string {
	t.Helper()
	var salida bytes.Buffer
	if err := json.Compact(&salida, []byte(texto)); err != nil {
		t.Fatalf("json inválido en la prueba: %s (%v)", texto, err)
	}
	return salida.String()
}
