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
	"fmt"
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
	case *PeticionCrearTodo:
		return ValidarCrearTodo(p, nil)
	case *PeticionActualizarTodo:
		return validarTodo(nombre, p)
	}
	return nuevoError(CodigoArgumentosInvalidos,
		"los argumentos de "+nombre+" no encajan con su contrato")
}

// Vocabularios de la lista de pasos de la sesión (SPEC-TOOLS).
var (
	estadosDeTodo = map[string]bool{
		"pendiente": true, "en_progreso": true, "completada": true, "cancelada": true,
	}
	prioridadesDeTodo = map[string]bool{"alta": true, "media": true, "baja": true}
)

// validarTodo comprueba el vocabulario de la lista. Una lista vacía es válida:
// es como se deja la lista en blanco. Un estado o una prioridad fuera del
// vocabulario es E_BAD_ARGS y vuelve al modelo para que corrija.
func validarTodo(nombre string, p *PeticionActualizarTodo) error {
	for i, e := range p.Elementos {
		if err := validarPaso(nombre, i, e.Contenido, e.Estado, e.Prioridad); err != nil {
			return err
		}
	}
	return nil
}

// validarPaso comprueba un paso contra el vocabulario compartido por las dos
// herramientas de la lista: contenido no vacío, estado y prioridad dentro de su
// vocabulario. Es la fuente única del vocabulario, para que `crear_todo` y
// `actualizar_todo` no puedan divergir.
func validarPaso(nombre string, i int, contenido, estado, prioridad string) error {
	if strings.TrimSpace(contenido) == "" {
		return nuevoError(CodigoArgumentosInvalidos,
			fmt.Sprintf("el elemento %d de %s no tiene `contenido`", i, nombre))
	}
	if !estadosDeTodo[estado] {
		return nuevoError(CodigoArgumentosInvalidos,
			fmt.Sprintf("el estado %q del elemento %d de %s no es válido: usa pendiente, en_progreso, completada o cancelada", estado, i, nombre))
	}
	if prioridad != "" && !prioridadesDeTodo[prioridad] {
		return nuevoError(CodigoArgumentosInvalidos,
			fmt.Sprintf("la prioridad %q del elemento %d de %s no es válida: usa alta, media o baja", prioridad, i, nombre))
	}
	return nil
}

// ValidarCrearTodo comprueba el paso de `crear_todo` contra el vocabulario de la
// lista y, cuando `existentes` no es nil, contra la regla de un solo
// `en_progreso`: añadir un segundo paso en curso es E_BAD_ARGS corregible que
// nombra `actualizar_todo` como la vía para cambiarlo. Con `existentes` nil solo
// se valida el vocabulario (es lo que corre durante la decodificación).
func ValidarCrearTodo(p *PeticionCrearTodo, existentes []ElementoTodo) error {
	if err := validarPaso("crear_todo", 0, p.Contenido, p.Estado, p.Prioridad); err != nil {
		return err
	}
	if p.Estado != "en_progreso" {
		return nil
	}
	for _, e := range existentes {
		if e.Estado == "en_progreso" {
			return nuevoError(CodigoArgumentosInvalidos,
				"ya hay un paso `en_progreso`: usa `actualizar_todo` para cambiar cuál está en curso antes de añadir otro")
		}
	}
	return nil
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

// DecodificarUsuario acepta el objeto genérico de una herramienta del usuario.
// El harness no conoce el ejecutable, así que no puede validar campos: solo
// comprueba que lo que llega es un objeto JSON (VALIDATION.md §1).
func DecodificarUsuario(datos []byte) (any, error) {
	var objeto map[string]any
	dec := json.NewDecoder(bytes.NewReader(datos))
	if err := dec.Decode(&objeto); err != nil {
		return nil, nuevoError(CodigoArgumentosInvalidos,
			"los argumentos de una herramienta del usuario deben ser un objeto JSON: "+err.Error())
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return nil, nuevoError(CodigoArgumentosInvalidos,
			"los argumentos de una herramienta del usuario llevan contenido de más")
	}
	if objeto == nil {
		return map[string]any{}, nil
	}
	return objeto, nil
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
