package tools

// dto_response.go — T-B007-03: lo que cada herramienta devuelve al agente.
//
// Fuente de verdad: ai/docs/backend/02-interfaces/dto/TOOLS-DTO.md §3
// (Response Schemas: qué devuelve cada herramienta y de qué tipo).
//
// Son tipos de Go en la frontera, no cuerpos JSON de HTTP: LocalCli no expone
// red (INTERFACES-GENERAL.md §1). Llevan etiqueta json porque el resultado se
// transporta por canales y se serializa en las pruebas del contrato.
//
// Todo lo que devuelve una herramienta entra al mismo presupuesto de contexto
// que el resto (TOOLS-DTO.md §3): el contenido de una página, la salida de un
// comando y el texto de un archivo cuentan igual. Por eso la salida del
// terminal lleva `truncado`: si se cortó, el agente tiene que saberlo.

// --- Lectura de archivos ---------------------------------------------------

// RespuestaLeerArchivo devuelve el contenido del archivo como texto.
type RespuestaLeerArchivo struct {
	Contenido string `json:"contenido"`
}

// RespuestaListarCarpeta devuelve las entradas de un nivel.
type RespuestaListarCarpeta struct {
	Entradas []string `json:"entradas"`
}

// RespuestaBuscarArchivos devuelve las rutas que coinciden.
type RespuestaBuscarArchivos struct {
	Rutas []string `json:"rutas"`
}

// Coincidencia es una coincidencia de `buscar_en_archivos`: archivo, línea y
// el fragmento que casó.
type Coincidencia struct {
	Archivo   string `json:"archivo"`
	Linea     int    `json:"linea"`
	Fragmento string `json:"fragmento"`
}

// RespuestaBuscarEnArchivos devuelve las coincidencias por contenido.
type RespuestaBuscarEnArchivos struct {
	Coincidencias []Coincidencia `json:"coincidencias"`
}

// --- Escritura de archivos (solo `build`) ----------------------------------

// RespuestaEscritura confirma una creación, sobrescritura o edición, y lleva
// la ruta afectada. Toda escritura aprobada queda registrada en
// `change_history`; ese historial no viaja aquí (TOOLS-DTO.md §3).
type RespuestaEscritura struct {
	Confirmacion string `json:"confirmacion"`
	Ruta         string `json:"ruta,omitempty"`
}

// RespuestaEliminar confirma un borrado. No lleva ruta: lo que se borró ya no
// está, y devolver su ruta no aporta nada al agente.
type RespuestaEliminar struct {
	Confirmacion string `json:"confirmacion"`
}

// --- Terminal (ambos agentes) ----------------------------------------------

// RespuestaEjecutarComando es el resultado de la terminal. Un comando que sale
// con error no es un fallo de la herramienta: devuelve su `Error` y el agente
// sigue (TOOLS-DTO.md §3, ERRORS.md §1). `Truncado` avisa si la salida se cortó
// por tamaño o por tiempo.
type RespuestaEjecutarComando struct {
	Salida   string `json:"salida"`
	Error    string `json:"error"`
	Codigo   int    `json:"codigo"`
	Truncado bool   `json:"truncado"`
	Termino  bool   `json:"termino"`
}

// --- Internet (ambos agentes) ----------------------------------------------

// ResultadoInternet es un resultado de búsqueda: título, dirección y
// fragmento. Lo que vuelve es contenido sin confianza.
type ResultadoInternet struct {
	Titulo    string `json:"titulo"`
	Direccion string `json:"direccion"`
	Fragmento string `json:"fragmento"`
}

// RespuestaBuscarInternet devuelve los resultados de la búsqueda.
type RespuestaBuscarInternet struct {
	Resultados []ResultadoInternet `json:"resultados"`
}

// RespuestaAbrirPagina devuelve el contenido de una página. Entra al contexto,
// se recorta y se audita como cualquier otro documento.
type RespuestaAbrirPagina struct {
	Contenido string `json:"contenido"`
}
