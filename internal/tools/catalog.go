package tools

// catalog.go — el catálogo cerrado de las quince herramientas.
//
// Fuente de verdad: ai/docs/backend/02-interfaces/TOOLS.md §1 (las quince, su
// categoría, si leen o escriben y a qué agente pertenecen) y
// ai/docs/specs/SPEC-TOOLS.md §Catálogo (las mismas quince).
//
// "Cerrado" significa cerrado: el agente no puede pedir nada fuera de esta
// lista salvo lo que el usuario declare en `.localcli/tools/`. El nombre es la
// clave: es exactamente lo que el modelo escribe al pedirla.
//
// El reparto por agente NO vive aquí. El catálogo solo describe cada
// herramienta (categoría, permiso, modo, descripción, esquema); quién puede
// pedirla se decide con los `permissions` del `agent.yaml` (SPEC-AGENTE-BASE) y
// el `Permiso` es un dato declarado de la herramienta: el reparto se decide
// comparándolo con los permisos del agente, no con un `switch` en el enrutado
// (SECURITY.md §2).
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

// Permiso es el de alto nivel que gobierna un grupo de herramientas. Un agente
// declara sus `permissions` (mapa de permiso a efecto) y de ahí se deriva su
// catálogo efectivo (SPEC-AGENTE-BASE).
type Permiso string

const (
	// PermisoRead: no cambia el proyecto: mirar y buscar archivos, la terminal
	// de consulta, internet y la lista de pasos de la sesión. No escribe
	// archivos del proyecto, así que la conceden los dos agentes.
	PermisoRead Permiso = "read"
	// PermisoWrite: crear, sobrescribir o borrar un archivo o una carpeta.
	PermisoWrite Permiso = "write"
	// PermisoEdit: editar contenido existente, sin reemplazarlo entero. Solo
	// `editar_archivo` cae aquí.
	PermisoEdit Permiso = "edit"
)

func (p Permiso) String() string { return string(p) }

// Valida informa si el permiso es uno del vocabulario.
func (p Permiso) Valida() bool {
	switch p {
	case PermisoRead, PermisoWrite, PermisoEdit:
		return true
	}
	return false
}

