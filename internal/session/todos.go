package session

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"localcli/internal/store"
	"localcli/internal/tools"
	"localcli/internal/tui"
)

// Vocabulario cerrado de la lista de pasos ([[database/01-schema/ENUMS]]).
var (
	estadosTodo     = map[string]bool{"pendiente": true, "en_progreso": true, "completada": true, "cancelada": true}
	prioridadesTodo = map[string]bool{"alta": true, "media": true, "baja": true}
)

// crearTodo es el handler de `crear_todo`: añade al final un paso suelto o una
// tarea principal con sus subtareas, y devuelve la lista resultante como
// checklist.
func (o *Orquestador) crearTodo(ctx context.Context, args any, c tools.Contexto) (tools.Resultado, error) {
	peticion, ok := args.(tools.PeticionCrearTodo)
	if !ok {
		return tools.Resultado{Error: tools.CodBadArgs + ": argumentos inesperados"}, nil
	}

	contenido := strings.TrimSpace(peticion.Contenido)
	titulo := strings.TrimSpace(peticion.Titulo)
	conSubtareas := len(peticion.Subtareas) > 0
	if contenido != "" && (titulo != "" || conSubtareas) {
		return tools.Resultado{Error: tools.CodBadArgs + ": usa `contenido` para un paso suelto o `titulo`+`subtareas` para una tarea principal, no ambos"}, nil
	}
	if contenido == "" && (titulo == "" || !conSubtareas) {
		return tools.Resultado{Error: tools.CodBadArgs + ": falta `contenido` o `titulo` con `subtareas`"}, nil
	}

	actuales, err := store.ListarTodos(o.db, o.projectID, c.SesionID)
	if err != nil {
		return tools.Resultado{}, err
	}
	hayEnProgreso := false
	for _, t := range actuales {
		if t.Estado == "en_progreso" {
			hayEnProgreso = true
			break
		}
	}

	if contenido != "" {
		paso, motivo := pasoValidado(peticion.Contenido, peticion.Estado, peticion.Prioridad)
		if motivo != "" {
			return tools.Resultado{Error: motivo}, nil
		}
		if paso.Estado == "en_progreso" && hayEnProgreso {
			return tools.Resultado{Error: tools.CodBadArgs + ": ya hay un paso `en_progreso`; usa `actualizar_todo` para cambiar cuál está en curso"}, nil
		}
		lista, err := store.CrearTodo(o.db, o.projectID, c.SesionID, paso, nil)
		if err != nil {
			return tools.Resultado{}, err
		}
		o.emitirLista(c.SesionID, lista)
		return tools.Resultado{Salida: checklist(lista), Meta: map[string]any{"medida": pluralPasos(len(lista))}}, nil
	}

	// Tarea principal con subtareas: un solo grupo atómico.
	estadoPrincipal, motivo := estadoValidado(peticion.Estado, "pendiente")
	if motivo != "" {
		return tools.Resultado{Error: motivo}, nil
	}
	prioridadPrincipal, motivo := prioridadValidada(peticion.Prioridad)
	if motivo != "" {
		return tools.Resultado{Error: motivo}, nil
	}
	enProgreso := 0
	if estadoPrincipal == "en_progreso" {
		enProgreso++
	}
	subtareas := make([]store.Todo, 0, len(peticion.Subtareas))
	for _, s := range peticion.Subtareas {
		subtarea, motivo := pasoValidado(s.Contenido, s.Estado, s.Prioridad)
		if motivo != "" {
			return tools.Resultado{Error: motivo}, nil
		}
		if subtarea.Estado == "en_progreso" {
			enProgreso++
		}
		subtareas = append(subtareas, subtarea)
	}
	if enProgreso > 1 {
		return tools.Resultado{Error: tools.CodBadArgs + ": solo un paso puede estar `en_progreso` a la vez, subtareas incluidas"}, nil
	}
	if enProgreso > 0 && hayEnProgreso {
		return tools.Resultado{Error: tools.CodBadArgs + ": ya hay un paso `en_progreso`; usa `actualizar_todo` para cambiar cuál está en curso"}, nil
	}

	principal := store.Todo{Contenido: titulo, Estado: estadoPrincipal, Prioridad: prioridadPrincipal}
	lista, err := store.CrearTodo(o.db, o.projectID, c.SesionID, principal, subtareas)
	if err != nil {
		return tools.Resultado{}, err
	}
	o.emitirLista(c.SesionID, lista)
	return tools.Resultado{Salida: checklist(lista), Meta: map[string]any{"medida": pluralPasos(len(lista))}}, nil
}

