package flow

// catalogo.go — el conjunto de flujos disponibles, indexados por su comando.
//
// Fuente de verdad: [[specs/SPEC-MOTOR-FLUJOS]] ("un flujo no arranca solo: lo
// solicita el usuario con un comando explícito") y
// [[specs/SPEC-FLUJO-PERSONALIZADO]]. Los flujos del proyecto son **los que
// declara `.localcli/flows/*.json`**: el catálogo los carga al arrancar y no
// registra ninguno cableado. Un comando sin archivo no existe y no se lista.

import (
	"path/filepath"
	"strings"
)

// comandoEjecutar es el único comando que no corre etapas: consume la cola del
// TODO (SPEC-COLA-TAREAS). No tiene flujo propio.
const comandoEjecutar = "/ejecutar"

// Catalogo es el conjunto de flujos disponibles en el proyecto.
type Catalogo struct {
	porComando map[string]Flujo
	orden      []string // comandos en orden de registro
}

// CatalogoPorDefecto devuelve un catálogo vacío. Los flujos son archivos, no
// una lista cableada, así que no hay oficiales que registrar; se mantiene para
// quien necesita un catálogo sin proyecto y como respaldo del arranque.
func CatalogoPorDefecto() *Catalogo {
	return &Catalogo{porComando: map[string]Flujo{}}
}

// CargarFlujos carga los flujos del proyecto desde `.localcli/flows/*.json` y
// los indexa por comando. Devuelve error si algún JSON está roto; quien llame
// decide si avisa y arranca con el catálogo vacío.
func CargarFlujos(raiz string) (*Catalogo, error) {
	c := CatalogoPorDefecto()
	flujos, err := CargarFlujosCarpeta(filepath.Join(raiz, ".localcli", "flows"))
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

// PorComando devuelve el flujo registrado con ese comando (p. ej. "/crear"). El
// segundo valor es false si no hay ninguno: lo usa la cola para resolver el
// flujo de un elemento por su acción, y quien llama decide qué hacer sin flujo
// en vez de recibir uno vacío.
func (c *Catalogo) PorComando(comando string) (Flujo, bool) {
	f, ok := c.porComando[comando]
	return f, ok
}

// Flujos devuelve los flujos del catálogo, en orden de registro.
func (c *Catalogo) Flujos() []Flujo {
	out := make([]Flujo, 0, len(c.orden))
	for _, comando := range c.orden {
		out = append(out, c.porComando[comando])
	}
	return out
}
