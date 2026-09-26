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
// es la clave: es exactamente lo que el modelo escribe al pedirla, y lo que
// el JSON del agente declara en `herramientas`.
//
// El reparto plan/build se apoya en UN solo dato por herramienta, `Modo`: las
// que escriben son "Solo build" y las que leen son "Ambos" (columna "Agente"
// de TOOLS.md §1). No hay un segundo campo "soloBuild" que pueda contradecir
// al primero y debilitar la garantía. La garantía en sí no vive aquí, vive en
// el JSON del agente: `plan` no lista ninguna herramienta de escritura, así
// que sencillamente no tiene cómo escribir (SECURITY.md §2, DECISIONS.md).

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

// Herramienta es una entrada del catálogo cerrado.
type Herramienta struct {
	Nombre    string    // nombre exacto con el que el modelo la pide
	Categoria Categoria // destino del enrutado
	Modo      Modo      // lee o escribe: de aquí sale el reparto por agente
}

// SoloBuild informa si la herramienta es de escritura y por tanto solo
// pertenece a `build` (columna "Agente" de TOOLS.md §1).
func (h Herramienta) SoloBuild() bool { return h.Modo == Escribe }

// catalogo es la lista cerrada, en el orden documentado: archivos de lectura,
// archivos de escritura, terminal e internet.
//
// `crear_archivo` y `escribir_archivo` están separadas a propósito: el agente
// no destruye algo por accidente cuando pretendía crear (TOOLS.md §1).
var catalogo = []Herramienta{
	// Archivos de lectura — ambos agentes.
	{Nombre: "leer_archivo", Categoria: CatArchivos, Modo: Lee},
	{Nombre: "listar_carpeta", Categoria: CatArchivos, Modo: Lee},
	{Nombre: "buscar_archivos", Categoria: CatArchivos, Modo: Lee},
	{Nombre: "buscar_en_archivos", Categoria: CatArchivos, Modo: Lee},

	// Archivos de escritura — solo `build`.
	{Nombre: "crear_archivo", Categoria: CatArchivos, Modo: Escribe},
	{Nombre: "escribir_archivo", Categoria: CatArchivos, Modo: Escribe},
	{Nombre: "editar_archivo", Categoria: CatArchivos, Modo: Escribe},
	{Nombre: "eliminar_archivo", Categoria: CatArchivos, Modo: Escribe},
	{Nombre: "crear_carpeta", Categoria: CatArchivos, Modo: Escribe},
	{Nombre: "eliminar_carpeta", Categoria: CatArchivos, Modo: Escribe},

	// Terminal — ambos agentes.
	{Nombre: "ejecutar_comando", Categoria: CatTerminal, Modo: Lee},

	// Internet — ambos agentes.
	{Nombre: "buscar_en_internet", Categoria: CatInternet, Modo: Lee},
	{Nombre: "abrir_pagina", Categoria: CatInternet, Modo: Lee},
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

// NombresCatalogo devuelve los trece nombres, en orden. Es lo que necesita
// `agent` para validar el campo `herramientas` de un JSON (DECISIONS.md: "La
// carga valida contra el catálogo cerrado y rechaza una herramienta
// inexistente").
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