// actualizarTodo es el handler de `actualizar_todo`: sustituye la lista entera y
// devuelve el checklist resultante. Cada elemento puede llevar sus subtareas.
func (o *Orquestador) actualizarTodo(ctx context.Context, args any, c tools.Contexto) (tools.Resultado, error) {
	peticion, ok := args.(tools.PeticionActualizarTodo)
	if !ok {
		return tools.Resultado{Error: tools.CodBadArgs + ": argumentos inesperados"}, nil
	}

	elementos := make([]store.Todo, 0, len(peticion.Elementos))
	enProgreso := 0
	for _, e := range peticion.Elementos {
		paso, motivo := pasoValidado(e.Contenido, e.Estado, e.Prioridad)
		if motivo != "" {
			return tools.Resultado{Error: motivo}, nil
		}
		if paso.Estado == "en_progreso" {
			enProgreso++
		}
		posicionPrincipal := len(elementos)
		elementos = append(elementos, paso)
		for _, s := range e.Subtareas {
			subtarea, motivo := pasoValidado(s.Contenido, s.Estado, s.Prioridad)
			if motivo != "" {
				return tools.Resultado{Error: motivo}, nil
			}
			if subtarea.Estado == "en_progreso" {
				enProgreso++
			}
			padre := posicionPrincipal
			subtarea.PosicionPadre = &padre
			elementos = append(elementos, subtarea)
		}
	}
	if enProgreso > 1 {
		return tools.Resultado{Error: tools.CodBadArgs + ": solo un paso puede estar `en_progreso` a la vez, subtareas incluidas"}, nil
	}

	lista, err := store.ReemplazarTodos(o.db, o.projectID, c.SesionID, elementos)
	if err != nil {
		return tools.Resultado{}, err
	}
	o.emitirLista(c.SesionID, lista)
	return tools.Resultado{Salida: checklist(lista), Meta: map[string]any{"medida": pluralPasos(len(lista))}}, nil
}

// pasoValidado normaliza un paso y comprueba su vocabulario. Devuelve el motivo
// como `E_BAD_ARGS` corregible si algo no encaja.
func pasoValidado(contenido, estado, prioridad string) (store.Todo, string) {
	contenido = strings.TrimSpace(contenido)
	if contenido == "" {
		return store.Todo{}, tools.CodBadArgs + ": el `contenido` no puede estar vacío"
	}
	estado, motivo := estadoValidado(estado, "")
	if motivo != "" {
		return store.Todo{}, motivo
	}
	prioridad, motivo = prioridadValidada(prioridad)
	if motivo != "" {
		return store.Todo{}, motivo
	}
	return store.Todo{Contenido: contenido, Estado: estado, Prioridad: prioridad}, ""
}

func estadoValidado(estado, porDefecto string) (string, string) {
	estado = strings.TrimSpace(estado)
	if estado == "" {
		estado = porDefecto
	}
	if !estadosTodo[estado] {
		return "", tools.CodBadArgs + ": el `estado` no está en pendiente/en_progreso/completada/cancelada"
	}
	return estado, ""
}

func prioridadValidada(prioridad string) (string, string) {
	prioridad = strings.TrimSpace(prioridad)
	if prioridad == "" {
		prioridad = "media"
	}
	if !prioridadesTodo[prioridad] {
		return "", tools.CodBadArgs + ": la `prioridad` no está en alta/media/baja"
	}
	return prioridad, ""
}

// emitirLista publica la lista completa para que el panel la repinte. Cada paso
// lleva su `padre`: la posición de la tarea principal de la que cuelga, o `null`
// si es un paso raíz.
func (o *Orquestador) emitirLista(sesionID string, lista []store.Todo) {
	type elementoTodo struct {
		Contenido string `json:"contenido"`
		Estado    string `json:"estado"`
		Padre     *int   `json:"padre"`
	}
	elementos := make([]elementoTodo, 0, len(lista))
	for _, t := range lista {
		elementos = append(elementos, elementoTodo{Contenido: t.Contenido, Estado: t.Estado, Padre: t.PosicionPadre})
	}
	datos, _ := json.Marshal(elementos)
	o.emitir(tui.Evento{Sesion: sesionID, Tipo: tui.EventoTodoActualizada, Datos: map[string]string{"elementos": string(datos)}})
}

// checklist compone la lista de pasos como texto, con el glifo de cada estado y
// las subtareas sangradas bajo su tarea principal.
func checklist(lista []store.Todo) string {
	var b strings.Builder
	for i, t := range lista {
		if i > 0 {
			b.WriteString("\n")
		}
		if t.PosicionPadre != nil {
			b.WriteString("  ")
		}
		b.WriteString(glifoDeEstado(t.Estado))
		b.WriteString(" ")
		b.WriteString(t.Contenido)
	}
	return b.String()
}

func glifoDeEstado(estado string) string {
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

func pluralPasos(n int) string {
	if n == 1 {
		return "1 paso"
	}
	return strconv.Itoa(n) + " pasos"
}
