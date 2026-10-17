package flow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Tipos de etapa.
const (
	// TipoLineal corre la etapa una sola vez.
	TipoLineal = "linear"
	// TipoLoop repite la etapa hasta que el agente responda `true`.
	TipoLoop = "loop"
)

// IteracionesLoopPorDefecto es la cota de vueltas de una etapa `loop` que no
// declara `max_iteraciones`.
const IteracionesLoopPorDefecto = 5

// definicionFlujo es el contrato JSON de un flujo, tal cual llega del archivo.
type definicionFlujo struct {
	Comando string            `json:"comando"`
	Reglas  []string          `json:"reglas"`
	Etapas  []definicionEtapa `json:"etapas"`
}

// definicionEtapa es el contrato JSON de una etapa del flujo.
type definicionEtapa struct {
	ID              string `json:"id"`
	Nombre          string `json:"nombre"`
	Orden           string `json:"orden"`
	Respuesta       string `json:"respuesta"`
	Type            string `json:"type"`
	Aprobacion      bool   `json:"aprobacion"`
	RespuestaEnChat bool   `json:"respuesta_en_chat"`
	MaxIteraciones  int    `json:"max_iteraciones"`
}

// CamposDelFlujo y CamposDeLaEtapa son los campos admitidos en el JSON.
var (
	CamposDelFlujo  = []string{"comando", "reglas", "etapas"}
	CamposDeLaEtapa = []string{"id", "nombre", "orden", "respuesta", "type", "aprobacion", "respuesta_en_chat", "max_iteraciones"}
)

// Etapa es un paso del flujo.
type Etapa struct {
	ID              string
	Nombre          string
	Orden           string // qué debe cumplir la etapa
	Respuesta       string // estructura de salida que recibe la siguiente etapa
	Type            string // TipoLineal o TipoLoop
	Aprobacion      bool   // detiene el flujo DESPUÉS de correr la etapa
	RespuestaEnChat bool   // pinta la respuesta sin que sea la del turno
	MaxIteraciones  int    // cota de vueltas de una etapa `loop`
}

// EsLoop dice si la etapa repite hasta que el agente responda `true`.
func (e Etapa) EsLoop() bool { return e.Type == TipoLoop }

// Flujo es una secuencia ordenada de etapas con nombre. Su identidad es el
// `comando`: es lo que lo invoca y lo que separa su bloque de contexto.
type Flujo struct {
	Comando string // comando que lo arranca, p. ej. "/resolver"
	Reglas  []string
	Etapas  []Etapa
	Origen  string // ruta del archivo; lo fija la carga
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
		Comando: strings.TrimSpace(d.Comando),
		Reglas:  limpiarLista(d.Reglas),
		Etapas:  make([]Etapa, 0, len(d.Etapas)),
	}
	for _, e := range d.Etapas {
		f.Etapas = append(f.Etapas, normalizarEtapa(e))
	}
	if err := f.Validar(); err != nil {
		return Flujo{}, err
	}
	return f, nil
}

// normalizarEtapa aplica los valores por defecto de una etapa: `linear` cuando
// no declara `type`, y la cota por defecto cuando es `loop` y no la declara.
func normalizarEtapa(e definicionEtapa) Etapa {
	tipo := strings.TrimSpace(e.Type)
	if tipo == "" {
		tipo = TipoLineal
	}
	max := e.MaxIteraciones
	if tipo == TipoLoop && max == 0 {
		max = IteracionesLoopPorDefecto
	}
	return Etapa{
		ID:              strings.TrimSpace(e.ID),
		Nombre:          strings.TrimSpace(e.Nombre),
		Orden:           strings.TrimSpace(e.Orden),
		Respuesta:       strings.TrimSpace(e.Respuesta),
		Type:            tipo,
		Aprobacion:      e.Aprobacion,
		RespuestaEnChat: e.RespuestaEnChat,
		MaxIteraciones:  max,
	}
}

// Validar comprueba que el flujo es encadenable: comando con `/`, al menos una
// etapa y, por etapa, identidad, `orden` y coherencia de `type`/`respuesta`.
func (f Flujo) Validar() error {
	if f.Comando == "" {
		return errorf("falta el campo obligatorio `comando`")
	}
	if !strings.HasPrefix(f.Comando, "/") {
		return errorf("el comando del flujo %s no empieza por / (%q)", f.Comando, f.Comando)
	}
	if len(f.Etapas) == 0 {
		return errorf("el flujo %s no tiene etapas", f.Comando)
	}
	for i, e := range f.Etapas {
		pos := fmt.Sprintf("la etapa %d del flujo %s", i, f.Comando)
		if e.ID == "" {
			return errorf("%s no tiene id", pos)
		}
		pos = "la etapa " + e.ID + " del flujo " + f.Comando
		if e.Nombre == "" {
			return errorf("%s no tiene nombre", pos)
		}
		if e.Orden == "" {
			return errorf("%s no declara `orden`", pos)
		}
		switch e.Type {
		case TipoLineal:
			if e.Respuesta == "" {
				return errorf("%s no declara `respuesta`", pos)
			}
			if e.MaxIteraciones != 0 {
				return errorf("%s declara `max_iteraciones` fuera de una etapa `loop`", pos)
			}
		case TipoLoop:
			if e.MaxIteraciones < 1 {
				return errorf("%s declara `max_iteraciones` menor que 1", pos)
			}
		default:
			return errorf("%s declara un `type` desconocido: %q", pos, e.Type)
		}
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
