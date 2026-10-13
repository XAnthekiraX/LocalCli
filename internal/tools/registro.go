package tools

import (
	"context"
	"encoding/json"
	"reflect"
	"time"
)

// Peticion es una ejecución pedida por el agente: qué herramienta, con qué
// argumentos, y el contexto con el que resolver permiso, aprobación y eventos.
type Peticion struct {
	Sesion     string
	Agente     string
	Permisos   []string
	Ask        func(ctx context.Context, s Solicitud) (Decision, error)
	Meta       func(clave string, valor any)
	Nombre     string
	Argumentos json.RawMessage
}

// Registro es el catálogo de herramientas y la capa universal de ejecución.
type Registro struct {
	Hooks   Hooks
	Eventos Eventos

	porNombre map[string]Herramienta
	orden     []string
}

// NuevoRegistro construye el registro a partir de una lista de herramientas.
func NuevoRegistro(catalogo []Herramienta) *Registro {
	r := &Registro{porNombre: make(map[string]Herramienta, len(catalogo))}
	for _, h := range catalogo {
		r.Agregar(h)
	}
	return r
}

// Agregar mete una herramienta si su nombre no estaba ya. Devuelve false si lo repite.
func (r *Registro) Agregar(h Herramienta) bool {
	if _, existe := r.porNombre[h.Nombre]; existe {
		return false
	}
	r.porNombre[h.Nombre] = h
	r.orden = append(r.orden, h.Nombre)
	return true
}

// Buscar devuelve la herramienta con ese nombre.
func (r *Registro) Buscar(nombre string) (Herramienta, bool) {
	h, ok := r.porNombre[nombre]
	return h, ok
}

// Instalar fija el handler de una herramienta del catálogo. Devuelve false si no
// existe: el cableado no inventa entradas.
func (r *Registro) Instalar(nombre string, e Ejecutar) bool {
	h, ok := r.porNombre[nombre]
	if !ok {
		return false
	}
	h.Ejecutar = e
	r.porNombre[nombre] = h
	return true
}

// Herramientas devuelve el catálogo en orden de alta.
func (r *Registro) Herramientas() []Herramienta {
	out := make([]Herramienta, 0, len(r.orden))
	for _, nombre := range r.orden {
		out = append(out, r.porNombre[nombre])
	}
	return out
}

// Definicion es la vista neutra de una herramienta para el agente: lo que ve el
// modelo, ya pasado por el hook DefinirHerramienta.
type Definicion struct {
	Nombre      string
	Descripcion string
	Permiso     Permiso
	Esquema     json.RawMessage
}

// Definiciones devuelve el catálogo con su esquema ya derivado y ajustado por el
// hook, listo para que el cableado lo entregue al agente.
func (r *Registro) Definiciones() []Definicion {
	herramientas := r.Herramientas()
	definiciones := make([]Definicion, 0, len(herramientas))
	for _, h := range herramientas {
		nombre, esquema := r.definir(h)
		definiciones = append(definiciones, Definicion{
			Nombre:      nombre,
			Descripcion: h.Descripcion,
			Permiso:     h.Permiso,
			Esquema:     esquemaJSON(esquema),
		})
	}
	return definiciones
}

func (r *Registro) definir(h Herramienta) (string, *Esquema) {
	nombre, esquema := h.Nombre, h.Esquema()
	if r.Hooks.DefinirHerramienta != nil {
		nombre, esquema = r.Hooks.DefinirHerramienta(nombre, h.Descripcion, esquema)
	}
	return nombre, esquema
}

func esquemaJSON(e *Esquema) json.RawMessage {
	if e == nil {
		return json.RawMessage(`{"type":"object"}`)
	}
	datos, err := json.Marshal(e)
	if err != nil {
		return json.RawMessage(`{"type":"object"}`)
	}
	return datos
}

