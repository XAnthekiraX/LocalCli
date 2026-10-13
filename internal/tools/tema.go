package tools

import (
	"reflect"
	"strings"
)

// TemaVisible extrae el objetivo declarado por el catálogo —el campo del
// argumento cuyo nombre es `h.Tema`— y lo deja colapsado y recortado para la
// línea del chat. El resto de los argumentos no se expone.
func TemaVisible(h Herramienta, args any) string {
	if h.Tema == "" {
		return ""
	}
	return colapsar(campoDeTexto(args, h.Tema))
}

// TemaFinal añade el rango real de una lectura parcial. El handler deja el rango
// en `Meta["rango"]` (por ejemplo `1-70`) cuando no leyó el archivo entero.
func TemaFinal(h Herramienta, tema string, meta map[string]any) string {
	if tema == "" || meta == nil {
		return tema
	}
	if rango, ok := meta["rango"].(string); ok && rango != "" {
		return tema + ":" + rango
	}
	return tema
}

func colapsar(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	const tope = 120
	if len(s) > tope {
		return s[:tope] + "…"
	}
	return s
}

// campoDeTexto lee el valor de texto del campo `nombre` (su tag `json`) de los
// argumentos. Una herramienta del usuario, cuyo argumento es un objeto genérico,
// se lee como mapa.
func campoDeTexto(args any, nombre string) string {
	if args == nil {
		return ""
	}
	if mapa, ok := args.(map[string]any); ok {
		if v, ok := mapa[nombre].(string); ok {
			return v
		}
		return ""
	}
	v := reflect.ValueOf(args)
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return ""
	}
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		if nombreDeTag(t.Field(i)) != nombre {
			continue
		}
		if v.Field(i).Kind() == reflect.String {
			return v.Field(i).String()
		}
	}
	return ""
}

func nombreDeTag(campo reflect.StructField) string {
	etiqueta := campo.Tag.Get("json")
	if etiqueta == "" {
		return campo.Name
	}
	return strings.Split(etiqueta, ",")[0]
}
