package flow

// comando.go — T-B016-02 y T-B016-03: reconocer los comandos explícitos que
// arrancan un flujo y proponer trabajo ordenado sin ejecutarlo.
//
// Fuente de verdad: [[specs/SPEC-INTERFAZ]] §Reglas de negocio ("Un flujo solo
// arranca con un comando explícito escrito en la entrada: /planificar, /crear,
// /actualizar, /eliminar, /resolver o /ejecutar") y [[specs/SPEC-COLA-TAREAS]]
// ("Al detectar una petición ordenada, el sistema propone el TODO; el usuario
// decide si se ejecuta").
//
// Un comando es una línea que empieza por `/`. Lo que no es comando se responde
// en el chat: `flow` no arranca etapas por su cuenta.

import (
	"strings"

	"localcli/internal/task"
)

// Comando es un comando explícito que arranca un flujo. `Consumir` distingue
// `/ejecutar`, que no corre una secuencia de etapas sino que consume la cola
// del TODO (SPEC-COLA-TAREAS).
type Comando struct {
	Nombre   string
	Flujo    Flujo
	Consumir bool
}

// ComandoDe reconoce un comando explícito al inicio de la línea. Devuelve el
// comando y true solo para los seis nombres de la spec; cualquier otro texto
// (incluido otro `/…`) es chat.
func ComandoDe(texto string) (Comando, bool) {
	campos := strings.Fields(strings.TrimSpace(texto))
	if len(campos) == 0 {
		return Comando{}, false
	}
	switch campos[0] {
	case "/planificar":
		return Comando{Nombre: campos[0], Flujo: FlujoPlanificacion()}, true
	case "/crear":
		return Comando{Nombre: campos[0], Flujo: FlujoTrabajo(task.AccionCrear)}, true
	case "/actualizar":
		return Comando{Nombre: campos[0], Flujo: FlujoTrabajo(task.AccionActualizar)}, true
	case "/eliminar":
		return Comando{Nombre: campos[0], Flujo: FlujoTrabajo(task.AccionEliminar)}, true
	case "/resolver":
		return Comando{Nombre: campos[0], Flujo: FlujoResolver()}, true
	case "/ejecutar":
		return Comando{Nombre: campos[0], Consumir: true}, true
	}
	return Comando{}, false
}

// ObjetivoDe quita el comando y devuelve el resto de la línea, que es el
// objetivo que recibe el flujo. Sin resto —por ejemplo `/planificar` a secas—
// el objetivo es el nombre del comando: nunca se entrega un objetivo vacío.
func ObjetivoDe(texto string) string {
	t := strings.TrimSpace(texto)
	i := strings.IndexAny(t, " \t\n")
	if i < 0 {
		return t
	}
	if resto := strings.TrimSpace(t[i+1:]); resto != "" {
		return resto
	}
	return t
}

// SugerenciaTrabajo propone ejecutar un flujo cuando la petición parece trabajo
// ordenado, sin arrancarlo. Devuelve el aviso y true solo si hay algo que
// proponer; un comando explícito ya no es una sugerencia.
func SugerenciaTrabajo(texto string) (string, bool) {
	if _, ok := ComandoDe(texto); ok {
		return "", false
	}
	if !EsTrabajoOrdenado(texto) {
		return "", false
	}
	return "Esta petición parece trabajo ordenado. Se responde como chat; escribe `/ejecutar` para ejecutar la cola.", true
}
