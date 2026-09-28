package flow

// catalogo.go — el conjunto de flujos disponibles, indexados por su comando.
//
// Fuente de verdad: [[specs/SPEC-MOTOR-FLUJOS]] ("los flujos oficiales vienen
// con la herramienta y funcionan sin configuración") y
// [[specs/SPEC-FLUJO-PERSONALIZADO]] ("un flujo propio se define con nombre,
// etapas y orden; solo se ve dentro de su proyecto"). Los oficiales se arman
// con los constructores de este paquete; los propios —y las personalizaciones
// de los oficiales— se cargan de `ai/flows/*.json` (flujo.go) y se aplican
// encima por `comando`: mismo comando sobreescribe, comando nuevo añade.

import (
	"path/filepath"
	"strings"

	"localcli/internal/task"
)

// comandoEjecutar es el único comando que no corre etapas: consume la cola del
// TODO (SPEC-COLA-TAREAS). No tiene flujo propio.
const comandoEjecutar = "/ejecutar"

// Catalogo es el conjunto de flujos disponibles en el proyecto.
type Catalogo struct {
	porComando map[string]Flujo
	orden      []string // comandos en orden de registro
}

// CatalogoPorDefecto arma el catálogo con los flujos oficiales. Es el respaldo
// cuando el proyecto no tiene `ai/flows/`: los flujos oficiales funcionan sin
// configuración.
func CatalogoPorDefecto() *Catalogo {
	c := &Catalogo{porComando: map[string]Flujo{}}
	c.registrar(FlujoPlanificacion())
	c.registrar(FlujoTrabajo(task.AccionCrear))
	c.registrar(FlujoTrabajo(task.AccionActualizar))
	c.registrar(FlujoTrabajo(task.AccionEliminar))
	c.registrar(FlujoResolver())
	return c
}

// CargarFlujos carga los flujos del proyecto: parte de los oficiales y aplica
// encima los `ai/flows/*.json`. Devuelve error si algún JSON está roto; quien
// llame decide si cae al catálogo por defecto (el arranque lo hace y avisa).
func CargarFlujos(raiz string) (*Catalogo, error) {
	c := CatalogoPorDefecto()
	flujos, err := CargarFlujosCarpeta(filepath.Join(raiz, "ai", "flows"))
	if err != nil {
		return nil, err
	}
	for _, f := range flujos {
		c.registrar(f)
	}
	return c, nil
}

// registrar mete o reemplaza un flujo por su comando. Un flujo sin comando no
// se registra: no hay forma de arrancarlo.
func (c *Catalogo) registrar(f Flujo) {
	if f.Comando == "" {
		return
	}
	if _, ok := c.porComando[f.Comando]; !ok {
		c.orden = append(c.orden, f.Comando)
	}
	c.porComando[f.Comando] = f
}

// De reconoce un comando explícito al inicio de la línea. Devuelve el comando
// y true solo si su nombre está en el catálogo (o si es `/ejecutar`); cualquier
// otro texto —incluido otro `/…`— es chat.
func (c *Catalogo) De(texto string) (Comando, bool) {
	campos := strings.Fields(strings.TrimSpace(texto))
	if len(campos) == 0 {
		return Comando{}, false
	}
	if campos[0] == comandoEjecutar {
		return Comando{Nombre: comandoEjecutar, Consumir: true}, true
	}
	if f, ok := c.porComando[campos[0]]; ok {
		return Comando{Nombre: campos[0], Flujo: f}, true
	}
	return Comando{}, false
}

// PorNombre devuelve el flujo con ese nombre (no con ese comando). El segundo
// valor es false si no está.
func (c *Catalogo) PorNombre(nombre string) (Flujo, bool) {
	for _, comando := range c.orden {
		if f := c.porComando[comando]; f.Nombre == nombre {
			return f, true
		}
	}
	return Flujo{}, false
}

// Flujos devuelve los flujos del catálogo, en orden de registro.
func (c *Catalogo) Flujos() []Flujo {
	out := make([]Flujo, 0, len(c.orden))
	for _, comando := range c.orden {
		out = append(out, c.porComando[comando])
	}
	return out
}
