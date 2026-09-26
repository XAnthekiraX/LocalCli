package context

// model.go — T-B011-01: el tipo SolicitudContexto y la fuente de candidatos.
//
// Fuente de verdad: [[specs/SPEC-NODO-CONTEXTO]] §Flujo principal (una etapa
// pide contexto para un objetivo; el sistema lee la documentación y las
// dependencias que declara) y ai/docs/backend/DECISIONS.md ("Grafo de
// documentación en memoria, reconstruido al arrancar").
//
// Los candidatos salen del grafo de `docs`: la semilla (los documentos de los
// que parte la etapa) y el cierre de sus dependencias declaradas. El grafo
// orienta; quien decide es el modelo (select.go).

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"localcli/internal/docs"
)

// VariableLimite es la variable que limita los tokens de contexto respetados al
// recortar (CONFIGURATION.md §2). Sin ella, se usa el límite del modelo.
const VariableLimite = "LOCALCLI_CONTEXT_LIMIT"

// SolicitudContexto es lo que pide una etapa.
type SolicitudContexto struct {
	Objetivo  string   // p. ej. "implementar la API de pedidos"
	Etapa     string   // identificador de la etapa (para la auditoría)
	SessionID string   // sesión que pide el contexto
	Semilla   []string // rutas de partida; vacío = todo el proyecto
	Limite    int      // tokens; 0 = el del entorno o sin tope
}

// DocumentoSeleccionado es un documento dentro del contexto, con su contenido.
type DocumentoSeleccionado struct {
	Ruta      string
	Contenido string
	Tokens    int
}

// ContextoArmado es el resultado: lo que se le entrega a la etapa.
type ContextoArmado struct {
	Objetivo    string
	Etapa       string
	Documentos  []DocumentoSeleccionado
	Descartados []Descarte
	Tokens      int
	Bloque      string
}

// Descarte es un documento que no entró, con el motivo.
type Descarte struct {
	Ruta   string
	Motivo string
}

// Grafo es lo que el nodo de contexto necesita del grafo de documentación.
type Grafo interface {
	// Rutas devuelve las rutas de todos los documentos, ordenadas.
	Rutas() []string
	// Contenido devuelve el cuerpo de un documento. Error E_DOC_NOT_FOUND si
	// no existe.
	Contenido(ruta string) (string, error)
	// Cierre devuelve las dependencias transitivas de una ruta.
	Cierre(ruta string) ([]string, error)
}

// GrafoDeDocs adapta el grafo real de `docs` a la interfaz del nodo.
func GrafoDeDocs(g *docs.Grafo) Grafo { return adaptadorDocs{g: g} }

type adaptadorDocs struct{ g *docs.Grafo }

func (a adaptadorDocs) Rutas() []string { return docs.Rutas(a.g.Docs()) }

func (a adaptadorDocs) Contenido(ruta string) (string, error) {
	d, err := a.g.Documento(ruta)
	if err != nil {
		return "", err
	}
	return d.Body, nil
}

func (a adaptadorDocs) Cierre(ruta string) ([]string, error) { return a.g.Cierre(ruta) }

// Candidatos arma la lista de candidatos: la semilla y el cierre de sus
// dependencias. Sin semilla, candidatos = todos los documentos. Ordenado y sin
// duplicados.
func Candidatos(g Grafo, semilla []string) ([]string, error) {
	if len(semilla) == 0 {
		return g.Rutas(), nil
	}
	visto := map[string]bool{}
	var out []string
	añadir := func(r string) {
		if r != "" && !visto[r] {
			visto[r] = true
			out = append(out, r)
		}
	}
	for _, s := range semilla {
		añadir(s)
		cierre, err := g.Cierre(s)
		if err != nil {
			return nil, err
		}
		for _, d := range cierre {
			añadir(d)
		}
	}
	sort.Strings(out)
	return out, nil
}

// LimiteDeEntorno lee LOCALCLI_CONTEXT_LIMIT. Devuelve 0 si no está o no es un
// número positivo.
func LimiteDeEntorno() int {
	v := strings.TrimSpace(os.Getenv(VariableLimite))
	if v == "" {
		return 0
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

// errNoExiste construye el E_DOC_NOT_FOUND documentado.
func errNoExiste(ruta string) error {
	return fmt.Errorf("E_DOC_NOT_FOUND: %s", ruta)
}
