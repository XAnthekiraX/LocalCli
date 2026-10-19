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
	Buscar    string `json:"buscar"    desc:"Fragmento existente copiado textualmente del archivo (misma sangría y espacios); debe aparecer una sola vez"`
	Reemplazo string `json:"reemplazo" desc:"Texto que reemplaza al fragmento; usa finales de línea LF"`
}

type PeticionEditarArchivo struct {
	Ruta       string   `json:"ruta"       desc:"Archivo a modificar"`
	Reemplazos []Cambio `json:"reemplazos" desc:"Lista no vacía de sustituciones aplicadas en orden; cada buscar debe coincidir exactamente una vez"`
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
	Comando string `json:"comando"          desc:"Comando a correr; pide confirmación si es destructivo, si sale a la red o si toca una ruta fuera de la carpeta"`
	Carpeta string `json:"carpeta,omitempty" desc:"Carpeta de trabajo; por defecto, la del proyecto"`
}

type PeticionBuscarEnInternet struct {
	Consulta string `json:"consulta" desc:"Qué buscar; es lo único que sale de la máquina"`
}

type PeticionAbrirPagina struct {
	Direccion string `json:"direccion" desc:"Dirección de la página a abrir"`
}

// PeticionCrearTodo acepta una de las dos formas, excluyentes: un paso suelto
// (`contenido`) o una tarea principal con sus subtareas (`titulo` + `subtareas`).
// Los campos van opcionales en el esquema porque la exclusión se comprueba en el
// handler; un campo obligatorio en el esquema impediría la otra forma.
type PeticionCrearTodo struct {
	Contenido string     `json:"contenido,omitempty" desc:"Qué hay que hacer; una acción concreta. Úsalo solo para un paso suelto, en vez de 'titulo'/'subtareas'"`
	Estado    string     `json:"estado,omitempty"    desc:"pendiente | en_progreso | completada | cancelada; obligatorio con 'contenido'" enum:"pendiente,en_progreso,completada,cancelada"`
	Prioridad string     `json:"prioridad,omitempty" desc:"alta | media | baja (opcional; por defecto media)" enum:"alta,media,baja"`
	Titulo    string     `json:"titulo,omitempty"    desc:"Título de la tarea principal. Úsalo con 'subtareas' en vez de 'contenido'"`
	Subtareas []Subtarea `json:"subtareas,omitempty" desc:"Fases que cuelgan de la tarea principal, en orden; un solo nivel"`
}

// Subtarea es un paso que cuelga de una tarea principal. No tiene `subtareas`:
// la jerarquía de la lista es de un solo nivel.
type Subtarea struct {
	Contenido string `json:"contenido"          desc:"Qué hay que hacer; una acción concreta"`
	Estado    string `json:"estado"             desc:"pendiente | en_progreso | completada | cancelada" enum:"pendiente,en_progreso,completada,cancelada"`
	Prioridad string `json:"prioridad,omitempty" desc:"alta | media | baja (opcional; por defecto media)" enum:"alta,media,baja"`
}

type ElementoTodo struct {
	Contenido string     `json:"contenido"          desc:"Qué hay que hacer; una acción concreta"`
	Estado    string     `json:"estado"             desc:"pendiente | en_progreso | completada | cancelada" enum:"pendiente,en_progreso,completada,cancelada"`
	Prioridad string     `json:"prioridad,omitempty" desc:"alta | media | baja (opcional; por defecto media)" enum:"alta,media,baja"`
	Subtareas []Subtarea `json:"subtareas,omitempty" desc:"Subtareas del elemento; un solo nivel"`
}

type PeticionActualizarTodo struct {
	Elementos []ElementoTodo `json:"elementos" desc:"La lista completa de pasos; reemplaza la anterior"`
}

type PeticionEjecutarFlujo struct {
	Flujo    string `json:"flujo"    desc:"Comando del flujo a lanzar; el enum se arma por turno con los que el agente puede lanzar"`
	Peticion string `json:"peticion" desc:"Objetivo del flujo, con tus palabras"`
}
