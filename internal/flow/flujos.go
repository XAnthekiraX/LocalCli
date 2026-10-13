package flow

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// CargarCarpeta lee los `*.json` de un directorio. Un archivo roto se salta con
// un aviso. Una carpeta inexistente no es un error.
func CargarCarpeta(dir string) (flujos []Flujo, avisos []string) {
	entradas, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, []string{"flujos: no se pudo leer " + dir + ": " + err.Error()}
	}

	for _, nombre := range archivosJSON(entradas) {
		ruta := filepath.Join(dir, nombre)
		datos, err := os.ReadFile(ruta)
		if err != nil {
			avisos = append(avisos, "flujos: no se pudo leer "+ruta+": "+err.Error())
			continue
		}
		f, err := DecodificarFlujo(datos)
		if err != nil {
			avisos = append(avisos, "flujos: se ignora `"+nombre+"`: "+err.Error())
			continue
		}
		f.Origen = ruta
		flujos = append(flujos, f)
	}
	return flujos, avisos
}

func archivosJSON(entradas []os.DirEntry) []string {
	nombres := make([]string, 0, len(entradas))
	for _, e := range entradas {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
			nombres = append(nombres, e.Name())
		}
	}
	sort.Strings(nombres)
	return nombres
}

// CargarAditivo carga los flujos oficiales y los propios. Ante una colisión de
// `comando`, avisa y conserva el oficial.
func CargarAditivo(baseDir, proyectoDir string) (*Catalogo, []string) {
	c := NuevoCatalogo()

	oficiales, avisos := CargarCarpeta(baseDir)
	enBase := make(map[string]bool, len(oficiales))
	for _, f := range oficiales {
		c.Agregar(f)
		enBase[f.Comando] = true
	}

	propios, av := CargarCarpeta(proyectoDir)
	avisos = append(avisos, av...)
	for _, f := range propios {
		existente, ok := c.PorComando(f.Comando)
		if !ok {
			c.Agregar(f)
			continue
		}
		if enBase[f.Comando] {
			avisos = append(avisos, "flujos: el flujo propio `"+f.Comando+"` ("+f.Origen+
				") repite un comando oficial ("+existente.Origen+"); se conserva el oficial")
			continue
		}
		avisos = append(avisos, "flujos: el flujo propio `"+f.Comando+"` ("+f.Origen+
			") repite el comando de otro flujo propio ("+existente.Origen+"); se conserva el primero")
	}
	return c, avisos
}

// Catalogo es el conjunto de flujos disponibles, indexado por `comando`.
type Catalogo struct {
	porComando map[string]Flujo
	orden      []string
}

// NuevoCatalogo devuelve un catálogo vacío.
func NuevoCatalogo() *Catalogo {
	return &Catalogo{porComando: map[string]Flujo{}}
}

// Agregar mete un flujo si su `comando` no estaba ya. Un flujo sin comando no se
// registra.
func (c *Catalogo) Agregar(f Flujo) bool {
	if f.Comando == "" {
		return false
	}
	if _, existe := c.porComando[f.Comando]; existe {
		return false
	}
	c.porComando[f.Comando] = f
	c.orden = append(c.orden, f.Comando)
	return true
}

// PorComando devuelve el flujo registrado con ese comando.
func (c *Catalogo) PorComando(comando string) (Flujo, bool) {
	f, ok := c.porComando[comando]
	return f, ok
}

// PorNombre devuelve el flujo con ese nombre (no con ese comando).
func (c *Catalogo) PorNombre(nombre string) (Flujo, bool) {
	for _, comando := range c.orden {
		if f := c.porComando[comando]; f.Nombre == nombre {
			return f, true
		}
	}
	return Flujo{}, false
}

// Flujos devuelve los flujos en orden de carga.
func (c *Catalogo) Flujos() []Flujo {
	out := make([]Flujo, 0, len(c.orden))
	for _, comando := range c.orden {
		out = append(out, c.porComando[comando])
	}
	return out
}

// Comandos devuelve los comandos en orden de carga.
func (c *Catalogo) Comandos() []string {
	out := make([]string, len(c.orden))
	copy(out, c.orden)
	return out
}

// Len devuelve cuántos flujos tiene el catálogo.
func (c *Catalogo) Len() int { return len(c.orden) }
