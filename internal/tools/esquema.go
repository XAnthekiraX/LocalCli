package tools

import (
	"reflect"
	"strings"
)

// Esquema es el JSON Schema que ve el modelo. Se deriva por reflexión del tipo de
// la petición: el nombre del campo es el tag `json`, la descripción el `desc`,
// `enum` declara el vocabulario cerrado y `omitempty` marca el campo como
// opcional.
type Esquema struct {
	Tipo        string               `json:"type"`
	Propiedades map[string]Propiedad `json:"properties,omitempty"`
	Requeridos  []string             `json:"required,omitempty"`
}

type Propiedad struct {
	Tipo        string               `json:"type"`
	Enum        []string             `json:"enum,omitempty"`
	Descripcion string               `json:"description,omitempty"`
	Items       *Propiedad           `json:"items,omitempty"`
	Propiedades map[string]Propiedad `json:"properties,omitempty"`
	Requeridos  []string             `json:"required,omitempty"`
}

// Derivar construye el esquema de un tipo de petición. Un tipo nulo o desconocido
// cae en un objeto genérico.
func Derivar(t reflect.Type) *Esquema {
	for t != nil && t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t == nil || t.Kind() != reflect.Struct {
		return &Esquema{Tipo: "object"}
	}
	propiedades, requeridos := propiedadesDe(t)
	return &Esquema{Tipo: "object", Propiedades: propiedades, Requeridos: requeridos}
}

func propiedadesDe(t reflect.Type) (map[string]Propiedad, []string) {
	propiedades := map[string]Propiedad{}
	var requeridos []string
	for i := 0; i < t.NumField(); i++ {
		campo := t.Field(i)
		if campo.PkgPath != "" {
			continue
		}
		nombre, opcional := nombreDeCampo(campo)
		if nombre == "" {
			continue
		}
		propiedad := propiedadDe(campo.Type, campo.Tag.Get("desc"))
		if valores := enumeracionDe(campo); len(valores) > 0 {
			propiedad.Enum = valores
		}
		propiedades[nombre] = propiedad
		if !opcional {
			requeridos = append(requeridos, nombre)
		}
	}
	return propiedades, requeridos
}

func nombreDeCampo(campo reflect.StructField) (string, bool) {
	etiqueta := campo.Tag.Get("json")
	if etiqueta == "-" {
		return "", false
	}
	partes := strings.Split(etiqueta, ",")
	nombre := partes[0]
	if nombre == "" {
		nombre = campo.Name
	}
	opcional := false
	for _, p := range partes[1:] {
		if p == "omitempty" {
			opcional = true
		}
	}
	return nombre, opcional
}

// enumeracionDe lee el vocabulario cerrado declarado en la etiqueta `enum`:
// `enum:"pendiente,en_progreso,..."`.
func enumeracionDe(campo reflect.StructField) []string {
	etiqueta := strings.TrimSpace(campo.Tag.Get("enum"))
	if etiqueta == "" {
		return nil
	}
	partes := strings.Split(etiqueta, ",")
	valores := make([]string, 0, len(partes))
	for _, p := range partes {
		if valor := strings.TrimSpace(p); valor != "" {
			valores = append(valores, valor)
		}
	}
	return valores
}

func propiedadDe(t reflect.Type, descripcion string) Propiedad {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	propiedad := Propiedad{Descripcion: descripcion}
	switch t.Kind() {
	case reflect.String:
		propiedad.Tipo = "string"
	case reflect.Bool:
		propiedad.Tipo = "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		propiedad.Tipo = "integer"
	case reflect.Float32, reflect.Float64:
		propiedad.Tipo = "number"
	case reflect.Slice, reflect.Array:
		propiedad.Tipo = "array"
		elemento := propiedadDe(t.Elem(), "")
		propiedad.Items = &elemento
	case reflect.Struct:
		propiedad.Tipo = "object"
		propiedad.Propiedades, propiedad.Requeridos = propiedadesDe(t)
	case reflect.Map, reflect.Interface:
		propiedad.Tipo = "object"
	default:
		propiedad.Tipo = "object"
	}
	return propiedad
}
