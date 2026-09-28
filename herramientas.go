// herramientas.go — T-B024-05, T-B024-06 y T-B024-09: el cableado de las
// herramientas.
//
// Fuente de verdad: ai/docs/backend/02-interfaces/TOOLS.md §7 (cada herramienta
// trae su handler propio; `fileops`, `exec` y el cliente de internet inyectan
// los suyos al cablear) y §9 (las herramientas del usuario son adaptadores
// finos sobre `exec`).
//
// Aquí vive lo que no puede vivir en `tools`: las llamadas a `fileops` y a
// `exec`. `tools` define los tipos y el pipeline; este archivo conecta los
// handlers y adapta la aprobación al único mecanismo, `tools.Contexto.Ask`.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	lcexec "localcli/internal/exec"
	"localcli/internal/fileops"
	"localcli/internal/session"
	"localcli/internal/store"
	"localcli/internal/tools"
	"localcli/internal/tui"
)

// registroDeHerramientas arma el registro de producción: las catorce incluidas
// con sus handlers, más las que declare el usuario en `.localcli/tools/`.
func registroDeHerramientas(ad *Adaptador, carpeta string, cambios *store.Cambios, todos *store.Todos, externo *lcexec.Ejecutor) (*tools.Registro, error) {
	handlers := map[string]tools.Ejecutar{
		"leer_archivo":       herramientaLeerArchivo(carpeta),
		"listar_carpeta":     herramientaListarCarpeta(carpeta),
		"buscar_archivos":    herramientaBuscarArchivos(carpeta),
		"buscar_en_archivos": herramientaBuscarEnArchivos(carpeta),
		"crear_archivo":      herramientaEscritura(carpeta, cambios, ad, "crear_archivo"),
		"escribir_archivo":   herramientaEscritura(carpeta, cambios, ad, "escribir_archivo"),
		"editar_archivo":     herramientaEscritura(carpeta, cambios, ad, "editar_archivo"),
		"eliminar_archivo":   herramientaEscritura(carpeta, cambios, ad, "eliminar_archivo"),
		"crear_carpeta":      herramientaEscritura(carpeta, cambios, ad, "crear_carpeta"),
		"eliminar_carpeta":   herramientaEscritura(carpeta, cambios, ad, "eliminar_carpeta"),
		"ejecutar_comando":   herramientaEjecutarComando(carpeta),
		"buscar_en_internet": herramientaBuscarInternet(),
		"abrir_pagina":       herramientaAbrirPagina(),
		"actualizar_todo":    herramientaActualizarTodo(ad, todos),
	}

	var impls []tools.Herramienta
	for _, nombre := range tools.NombresCatalogo() {
		h, ok := tools.NuevaHerramienta(nombre, handlers[nombre])
		if !ok {
			return nil, fmt.Errorf("arranque: %s no está en el catálogo", nombre)
		}
		impls = append(impls, h)
	}

	// Herramientas del usuario: se declaran, no se ejecutan al leerlas. Un JSON
	// inválido se avisa y se salta; el arranque continúa.
	declaradas, avisos := tools.CargarHerramientasUsuario(tools.DirHerramientasDe(carpeta))
	for _, aviso := range avisos {
		fmt.Fprintln(os.Stderr, "aviso:", aviso)
	}
	for _, u := range declaradas {
		impls = append(impls, tools.NuevaHerramientaUsuario(u, externo))
	}

	registro, err := tools.NuevoRegistro(impls)
	if err != nil {
		return nil, err
	}
	// La aprobación vive aquí, no dentro de cada handler: toda herramienta que
	// la necesita la pide por el mismo camino (DECISIONS.md).
	registro.Ask = ad.pedirAprobacion
	registro.Eventos = publicadorBus{bus: ad.bus}
	return registro, nil
}

// publicadorBus lleva los eventos de la capa universal al bus de la TUI. Si no
// hay suscriptores, el bus los descarta: no se bloquea nada.
type publicadorBus struct{ bus *session.Bus }

func (p publicadorBus) HerramientaInvocada(nombre, agente, verbo, tema string) {
	p.bus.Emitir(tui.Evento{Nombre: tui.EventoHerramientaInvocada, Datos: map[string]string{
		"herramienta": nombre,
		"agente":      agente,
		"verbo":       verbo,
		"tema":        tema,
	}})
}

func (p publicadorBus) HerramientaResultado(nombre string, ok bool, err string, truncado bool, medida string) {
	datos := map[string]string{
		"herramienta": nombre,
		"ok":          boolTexto(ok),
		"truncado":    boolTexto(truncado),
		"medida":      medida,
	}
	if err != "" {
		datos["error"] = err
	}
	p.bus.Emitir(tui.Evento{Nombre: tui.EventoHerramientaResultado, Datos: datos})
}

