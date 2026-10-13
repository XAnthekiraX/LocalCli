package agent

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Cargar lee y valida un agente de su carpeta. `agent.yaml` y `prompt.md` son
// obligatorios; un prompt vacío no carga.
func Cargar(dir string) (Agente, error) {
	datos, err := os.ReadFile(filepath.Join(dir, ArchivoConfig))
	if err != nil {
		return Agente{}, errorf("falta `%s` en %s: %w", ArchivoConfig, dir, err)
	}
	a, err := DecodificarAgente(datos)
	if err != nil {
		return Agente{}, errorf("%s: %w", dir, err)
	}
	prompt, err := os.ReadFile(filepath.Join(dir, ArchivoPrompt))
	if err != nil {
		return Agente{}, errorf("falta `%s` en %s: %w", ArchivoPrompt, dir, err)
	}
	if strings.TrimSpace(string(prompt)) == "" {
		return Agente{}, errorf("`%s` está vacío en %s", ArchivoPrompt, dir)
	}
	a.Prompt = string(prompt)
	a.Origen = dir
	return a, nil
}

// CargarCarpeta carga cada subcarpeta como un agente. Una carpeta inválida se
// salta con un aviso, igual que un `*.json` suelto (formato antiguo). Una
// carpeta inexistente no es un error.
func CargarCarpeta(dir string) (agentes []Agente, avisos []string) {
	entradas, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, []string{"agentes: no se pudo leer " + dir + ": " + err.Error()}
	}

	for _, e := range entradas {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
			avisos = append(avisos, "agentes: `"+e.Name()+"` usa el formato antiguo; "+
				"conviértelo a una carpeta con `"+ArchivoConfig+"` y `"+ArchivoPrompt+"`")
		}
	}

	for _, nombre := range subcarpetas(entradas) {
		a, err := Cargar(filepath.Join(dir, nombre))
		if err != nil {
			avisos = append(avisos, "agentes: se ignora la carpeta `"+nombre+"`: "+err.Error())
			continue
		}
		agentes = append(agentes, a)
	}
	return agentes, avisos
}

func subcarpetas(entradas []os.DirEntry) []string {
	nombres := make([]string, 0, len(entradas))
	for _, e := range entradas {
		if e.IsDir() {
			nombres = append(nombres, e.Name())
		}
	}
	sort.Strings(nombres)
	return nombres
}

// CargarAditivo carga los agentes base y los propios. Ante una colisión de
// `name`, avisa y conserva el base.
func CargarAditivo(baseDir, proyectoDir string) (*Catalogo, []string) {
	c := NuevoCatalogo()

	base, avisos := CargarCarpeta(baseDir)
	enBase := make(map[string]bool, len(base))
	for _, a := range base {
		c.Agregar(a)
		enBase[a.Nombre] = true
	}

	propios, av := CargarCarpeta(proyectoDir)
	avisos = append(avisos, av...)
	for _, a := range propios {
		existente, ok := c.Buscar(a.Nombre)
		if !ok {
			c.Agregar(a)
			continue
		}
		if enBase[a.Nombre] {
			avisos = append(avisos, "agentes: el agente propio `"+a.Nombre+"` ("+a.Origen+
				") repite el nombre de un agente base ("+existente.Origen+"); se conserva el base")
			continue
		}
		avisos = append(avisos, "agentes: el agente propio `"+a.Nombre+"` ("+a.Origen+
			") repite el nombre de otro agente propio ("+existente.Origen+"); se conserva el primero")
	}
	return c, avisos
}

// Catalogo es el conjunto de agentes disponibles, indexado por `name`.
type Catalogo struct {
	porNombre map[string]Agente
	orden     []string
}

// NuevoCatalogo devuelve un catálogo vacío.
func NuevoCatalogo() *Catalogo {
	return &Catalogo{porNombre: map[string]Agente{}}
}

// Agregar mete un agente si su `name` no estaba ya. Devuelve false si lo repite.
func (c *Catalogo) Agregar(a Agente) bool {
	if _, existe := c.porNombre[a.Nombre]; existe {
		return false
	}
	c.porNombre[a.Nombre] = a
	c.orden = append(c.orden, a.Nombre)
	return true
}

// Buscar devuelve el agente con ese `name` exacto.
func (c *Catalogo) Buscar(nombre string) (Agente, bool) {
	a, ok := c.porNombre[nombre]
	return a, ok
}

// Nombres devuelve los nombres en orden de carga.
func (c *Catalogo) Nombres() []string {
	out := make([]string, len(c.orden))
	copy(out, c.orden)
	return out
}

// Agentes devuelve los agentes en orden de carga.
func (c *Catalogo) Agentes() []Agente {
	out := make([]Agente, 0, len(c.orden))
	for _, nombre := range c.orden {
		out = append(out, c.porNombre[nombre])
	}
	return out
}

// Len devuelve cuántos agentes tiene el catálogo.
func (c *Catalogo) Len() int { return len(c.orden) }
