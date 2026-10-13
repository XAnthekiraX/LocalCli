package agent

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

// Archivos de una carpeta de agente.
const (
	ArchivoConfig = "agent.yaml"
	ArchivoPrompt = "prompt.md"
)

// Permisos admitidos.
const (
	PermisoRead  = "read"
	PermisoWrite = "write"
	PermisoEdit  = "edit"
)

// Efectos de un permiso.
const (
	EfectoAllow = "allow"
	EfectoDeny  = "deny"
)

// ClaveDefault solo admite `deny`.
const ClaveDefault = "default"

// CamposDelAgente son los campos admitidos en `agent.yaml`.
var CamposDelAgente = []string{"name", "description", "permissions"}

// PermisosDelAgente es el vocabulario cerrado de permisos.
var PermisosDelAgente = []string{PermisoRead, PermisoWrite, PermisoEdit}

// Agente es una definición cargada desde una carpeta de `agents/`.
type Agente struct {
	Nombre      string            `yaml:"name"`
	Descripcion string            `yaml:"description"`
	Permissions map[string]string `yaml:"permissions,omitempty"`
	Prompt      string            `yaml:"-"` // se rellena leyendo prompt.md
	Origen      string            `yaml:"-"` // carpeta de la que se cargó
}

// SoloConversacion indica si el agente no concede ningún permiso.
func (a Agente) SoloConversacion() bool { return len(a.Permisos()) == 0 }

// Permisos devuelve los permisos concedidos (`allow`), en orden del vocabulario.
func (a Agente) Permisos() []string {
	var out []string
	for _, p := range PermisosDelAgente {
		if strings.TrimSpace(a.Permissions[p]) == EfectoAllow {
			out = append(out, p)
		}
	}
	return out
}

// DecodificarAgente lee el YAML con campos estrictos y valida identidad y
// permisos. No exige `prompt.md`.
func DecodificarAgente(datos []byte) (Agente, error) {
	var a Agente
	dec := yaml.NewDecoder(bytes.NewReader(datos))
	dec.KnownFields(true)
	if err := dec.Decode(&a); err != nil {
		return Agente{}, errorf("no se pudo leer %s: %w", ArchivoConfig, err)
	}
	var sobra yaml.Node
	if err := dec.Decode(&sobra); err != io.EOF {
		return Agente{}, errorf("%s lleva contenido de más después del objeto", ArchivoConfig)
	}
	if err := a.validar(); err != nil {
		return Agente{}, err
	}
	return a, nil
}

func (a Agente) validar() error {
	if strings.TrimSpace(a.Nombre) == "" {
		return errorf("falta el campo obligatorio `name`")
	}
	return validarPermisos(a.Permissions)
}

func validarPermisos(permissions map[string]string) error {
	for clave, bruto := range permissions {
		efecto := strings.TrimSpace(bruto)
		if clave == ClaveDefault {
			if efecto != EfectoDeny {
				return errorf("`default` solo admite `deny`")
			}
			continue
		}
		if !permisoConocido(clave) {
			return errorf("permiso desconocido: %s", clave)
		}
		if efecto != EfectoAllow && efecto != EfectoDeny {
			return errorf("efecto desconocido para `%s`: %s", clave, bruto)
		}
	}
	return nil
}

func permisoConocido(clave string) bool {
	for _, p := range PermisosDelAgente {
		if clave == p {
			return true
		}
	}
	return false
}

func errorf(format string, args ...any) error {
	return fmt.Errorf("agente: "+format, args...)
}
