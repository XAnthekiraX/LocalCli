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
			nombre: "dos objetos",
			texto:  `{"name":"leer_archivo","arguments":{}}{"name":"leer_archivo","arguments":{}}`,
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
			llamada, ok := extraerLlamadaDeTexto(c.texto, permitidas)
			if ok != c.ok {
				t.Fatalf("ok = %v, se esperaba %v (llamada %+v)", ok, c.ok, llamada)
			}
			if !ok {
				return
			}
			if llamada.Nombre == "" {
				t.Fatal("la llamada debería llevar nombre")
			}
			if compacto(t, llamada.Argumentos) != compacto(t, c.esperado) {
				t.Fatalf("argumentos = %s, se esperaba %s", llamada.Argumentos, c.esperado)
			}
		})
	}
}

func TestExtraerLlamadaDeTextoIgnoraElCatalogoVacio(t *testing.T) {
	if _, ok := extraerLlamadaDeTexto(`{"name":"leer_archivo","arguments":{}}`, nil); ok {
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
