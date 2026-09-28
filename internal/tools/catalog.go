package tools

// catalog.go — el catálogo cerrado de las catorce herramientas.
//
// Fuente de verdad: ai/docs/backend/02-interfaces/TOOLS.md §1 (las catorce, su
// categoría, si leen o escriben y a qué agente pertenecen) y
// ai/docs/specs/SPEC-TOOLS.md §Catálogo (las mismas catorce).
//
// "Cerrado" significa cerrado: el agente no puede pedir nada fuera de esta
// lista salvo lo que el usuario declare en `.localcli/tools/`. El nombre es la
// clave: es exactamente lo que el modelo escribe al pedirla.
//
// El reparto por agente NO vive aquí. El catálogo solo describe cada
// herramienta (categoría, modo, descripción, esquema); quién puede pedirla se
// decide con los `permisos` del JSON del agente (SPEC-AGENTE-BASE) y la `Accion`
// se DEDUCE de su categoría y su modo: no hay un campo aparte que pueda
// contradecirlos y debilitar la garantía (SECURITY.md §2).
//
// El `Ejecutar` de cada herramienta es su handler propio, no una categoría: se
// rellena en el cableado y una herramienta nueva no obliga a tocar un `switch`
// (TOOLS.md §7).

// Categoria dice qué es la herramienta y de dónde viene su implementación.
type Categoria int

const (
	CatArchivos Categoria = iota
	CatTerminal
	CatInternet
	// CatTareas: la lista de pasos de la sesión. No toca archivos del proyecto
	// —escribe estado de la sesión—, así que la tienen los dos agentes.
	CatTareas
	// CatUsuario: una herramienta declarada en `.localcli/tools/`. Su handler es
	// un adaptador fino sobre `exec` y siempre pide aprobación.
	CatUsuario
)

