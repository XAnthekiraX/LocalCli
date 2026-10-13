package tools

import "strings"

// HerramientaUsuario es una herramienta declarada por el usuario en un `*.json`.
// No es código: LocalCli la ejecuta con su propio ejecutor.
type HerramientaUsuario struct {
	Nombre          string   `json:"nombre"`
	Descripcion     string   `json:"descripcion"`
	Modo            Modo     `json:"modo"`
	Equipo          []string `json:"equipo"`
	TimeoutSegundos int      `json:"timeout_segundos,omitempty"`
}

// ComoHerramienta proyecta la declaración al catálogo. Una herramienta del
// usuario siempre es de lectura (`modo: lee`), así que entra como permiso `read`.
func (u HerramientaUsuario) ComoHerramienta() Herramienta {
	modo := u.Modo
	if modo == "" {
		modo = ModoLee
	}
	return Herramienta{
		Nombre:      u.Nombre,
		Descripcion: u.Descripcion,
		Categoria:   CategoriaUsuario,
		Permiso:     PermisoRead,
		Modo:        modo,
		Verbo:       strings.ToUpper(u.Nombre),
		// El contrato es un objeto genérico: no hay tipo de petición que derive.
		TipoPeticion: nil,
	}
}

// ValidaDeclaracion comprueba los campos obligatorios. `modo: escribe` se rechaza.
func (u HerramientaUsuario) ValidaDeclaracion() string {
	if strings.TrimSpace(u.Nombre) == "" {
		return "falta el campo obligatorio `nombre`"
	}
	if strings.TrimSpace(u.Descripcion) == "" {
		return "falta el campo obligatorio `descripcion`"
	}
	if len(u.Equipo) == 0 {
		return "falta el campo obligatorio `equipo` (no vacío)"
	}
	if u.Modo == ModoEscribe {
		return "`modo: escribe` se rechaza: la única vía sancionada de escritura son las herramientas de archivo"
	}
	if u.Modo != "" && u.Modo != ModoLee {
		return "`modo` desconocido: " + string(u.Modo)
	}
	return ""
}
