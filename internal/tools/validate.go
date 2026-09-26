package tools

// validate.go — T-B007-08: validar el payload de cada request antes de
// enrutar.
//
// Fuente de verdad: ai/docs/backend/02-interfaces/dto/TOOLS-DTO.md §2 (campo,
// obligatoriedad) y ai/docs/backend/05-quality/VALIDATION.md §1 y §3
// ("Los argumentos se validan contra el contrato de la herramienta antes de
// aplicarse"; "Una herramienta sin su campo obligatorio no se ejecuta").
//
// La validación no decide si una operación es segura —eso es de `fileops` y
// `exec`— ni comprueba permisos. Solo verifica que el payload encaja con el
// contrato: campos obligatorios presentes y sin campos que no existen. Un
// payload incompleto es E_BAD_ARGS y no llega al handler.

import (
	"bytes"
	"encoding/json"
	"io"
	"reflect"
	"strings"
)

// Validar comprueba que el valor recibido es del tipo que corresponde a la
// herramienta y que sus campos obligatorios están presentes.
func Validar(nombre string, peticion any) error {
	esperado, ok := NuevaPeticion(nombre)
	if !ok {
		return nuevoError(CodigoHerramientaDesconocida,
			"la herramienta "+nombre+" no está en el catálogo cerrado")
	}
	// El tipo tiene que ser exactamente el del contrato: mezclar una petición
	// con otra herramienta daría una validación que no valida nada.
	if reflect.TypeOf(esperado) != reflect.TypeOf(peticion) {
		return nuevoError(CodigoArgumentosInvalidos,
			"los argumentos de "+nombre+" no son de su tipo de contrato")
	}

	switch p := peticion.(type) {
	case *PeticionLeerArchivo:
		return requerido(nombre, "ruta", p.Ruta)
	case *PeticionListarCarpeta:
		return requerido(nombre, "ruta", p.Ruta)
	case *PeticionBuscarArchivos:
		return requerido(nombre, "patron", p.Patron)
	case *PeticionBuscarEnArchivos:
		return requerido(nombre, "patron", p.Patron)
	case *PeticionCrearArchivo:
		if err := requerido(nombre, "ruta", p.Ruta); err != nil {
			return err
		}
		return requerido(nombre, "contenido", p.Contenido)
	case *PeticionEscribirArchivo:
		if err := requerido(nombre, "ruta", p.Ruta); err != nil {
			return err
		}
		return requerido(nombre, "contenido", p.Contenido)
	case *PeticionEditarArchivo:
		if err := requerido(nombre, "ruta", p.Ruta); err != nil {
			return err
		}
		return requerido(nombre, "cambio", p.Cambio)
	case *PeticionEliminarArchivo:
		return requerido(nombre, "ruta", p.Ruta)
	case *PeticionCrearCarpeta:
		return requerido(nombre, "ruta", p.Ruta)
	case *PeticionEliminarCarpeta:
		return requerido(nombre, "ruta", p.Ruta)
	case *PeticionEjecutarComando:
		return requerido(nombre, "comando", p.Comando)
	case *PeticionBuscarInternet:
		return requerido(nombre, "consulta", p.Consulta)
	case *PeticionAbrirPagina:
		return requerido(nombre, "direccion", p.Direccion)
	}
	return nuevoError(CodigoArgumentosInvalidos,
		"los argumentos de "+nombre+" no encajan con su contrato")
}

// Decodificar convierte el texto que produjo el modelo en el tipo de
// argumentos de la herramienta, rechazando campos desconocidos, y valida el
// resultado. Es el punto de entrada de la frontera: nada se enruta sin pasar
// por aquí.
func Decodificar(nombre string, datos []byte) (any, error) {
	v, ok := NuevaPeticion(nombre)
	if !ok {
		return nil, nuevoError(CodigoHerramientaDesconocida,
			"la herramienta "+nombre+" no está en el catálogo cerrado")
	}
	dec := json.NewDecoder(bytes.NewReader(datos))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return nil, nuevoError(CodigoArgumentosInvalidos,
			"los argumentos de "+nombre+" no encajan con su contrato: "+err.Error())
	}
	// Datos sobrantes tras el objeto también son un contrato roto.
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return nil, nuevoError(CodigoArgumentosInvalidos,
			"los argumentos de "+nombre+" llevan contenido de más")
	}
	if err := Validar(nombre, v); err != nil {
		return nil, err
	}
	return v, nil
}

// requerido comprueba que un campo obligatorio no esté vacío. Un valor en
// blanco es una omisión disfrazada.
func requerido(nombre, campo, valor string) error {
	if strings.TrimSpace(valor) == "" {
		return nuevoError(CodigoArgumentosInvalidos,
			"a "+nombre+" le falta el campo obligatorio `"+campo+"`")
	}
	return nil
}