func (c Categoria) String() string {
	switch c {
	case CatArchivos:
		return "archivos"
	case CatTerminal:
		return "terminal"
	case CatInternet:
		return "internet"
	case CatTareas:
		return "tareas"
	case CatUsuario:
		return "usuario"
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
// su catálogo efectivo (SPEC-AGENTE-BASE).
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
	// AccionTareas: llevar la lista de pasos de la sesión. No escribe archivos
	// del proyecto, así que no entra en la garantía de escritura y la conceden
	// los dos agentes.
	AccionTareas Accion = "tareas"
)

func (a Accion) String() string { return string(a) }

// Valida informa si la acción es una de las del catálogo.
func (a Accion) Valida() bool {
	switch a {
	case AccionLeer, AccionEditar, AccionEjecutar, AccionInternet, AccionTareas:
		return true
	}
	return false
}

// Permitida informa si la acción está entre las concedidas.
func (a Accion) Permitida(permisos []Accion) bool {
	for _, p := range permisos {
		if p == a {
			return true
		}
	}
	return false
}

// Herramienta es una entrada del catálogo: su descripción (lo que ve el modelo)
// y su handler propio (lo que ejecuta LocalCli).
type Herramienta struct {
	Nombre      string
	Descripcion string
	Categoria   Categoria
	Modo        Modo
	// Verbo es la etiqueta corta con la que la TUI nombra la herramienta
	// («LEER», «EJEC»). No viaja al modelo: es presentación.
	Verbo string
	// Tema es el campo JSON cuyo valor es el objetivo que se muestra en la
	// línea («ruta», «patron», «comando»). Vacío = la herramienta no tiene un
	// objetivo único que enseñar.
	Tema string
	// Unidad es cómo se mide el resultado para la línea («línea», «entrada»,
	// «coincidencia»). Vacío = sin medida (p. ej. una escritura).
	Unidad string
	// Esquema es el JSON Schema de sus argumentos, derivado del DTO.
	Esquema *Esquema
	// Ejecutar es el handler propio. nil significa "sin implementar": la capa
	// universal lo dice en vez de reventar.
	Ejecutar Ejecutar
}

// SoloBuild informa si la herramienta modifica el proyecto y por tanto solo
// pertenece a `build` (columna "Agente" de TOOLS.md §1). Lo decide la acción
// `editar`, no el modo: la lista de pasos de la sesión (`tareas`) escribe estado
// de la sesión, no archivos, y la tienen los dos agentes.
func (h Herramienta) SoloBuild() bool { return h.Accion() == AccionEditar }

// Accion deduce el permiso de alto nivel de la herramienta a partir de su
// categoría y su modo. Es la única fuente de la acción.
func (h Herramienta) Accion() Accion {
	switch h.Categoria {
	case CatTerminal:
		return AccionEjecutar
	case CatInternet:
		return AccionInternet
	case CatTareas:
		return AccionTareas
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
	{Nombre: "leer_archivo", Categoria: CatArchivos, Modo: Lee, Verbo: "LEER", Tema: "ruta", Unidad: "línea", Descripcion: "Lee el contenido de un archivo del proyecto."},
	{Nombre: "listar_carpeta", Categoria: CatArchivos, Modo: Lee, Verbo: "LISTAR", Tema: "ruta", Unidad: "entrada", Descripcion: "Lista las entradas de un nivel de una carpeta."},
	{Nombre: "buscar_archivos", Categoria: CatArchivos, Modo: Lee, Verbo: "BUSCAR", Tema: "patron", Unidad: "coincidencia", Descripcion: "Busca archivos por nombre."},
	{Nombre: "buscar_en_archivos", Categoria: CatArchivos, Modo: Lee, Verbo: "GREP", Tema: "patron", Unidad: "coincidencia", Descripcion: "Busca texto dentro del contenido de los archivos."},

	// Archivos de escritura — acción `editar`.
	{Nombre: "crear_archivo", Categoria: CatArchivos, Modo: Escribe, Verbo: "CREAR", Tema: "ruta", Descripcion: "Crea un archivo nuevo; falla si ya existe."},
	{Nombre: "escribir_archivo", Categoria: CatArchivos, Modo: Escribe, Verbo: "ESCRIBIR", Tema: "ruta", Descripcion: "Sobrescribe el contenido entero de un archivo."},
	{Nombre: "editar_archivo", Categoria: CatArchivos, Modo: Escribe, Verbo: "EDITAR", Tema: "ruta", Descripcion: "Aplica una edición parcial a un archivo."},
	{Nombre: "eliminar_archivo", Categoria: CatArchivos, Modo: Escribe, Verbo: "BORRAR", Tema: "ruta", Descripcion: "Borra un archivo del proyecto."},
	{Nombre: "crear_carpeta", Categoria: CatArchivos, Modo: Escribe, Verbo: "MKDIR", Tema: "ruta", Descripcion: "Crea una carpeta; falla si ya existe."},
	{Nombre: "eliminar_carpeta", Categoria: CatArchivos, Modo: Escribe, Verbo: "RMDIR", Tema: "ruta", Descripcion: "Borra una carpeta del proyecto."},

	// Terminal — acción `ejecutar`.
	{Nombre: "ejecutar_comando", Categoria: CatTerminal, Modo: Lee, Verbo: "EJEC", Tema: "comando", Unidad: "línea", Descripcion: "Ejecuta un comando de la lista blanca dentro del proyecto."},

	// Internet — acción `internet`.
	{Nombre: "buscar_en_internet", Categoria: CatInternet, Modo: Lee, Verbo: "WEB", Tema: "consulta", Unidad: "resultado", Descripcion: "Busca en internet; solo sale la consulta de la máquina."},
	{Nombre: "abrir_pagina", Categoria: CatInternet, Modo: Lee, Verbo: "ABRIR", Tema: "direccion", Unidad: "línea", Descripcion: "Descarga y devuelve el texto de una página."},

	// Sesión — acción `tareas` (la tienen los dos agentes).
	{Nombre: "actualizar_todo", Categoria: CatTareas, Modo: Escribe, Verbo: "TODO", Unidad: "paso", Descripcion: descripcionActualizarTodo},
}

// descripcionActualizarTodo es la política que ve el modelo: no describe la
// implementación, dice cuándo usar la lista y cómo mantenerla (SPEC-TOOLS).
const descripcionActualizarTodo = "Crea y mantiene la lista de pasos de la sesión para trabajo de varios pasos. " +
	"Reemplaza la lista ENTERA: manda todos los elementos, no solo los que cambian. " +
	"Úsala cuando el trabajo tenga tres o más pasos distintos, cuando haya que coordinar varias tareas " +
	"o cuando el usuario te dé una lista de cosas que hacer. No la uses para un paso único y trivial ni para conversar. " +
	"Estados: `pendiente`, `en_progreso`, `completada` y `cancelada`. Solo un elemento puede estar `en_progreso` a la vez. " +
	"Marca `completada` únicamente cuando el paso, incluida su verificación, esté realmente hecho; nunca por intención. " +
	"Si algo bloquea un paso, déjalo en `en_progreso` y añade otro elemento que describa el bloqueo. " +
	"Prioridad opcional: `alta`, `media` o `baja`."

// El esquema de cada una de las catorce se deriva de su DTO al cargar el paquete:
// no hay un esquema escrito a mano al lado que pueda divergir.
func init() {
	for i := range catalogo {
		if v, ok := NuevaPeticion(catalogo[i].Nombre); ok {
			catalogo[i].Esquema = EsquemaDe(v)
		}
	}
}

// Herramientas devuelve una copia del catálogo cerrado, en el orden
// documentado. Es una copia: quien la reciba no puede alterar el catálogo.
func Herramientas() []Herramienta {
	out := make([]Herramienta, len(catalogo))
	copy(out, catalogo)
	return out
}

// NuevaHerramienta devuelve la entrada del catálogo cerrado con su handler ya
// conectado. El segundo valor es false si el nombre no está en el catálogo.
func NuevaHerramienta(nombre string, e Ejecutar) (Herramienta, bool) {
	h, ok := buscarCatalogo(nombre)
	if !ok {
		return Herramienta{}, false
	}
	h.Ejecutar = e
	return h, true
}

// Buscar devuelve la herramienta con ese nombre exacto del catálogo cerrado.
func Buscar(nombre string) (Herramienta, bool) { return buscarCatalogo(nombre) }

func buscarCatalogo(nombre string) (Herramienta, bool) {
	for _, h := range catalogo {
		if h.Nombre == nombre {
			return h, true
		}
	}
	return Herramienta{}, false
}

// Existe informa si el nombre está en el catálogo cerrado.
func Existe(nombre string) bool {
	_, ok := buscarCatalogo(nombre)
	return ok
}

// NombresCatalogo devuelve los catorce nombres, en orden.
func NombresCatalogo() []string {
	out := make([]string, 0, len(catalogo))
	for _, h := range catalogo {
		out = append(out, h.Nombre)
	}
	return out
}

// NombresDeCategoria devuelve los nombres de una categoría, en orden.
func NombresDeCategoria(c Categoria) []string {
	var out []string
	for _, h := range catalogo {
		if h.Categoria == c {
			out = append(out, h.Nombre)
		}
	}
	return out
}

// NombresDeAccion devuelve los nombres de las herramientas de una acción, en el
// orden del catálogo.
func NombresDeAccion(a Accion) []string {
	var out []string
	for _, h := range catalogo {
		if h.Accion() == a {
			out = append(out, h.Nombre)
		}
	}
	return out
}

// AccionDe devuelve la acción de una herramienta del catálogo. El segundo valor
// es false si el nombre no existe.
func AccionDe(nombre string) (Accion, bool) {
	h, ok := buscarCatalogo(nombre)
	if !ok {
		return "", false
	}
	return h.Accion(), true
}
