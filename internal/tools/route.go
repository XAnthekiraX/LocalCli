package tools

// route.go — T-B007-06 y T-B007-07: enrutar una petición a su destino.
//
// Fuente de verdad: ai/docs/backend/02-interfaces/TOOLS.md §7 (los pasos del
// enrutado) y ai/docs/backend/04-infrastructure/CONFIGURATION.md §2
// (`LOCALCLI_ALLOW_INTERNET`, desactivado por defecto).
//
// El orden importa y es el documentado: primero se comprueba que la
// herramienta existe y que el agente la tiene, después se validan los
// argumentos, y solo entonces se enruta. Una petición rechazada no llega nunca
// al handler.

import (
	"context"
	"os"
	"strings"
)

// VarInternet es la variable que habilita las herramientas de internet.
// Desactivado no es un descuido: es la única integración que saca información
// de la máquina, así que el usuario la habilita a propósito (SECURITY.md §4).
const VarInternet = "LOCALCLI_ALLOW_INTERNET"

// InternetPermitida informa si el usuario habilitó las herramientas de
// internet. Cualquier valor de la lista se considera un sí; ausente o
// cualquier otra cosa, no. La comparación es insensible a mayúsculas.
func InternetPermitida() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(VarInternet))) {
	case "1", "true", "yes", "on", "si", "sí":
		return true
	}
	return false
}

// Enrutar ejecuta el pipeline completo de `tools`: comprobar permiso, validar
// el payload y despachar al destino de la categoría. Es lo que llama `agent`.
//
// `tools` no aplica el permiso concreto: `fileops` y `exec` reciben la
// petición ya filtrada y deciden si el efecto se aplica.
func (r *Registro) Enrutar(ctx context.Context, p Peticion) (any, error) {
	h, ok := Buscar(p.Herramienta)
	if !ok {
		return nil, nuevoError(CodigoHerramientaDesconocida,
			"la herramienta "+p.Herramienta+" no está en el catálogo cerrado")
	}
	if err := ComprobarPermiso(p.Permitidas, p.Herramienta); err != nil {
		return nil, err
	}
	if err := Validar(p.Herramienta, p.Argumentos); err != nil {
		return nil, err
	}
	// Las de internet son las únicas que salen de la máquina: sin la variable,
	// ni siquiera se llega al cliente.
	if h.Categoria == CatInternet && !InternetPermitida() {
		return nil, nuevoError(CodigoHerramientaNoPermitida,
			"las herramientas de internet están desactivadas; habilítalas con "+
				VarInternet+"=1")
	}
	destino, err := r.Handler(p.Herramienta)
	if err != nil {
		return nil, err
	}
	return destino(ctx, p)
}
