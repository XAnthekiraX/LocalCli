package context

// nodo.go — T-B011: el nodo de contexto, que orquesta los pasos.
//
// Fuente de verdad: [[specs/SPEC-NODO-CONTEXTO]] §Flujo principal (1-7) y
// ai/docs/backend/01-domain/BUSINESS_RULES.md §Nodo de contexto.
//
// Encadena: candidatos desde el grafo → selección del modelo → carga y
// estimación → recorte al límite → auditoría → ensamblado. No llama al modelo
// por su cuenta: se lo pide una etapa, y el nodo hace exactamente un paso de
// selección por petición.

import (
	"context"
	"fmt"

	"localcli/internal/docs"
)

// EtapaChat es la etapa que usa `flow.Motor.Conversar` para el camino de chat
// (mismo literal que flow.EtapaChat; un test de integración asegura que
// coinciden). El chat no es una etapa de flujo: marca dónde no aplican las
// reglas de una (p. ej. el respaldo a todo el proyecto).
const EtapaChat = "chat"

// Nodo es el nodo de contexto de una sesión/etapa.
type Nodo struct {
	Grafo   Grafo
	Modelo  Modelo
	Auditor Auditor
	// SessionID y Etapa se usan cuando el nodo se consume como la interfaz que
	// espera flow (`ContextoPara`); la etapa que trae la petición manda sobre
	// la del nodo.
	SessionID string
	Etapa     string
	// Semilla son las rutas de partida; vacío = todo el proyecto.
	Semilla []string
	// Limite en tokens; 0 = entorno o sin tope.
	Limite int
}

// Preparar hace el trabajo completo para una solicitud.
func (n *Nodo) Preparar(ctx context.Context, s SolicitudContexto) (ContextoArmado, error) {
	if n.Grafo == nil || n.Modelo == nil {
		return ContextoArmado{}, fmt.Errorf("context: el nodo no tiene grafo y modelo conectados")
	}
	semilla := s.Semilla
	if semilla == nil {
		semilla = n.Semilla
	}
	candidatos, err := Candidatos(n.Grafo, semilla)
	if err != nil {
		return ContextoArmado{}, err
	}

	// El chat sin semilla no consulta al modelo qué documentos quiere: su
	// conversación es casual y su bloque va solo con el objetivo, así que la
	// consulta solo añadía una generación entera al turno antes de la respuesta.
	// Las etapas de un flujo nunca caen aquí: en ellas el modelo sigue decidiendo
	// qué es relevante (SPEC-NODO-CONTEXTO §Reglas y §Flujos alternativos).
	var seleccion []string
	if !n.omiteSeleccionDeChat(s) {
		seleccion, err = n.Modelo.Seleccionar(ctx, s.Objetivo, candidatos)
		if err != nil {
			return ContextoArmado{}, err
		}
	}
	seleccion = n.conFallback(s, seleccion, candidatos)

	// Un documento pedido que no existe detiene la etapa (SPEC-NODO-CONTEXTO):
	// no se continúa suponiendo.
	enCandidatos := map[string]bool{}
	for _, c := range candidatos {
		enCandidatos[c] = true
	}
	cargados := make([]DocumentoSeleccionado, 0, len(seleccion))
	for _, ruta := range seleccion {
		if !enCandidatos[ruta] {
			return ContextoArmado{}, errNoExiste(ruta)
		}
		contenido, cErr := n.Grafo.Contenido(ruta)
		if cErr != nil {
			return ContextoArmado{}, errNoExiste(ruta)
		}
		cargados = append(cargados, DocumentoSeleccionado{
			Ruta:      ruta,
			Contenido: contenido,
			Tokens:    EstimarTokens(contenido),
		})
	}

	limite := s.Limite
	if limite == 0 {
		limite = n.Limite
	}
	if limite == 0 {
		limite = LimiteDeEntorno()
	}
	incluidos, descartados, tokens := Recortar(cargados, limite)

	if err := Auditar(n.Auditor, s.SessionID, s.Etapa, incluidos, descartados); err != nil {
		return ContextoArmado{}, err
	}

	return ContextoArmado{
		Objetivo:    s.Objetivo,
		Etapa:       s.Etapa,
		Documentos:  incluidos,
		Descartados: descartados,
		Tokens:      tokens,
		Bloque:      Ensamblar(s.Objetivo, s.Etapa, incluidos),
	}, nil
}

// omiteSeleccionDeChat dice si el nodo puede ahorrarse la consulta al modelo.
// Es el caso del chat sin semilla: su bloque va solo con el objetivo, así que
// preguntar al modelo no cambiaría lo que se entrega y sí costaría una
// generación completa por turno. Fuera del chat —o con semilla— el modelo sigue
// decidiendo qué es relevante.
func (n *Nodo) omiteSeleccionDeChat(s SolicitudContexto) bool {
	if s.Etapa != EtapaChat {
		return false
	}
	return len(s.Semilla) == 0 && len(n.Semilla) == 0
}

// conFallback aplica el valor por defecto cuando el modelo no decide: se usan
// la semilla o, si no hay, los candidatos. Es el "objetivo declarado por la
// etapa" llevado a lo que el nodo puede entregar sin inventar. La excepción es
// el chat: sin selección y sin semilla no se vuelca el proyecto entero —el
// modelo solo vería la estructura y respondería con un resumen—, así que su
// bloque queda sin documentos (el modelo nunca ve el proyecto entero,
// BUSINESS_RULES §Invariantes).
func (n *Nodo) conFallback(s SolicitudContexto, seleccion, candidatos []string) []string {
	if len(seleccion) > 0 {
		return seleccion
	}
	semilla := s.Semilla
	if semilla == nil {
		semilla = n.Semilla
	}
	if len(semilla) > 0 {
		return semilla
	}
	if s.Etapa == EtapaChat {
		return nil
	}
	return candidatos
}

// ContextoPara cumple la interfaz que consume flow: devuelve el bloque de
// contexto ya ensamblado para un objetivo de la etapa indicada. Una etapa
// vacía cae en la del nodo, por compatibilidad con quien lo monta.
func (n *Nodo) ContextoPara(ctx context.Context, etapa, objetivo string) (string, error) {
	if etapa == "" {
		etapa = n.Etapa
	}
	armado, err := n.Preparar(ctx, SolicitudContexto{
		Objetivo:  objetivo,
		Etapa:     etapa,
		SessionID: n.SessionID,
		Limite:    n.Limite,
	})
	if err != nil {
		return "", err
	}
	return armado.Bloque, nil
}

// GrafoDeDocumentos construye el grafo real a partir de los documentos
// cargados (atajo de composición).
func GrafoDeDocumentos(raiz string) (Grafo, error) {
	docsCargados, errs, err := docs.CargarDocs(raiz)
	if err != nil {
		return nil, err
	}
	if len(errs) > 0 {
		// Un frontmatter roto es un error localizado: el resto se carga.
		// El nodo no puede avisar de todos sin parar, así que solo se
		// propaga el fallo global; los localizados quedan fuera del grafo.
		_ = errs
	}
	return GrafoDeDocs(docs.ConstruirGrafo(docsCargados)), nil
}