func boolTexto(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

// --- herramientas de archivo -------------------------------------------------

// aprobadorDeContexto traduce el `Ask` de la capa universal a lo que espera
// `fileops`. Sin `Ask`, no hay aprobador y ninguna escritura se aplica: el
// estado por defecto es cerrado.
func aprobadorDeContexto(c tools.Contexto) fileops.Aprobador {
	if c.Ask == nil {
		return nil
	}
	return fileops.AprobadorFunc(func(ctx context.Context, s fileops.SolicitudAprobacion) (fileops.Decision, error) {
		d, err := c.Ask(ctx, tools.Solicitud{Descripcion: s.Descripcion, Borrado: s.Borrado})
		if err != nil {
			return fileops.Decision{}, err
		}
		return fileops.Decision{Aprobada: d.Aprobada, Explicita: d.Explicita}, nil
	})
}

func opsDe(carpeta string, cambios *store.Cambios, ad *Adaptador, c tools.Contexto) *fileops.Ops {
	return &fileops.Ops{
		Proyecto:  carpeta,
		Historial: cambios,
		SesionID:  ad.sesionActual(),
		Aprobador: aprobadorDeContexto(c),
	}
}

func herramientaLeerArchivo(carpeta string) tools.Ejecutar {
	return func(ctx context.Context, args any, _ tools.Contexto) (tools.Resultado, error) {
		p, ok := args.(*tools.PeticionLeerArchivo)
		if !ok {
			return tools.Resultado{}, fmt.Errorf("arranque: petición de lectura desconocida %T", args)
		}
		resp, err := fileops.LeerArchivo(carpeta, p.Ruta)
		if err != nil {
			return corregible(err)
		}
		return tools.Resultado{Salida: resp.Contenido, Meta: map[string]any{"ruta": p.Ruta}}, nil
	}
}

func herramientaListarCarpeta(carpeta string) tools.Ejecutar {
	return func(ctx context.Context, args any, _ tools.Contexto) (tools.Resultado, error) {
		p, ok := args.(*tools.PeticionListarCarpeta)
		if !ok {
			return tools.Resultado{}, fmt.Errorf("arranque: petición de listado desconocida %T", args)
		}
		resp, err := fileops.ListarCarpeta(carpeta, p.Ruta)
		if err != nil {
			return corregible(err)
		}
		return tools.Resultado{Salida: strings.Join(resp.Entradas, "\n")}, nil
	}
}

func herramientaBuscarArchivos(carpeta string) tools.Ejecutar {
	return func(ctx context.Context, args any, _ tools.Contexto) (tools.Resultado, error) {
		p, ok := args.(*tools.PeticionBuscarArchivos)
		if !ok {
			return tools.Resultado{}, fmt.Errorf("arranque: petición de búsqueda desconocida %T", args)
		}
		resp, err := fileops.BuscarArchivos(carpeta, p.Patron)
		if err != nil {
			return corregible(err)
		}
		return tools.Resultado{Salida: strings.Join(resp.Rutas, "\n")}, nil
	}
}

func herramientaBuscarEnArchivos(carpeta string) tools.Ejecutar {
	return func(ctx context.Context, args any, _ tools.Contexto) (tools.Resultado, error) {
		p, ok := args.(*tools.PeticionBuscarEnArchivos)
		if !ok {
			return tools.Resultado{}, fmt.Errorf("arranque: petición de búsqueda desconocida %T", args)
		}
		resp, err := fileops.BuscarEnArchivos(carpeta, p.Patron, p.Ruta)
		if err != nil {
			return corregible(err)
		}
		var lineas []string
		for _, c := range resp.Coincidencias {
			lineas = append(lineas, fmt.Sprintf("%s:%d: %s", c.Archivo, c.Linea, c.Fragmento))
		}
		return tools.Resultado{Salida: strings.Join(lineas, "\n")}, nil
	}
}

// herramientaEscritura cubre las seis de escritura: todas pasan por `fileops`,
// que aplica la frontera de rutas, la aprobación y el registro del cambio.
func herramientaEscritura(carpeta string, cambios *store.Cambios, ad *Adaptador, operacion string) tools.Ejecutar {
	return func(ctx context.Context, args any, c tools.Contexto) (tools.Resultado, error) {
		ops := opsDe(carpeta, cambios, ad, c)
		var salida string
		var err error
		switch operacion {
		case "crear_archivo":
			p, ok := args.(*tools.PeticionCrearArchivo)
			if !ok {
				return tools.Resultado{}, fmt.Errorf("arranque: petición desconocida %T", args)
			}
			var resp tools.RespuestaEscritura
			if resp, err = ops.CrearArchivo(ctx, p.Ruta, p.Contenido); err == nil {
				salida = confirmacion(resp)
			}
		case "escribir_archivo":
			p, ok := args.(*tools.PeticionEscribirArchivo)
			if !ok {
				return tools.Resultado{}, fmt.Errorf("arranque: petición desconocida %T", args)
			}
			var resp tools.RespuestaEscritura
			if resp, err = ops.EscribirArchivo(ctx, p.Ruta, p.Contenido); err == nil {
				salida = confirmacion(resp)
			}
		case "editar_archivo":
			p, ok := args.(*tools.PeticionEditarArchivo)
			if !ok {
				return tools.Resultado{}, fmt.Errorf("arranque: petición desconocida %T", args)
			}
			var resp tools.RespuestaEscritura
			if resp, err = ops.EditarArchivo(ctx, p.Ruta, p.Cambio); err == nil {
				salida = confirmacion(resp)
			}
		case "eliminar_archivo":
			p, ok := args.(*tools.PeticionEliminarArchivo)
			if !ok {
				return tools.Resultado{}, fmt.Errorf("arranque: petición desconocida %T", args)
			}
			var resp tools.RespuestaEliminar
			if resp, err = ops.EliminarArchivo(ctx, p.Ruta); err == nil {
				salida = resp.Confirmacion
			}
		case "crear_carpeta":
			p, ok := args.(*tools.PeticionCrearCarpeta)
			if !ok {
				return tools.Resultado{}, fmt.Errorf("arranque: petición desconocida %T", args)
			}
			var resp tools.RespuestaEscritura
			if resp, err = ops.CrearCarpeta(ctx, p.Ruta); err == nil {
				salida = confirmacion(resp)
			}
		case "eliminar_carpeta":
			p, ok := args.(*tools.PeticionEliminarCarpeta)
			if !ok {
				return tools.Resultado{}, fmt.Errorf("arranque: petición desconocida %T", args)
			}
			var resp tools.RespuestaEliminar
			if resp, err = ops.EliminarCarpeta(ctx, p.Ruta); err == nil {
				salida = resp.Confirmacion
			}
		}
		if err != nil {
			return corregible(err)
		}
		return tools.Resultado{Salida: salida}, nil
	}
}

func confirmacion(r tools.RespuestaEscritura) string {
	if r.Ruta != "" {
		return r.Confirmacion + ": " + r.Ruta
	}
	return r.Confirmacion
}

// --- terminal -----------------------------------------------------------------

func herramientaEjecutarComando(carpeta string) tools.Ejecutar {
	return func(ctx context.Context, args any, c tools.Contexto) (tools.Resultado, error) {
		p, ok := args.(*tools.PeticionEjecutarComando)
		if !ok {
			return tools.Resultado{}, fmt.Errorf("arranque: petición de terminal desconocida %T", args)
		}
		// T-B024-01: el ejecutor corre SIEMPRE dentro de la carpeta del
		// proyecto; la carpeta de trabajo del payload es una subcarpeta suya.
		ejecutor := &lcexec.Ejecutor{Proyecto: carpeta}
		if c.Ask != nil {
			ejecutor.Aprobador = lcexec.AprobadorFunc(func(ctx context.Context, descripcion string) (bool, error) {
				d, err := c.Ask(ctx, tools.Solicitud{Descripcion: descripcion})
				if err != nil {
					return false, err
				}
				return d.Aprobada, nil
			})
		}
		res, err := ejecutor.Ejecutar(ctx, p.Comando, p.Carpeta)
		if err != nil {
			return corregible(err)
		}
		return tools.Resultado{
			Salida:   salidaDeComando(res),
			Truncado: res.Truncado,
			Meta:     map[string]any{"codigo": res.Codigo, "termino": res.Termino},
		}, nil
	}
}

// salidaDeComando junta la salida y la de error: un comando que falla no es un
// fallo de la herramienta, devuelve lo que dijo y el agente sigue.
func salidaDeComando(r tools.RespuestaEjecutarComando) string {
	var b strings.Builder
	if s := strings.TrimRight(r.Salida, "\n"); s != "" {
		b.WriteString(s)
	}
	if e := strings.TrimRight(r.Error, "\n"); e != "" {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString("[error] " + e)
	}
	if !r.Termino {
		b.WriteString("\n[el comando no terminó]")
	}
	if b.Len() == 0 {
		return "(sin salida)"
	}
	return b.String()
}

// --- sesión: la lista de pasos ------------------------------------------------

// herramientaActualizarTodo reescribe la lista de pasos de la sesión activa y
// la anuncia al bus para que la vea el panel. El modelo ve la lista resultante
// en el propio resultado, así que no necesita una herramienta de lectura.
func herramientaActualizarTodo(ad *Adaptador, todos *store.Todos) tools.Ejecutar {
	return func(ctx context.Context, args any, _ tools.Contexto) (tools.Resultado, error) {
		p, ok := args.(*tools.PeticionActualizarTodo)
		if !ok {
			return tools.Resultado{}, fmt.Errorf("arranque: petición de TODO desconocida %T", args)
		}
		sesion := ad.sesionActual()
		if sesion == "" {
			return corregible(fmt.Errorf("no hay una sesión activa donde guardar la lista de pasos"))
		}
		items := make([]store.Todo, len(p.Elementos))
		for i, e := range p.Elementos {
			items[i] = store.Todo{Contenido: e.Contenido, Estado: e.Estado, Prioridad: e.Prioridad}
		}
		if err := todos.Reemplazar(sesion, items); err != nil {
			return corregible(err)
		}
		ad.bus.Emitir(tui.Evento{Nombre: tui.EventoTodoActualizada, Datos: map[string]string{
			"sesion":    sesion,
			"elementos": codificarTodo(items),
		}})
		return tools.Resultado{Salida: textoDeTodo(items)}, nil
	}
}

// todoDeEvento es la forma mínima de un paso en el evento todo_actualizada: lo
// que el panel necesita para pintarlo. No lleva prioridad: no se muestra.
type todoDeEvento struct {
	Contenido string `json:"contenido"`
	Estado    string `json:"estado"`
}

// codificarTodo serializa la lista para el evento, en un solo campo de texto.
func codificarTodo(items []store.Todo) string {
	vista := make([]todoDeEvento, len(items))
	for i, it := range items {
		vista[i] = todoDeEvento{Contenido: it.Contenido, Estado: it.Estado}
	}
	b, err := json.Marshal(vista)
	if err != nil {
		return "[]"
	}
	return string(b)
}

// textoDeTodo arma el checklist que vuelve al modelo, para que vea la lista
// resultante sin tener que pedirla otra vez.
func textoDeTodo(items []store.Todo) string {
	if len(items) == 0 {
		return "lista de pasos vacía"
	}
	var b strings.Builder
	for _, it := range items {
		b.WriteString(marcaDeTodo(it.Estado) + " " + it.Contenido + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func marcaDeTodo(estado string) string {
	switch estado {
	case "en_progreso":
		return "[•]"
	case "completada":
		return "[✓]"
	case "cancelada":
		return "[x]"
	default:
		return "[ ]"
	}
}

// --- internet --------------------------------------------------------------

func herramientaBuscarInternet() tools.Ejecutar {
	return func(ctx context.Context, args any, _ tools.Contexto) (tools.Resultado, error) {
		p, ok := args.(*tools.PeticionBuscarInternet)
		if !ok {
			return tools.Resultado{}, fmt.Errorf("arranque: petición de internet desconocida %T", args)
		}
		resp, err := buscarEnInternet(ctx, p.Consulta)
		if err != nil {
			return corregible(err)
		}
		var lineas []string
		for _, r := range resp.Resultados {
			lineas = append(lineas, r.Titulo+" — "+r.Direccion+"\n"+r.Fragmento)
		}
		return tools.Resultado{Salida: strings.Join(lineas, "\n\n")}, nil
	}
}

func herramientaAbrirPagina() tools.Ejecutar {
	return func(ctx context.Context, args any, _ tools.Contexto) (tools.Resultado, error) {
		p, ok := args.(*tools.PeticionAbrirPagina)
		if !ok {
			return tools.Resultado{}, fmt.Errorf("arranque: petición de internet desconocida %T", args)
		}
		resp, err := abrirPagina(ctx, p.Direccion)
		if err != nil {
			return corregible(err)
		}
		return tools.Resultado{Salida: resp.Contenido}, nil
	}
}

// corregible convierte un error de negocio en un resultado que el modelo puede
// leer y corregir: no corta el turno (VALIDATION.md §2.1).
func corregible(err error) (tools.Resultado, error) {
	if err == nil {
		return tools.Resultado{}, nil
	}
	return tools.Resultado{Error: err.Error()}, nil
}
