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

// crearTodo es el handler de `crear_todo`: añade un paso al final y devuelve la
// lista resultante como checklist.
func (o *Orquestador) crearTodo(ctx context.Context, args any, c tools.Contexto) (tools.Resultado, error) {
	peticion, ok := args.(tools.PeticionCrearTodo)
	if !ok {
		return tools.Resultado{Error: tools.CodBadArgs + ": argumentos inesperados"}, nil
	}
	contenido := strings.TrimSpace(peticion.Contenido)
	if contenido == "" {
		return tools.Resultado{Error: tools.CodBadArgs + ": el `contenido` no puede estar vacío"}, nil
	}
	estado := strings.TrimSpace(peticion.Estado)
	if !estadosTodo[estado] {
		return tools.Resultado{Error: tools.CodBadArgs + ": el `estado` no está en pendiente/en_progreso/completada/cancelada"}, nil
	}
	prioridad := strings.TrimSpace(peticion.Prioridad)
	if prioridad == "" {
		prioridad = "media"
	}
	if !prioridadesTodo[prioridad] {
		return tools.Resultado{Error: tools.CodBadArgs + ": la `prioridad` no está en alta/media/baja"}, nil
	}

	actuales, err := store.ListarTodos(o.db, o.projectID, c.SesionID)
	if err != nil {
		return tools.Resultado{}, err
	}
	if estado == "en_progreso" {
		for _, t := range actuales {
			if t.Estado == "en_progreso" {
				return tools.Resultado{Error: tools.CodBadArgs + ": ya hay un paso `en_progreso`; usa `actualizar_todo` para cambiar cuál está en curso"}, nil
			}
		}
	}

	lista, err := store.CrearTodo(o.db, o.projectID, c.SesionID, contenido, estado, prioridad)
	if err != nil {
		return tools.Resultado{}, err
	}
	o.emitirLista(c.SesionID, lista)
	return tools.Resultado{Salida: checklist(lista), Meta: map[string]any{"medida": pluralPasos(len(lista))}}, nil
}

// actualizarTodo es el handler de `actualizar_todo`: sustituye la lista entera y
// devuelve el checklist resultante.
func (o *Orquestador) actualizarTodo(ctx context.Context, args any, c tools.Contexto) (tools.Resultado, error) {
	peticion, ok := args.(tools.PeticionActualizarTodo)
	if !ok {
		return tools.Resultado{Error: tools.CodBadArgs + ": argumentos inesperados"}, nil
	}

	elementos := make([]store.Todo, 0, len(peticion.Elementos))
	enProgreso := 0
	for _, e := range peticion.Elementos {
		contenido := strings.TrimSpace(e.Contenido)
		if contenido == "" {
			return tools.Resultado{Error: tools.CodBadArgs + ": un paso sin `contenido` no es válido"}, nil
		}
		estado := strings.TrimSpace(e.Estado)
		if !estadosTodo[estado] {
			return tools.Resultado{Error: tools.CodBadArgs + ": el `estado` no está en pendiente/en_progreso/completada/cancelada"}, nil
		}
		prioridad := strings.TrimSpace(e.Prioridad)
		if prioridad == "" {
			prioridad = "media"
		}
		if !prioridadesTodo[prioridad] {
			return tools.Resultado{Error: tools.CodBadArgs + ": la `prioridad` no está en alta/media/baja"}, nil
		}
		if estado == "en_progreso" {
			enProgreso++
		}
		elementos = append(elementos, store.Todo{Contenido: contenido, Estado: estado, Prioridad: prioridad})
	}
	if enProgreso > 1 {
		return tools.Resultado{Error: tools.CodBadArgs + ": solo un paso puede estar `en_progreso` a la vez"}, nil
	}

	lista, err := store.ReemplazarTodos(o.db, o.projectID, c.SesionID, elementos)
	if err != nil {
		return tools.Resultado{}, err
	}
	o.emitirLista(c.SesionID, lista)
	return tools.Resultado{Salida: checklist(lista), Meta: map[string]any{"medida": pluralPasos(len(lista))}}, nil
}

// emitirLista publica la lista completa para que el panel la repinte.
func (o *Orquestador) emitirLista(sesionID string, lista []store.Todo) {
	elementos := make([]map[string]string, 0, len(lista))
	for _, t := range lista {
		elementos = append(elementos, map[string]string{"contenido": t.Contenido, "estado": t.Estado})
	}
	datos, _ := json.Marshal(elementos)
	o.emitir(tui.Evento{Sesion: sesionID, Tipo: tui.EventoTodoActualizada, Datos: map[string]string{"elementos": string(datos)}})
}

// checklist compone la lista de pasos como texto, con el glifo de cada estado.
func checklist(lista []store.Todo) string {
	var b strings.Builder
	for i, t := range lista {
		if i > 0 {
			b.WriteString("\n")
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
