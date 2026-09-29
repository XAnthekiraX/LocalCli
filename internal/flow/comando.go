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
)

// Comando es un comando explícito que arranca un flujo. `Consumir` distingue
// `/ejecutar`, que no corre una secuencia de etapas sino que consume la cola
// del TODO (SPEC-COLA-TAREAS).
type Comando struct {
	Nombre   string
	Flujo    Flujo
	Consumir bool
}

// Objetivo devuelve la petición que acompaña al comando. Sin texto, usa la
// `peticion` declarada por el flujo o, en su defecto, el nombre del comando:
// nunca se entrega un objetivo vacío.
func (c Comando) Objetivo(texto string) string {
	if resto := quitarComando(texto); resto != "" {
		return resto
	}
	if c.Flujo.Peticion != "" {
		return c.Flujo.Peticion
	}
	return c.Nombre
}

// ObjetivoDe quita el comando y devuelve el resto de la línea, que es el
// objetivo que recibe el flujo. Sin resto —por ejemplo `/planificar` a secas—
// el objetivo es el nombre del comando.
func ObjetivoDe(texto string) string {
	t := strings.TrimSpace(texto)
	if resto := quitarComando(t); resto != "" {
		return resto
	}
	return t
}

// quitarComando devuelve el texto que sigue al primer campo (el comando), sin
// espacios. Vacío si la línea es solo el comando.
func quitarComando(texto string) string {
	t := strings.TrimSpace(texto)
	i := strings.IndexAny(t, " \t\n")
	if i < 0 {
		return ""
	}
	return strings.TrimSpace(t[i+1:])
}

// SugerenciaTrabajo propone ejecutar un flujo cuando la petición parece trabajo
// ordenado, sin arrancarlo. Devuelve el aviso y true solo si hay algo que
// proponer; una línea que empieza por `/` es un comando explícito, no una
// sugerencia.
func SugerenciaTrabajo(texto string) (string, bool) {
	if esComando(texto) {
		return "", false
	}
	if !EsTrabajoOrdenado(texto) {
		return "", false
	}
	return "Esta petición parece trabajo ordenado. Se responde como chat; escribe `/ejecutar` para ejecutar la cola.", true
}

// esComando dice si la línea empieza por un comando (`/…`). No consulta el
// catálogo: qué comandos existen lo sabe el motor, que es quien lo carga de
// `.localcli/flows/`.
func esComando(texto string) bool {
	return strings.HasPrefix(strings.TrimSpace(texto), "/")
}
