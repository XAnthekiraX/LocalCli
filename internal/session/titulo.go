package session

import (
	"context"
	"strings"

	"localcli/internal/llm/openai"
	"localcli/internal/store"
	"localcli/internal/tui"
)

// nombreProvisional es el nombre con el que nace una sesión.
const nombreProvisional = "Nueva sesión"

const promptTitulo = "Redacta un título breve, de como mucho seis palabras, que resuma " +
	"la petición del usuario. Devuelve solo el título, sin comillas ni punto final."

// opcionesTitulo acotan el título: corto y sin razonamiento, para que no gaste el
// turno pensando.
var opcionesTitulo = openai.Opciones{MaxTokens: 64, SinRazonamiento: true}

// titular renombra una sesión recién creada con un título del modelo.
func (o *Orquestador) titular(ctx context.Context, sesionID, texto string) {
	s, err := store.ObtenerSesion(o.db, o.projectID, sesionID)
	if err != nil || s.Name != nombreProvisional {
		return
	}
	titulo, ok := o.tituloGenerado(ctx, texto)
	if !ok {
		return
	}
	if err := store.ActualizarNombreSesion(o.db, o.projectID, sesionID, titulo); err != nil {
		return
	}
	o.emitir(tui.Evento{Sesion: sesionID, Tipo: tui.EventoTituloSesion, Datos: map[string]string{"titulo": titulo}})
}

// tituloGenerado pide un título al modelo; false si falla o viene vacío.
func (o *Orquestador) tituloGenerado(ctx context.Context, texto string) (string, bool) {
	if o.llm == nil {
		return "", false
	}
	mensajes := []openai.Mensaje{
		{Rol: "system", Contenido: promptTitulo},
		{Rol: "user", Contenido: texto},
	}
	var b strings.Builder
	if _, err := o.llm.Completar(ctx, o.modeloActual(), mensajes, nil, opcionesTitulo, func(d openai.Delta) error {
		if !d.Razonamiento {
			b.WriteString(d.Texto)
		}
		return nil
	}); err != nil {
		return "", false
	}
	titulo := LimpiarTitulo(b.String())
	if titulo == "" {
		return "", false
	}
	return titulo, true
}

// LimpiarTitulo deja el título en una sola línea corta y sin comillas.
func LimpiarTitulo(s string) string {
	s = strings.TrimSpace(s)
	if corte := strings.IndexByte(s, '\n'); corte >= 0 {
		s = strings.TrimSpace(s[:corte])
	}
	s = strings.Trim(s, "\"'“”‘’`")
	const ancho = 60
	if r := []rune(s); len(r) > ancho {
		s = string(r[:ancho]) + "…"
	}
	return s
}
