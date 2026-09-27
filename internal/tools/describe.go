package tools

// describe.go — describir el catálogo para el prompt de sistema del agente.
//
// Fuente de verdad: ai/docs/specs/SPEC-TOOLS.md (el agente responde con las
// herramientas de su catálogo) y SPEC-AGENTE-BASE.md (el agente recibe solo su
// contexto y sus herramientas).
//
// El modelo local no tiene function-calling nativo: hay que decirle, en el
// mensaje de sistema, QUÉ herramientas tiene y CÓMO pedirlas. El esquema de
// argumentos no se escribe a mano —eso se quedaría obsoleto en cuanto cambie
// un DTO—: se DEDUCE por reflexión del contrato real de `dto_request.go`.

import (
	"reflect"
	"sort"
	"strings"
)

// Campo es un argumento del contrato de una herramienta, tal como aparece en
// su DTO.
type Campo struct {
	Nombre      string
	Obligatorio bool
}

// CamposDe devuelve los argumentos del contrato de una herramienta, en el
// orden en que están declarados en su DTO. La obligatoriedad sale del tag
// `json`: un campo con `,omitempty` es opcional. El segundo valor es false si
// la herramienta no existe.
func CamposDe(nombre string) ([]Campo, bool) {
	v, ok := NuevaPeticion(nombre)
	if !ok {
		return nil, false
	}
	t := reflect.TypeOf(v).Elem() // *PeticionX → PeticionX
	campos := make([]Campo, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := f.Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		partes := strings.Split(tag, ",")
		opcional := false
		for _, p := range partes[1:] {
			if p == "omitempty" {
				opcional = true
			}
		}
		campos = append(campos, Campo{Nombre: partes[0], Obligatorio: !opcional})
	}
	return campos, true
}

// ArgumentosDe arma la línea de argumentos de una herramienta, o "" si no
// declara ninguno.
func ArgumentosDe(nombre string) string {
	campos, ok := CamposDe(nombre)
	if !ok || len(campos) == 0 {
		return ""
	}
	partes := make([]string, 0, len(campos))
	for _, c := range campos {
		if c.Obligatorio {
			partes = append(partes, c.Nombre+" (obligatorio)")
			continue
		}
		partes = append(partes, c.Nombre+" (opcional)")
	}
	return strings.Join(partes, ", ")
}

// CatalogoTexto describe las herramientas indicadas para el mensaje de
// sistema: nombre, para qué sirve y qué argumentos acepta. Solo se incluyen
// los nombres que existen en el catálogo cerrado, en orden alfabético para que
// la salida sea estable.
func CatalogoTexto(nombres []string) string {
	unicos := map[string]bool{}
	for _, n := range nombres {
		if Existe(n) {
			unicos[n] = true
		}
	}
	orden := make([]string, 0, len(unicos))
	for n := range unicos {
		orden = append(orden, n)
	}
	sort.Strings(orden)

	var b strings.Builder
	for _, n := range orden {
		h, _ := Buscar(n)
		b.WriteString("- " + h.Nombre + " — " + h.Descripcion)
		if args := ArgumentosDe(n); args != "" {
			b.WriteString(" Argumentos: " + args + ".")
		}
		b.WriteString("\n")
	}
	return b.String()
}
