// Package context decide qué parte de la documentación del proyecto se entrega a
// una etapa: deja que el modelo elija, carga solo lo elegido, lo recorta y audita
// qué entró y qué salió. La lectura de la documentación y su grafo vienen de fuera
// (módulo `docs`) por una fuente inyectada.
package context

import "context"

// CodDocNotFound es el código de un documento que el modelo pidió y no existe.
const CodDocNotFound = "E_DOC_NOT_FOUND"

// Documento es una entrada del grafo: su ruta y las dependencias que declara.
type Documento struct {
	Ruta         string
	Dependencias []string
}

// Fuente entrega la documentación del proyecto. La implementación real vive en el
// módulo de documentación; aquí solo se consume.
type Fuente interface {
	Documentos() []Documento
	Leer(ruta string) (string, error)
}

// Modelo decide qué documentos son relevantes para un objetivo.
type Modelo interface {
	Elegir(ctx context.Context, objetivo string, catalogo []string) ([]string, error)
}

// Peticion pide el contexto de una etapa.
type Peticion struct {
	Sesion   string
	Etapa    string
	Objetivo string
	// SinSemilla marca el chat sin semilla: su bloque va solo con el objetivo y
	// no consulta al modelo.
	SinSemilla bool
	// YaEntregados son los documentos de etapas anteriores de la misma ejecución.
	YaEntregados []string
}

// Bloque es el contexto armado para una etapa.
type Bloque struct {
	Objetivo   string
	Documentos []DocumentoContenido
	SinSemilla bool
}

// DocumentoContenido es un documento ya leído y recortado.
type DocumentoContenido struct {
	Ruta      string
	Contenido string
	Truncado  bool
}

// Nodo arma el contexto de una etapa.
type Nodo struct {
	Fuente      Fuente
	Modelo      Modelo
	Auditoria   Auditoria
	Presupuesto int
	Evento      func(sesion, etapa string)
	Aviso       func(motivo string)
}

// Preparar arma el bloque de una etapa siguiendo el flujo documentado.
func (n *Nodo) Preparar(ctx context.Context, p Peticion) (Bloque, error) {
	if p.SinSemilla {
		n.registrar(p, "(objetivo)", Incluido, "chat sin semilla: solo el objetivo", 0)
		n.emitir(p)
		return Bloque{Objetivo: p.Objetivo, SinSemilla: true}, nil
	}

	catalogo := n.Fuente.Documentos()
	indice := indexar(catalogo)
	bloque := Bloque{Objetivo: p.Objetivo}
	restante := n.presupuesto()
	yaEntregados := conjunto(p.YaEntregados)

	for _, ruta := range n.elegir(ctx, p.Objetivo, nombres(catalogo)) {
		if yaEntregados[ruta] {
			continue
		}
		if _, ok := indice[ruta]; !ok {
			return Bloque{}, errorf("%s: el modelo pidió un documento que no existe: %s", CodDocNotFound, ruta)
		}
		contenido, err := n.Fuente.Leer(ruta)
		if err != nil {
			return Bloque{}, errorf("%s: no se pudo leer %s", CodDocNotFound, ruta)
		}
		recortado, truncado, tokens := RecortarPorSeccion(contenido, restante)
		if tokens == 0 {
			n.registrar(p, ruta, Descartado, "no cabe en el presupuesto de contexto", 0)
			continue
		}
		restante -= tokens
		bloque.Documentos = append(bloque.Documentos, DocumentoContenido{Ruta: ruta, Contenido: recortado, Truncado: truncado})
		n.registrar(p, ruta, Incluido, "", tokens)
	}

	n.avisarDocumentosSinDependencias(catalogo)
	n.emitir(p)
	return bloque, nil
}

func (n *Nodo) presupuesto() int {
	if n.Presupuesto > 0 {
		return n.Presupuesto
	}
	return presupuestoDefecto
}

func indexar(documentos []Documento) map[string]Documento {
	indice := make(map[string]Documento, len(documentos))
	for _, d := range documentos {
		indice[d.Ruta] = d
	}
	return indice
}

func nombres(documentos []Documento) []string {
	nombres := make([]string, 0, len(documentos))
	for _, d := range documentos {
		nombres = append(nombres, d.Ruta)
	}
	return nombres
}

func conjunto(valores []string) map[string]bool {
	c := make(map[string]bool, len(valores))
	for _, v := range valores {
		c[v] = true
	}
	return c
}
