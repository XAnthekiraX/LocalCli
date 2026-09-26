package context

// select.go — T-B011-02: preguntar al modelo qué documentos son relevantes.
//
// Fuente de verdad: [[specs/SPEC-NODO-CONTEXTO]] §Flujo principal pasos 3-4 y
// §Reglas ("la decisión de qué es relevante la toma siempre el modelo").
//
// El nodo no llama al modelo por su cuenta: lo hace cuando una etapa le pide
// contexto. La decisión es del modelo; el sistema solo le presenta los
// candidatos y recoge su respuesta. Una respuesta que no reconoce ningún
// candidato no inventa nada: deja la selección vacía y el nodo aplica el
// objetivo declarado.

import (
	"context"
	"sort"
	"strings"

	"localcli/internal/ollama"
)

// Modelo decide qué documentos necesita el modelo para un objetivo.
type Modelo interface {
	Seleccionar(ctx context.Context, objetivo string, candidatos []string) ([]string, error)
}

// ModeloOllama pregunta al modelo local. Solo se le pasan los nombres de los
// candidatos y el objetivo, nunca el contenido de los documentos.
type ModeloOllama struct {
	Cliente *ollama.Client
	Modelo  string
}

// Seleccionar lanza la consulta y devuelve los candidatos que el modelo nombró,
// en el orden en que aparecieron. Un fallo de Ollama se propaga; una respuesta
// sin candidatos reconocibles devuelve una lista vacía (sin error), para que el
// nodo pueda aplicar su valor por defecto.
func (m *ModeloOllama) Seleccionar(ctx context.Context, objetivo string, candidatos []string) ([]string, error) {
	ch, err := m.Cliente.Chat(ctx, ollama.GenerarRequest{
		Model:    m.Modelo,
		Messages: []ollama.Mensaje{{Role: "user", Content: PromptSeleccion(objetivo, candidatos)}},
	})
	if err != nil {
		return nil, err
	}
	var b strings.Builder
	for ev := range ch {
		switch ev.Tipo {
		case ollama.EventoToken:
			b.WriteString(ev.Texto)
		case ollama.EventoError:
			return nil, ev.Error
		}
	}
	return InterpretarSeleccion(b.String(), candidatos), nil
}

// PromptSeleccion redacta la consulta. Es el único texto que sale hacia el
// modelo desde aquí: el objetivo y la lista de rutas candidatas.
func PromptSeleccion(objetivo string, candidatos []string) string {
	var b strings.Builder
	b.WriteString("Objetivo: " + objetivo + "\n\n")
	b.WriteString("Documentos disponibles (rutas relativas a ai/docs):\n")
	for _, c := range candidatos {
		b.WriteString("- " + c + "\n")
	}
	b.WriteString("\nResponde SOLO con las rutas de los documentos que necesites, una por línea. ")
	b.WriteString("No incluyas rutas que no estén en la lista.")
	return b.String()
}

// InterpretarSeleccion extrae de la respuesta las rutas candidatas nombradas,
// en orden de aparición y sin duplicados. Es tolerante a viñetas y a líneas de
// relleno: una ruta cuenta si aparece como tal.
func InterpretarSeleccion(respuesta string, candidatos []string) []string {
	set := make(map[string]bool, len(candidatos))
	for _, c := range candidatos {
		set[c] = true
	}
	visto := map[string]bool{}
	var out []string
	for _, linea := range strings.Split(respuesta, "\n") {
		limpia := strings.TrimSpace(linea)
		limpia = strings.TrimPrefix(limpia, "- ")
		limpia = strings.Trim(limpia, "`\"'* ")
		if set[limpia] && !visto[limpia] {
			visto[limpia] = true
			out = append(out, limpia)
		}
	}
	return out
}

// OrdenarSeleccion ordena una selección alfabéticamente (para pruebas y para
// comparaciones estables).
func OrdenarSeleccion(sel []string) []string {
	out := append([]string(nil), sel...)
	sort.Strings(out)
	return out
}