// Ejecutar es la capa universal: buscar, permiso, validar, emitir invocada,
// ejecutar y medir, recortar y emitir resultado. El permiso se comprueba antes de
// validar los argumentos.
func (r *Registro) Ejecutar(ctx context.Context, p Peticion) Resultado {
	h, ok := r.Buscar(p.Nombre)
	if !ok {
		return Resultado{Error: CodToolUnknown + ": la herramienta `" + p.Nombre + "` no existe"}
	}
	if !permitida(h, p.Permisos) {
		return Resultado{Error: CodToolNotAllowed + ": el agente no tiene la herramienta `" + p.Nombre + "`"}
	}
	args, errArgs := validarArgs(h, p.Argumentos)
	if errArgs != "" {
		return Resultado{Error: errArgs}
	}
	if r.Hooks.AntesDeEjecutar != nil {
		r.Hooks.AntesDeEjecutar(h.Nombre, args, nil)
	}

	tema := TemaVisible(h, args)
	r.Eventos.invocada(EventoInvocacion{
		Sesion: p.Sesion, Agente: p.Agente, Nombre: h.Nombre, Verbo: h.Verbo, Tema: tema,
	})

	contexto := Contexto{
		Ctx: ctx, SesionID: p.Sesion, Agente: p.Agente,
		Permisos: p.Permisos, Ask: p.Ask, Meta: p.Meta,
	}
	inicio := time.Now()
	res, err := ejecutarHandler(h, ctx, args, contexto)
	duracion := time.Since(inicio)
	if err != nil {
		res = Resultado{Error: err.Error()}
	}
	if !res.Truncado {
		res.Salida, res.Truncado = Recortar(res.Salida)
	}
	if r.Hooks.DespuesDeEjecutar != nil {
		r.Hooks.DespuesDeEjecutar(h.Nombre, res, err, nil)
	}
	r.Eventos.resultado(EventoResultado{
		Sesion: p.Sesion, Nombre: h.Nombre, Verbo: h.Verbo,
		Tema:  TemaFinal(h, tema, res.Meta),
		OK:    err == nil && res.Error == "",
		Error: res.Error, Medida: medidaDe(h, res),
		Truncado: res.Truncado, Duracion: duracion,
	})
	return res
}

// medidaDe prefiere la medida que el handler declara en `Meta` —una lectura
// devuelve un objeto y sus líneas no son las del JSON— y, si no la hay, cuenta
// las unidades de la salida.
func medidaDe(h Herramienta, res Resultado) string {
	if medida, ok := res.Meta["medida"].(string); ok && medida != "" {
		return medida
	}
	return Medir(h.Unidad, res.Salida)
}

func ejecutarHandler(h Herramienta, ctx context.Context, args any, c Contexto) (Resultado, error) {
	if h.Ejecutar == nil {
		return Resultado{Error: "la herramienta `" + h.Nombre + "` no está implementada"}, nil
	}
	return h.Ejecutar(ctx, args, c)
}

func permitida(h Herramienta, permisos []string) bool {
	if h.Permiso == PermisoDerivado {
		return len(permisos) > 0
	}
	for _, p := range permisos {
		if Permiso(p) == h.Permiso {
			return true
		}
	}
	return false
}

func validarArgs(h Herramienta, raw json.RawMessage) (any, string) {
	if h.TipoPeticion == nil {
		objeto := map[string]any{}
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &objeto); err != nil {
				return nil, CodBadArgs + ": los argumentos no son un objeto"
			}
		}
		return objeto, ""
	}

	objeto := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &objeto); err != nil {
			return nil, CodBadArgs + ": los argumentos no son un objeto válido"
		}
	}
	if falta := FaltanCampos(h.Esquema(), objeto); falta != "" {
		return nil, CodBadArgs + ": falta el campo obligatorio \"" + falta + "\""
	}
	valor := reflect.New(h.TipoPeticion).Interface()
	if err := json.Unmarshal(raw, valor); err != nil {
		return nil, CodBadArgs + ": los argumentos no encajan con `" + h.Nombre + "`"
	}
	return reflect.ValueOf(valor).Elem().Interface(), ""
}
