package tools

// route.go — el interruptor de las herramientas de internet.
//
// Fuente de verdad: ai/docs/backend/04-infrastructure/CONFIGURATION.md §2
// (`LOCALCLI_ALLOW_INTERNET`, desactivado por defecto) y
// ai/docs/backend/03-security/SECURITY.md §4 (es la única integración que saca
// información de la máquina).
//
// El enrutado por categorías desapareció (TOOLS.md §7): cada herramienta trae
// su handler y la capa universal los ejecuta. Lo que queda aquí es el control
// que no es de ninguna herramienta: sin la variable, ninguna petición de
// internet llega a salir de la máquina.

import (
	"os"
	"strings"
)

// VarInternet es la variable que habilita las herramientas de internet.
const VarInternet = "LOCALCLI_ALLOW_INTERNET"

// InternetPermitida informa si el usuario habilitó las herramientas de
// internet. Cualquier valor de la lista se considera un sí; ausente o cualquier
// otra cosa, no. La comparación es insensible a mayúsculas.
func InternetPermitida() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(VarInternet))) {
	case "1", "true", "yes", "on", "si", "sí":
		return true
	}
	return false
}
