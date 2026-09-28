package tools

// esquema.go — T-B024-02: el JSON Schema se deriva del tipo de petición por
// reflexión.
//
// Fuente de verdad: ai/docs/backend/02-interfaces/TOOLS.md §7 ("El esquema se
// deriva, no se escribe") y ai/docs/backend/02-interfaces/dto/TOOLS-DTO.md §2
// (reglas de derivación: nombre del tag `json`, tipo del campo, obligatorio si
// falta `omitempty`, descripción del tag `desc`).
//
// No hay un esquema escrito a mano al lado de los DTO: si los dos existieran,
// divergirían sin que nada lo notase. La descripción de cada campo se escribe
// para el modelo —dice cuándo usar el argumento—, no para quien lee el código.

import (
	"encoding/json"
	"reflect"
	"strings"
)

// Propiedad es un campo del esquema. Se anida para los tipos compuestos
// (`[]Cambio`, un objeto), de modo que un tipo anidado exponga su esquema
// entero y no un `object` vacío.
type Propiedad struct {
	Type        string               `json:"type"`
	Description string               `json:"description,omitempty"`
	Items       *Propiedad           `json:"items,omitempty"`
	Properties  map[string]Propiedad `json:"properties,omitempty"`
	Required    []string             `json:"required,omitempty"`
}

// Esquema es el JSON Schema que ve el modelo para los argumentos de una
// herramienta.
type Esquema struct {
	Type       string               `json:"type"`
	Properties map[string]Propiedad `json:"properties,omitempty"`
	Required   []string             `json:"required,omitempty"`
}

// tipoJSON traduce un tipo de Go al nombre JSON Schema.
func tipoJSON(t reflect.Type) (string, bool) {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.String:
		return "string", true
	case reflect.Bool:
		return "boolean", true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return "number", true
	case reflect.Slice, reflect.Array:
		return "array", true
	case reflect.Map, reflect.Struct, reflect.Interface:
		return "object", true
	}
	return "string", false
}

// propiedadDe arma la propiedad de un tipo, descendiendo en los compuestos.
func propiedadDe(t reflect.Type) Propiedad {
	nombre, _ := tipoJSON(t)
	p := Propiedad{Type: nombre}
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Slice, reflect.Array:
		item := propiedadDe(t.Elem())
		p.Items = &item
	case reflect.Map:
		// Un mapa es un objeto libre: sus claves no se conocen de antemano.
	case reflect.Struct:
		if t == reflect.TypeOf(json.RawMessage(nil)) {
			// json.RawMessage es []byte: un objeto libre, no una cadena.
			p.Type = "object"
			break
		}
		props, req := propiedadesDe(t)
		p.Properties = props
		p.Required = req
	}
	return p
}

// propiedadesDe recorre los campos exportados de un struct y arma el mapa de
// propiedades y la lista de obligatorios.
func propiedadesDe(t reflect.Type) (map[string]Propiedad, []string) {
	props := map[string]Propiedad{}
	var req []string
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" { // no exportado
			continue
		}
		tag := f.Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		partes := strings.Split(tag, ",")
		nombre := partes[0]
		if nombre == "" {
			continue
		}
		opcional := false
		for _, p := range partes[1:] {
			if p == "omitempty" {
				opcional = true
			}
		}
		prop := propiedadDe(f.Type)
		prop.Description = f.Tag.Get("desc")
		props[nombre] = prop
		if !opcional {
			req = append(req, nombre)
		}
	}
	return props, req
}

// EsquemaDe deriva el JSON Schema del tipo de petición de una herramienta. Se
// le pasa el valor que devuelve NuevaPeticion (un puntero a struct).
func EsquemaDe(v any) *Esquema {
	if v == nil {
		return &Esquema{Type: "object"}
	}
	t := reflect.TypeOf(v)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		// Una petición genérica (herramienta del usuario) acepta cualquier
		// objeto: no hay campos que describir.
		return &Esquema{Type: "object"}
	}
	props, req := propiedadesDe(t)
	return &Esquema{Type: "object", Properties: props, Required: req}
}

// EsquemaObjeto es el esquema de un argumento genérico: un objeto libre, sin
// campos conocidos. Es el de una herramienta del usuario, cuyo ejecutable y
// contrato LocalCli no conoce (TOOLS-DTO.md §3).
func EsquemaObjeto() *Esquema { return &Esquema{Type: "object"} }
