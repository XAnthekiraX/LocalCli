package tools

// catalog.go — T-B007-01: el catálogo cerrado de las trece herramientas.
//
// Fuente de verdad: ai/docs/backend/02-interfaces/TOOLS.md §1 (las trece, su
// categoría, si leen o escriben y a qué agente pertenecen) y
// ai/docs/specs/SPEC-TOOLS.md §Catálogo (las mismas trece).
//
// "Cerrado" significa cerrado: el agente no puede pedir nada fuera de esta
// lista (TOOLS.md §1, BUSINESS_RULES.md §Herramientas y terminal: "El
// catálogo es cerrado: el agente no puede inventar herramientas"). El nombre
// es la clave: es exactamente lo que el modelo escribe al pedirla.
//
// El reparto por agente NO vive aquí. El catálogo solo describe cada
// herramienta (categoría, modo, descripción); quién puede pedirla se decide
// con los `permisos` del JSON del agente (SPEC-AGENTE-BASE). La `Accion` de
// cada herramienta se DEDUCE de su categoría y su modo: no hay un campo aparte
// que pueda contradecirlos y debilitar la garantía (SECURITY.md §2).

// Categoria es el destino del enrutado (TOOLS.md §7): archivos → fileops,
// terminal → exec, internet → el cliente de internet.
type Categoria int

const (
	CatArchivos Categoria = iota
	CatTerminal
	CatInternet
)

func (c Categoria) String() string {
	switch c {
	case CatArchivos:
		return "archivos"
	case CatTerminal:
		return "terminal"
	case CatInternet:
		return "internet"
	}
	return "desconocida"
}

// Modo dice si la herramienta lee o escribe.
type Modo int

const (
	Lee Modo = iota
	Escribe
)

func (m Modo) String() string {
	if m == Escribe {
		return "escribe"
	}
	return "lee"
}

// Accion es el permiso de alto nivel que gobierna un grupo de herramientas. Un
// agente declara permisos por acción (`permitir`/`denegar`) y de ahí se deriva
// su catálogo efectivo (SPEC-AGENTE-BASE). Las cuatro acciones agrupan las
// trece herramientas por lo que hacen, no por dónde viven.
type Accion string

const (
	// AccionLeer: mirar archivos del proyecto.
	AccionLeer Accion = "leer"
	// AccionEditar: crear, modificar o borrar archivos y carpetas.
	AccionEditar Accion = "editar"
	// AccionEjecutar: lanzar comandos de consulta en el proyecto.
	AccionEjecutar Accion = "ejecutar"
	// AccionInternet: salir de la máquina (con LOCALCLI_ALLOW_INTERNET).
	AccionInternet Accion = "internet"
)

func (a Accion) String() string { return string(a) }

// Valida informa si la acción es una de las cuatro del catálogo. Un permiso
// sobre una acción desconocida no se admite.
func (a Accion) Valida() bool {
	switch a {
	case AccionLeer, AccionEditar, AccionEjecutar, AccionInternet:
		return true
	}
	return false
}

// Herramienta es una entrada del catálogo cerrado.
type Herramienta struct {
	Nombre      string    // nombre exacto con el que el modelo la pide
	Categoria   Categoria // destino del enrutado
	Modo        Modo      // lee o escribe
	Descripcion string    // frase corta para el catálogo que ve el modelo
}

// SoloBuild informa si la herramienta es de escritura y por tanto solo
// pertenece a `build` (columna "Agente" de TOOLS.md §1).
func (h Herramienta) SoloBuild() bool { return h.Modo == Escribe }

// Accion deduce el permiso de alto nivel de la herramienta a partir de su
// categoría y su modo. Es la única fuente de la acción: un campo aparte podría
// contradecir a `Modo` o a `Categoria` y ahí se iría la garantía.
func (h Herramienta) Accion() Accion {
	switch h.Categoria {
	case CatTerminal:
		return AccionEjecutar
	case CatInternet:
		return AccionInternet
	}
	if h.Modo == Escribe {
		return AccionEditar
	}
	return AccionLeer
}

