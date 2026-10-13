package flow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// definicionFlujo es el contrato JSON de un flujo, tal cual llega del archivo.
type definicionFlujo struct {
	Comando     string            `json:"comando"`
	Nombre      string            `json:"nombre"`
	Descripcion string            `json:"descripcion"`
	Peticion    string            `json:"peticion"`
	Pregunta    string            `json:"pregunta"`
	Reglas      []string          `json:"reglas"`
	Etapas      []definicionEtapa `json:"etapas"`
}

// definicionEtapa es el contrato JSON de una etapa del flujo.
type definicionEtapa struct {
	ID              string `json:"id"`
	Nombre          string `json:"nombre"`
	Agente          string `json:"agente"`
	Pregunta        string `json:"pregunta"`
	Instruccion     string `json:"instruccion"`
	Continuacion    bool   `json:"continuacion"`
	Entrega         bool   `json:"entrega"`
	RespuestaEnChat bool   `json:"respuesta_en_chat"`
	Aprobacion      bool   `json:"aprobacion"`
}

// CamposDelFlujo y CamposDeLaEtapa son los campos admitidos en el JSON.
var (
	CamposDelFlujo  = []string{"comando", "nombre", "descripcion", "peticion", "pregunta", "reglas", "etapas"}
	CamposDeLaEtapa = []string{"id", "nombre", "agente", "pregunta", "instruccion", "continuacion", "entrega", "respuesta_en_chat", "aprobacion"}
)

// Etapa es un paso del flujo.
type Etapa struct {
	ID              string
	Nombre          string
	Agente          string // agente que la corre (`plan` o `build`)
	Pregunta        string // lo que la etapa responde; obligatoria
	Instruccion     string // se une a las Reglas y viaja con el contexto de la etapa
	Continuacion    bool   // detiene el flujo ANTES de correr la etapa
	Aprobacion      bool   // detiene el flujo DESPUÉS de correr la etapa
	Entrega         bool   // etapa de cierre; exactamente una, y la última
	RespuestaEnChat bool   // pinta la respuesta sin meterla al contexto
}

// Flujo es una secuencia ordenada de etapas con nombre.
type Flujo struct {
	Nombre      string
	Comando     string // comando que lo arranca, p. ej. "/resolver"
	Descripcion string
	Peticion    string // objetivo por defecto cuando el comando va sin texto
	Pregunta    string // qué entrega el flujo; obligatoria
	Reglas      []string
	Etapas      []Etapa
	Origen      string // ruta del archivo; lo fija la carga
}

// DecodificarFlujo lee el JSON con campos estrictos y valida el flujo.
func DecodificarFlujo(datos []byte) (Flujo, error) {
	var d definicionFlujo
	dec := json.NewDecoder(bytes.NewReader(datos))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&d); err != nil {
		return Flujo{}, errorf("%w", err)
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return Flujo{}, errorf("el archivo lleva contenido de más después del objeto")
	}

	f := Flujo{
		Nombre:      strings.TrimSpace(d.Nombre),
		Comando:     strings.TrimSpace(d.Comando),
		Descripcion: strings.TrimSpace(d.Descripcion),
		Peticion:    strings.TrimSpace(d.Peticion),
		Pregunta:    strings.TrimSpace(d.Pregunta),
		Reglas:      limpiarLista(d.Reglas),
		Etapas:      make([]Etapa, 0, len(d.Etapas)),
	}
	for _, e := range d.Etapas {
		f.Etapas = append(f.Etapas, Etapa{
			ID:              strings.TrimSpace(e.ID),
			Nombre:          strings.TrimSpace(e.Nombre),
			Agente:          strings.TrimSpace(e.Agente),
			Pregunta:        strings.TrimSpace(e.Pregunta),
			Instruccion:     strings.TrimSpace(e.Instruccion),
			Continuacion:    e.Continuacion,
			Aprobacion:      e.Aprobacion,
			Entrega:         e.Entrega,
			RespuestaEnChat: e.RespuestaEnChat,
		})
	}
	if err := f.Validar(); err != nil {
		return Flujo{}, err
	}
	return f, nil
}

// Validar comprueba que el flujo es encadenable: comando con `/`, nombre,
// `pregunta`, al menos una etapa y una sola `entrega`, que es la última.
func (f Flujo) Validar() error {
	if f.Comando == "" {
		return errorf("falta el campo obligatorio `comando`")
	}
	if !strings.HasPrefix(f.Comando, "/") {
		return errorf("el comando del flujo %s no empieza por / (%q)", f.Nombre, f.Comando)
	}
	if f.Nombre == "" {
		return errorf("falta el campo obligatorio `nombre`")
	}
	if strings.TrimSpace(f.Pregunta) == "" {
		return errorf("el flujo %s no declara `pregunta` (qué entrega)", f.Nombre)
	}
	if len(f.Etapas) == 0 {
		return errorf("el flujo %s no tiene etapas", f.Nombre)
	}
	entregas := 0
	for i, e := range f.Etapas {
		if e.ID == "" {
			return errorf("la etapa %d del flujo %s no tiene id", i, f.Nombre)
		}
		if e.Agente == "" {
			return errorf("la etapa %s no declara agente", e.ID)
		}
		if e.Pregunta == "" {
			return errorf("la etapa %s del flujo %s no declara `pregunta`", e.ID, f.Nombre)
		}
		if e.Entrega {
			entregas++
		}
	}
	if entregas != 1 {
		return errorf("el flujo %s declara %d etapas `entrega`, quiero exactamente una", f.Nombre, entregas)
	}
	if !f.Etapas[len(f.Etapas)-1].Entrega {
		return errorf("en el flujo %s la etapa `entrega` debe ser la última", f.Nombre)
	}
	return nil
}

// limpiarLista recorta cada elemento y descarta los vacíos.
func limpiarLista(xs []string) []string {
	var out []string
	for _, x := range xs {
		if x = strings.TrimSpace(x); x != "" {
			out = append(out, x)
		}
	}
	return out
}

func errorf(format string, args ...any) error {
	return fmt.Errorf("flujo: "+format, args...)
}
