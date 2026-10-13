package tools

// Tipos de petición de las herramientas incluidas. Son la fuente del esquema que
// ve el modelo; los handlers de fileops y exec los consumen.

type PeticionLeerArchivo struct {
	Ruta        string `json:"ruta"                  desc:"Archivo a leer; usa la ruta relativa al directorio de trabajo conservando las subcarpetas (p. ej. docs/specs/X.md), no la ruta absoluta completa ni ~ o $HOME"`
	InicioLinea int    `json:"inicio_linea,omitempty" desc:"Línea inicial, contando desde 1"`
	FinLinea    int    `json:"fin_linea,omitempty"    desc:"Línea final, inclusive"`
	Motivo      string `json:"motivo,omitempty"       desc:"Solo si la ruta sale de la carpeta del proyecto: explica en una línea por qué la buscas"`
}

type PeticionListarCarpeta struct {
	Ruta   string `json:"ruta"            desc:"Carpeta a listar, un nivel; usa la ruta relativa conservando las subcarpetas"`
	Motivo string `json:"motivo,omitempty" desc:"Solo si la ruta sale de la carpeta del proyecto: explica en una línea por qué la buscas"`
}

type PeticionBuscarArchivos struct {
	Patron string `json:"patron" desc:"Texto contra el que se comparan los nombres de archivo"`
}

type PeticionBuscarEnArchivos struct {
	Patron string `json:"patron"          desc:"Texto a buscar dentro de los archivos"`
	Ruta   string `json:"ruta,omitempty"   desc:"Limita la búsqueda a una subcarpeta"`
	Motivo string `json:"motivo,omitempty" desc:"Solo si la ruta sale de la carpeta del proyecto: explica en una línea por qué la buscas"`
}

type PeticionCrearArchivo struct {
	Ruta      string `json:"ruta"      desc:"Archivo a crear; falla si ya existe"`
	Contenido string `json:"contenido" desc:"Texto completo del archivo"`
}

type PeticionEscribirArchivo struct {
	Ruta      string `json:"ruta"      desc:"Archivo a sobrescribir"`
	Contenido string `json:"contenido" desc:"Texto que reemplaza el contenido entero"`
}

// Cambio es una sustitución de editar_archivo.
type Cambio struct {
	Buscar    string `json:"buscar"    desc:"Fragmento existente que se sustituye"`
	Reemplazo string `json:"reemplazo" desc:"Texto que reemplaza al fragmento encontrado"`
}

type PeticionEditarArchivo struct {
	Ruta       string   `json:"ruta"       desc:"Archivo a modificar"`
	Reemplazos []Cambio `json:"reemplazos" desc:"Cambios a aplicar, en orden; cada uno sustituye un fragmento existente"`
}

type PeticionEliminarArchivo struct {
	Ruta   string `json:"ruta"            desc:"Archivo a borrar; pide confirmación explícita"`
	Motivo string `json:"motivo,omitempty" desc:"Solo si la ruta sale de la carpeta del proyecto: explica en una línea por qué la buscas"`
}

type PeticionCrearCarpeta struct {
	Ruta   string `json:"ruta"            desc:"Carpeta a crear; falla si ya existe"`
	Motivo string `json:"motivo,omitempty" desc:"Solo si la ruta sale de la carpeta del proyecto: explica en una línea por qué la buscas"`
}

type PeticionEliminarCarpeta struct {
	Ruta   string `json:"ruta"            desc:"Carpeta a borrar; pide confirmación explícita"`
	Motivo string `json:"motivo,omitempty" desc:"Solo si la ruta sale de la carpeta del proyecto: explica en una línea por qué la buscas"`
}

type PeticionEjecutarComando struct {
	Comando string `json:"comando"          desc:"Comando a correr; se valida contra la lista blanca, no contra su texto"`
	Carpeta string `json:"carpeta,omitempty" desc:"Carpeta de trabajo; por defecto, la del proyecto"`
}

type PeticionBuscarEnInternet struct {
	Consulta string `json:"consulta" desc:"Qué buscar; es lo único que sale de la máquina"`
}

type PeticionAbrirPagina struct {
	Direccion string `json:"direccion" desc:"Dirección de la página a abrir"`
}

type PeticionCrearTodo struct {
	Contenido string `json:"contenido"          desc:"Qué hay que hacer; una acción concreta"`
	Estado    string `json:"estado"             desc:"pendiente | en_progreso | completada | cancelada"`
	Prioridad string `json:"prioridad,omitempty" desc:"alta | media | baja (opcional; por defecto media)"`
}

type ElementoTodo struct {
	Contenido string `json:"contenido"          desc:"Qué hay que hacer; una acción concreta"`
	Estado    string `json:"estado"             desc:"pendiente | en_progreso | completada | cancelada"`
	Prioridad string `json:"prioridad,omitempty" desc:"alta | media | baja (opcional; por defecto media)"`
}

type PeticionActualizarTodo struct {
	Elementos []ElementoTodo `json:"elementos" desc:"La lista completa de pasos; reemplaza la anterior"`
}

type PeticionEjecutarFlujo struct {
	Flujo    string `json:"flujo"    desc:"Comando del flujo a lanzar; el enum se arma por turno con los que el agente puede lanzar"`
	Peticion string `json:"peticion" desc:"Objetivo del flujo, con tus palabras"`
}
