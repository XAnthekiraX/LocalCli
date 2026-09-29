package flow

// flujo.go — cargar la definición de un flujo desde un JSON.
//
// Fuente de verdad: [[specs/SPEC-FLUJO-PERSONALIZADO]] (definir flujos propios
// con nombre, etapas y orden) y [[specs/SPEC-MOTOR-FLUJOS]]. Los flujos del
// proyecto son los que declara `.localcli/flows/*.json`; los constructores de
// este paquete quedan como valor por defecto interno, no como catálogo.
//
// La definición es DATO: aquí solo se lee y se valida. El contrato es cerrado
// (json.Decoder con DisallowUnknownFields): un campo de más no se carga, para
// que un formato viejo no se interprete a medias.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// definicionFlujo es el contrato JSON de un flujo.
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

// CamposDelFlujo y CamposDeLaEtapa son los nombres de campo admitidos. Son la
// lista cerrada del contrato; sirven para documentar y para los tests.
var (
	CamposDelFlujo  = []string{"comando", "nombre", "descripcion", "peticion", "pregunta", "reglas", "etapas"}
	CamposDeLaEtapa = []string{"id", "nombre", "agente", "pregunta", "instruccion", "continuacion", "entrega", "respuesta_en_chat", "aprobacion"}
)

// errFlujo construye un error localizado de carga de flujo. No usa un código
// `E_`: ERRORS.md no define uno para un JSON de flujo inválido, y fingir un
// código ajeno sería peor que un mensaje claro.
func errFlujo(msg string) error { return fmt.Errorf("flujo: %s", msg) }

// DecodificarFlujo convierte el JSON de un flujo en un Flujo, rechazando
// campos desconocidos y comprobando los obligatorios y la secuencia.
func DecodificarFlujo(datos []byte) (Flujo, error) {
	var d definicionFlujo
	dec := json.NewDecoder(bytes.NewReader(datos))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&d); err != nil {
		return Flujo{}, err
	}
	// Datos sobrantes tras el objeto también son un archivo mal formado.
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return Flujo{}, errFlujo("el archivo lleva contenido de más después del objeto")
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
	if f.Comando == "" {
		return Flujo{}, errFlujo("falta el campo obligatorio `comando`")
	}
	if f.Nombre == "" {
		return Flujo{}, errFlujo("falta el campo obligatorio `nombre`")
	}
	if err := f.Validar(); err != nil {
		return Flujo{}, err
	}
	return f, nil
}

// limpiarLista recorta cada elemento y descarta los vacíos; devuelve nil para
// una lista sin contenido, para que el JSON y el round-trip no diverjan.
func limpiarLista(xs []string) []string {
	var out []string
	for _, x := range xs {
		if x = strings.TrimSpace(x); x != "" {
			out = append(out, x)
		}
	}
	return out
}

// CargarFlujosCarpeta lee los `*.json` de un directorio, en orden alfabético.
// Una carpeta inexistente no es un error (un proyecto puede no tener flujos
// propios). Un archivo roto detiene la carga: no se arranca con un catálogo de
// flujos a medias sin avisar (misma regla que agent.CargarCarpeta).
func CargarFlujosCarpeta(dir string) ([]Flujo, error) {
	entradas, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, errFlujo("no se pudo leer la carpeta de flujos " + dir + ": " + err.Error())
	}
	nombres := make([]string, 0, len(entradas))
	for _, e := range entradas {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		nombres = append(nombres, e.Name())
	}
	sort.Strings(nombres)

	flujos := make([]Flujo, 0, len(nombres))
	for _, nombre := range nombres {
		ruta := filepath.Join(dir, nombre)
		datos, err := os.ReadFile(ruta)
		if err != nil {
			return nil, errFlujo("no se pudo leer " + ruta + ": " + err.Error())
		}
		f, err := DecodificarFlujo(datos)
		if err != nil {
			return nil, errFlujo(ruta + ": " + err.Error())
		}
		flujos = append(flujos, f)
	}
	return flujos, nil
}
