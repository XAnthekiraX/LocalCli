package tools

import "reflect"

// Catalogo devuelve el catálogo cerrado de las dieciséis herramientas incluidas.
// Los handlers van a nil: se inyectan al cablear (fileops y exec). `ejecutar_flujo`
// es la única con permiso derivado y su esquema se arma por turno.
func Catalogo() []Herramienta {
	return []Herramienta{
		archivo("leer_archivo", "Lee un archivo de texto por rangos de línea. Devuelve el contenido y los metadatos de la lectura.",
			PermisoRead, "LEER", "ruta", UnidadLinea, PeticionLeerArchivo{}),
		archivo("listar_carpeta", "Lista un nivel de una carpeta: los nombres de sus entradas.",
			PermisoRead, "LISTAR", "ruta", UnidadEntrada, PeticionListarCarpeta{}),
		archivo("buscar_archivos", "Busca archivos por su nombre en el proyecto.",
			PermisoRead, "BUSCAR", "patron", UnidadCoincidencia, PeticionBuscarArchivos{}),
		archivo("buscar_en_archivos", "Busca un texto dentro de los archivos del proyecto.",
			PermisoRead, "GREP", "patron", UnidadCoincidencia, PeticionBuscarEnArchivos{}),
		archivo("crear_archivo", "Crea un archivo nuevo con su contenido; falla si ya existe.",
			PermisoWrite, "CREAR", "ruta", "", PeticionCrearArchivo{}),
		archivo("escribir_archivo", "Sobrescribe un archivo con el contenido dado.",
			PermisoWrite, "ESCRIBIR", "ruta", "", PeticionEscribirArchivo{}),
		{ // editar_archivo es la única de edición parcial: permiso `edit`.
			Nombre: "editar_archivo", Descripcion: "Aplica sustituciones parciales a un archivo existente; cada buscar debe coincidir exactamente una vez (se toleran finales de línea, sangría y Unicode tipográfico).",
			Categoria: CategoriaArchivos, Permiso: PermisoEdit, Modo: ModoEscribe,
			Verbo: "EDITAR", Tema: "ruta", TipoPeticion: reflect.TypeOf(PeticionEditarArchivo{}),
		},
		archivo("eliminar_archivo", "Borra un archivo; pide confirmación explícita.",
			PermisoWrite, "BORRAR", "ruta", "", PeticionEliminarArchivo{}),
		archivo("crear_carpeta", "Crea una carpeta; falla si ya existe.",
			PermisoWrite, "MKDIR", "ruta", "", PeticionCrearCarpeta{}),
		archivo("eliminar_carpeta", "Borra una carpeta; pide confirmación explícita.",
			PermisoWrite, "RMDIR", "ruta", "", PeticionEliminarCarpeta{}),
		{
			Nombre: "ejecutar_comando", Descripcion: "Ejecuta un comando de terminal en el proyecto; solo los de la lista blanca corren sin preguntar.",
			Categoria: CategoriaTerminal, Permiso: PermisoRead, Modo: ModoLee,
			Verbo: "EJEC", Tema: "comando", Unidad: UnidadLinea, TipoPeticion: reflect.TypeOf(PeticionEjecutarComando{}),
		},
		{
			Nombre: "buscar_en_internet", Descripcion: "Busca en internet. Solo sale de la máquina la consulta.",
			Categoria: CategoriaInternet, Permiso: PermisoRead, Modo: ModoLee,
			Verbo: "WEB", Tema: "consulta", Unidad: UnidadResultado, TipoPeticion: reflect.TypeOf(PeticionBuscarEnInternet{}),
		},
		{
			Nombre: "abrir_pagina", Descripcion: "Abre una página web y devuelve su contenido.",
			Categoria: CategoriaInternet, Permiso: PermisoRead, Modo: ModoLee,
			Verbo: "ABRIR", Tema: "direccion", Unidad: UnidadLinea, TipoPeticion: reflect.TypeOf(PeticionAbrirPagina{}),
		},
		{
			Nombre: "crear_todo", Descripcion: "Añade un paso al final de la lista de pasos de la sesión y devuelve la lista resultante.",
			Categoria: CategoriaTareas, Permiso: PermisoRead, Modo: ModoLee,
			Verbo: "TODO", Tema: "contenido", Unidad: UnidadPaso, TipoPeticion: reflect.TypeOf(PeticionCrearTodo{}),
		},
		{
			Nombre: "actualizar_todo", Descripcion: "Reemplaza la lista entera de pasos de la sesión y devuelve el checklist.",
			Categoria: CategoriaTareas, Permiso: PermisoRead, Modo: ModoLee,
			Verbo: "TODO", Unidad: UnidadPaso, TipoPeticion: reflect.TypeOf(PeticionActualizarTodo{}),
		},
		{
			Nombre: "ejecutar_flujo", Descripcion: "Lanza un flujo oficial o propio como sub-proceso y devuelve su entrega más el bloque resumido.",
			Categoria: CategoriaFlujos, Permiso: PermisoDerivado, Modo: ModoLee,
			Verbo: "FLUJO", Tema: "flujo", Unidad: UnidadEtapa, TipoPeticion: reflect.TypeOf(PeticionEjecutarFlujo{}),
		},
	}
}

func archivo(nombre, descripcion string, permiso Permiso, verbo, tema string, unidad Unidad, tipo any) Herramienta {
	modo := ModoEscribe
	if permiso == PermisoRead {
		modo = ModoLee
	}
	return Herramienta{
		Nombre: nombre, Descripcion: descripcion,
		Categoria: CategoriaArchivos, Permiso: permiso, Modo: modo,
		Verbo: verbo, Tema: tema, Unidad: unidad, TipoPeticion: reflect.TypeOf(tipo),
	}
}

// NombresIncluidos lista los nombres del catálogo incluido; sirve para validar que
// una herramienta del usuario no choca con una incluida.
func NombresIncluidos() map[string]bool {
	nombres := map[string]bool{}
	for _, h := range Catalogo() {
		nombres[h.Nombre] = true
	}
	return nombres
}
