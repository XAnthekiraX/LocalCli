package agent

// loader.go — T-B006-02: cargar los JSON de `ai/agents/*.json` validando
// estructura y tipos.
//
// Fuente de verdad: ai/docs/specs/SPEC-AGENTE-BASE.md §Dónde se definen (cada
// agente es un `ai/agents/*.json`) y ai/docs/backend/01-domain/DOMAIN.md ("El
// agente base existe, pero no está hardcodeado. No está en Go: vive en un
// JSON"). Los agentes son datos: este módulo solo los lee.
//
// Cada error de carga nombra el archivo, para que un JSON roto no se confunda
// con otro. La estructura y los tipos los impone DecodificarAgente (que además
// deriva el catálogo efectivo); los permisos los valida ValidarPermisos.

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"localcli/internal/tools"
)

// Cargar lee y valida un agente desde una ruta concreta.
func Cargar(ruta string) (Agente, error) {
	datos, err := os.ReadFile(ruta)
	if err != nil {
		return Agente{}, errAgente("no se pudo leer " + ruta + ": " + err.Error())
	}
	a, err := DecodificarAgente(datos)
	if err != nil {
		return Agente{}, errAgente(ruta + ": " + err.Error())
	}
	if err := ValidarPermisos(a); err != nil {
		return Agente{}, errAgente(ruta + ": " + err.Error())
	}
	return a, nil
}

// CargarCarpeta lee todos los `*.json` de un directorio, en orden alfabético.
// Un archivo roto detiene la carga: no se arranca con un catálogo de agentes a
// medias sin avisar.
func CargarCarpeta(dir string) ([]Agente, error) {
	entradas, err := os.ReadDir(dir)
	if err != nil {
		return nil, errAgente("no se pudo leer la carpeta de agentes " + dir + ": " + err.Error())
	}
	nombres := make([]string, 0, len(entradas))
	for _, e := range entradas {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		nombres = append(nombres, e.Name())
	}
	sort.Strings(nombres)

	agentes := make([]Agente, 0, len(nombres))
	for _, nombre := range nombres {
		a, err := Cargar(filepath.Join(dir, nombre))
		if err != nil {
			return nil, err
		}
		agentes = append(agentes, a)
	}
	return agentes, nil
}

// PorNombre devuelve el agente con ese nombre exacto. El segundo valor es
// false si no está: el relevo `plan`→`build` falla si falta cualquiera de los
// dos, en vez de seguir con un agente a medias.
func PorNombre(agentes []Agente, nombre string) (Agente, bool) {
	for _, a := range agentes {
		if a.Nombre == nombre {
			return a, true
		}
	}
	return Agente{}, false
}

// OrdenarNombres devuelve los nombres de un catálogo de agentes en un orden
// estable para presentarlos: `plan` y `build` primero —son los agentes base y
// los que arrancan los flujos por defecto— y el resto alfabético. Un catálogo
// vacío devuelve una lista vacía.
func OrdenarNombres(agentes map[string]Agente) []string {
	primeros := []string{tools.AgentePlan, tools.AgenteBuild}
	out := make([]string, 0, len(agentes))
	for _, n := range primeros {
		if _, ok := agentes[n]; ok {
			out = append(out, n)
		}
	}
	resto := make([]string, 0, len(agentes))
	for n := range agentes {
		if n == tools.AgentePlan || n == tools.AgenteBuild {
			continue
		}
		resto = append(resto, n)
	}
	sort.Strings(resto)
	return append(out, resto...)
}