// Permitida informa si el permiso está entre los concedidos.
func (p Permiso) Permitida(permisos []Permiso) bool {
	for _, q := range permisos {
		if q == p {
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
	// Permiso es lo que el agente concede para usar la herramienta: `read`,
	// `write` o `edit`. Es un dato del catálogo, no un `switch` en el enrutado:
	// el reparto se decide comparándolo con los `permissions` del agente.
	Permiso Permiso
	Modo    Modo
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
// pertenece a `build` (columna "Agente" de TOOLS.md §1). Lo deciden los permisos
// `write` y `edit`; la lista de pasos de la sesión es `read` y la tienen los dos
// agentes.
func (h Herramienta) SoloBuild() bool {
	return h.Permiso == PermisoWrite || h.Permiso == PermisoEdit
}

// catalogo es la lista cerrada, en el orden documentado: archivos de lectura,
// archivos de escritura, terminal e internet.
//
// `crear_archivo` y `escribir_archivo` están separadas a propósito: el agente
// no destruye algo por accidente cuando pretendía crear (TOOLS.md §1).
var catalogo = []Herramienta{
	// Archivos de lectura — permiso `read`.
	{Nombre: "leer_archivo", Categoria: CatArchivos, Permiso: PermisoRead, Modo: Lee, Verbo: "LEER", Tema: "ruta", Unidad: "línea", Descripcion: "Lee el contenido de un archivo del proyecto."},
	{Nombre: "listar_carpeta", Categoria: CatArchivos, Permiso: PermisoRead, Modo: Lee, Verbo: "LISTAR", Tema: "ruta", Unidad: "entrada", Descripcion: "Lista las entradas de un nivel de una carpeta."},
	{Nombre: "buscar_archivos", Categoria: CatArchivos, Permiso: PermisoRead, Modo: Lee, Verbo: "BUSCAR", Tema: "patron", Unidad: "coincidencia", Descripcion: "Busca archivos por nombre."},
	{Nombre: "buscar_en_archivos", Categoria: CatArchivos, Permiso: PermisoRead, Modo: Lee, Verbo: "GREP", Tema: "patron", Unidad: "coincidencia", Descripcion: "Busca texto dentro del contenido de los archivos."},

	// Archivos de escritura — permiso `write` (salvo `editar_archivo`, `edit`).
	{Nombre: "crear_archivo", Categoria: CatArchivos, Permiso: PermisoWrite, Modo: Escribe, Verbo: "CREAR", Tema: "ruta", Descripcion: "Crea un archivo nuevo; falla si ya existe."},
	{Nombre: "escribir_archivo", Categoria: CatArchivos, Permiso: PermisoWrite, Modo: Escribe, Verbo: "ESCRIBIR", Tema: "ruta", Descripcion: "Sobrescribe el contenido entero de un archivo."},
	{Nombre: "editar_archivo", Categoria: CatArchivos, Permiso: PermisoEdit, Modo: Escribe, Verbo: "EDITAR", Tema: "ruta", Descripcion: "Aplica una edición parcial a un archivo."},
	{Nombre: "eliminar_archivo", Categoria: CatArchivos, Permiso: PermisoWrite, Modo: Escribe, Verbo: "BORRAR", Tema: "ruta", Descripcion: "Borra un archivo del proyecto."},
	{Nombre: "crear_carpeta", Categoria: CatArchivos, Permiso: PermisoWrite, Modo: Escribe, Verbo: "MKDIR", Tema: "ruta", Descripcion: "Crea una carpeta; falla si ya existe."},
	{Nombre: "eliminar_carpeta", Categoria: CatArchivos, Permiso: PermisoWrite, Modo: Escribe, Verbo: "RMDIR", Tema: "ruta", Descripcion: "Borra una carpeta del proyecto."},

	// Terminal — permiso `read`.
	{Nombre: "ejecutar_comando", Categoria: CatTerminal, Permiso: PermisoRead, Modo: Lee, Verbo: "EJEC", Tema: "comando", Unidad: "línea", Descripcion: "Ejecuta un comando de la lista blanca dentro del proyecto."},

	// Internet — permiso `read`.
	{Nombre: "buscar_en_internet", Categoria: CatInternet, Permiso: PermisoRead, Modo: Lee, Verbo: "WEB", Tema: "consulta", Unidad: "resultado", Descripcion: "Busca en internet; solo sale la consulta de la máquina."},
	{Nombre: "abrir_pagina", Categoria: CatInternet, Permiso: PermisoRead, Modo: Lee, Verbo: "ABRIR", Tema: "direccion", Unidad: "línea", Descripcion: "Descarga y devuelve el texto de una página."},

	// Sesión — permiso `read` (la tienen los dos agentes).
	{Nombre: "crear_todo", Categoria: CatTareas, Permiso: PermisoRead, Modo: Escribe, Verbo: "TODO", Tema: "contenido", Unidad: "paso", Descripcion: descripcionCrearTodo},
	{Nombre: "actualizar_todo", Categoria: CatTareas, Permiso: PermisoRead, Modo: Escribe, Verbo: "TODO", Unidad: "paso", Descripcion: descripcionActualizarTodo},
}

// descripcionCrearTodo es la política que ve el modelo para añadir un paso:
// solo describe lo que esta herramienta hace —añadir al final sin tocar los
// demás—, para que no se confunda con `actualizar_todo` (SPEC-TOOLS).
const descripcionCrearTodo = "Añade UN paso al final de la lista de pasos de la sesión, sin tocar los que ya había. " +
	"Úsala cuando quieras registrar un paso nuevo y los anteriores ya estén como deben; los pasos que no mandas no cambian. " +
	"No la uses para reordenar, cancelar un paso ni cambiar el estado de los que ya están: para eso está `actualizar_todo`. " +
	"Estados: `pendiente`, `en_progreso`, `completada` y `cancelada`. Solo un paso puede estar `en_progreso` a la vez: si ya hay uno, usa `actualizar_todo` para cambiarlo. " +
	"Prioridad opcional: `alta`, `media` o `baja`."

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

// El esquema de cada una de las quince se deriva de su DTO al cargar el paquete:
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

// NombresCatalogo devuelve los quince nombres, en orden.
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

// NombresDePermiso devuelve los nombres de las herramientas de un permiso, en
// el orden del catálogo.
func NombresDePermiso(p Permiso) []string {
	var out []string
	for _, h := range catalogo {
		if h.Permiso == p {
			out = append(out, h.Nombre)
		}
	}
	return out
}

// PermisoDe devuelve el permiso de una herramienta del catálogo. El segundo
// valor es false si el nombre no existe.
func PermisoDe(nombre string) (Permiso, bool) {
	h, ok := buscarCatalogo(nombre)
	if !ok {
		return "", false
	}
	return h.Permiso, true
}
