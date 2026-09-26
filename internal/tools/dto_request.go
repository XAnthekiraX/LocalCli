package tools

// dto_request.go — T-B007-02: los argumentos que el modelo pasa a cada
// herramienta.
//
// Fuente de verdad: ai/docs/backend/02-interfaces/dto/TOOLS-DTO.md §2
// (Request Schemas: campo, tipo, obligatoriedad y notas, herramienta por
// herramienta) y ai/docs/backend/05-quality/VALIDATION.md §3 (campos
// obligatorios y opcionales).
//
// Son los tipos de Go en la frontera, no cuerpos JSON de HTTP: LocalCli no
// expone red (INTERFACES-GENERAL.md §1). Aun así llevan etiqueta json porque
// el modelo produce texto que hay que decodificar en la frontera, y
// decodificar con campos desconocidos se rechaza (VALIDATION.md §1: "Los
// argumentos se validan contra el contrato de la herramienta antes de
// aplicarse").
//
// Una omisión deliberada: los borrados (`eliminar_archivo`,
// `eliminar_carpeta`) NO llevan un campo `confirmar`. La confirmación
// explícita es una decisión de la persona y vive en `approvals`, no en los
// argumentos del modelo: si `confirmar` fuera un campo, el modelo se
// confirmaría a sí mismo y la garantía se iría por el desagüe. TOOLS-DTO.md
// lo anota en la columna "Notas" de `ruta`, no como campo. Ver
// SECURITY.md §3 y VALIDATION.md §5 ("El modelo no puede concederse
// permisos").

// --- Archivos de lectura (ambos agentes) -----------------------------------

// PeticionLeerArchivo pide el contenido de un archivo del proyecto.
type PeticionLeerArchivo struct {
	Ruta string `json:"ruta"` // obligatoria; relativa a la carpeta del proyecto
}

// PeticionListarCarpeta pide las entradas de un nivel de una carpeta.
type PeticionListarCarpeta struct {
	Ruta string `json:"ruta"` // obligatoria; relativa; un nivel
}

// PeticionBuscarArchivos busca rutas por nombre de archivo.
type PeticionBuscarArchivos struct {
	Patron string `json:"patron"` // obligatorio; coincide contra nombres de archivo
}

// PeticionBuscarEnArchivos busca por contenido. Ruta es opcional y acota la
// búsqueda a una subcarpeta (TOOLS-DTO.md §2).
type PeticionBuscarEnArchivos struct {
	Patron string `json:"patron"`         // obligatorio; coincide contra el contenido
	Ruta   string `json:"ruta,omitempty"` // opcional; limita a una subcarpeta
}

// --- Archivos de escritura (solo `build`) ---------------------------------

// PeticionCrearArchivo crea un archivo. Falla si ya existe.
type PeticionCrearArchivo struct {
	Ruta      string `json:"ruta"`      // obligatoria
	Contenido string `json:"contenido"` // obligatorio; texto plano, cualquier extensión
}

// PeticionEscribirArchivo sobrescribe el contenido entero de un archivo.
type PeticionEscribirArchivo struct {
	Ruta      string `json:"ruta"`      // obligatoria
	Contenido string `json:"contenido"` // obligatorio
}

// PeticionEditarArchivo aplica una edición parcial. El cambio va como texto
// porque el contrato lo define así; cómo lo aplica `fileops` es cosa suya.
type PeticionEditarArchivo struct {
	Ruta   string `json:"ruta"`   // obligatoria
	Cambio string `json:"cambio"` // obligatorio; la edición a aplicar
}

// PeticionEliminarArchivo borra un archivo. La confirmación explícita no viaja
// aquí: la pide el motor a la persona.
type PeticionEliminarArchivo struct {
	Ruta string `json:"ruta"` // obligatoria
}

// PeticionCrearCarpeta crea una carpeta. Falla si ya existe.
type PeticionCrearCarpeta struct {
	Ruta string `json:"ruta"` // obligatoria
}

// PeticionEliminarCarpeta borra una carpeta. La confirmación explícita no
// viaja aquí, igual que en el borrado de archivos.
type PeticionEliminarCarpeta struct {
	Ruta string `json:"ruta"` // obligatoria
}

// --- Terminal (ambos agentes) ---------------------------------------------

// PeticionEjecutarComando lanza un comando. El texto NO se usa para decidir si
// es seguro: la lista blanca y Landlock lo deciden (TOOLS.md §5, VALIDATION.md
// §5).
type PeticionEjecutarComando struct {
	Comando string `json:"comando"`           // obligatorio
	Carpeta string `json:"carpeta,omitempty"` // opcional; por defecto, la del proyecto
}

// --- Internet (ambos agentes) ---------------------------------------------

// PeticionBuscarInternet lleva la consulta a internet. Es lo único que sale
// de la máquina (SECURITY.md §4), así que el tipo no admite ningún campo
// adjunto: no hay forma de mandar contenido del proyecto.
type PeticionBuscarInternet struct {
	Consulta string `json:"consulta"` // obligatoria
}

// PeticionAbrirPagina pide el contenido de una página.
type PeticionAbrirPagina struct {
	Direccion string `json:"direccion"` // obligatoria; URL de la página
}

// peticiones asocia cada nombre del catálogo con un valor vacío de su tipo.
// Es el punto único del contrato: el enrutado y la validación parten de aquí,
// así que una herramienta sin entrada en esta tabla no se puede enrutar ni
// validar.
var peticiones = map[string]func() any{
	"leer_archivo":       func() any { return &PeticionLeerArchivo{} },
	"listar_carpeta":     func() any { return &PeticionListarCarpeta{} },
	"buscar_archivos":    func() any { return &PeticionBuscarArchivos{} },
	"buscar_en_archivos": func() any { return &PeticionBuscarEnArchivos{} },
	"crear_archivo":      func() any { return &PeticionCrearArchivo{} },
	"escribir_archivo":   func() any { return &PeticionEscribirArchivo{} },
	"editar_archivo":     func() any { return &PeticionEditarArchivo{} },
	"eliminar_archivo":   func() any { return &PeticionEliminarArchivo{} },
	"crear_carpeta":      func() any { return &PeticionCrearCarpeta{} },
	"eliminar_carpeta":   func() any { return &PeticionEliminarCarpeta{} },
	"ejecutar_comando":   func() any { return &PeticionEjecutarComando{} },
	"buscar_en_internet": func() any { return &PeticionBuscarInternet{} },
	"abrir_pagina":       func() any { return &PeticionAbrirPagina{} },
}

// NuevaPeticion devuelve un valor vacío del tipo de argumentos de la
// herramienta indicada, listo para decodificar encima. El segundo valor es
// false si la herramienta no existe o si su tipo no está declarado: en ese
// caso no hay contrato con el que validar y la petición se rechaza.
func NuevaPeticion(nombre string) (any, bool) {
	nueva, ok := peticiones[nombre]
	if !ok {
		return nil, false
	}
	return nueva(), true
}
