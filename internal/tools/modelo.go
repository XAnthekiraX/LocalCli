// Package tools define el contrato de las herramientas, el catálogo cerrado y la
// capa universal por la que pasa toda ejecución. No importa a los módulos que las
// implementan: sus handlers se inyectan al cablear.
package tools

import (
	"context"
	"reflect"
)

// Categoria agrupa las herramientas para la interfaz.
type Categoria string

const (
	CategoriaArchivos Categoria = "archivos"
	CategoriaTerminal Categoria = "terminal"
	CategoriaInternet Categoria = "internet"
	CategoriaTareas   Categoria = "tareas"
	CategoriaFlujos   Categoria = "flujos"
	CategoriaUsuario  Categoria = "usuario"
)

// Permiso es lo que hay que conceder para usar una herramienta.
type Permiso string

const (
	PermisoRead  Permiso = "read"
	PermisoWrite Permiso = "write"
	PermisoEdit  Permiso = "edit"
	// PermisoDerivado solo lo usa ejecutar_flujo: se resuelve del flujo pedido.
	PermisoDerivado Permiso = "derivado"
)

// Modo distingue una herramienta incluida de una del usuario.
type Modo string

const (
	ModoLee     Modo = "lee"
	ModoEscribe Modo = "escribe"
)

// Unidad es cómo se mide el resultado de una herramienta.
type Unidad string

const (
	UnidadLinea        Unidad = "linea"
	UnidadEntrada      Unidad = "entrada"
	UnidadCoincidencia Unidad = "coincidencia"
	UnidadResultado    Unidad = "resultado"
	UnidadPaso         Unidad = "paso"
	UnidadEtapa        Unidad = "etapa"
)

// Decision resuelve una petición de aprobación.
type Decision int

const (
	Aprobar Decision = iota + 1
	Declinar
)

// Solicitud es lo que se le muestra al usuario para que decida.
type Solicitud struct {
	Titulo      string
	Descripcion string
	Motivo      string
	// Peligrosa exige confirmación explícita (borrados).
	Peligrosa bool
}

// Contexto es lo que recibe el handler de una herramienta.
type Contexto struct {
	Ctx      context.Context
	SesionID string
	Agente   string
	Permisos []string
	Ask      func(ctx context.Context, s Solicitud) (Decision, error)
	Meta     func(clave string, valor any)
}

// Resultado es lo que devuelve un handler. `Error` es corregible: el modelo lo lee
// y puede reintentar; un `error` de Go corta el turno.
type Resultado struct {
	Salida   string
	Meta     map[string]any
	Truncado bool
	Error    string
}

// Ejecutar es el handler propio de una herramienta.
type Ejecutar func(ctx context.Context, args any, c Contexto) (Resultado, error)

// Herramienta es una entrada del catálogo. `TipoPeticion` es la fuente del esquema.
type Herramienta struct {
	Nombre       string
	Descripcion  string
	Categoria    Categoria
	Permiso      Permiso
	Modo         Modo
	Verbo        string
	Tema         string
	Unidad       Unidad
	TipoPeticion reflect.Type
	Ejecutar     Ejecutar
}

// Esquema deriva el JSON Schema del tipo de petición.
func (h Herramienta) Esquema() *Esquema { return Derivar(h.TipoPeticion) }
