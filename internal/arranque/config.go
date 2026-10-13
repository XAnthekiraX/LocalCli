package arranque

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// APIProveedorPorDefecto es la dirección base que se usa si el archivo no la trae.
const APIProveedorPorDefecto = "http://localhost:8080"

// LimiteContextoPorDefecto es la ventana de contexto cuando el modelo no declara
// la suya y no hay override del usuario.
const LimiteContextoPorDefecto = 16384

// LimiteContextoOverride devuelve el valor de `LOCALCLI_CONTEXT_LIMIT`, o 0 si no
// está definido o no es válido. El 0 significa «sin override»: la ventana sale
// entonces del contexto real del modelo y, si no lo declara, del defecto.
func LimiteContextoOverride() int {
	valor := strings.TrimSpace(os.Getenv("LOCALCLI_CONTEXT_LIMIT"))
	if valor == "" {
		return 0
	}
	n, err := strconv.Atoi(valor)
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

// Configuracion es la configuración global que lee el arranque.
type Configuracion struct {
	Proveedor string `json:"proveedor"`
	API       string `json:"api"`
}

// LeerConfiguracion lee `~/.config/localcli/localcli.json`. Si no está o es
// ilegible, devuelve la dirección por defecto.
func LeerConfiguracion() (Configuracion, error) {
	raiz, err := RaizMaquina()
	if err != nil {
		return Configuracion{}, err
	}
	datos, err := os.ReadFile(filepath.Join(raiz, "localcli.json"))
	if os.IsNotExist(err) {
		return Configuracion{API: APIProveedorPorDefecto}, nil
	}
	if err != nil {
		return Configuracion{}, fmt.Errorf("arranque: no se pudo leer localcli.json: %w", err)
	}
	var c Configuracion
	if err := json.Unmarshal(datos, &c); err != nil {
		return Configuracion{}, fmt.Errorf("arranque: localcli.json inválido: %w", err)
	}
	if c.API == "" {
		c.API = APIProveedorPorDefecto
	}
	return c, nil
}
