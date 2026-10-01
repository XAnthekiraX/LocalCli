package agent

// loader.go — T-B034-05: cargar un agente desde su CARPETA.
//
// Fuente de verdad: ai/docs/specs/SPEC-AGENTE-BASE.md §Dónde se definen (cada
// agente es una carpeta con `agent.yaml` y `prompt.md`; los dos obligatorios) y
// ai/docs/backend/04-infrastructure/CONFIGURATION.md §Agentes del proyecto.
//
// Los agentes son datos: este módulo solo los lee. Cada error de carga nombra la
// carpeta, para que un YAML roto no se confunda con otro. El nombre del agente
// sale del campo `name`, no del nombre de la carpeta. El `prompt.md` se lee
// íntegro, sin recortar, y un archivo vacío hace que el agente no cargue.

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"localcli/internal/tools"
)

// Los dos archivos de una carpeta de agente.
const (
	ArchivoConfig = "agent.yaml"
	ArchivoPrompt = "prompt.md"
)

// Cargar lee y valida un agente desde una carpeta: `agent.yaml` y `prompt.md`
// son los dos obligatorios. El nombre del agente sale de `name`, no del nombre
// de la carpeta; el `prompt.md` se lee íntegro, sin recortar, y un archivo vacío
// hace que el agente no cargue.
func Cargar(dir string) (Agente, error) {
	rutaConfig := filepath.Join(dir, ArchivoConfig)
	datos, err := os.ReadFile(rutaConfig)
	if err != nil {
		return Agente{}, errAgente("no se pudo leer " + rutaConfig + ": " + err.Error())
	}
	a, err := DecodificarAgente(datos)
	if err != nil {
		return Agente{}, errAgente(dir + ": " + err.Error())
	}
	rutaPrompt := filepath.Join(dir, ArchivoPrompt)
	prompt, err := os.ReadFile(rutaPrompt)
	if err != nil {
		return Agente{}, errAgente(dir + ": falta `" + ArchivoPrompt + "`: " + err.Error())
	}
	if strings.TrimSpace(string(prompt)) == "" {
		return Agente{}, errAgente(dir + ": `" + ArchivoPrompt + "` está vacío")
	}
	a.Prompt = string(prompt)
	return a, nil
}

// CargarCarpeta lee todos los agentes de un directorio de agentes: cada
// SUBcarpeta es un agente, en orden alfabético. Un archivo suelto se ignora (el
// arranque avisa aparte de los `*.json` del formato antiguo). Un agente roto
// detiene la carga: no se arranca con un catálogo a medias sin avisar.
func CargarCarpeta(dir string) ([]Agente, error) {
	entradas, err := os.ReadDir(dir)
	if err != nil {
		return nil, errAgente("no se pudo leer la carpeta de agentes " + dir + ": " + err.Error())
	}
	nombres := make([]string, 0, len(entradas))
	for _, e := range entradas {
		if !e.IsDir() {
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