// catalogo es la lista cerrada, en el orden documentado: archivos de lectura,
// archivos de escritura, terminal e internet.
//
// `crear_archivo` y `escribir_archivo` están separadas a propósito: el agente
// no destruye algo por accidente cuando pretendía crear (TOOLS.md §1).
var catalogo = []Herramienta{
	// Archivos de lectura — acción `leer`.
	{Nombre: "leer_archivo", Categoria: CatArchivos, Modo: Lee, Descripcion: "Lee el contenido de un archivo del proyecto."},
	{Nombre: "listar_carpeta", Categoria: CatArchivos, Modo: Lee, Descripcion: "Lista las entradas de un nivel de una carpeta."},
	{Nombre: "buscar_archivos", Categoria: CatArchivos, Modo: Lee, Descripcion: "Busca archivos por nombre."},
	{Nombre: "buscar_en_archivos", Categoria: CatArchivos, Modo: Lee, Descripcion: "Busca texto dentro del contenido de los archivos."},

	// Archivos de escritura — acción `editar`.
	{Nombre: "crear_archivo", Categoria: CatArchivos, Modo: Escribe, Descripcion: "Crea un archivo nuevo; falla si ya existe."},
	{Nombre: "escribir_archivo", Categoria: CatArchivos, Modo: Escribe, Descripcion: "Sobrescribe el contenido entero de un archivo."},
	{Nombre: "editar_archivo", Categoria: CatArchivos, Modo: Escribe, Descripcion: "Aplica una edición parcial a un archivo."},
	{Nombre: "eliminar_archivo", Categoria: CatArchivos, Modo: Escribe, Descripcion: "Borra un archivo del proyecto."},
	{Nombre: "crear_carpeta", Categoria: CatArchivos, Modo: Escribe, Descripcion: "Crea una carpeta; falla si ya existe."},
	{Nombre: "eliminar_carpeta", Categoria: CatArchivos, Modo: Escribe, Descripcion: "Borra una carpeta del proyecto."},

	// Terminal — acción `ejecutar`.
	{Nombre: "ejecutar_comando", Categoria: CatTerminal, Modo: Lee, Descripcion: "Ejecuta un comando de la lista blanca dentro del proyecto."},

	// Internet — acción `internet`.
	{Nombre: "buscar_en_internet", Categoria: CatInternet, Modo: Lee, Descripcion: "Busca en internet; solo sale la consulta de la máquina."},
	{Nombre: "abrir_pagina", Categoria: CatInternet, Modo: Lee, Descripcion: "Descarga y devuelve el texto de una página."},
}

// Herramientas devuelve una copia del catálogo cerrado, en el orden
// documentado. Es una copia: quien la reciba no puede alterar el catálogo.
func Herramientas() []Herramienta {
	out := make([]Herramienta, len(catalogo))
	copy(out, catalogo)
	return out
}

// Buscar devuelve la herramienta con ese nombre exacto. El segundo valor es
// false si el nombre no está en el catálogo: una herramienta inventada no
// existe (TOOLS.md §1).
func Buscar(nombre string) (Herramienta, bool) {
	for _, h := range catalogo {
		if h.Nombre == nombre {
			return h, true
		}
	}
	return Herramienta{}, false
}

// Existe informa si el nombre está en el catálogo cerrado.
func Existe(nombre string) bool {
	_, ok := Buscar(nombre)
	return ok
}

// NombresCatalogo devuelve los trece nombres, en orden.
func NombresCatalogo() []string {
	out := make([]string, 0, len(catalogo))
	for _, h := range catalogo {
		out = append(out, h.Nombre)
	}
	return out
}

// NombresDeCategoria devuelve los nombres de una categoría, en orden. Lo usa
// el enrutado para saber a qué destino pertenece cada herramienta sin
// repetir la tabla.
func NombresDeCategoria(c Categoria) []string {
	var out []string
	for _, h := range catalogo {
		if h.Categoria == c {
			out = append(out, h.Nombre)
		}
	}
	return out
}

// NombresDeAccion devuelve los nombres de las herramientas de una acción, en
// el orden del catálogo. Es lo que usa `agent` para derivar el catálogo
// efectivo de un agente a partir de sus permisos.
func NombresDeAccion(a Accion) []string {
	var out []string
	for _, h := range catalogo {
		if h.Accion() == a {
			out = append(out, h.Nombre)
		}
	}
	return out
}

// AccionDe devuelve la acción de una herramienta del catálogo. El segundo
// valor es false si el nombre no existe.
func AccionDe(nombre string) (Accion, bool) {
	h, ok := Buscar(nombre)
	if !ok {
		return "", false
	}
	return h.Accion(), true
}
