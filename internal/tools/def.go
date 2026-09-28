package tools

// def.go — T-B024-03 y T-B024-04: la forma de una herramienta y la capa
// universal que envuelve toda ejecución.
//
// Fuente de verdad: ai/docs/backend/02-interfaces/TOOLS.md §7 (la forma de una
// herramienta) y §8 (la capa universal: buscar, permiso, validar, ejecutar,
// recortar, avisar).
//
// `tools` define el contrato y NO importa `fileops`, `exec` ni el cliente de
// internet: sus tipos son la frontera y las implementaciones se inyectan en el
// cableado. Es lo que permite que el enrutado deje de ser un `switch` de tres
// categorías: cada herramienta trae su propio `Ejecutar` y una herramienta
// nueva es un caso más, sin tocar el motor.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// Contexto es lo que recibe un handler al ejecutarse. `Ask` vive aquí y no
// dentro del handler: la aprobación deja de estar acoplada a `fileops` y `exec`
// y cualquier herramienta —incluida una del usuario— la pide por el mismo
// camino (DECISIONS.md).
type Contexto struct {
	Ctx      context.Context
	SesionID string
	Agente   string
	// Permisos son las acciones del agente activo, ya resueltas.
	Permisos []Accion
	// Ask pide la decisión del usuario. Puede ser nil (sin aprobador): entonces
	// una herramienta que lo necesite falla cerrada.
	Ask func(ctx context.Context, s Solicitud) (Decision, error)
	// Meta publica una anotación para el panel. Puede ser nil.
	Meta func(titulo string, meta map[string]any)
}

// Resultado es lo que devuelve una herramienta. `Error` poblado es un error de
// negocio: el modelo puede corregir la llamada y reintentar. Un error de Go es
// un fallo del harness y no es corregible por el modelo.
type Resultado struct {
	Salida   string
	Meta     map[string]any
	Truncado bool
	// Error es el motivo por el que la llamada no produjo salida, en un texto
	// que el modelo puede entender y corregir.
	Error string
}

// Ejecutar es el handler propio de una herramienta. `args` es el valor ya
// decodificado y validado contra su esquema (para una herramienta del usuario,
// un objeto genérico).
type Ejecutar func(ctx context.Context, args any, c Contexto) (Resultado, error)

// Solicitud es lo que una herramienta pide antes de aplicar un efecto.
type Solicitud struct {
	Descripcion string
	Borrado     bool
}

// Decision es la respuesta del usuario.
type Decision struct {
	Aprobada bool
	// Explicita es la confirmación explícita, obligatoria para borrar.
	Explicita bool
}

// Peticion es lo que llega a la capa universal: quién pide, con qué permisos,
// qué herramienta y con qué argumentos (JSON crudo, tal como los formó el
// modelo).
type Peticion struct {
	Agente   string
	SesionID string
	// Permisos son las acciones del agente activo: de ellas depende la
	// comprobación de permiso, que vive ANTES de ejecutar nada.
	Permisos    []Accion
	Herramienta string
	Argumentos  any
}

// Publicador recibe los avisos de la capa universal. Lo implementa el cableado
// (el bus de eventos); `tools` no conoce la TUI ni `session`.
type Publicador interface {
	// HerramientaInvocada anuncia la llamada con su verbo de pantalla y su
	// objetivo (la ruta, el patrón o el comando sobre el que actúa).
	HerramientaInvocada(nombre, agente, verbo, tema string)
	// HerramientaResultado cierra la llamada con su desenlace y la medida del
	// resultado («70 líneas», «3 coincidencias»).
	HerramientaResultado(nombre string, ok bool, err string, truncado bool, medida string)
}

// Registro es el conjunto de herramientas disponibles (las trece incluidas más
// las que declare el usuario) y la capa universal que las ejecuta.
type Registro struct {
	herramientas map[string]Herramienta
	orden        []string

	// Ask obtiene la decisión del usuario para una herramienta que la requiere.
	// Lo conecta el cableado; sin él, esas herramientas fallan cerradas.
	Ask func(ctx context.Context, s Solicitud) (Decision, error)
	// Meta publica una anotación del handler. Opcional.
	Meta func(titulo string, meta map[string]any)
	// Eventos publica `herramienta_invocada` y `herramienta_resultado`.
	Eventos Publicador
	// Hooks son los puntos de enganche opcionales (§10).
	Hooks Hooks
	// LimiteTokens es el presupuesto de la salida. Cero usa el por defecto.
	LimiteTokens int
}

// Definicion es una herramienta tal como la ve el modelo: nombre, descripción y
// esquema de argumentos.
type Definicion struct {
	Nombre      string
	Descripcion string
	Esquema     *Esquema
}

// NuevoRegistro construye el registro con las herramientas dadas. Falla si
// viene vacío o si hay un nombre repetido: un registro sin herramientas o con
// dos entradas del mismo nombre es un error de cableado, no un caso de uso.
func NuevoRegistro(herramientas []Herramienta) (*Registro, error) {
	r := &Registro{herramientas: map[string]Herramienta{}, LimiteTokens: LimiteTokensSalida}
	for _, h := range herramientas {
		if strings.TrimSpace(h.Nombre) == "" {
			continue
		}
		if _, dup := r.herramientas[h.Nombre]; dup {
			return nil, fmt.Errorf("tools: la herramienta %q está declarada dos veces", h.Nombre)
		}
		if h.Esquema == nil {
			h.Esquema = EsquemaObjeto()
		}
		r.herramientas[h.Nombre] = h
		r.orden = append(r.orden, h.Nombre)
	}
	if len(r.herramientas) == 0 {
		return nil, errors.New("tools: el registro necesita al menos una herramienta")
	}
	return r, nil
}

// Buscar devuelve la herramienta del registro por su nombre exacto. El segundo
// valor es false si no existe.
func (r *Registro) Buscar(nombre string) (Herramienta, bool) {
	if r == nil {
		return Herramienta{}, false
	}
	h, ok := r.herramientas[nombre]
	return h, ok
}

// Nombres devuelve los nombres del registro, en el orden en que se declararon.
func (r *Registro) Nombres() []string {
	if r == nil {
		return nil
	}
	out := make([]string, len(r.orden))
	copy(out, r.orden)
	return out
}

// Definiciones devuelve las herramientas que corresponden a las acciones dadas,
// listas para viajar al modelo. El filtro aplica a las trece y a las del
// usuario por igual: una declarada de lectura llega a los dos agentes.
func (r *Registro) Definiciones(permisos []Accion) []Definicion {
	if r == nil {
		return nil
	}
	var out []Definicion
	for _, nombre := range r.orden {
		h := r.herramientas[nombre]
		if !h.Accion().Permitida(permisos) {
			continue
		}
		d := Definicion{Nombre: h.Nombre, Descripcion: h.Descripcion, Esquema: h.Esquema}
		if r.Hooks.DefinirHerramienta != nil {
			if n, e := r.Hooks.DefinirHerramienta(d.Nombre, d.Descripcion, d.Esquema); n != "" || e != nil {
				if n != "" {
					d.Nombre = n
				}
				if e != nil {
					d.Esquema = e
				}
			}
		}
		out = append(out, d)
	}
	return out
}

// Ejecutar es la capa universal: una sola función envuelve TODA ejecución, sin
// excepciones. Un error de Go es un fallo duro; un `Resultado.Error` es un
// motivo que el modelo puede corregir.
func (r *Registro) Ejecutar(ctx context.Context, p Peticion) (Resultado, error) {
	h, ok := r.herramientas[p.Herramienta]
	if !ok {
		return Resultado{Error: "no existe ninguna herramienta llamada `" + p.Herramienta + "`"}, nil
	}
	// El permiso va ANTES de validar: si el agente no tiene la herramienta, no
	// tiene sentido explicarle qué le falta a sus argumentos.
	if !h.Accion().Permitida(p.Permisos) {
		return Resultado{}, fmt.Errorf("%w: el agente %q no tiene la herramienta %q (`%s`)",
			ErrHerramientaNoPermitida, p.Agente, h.Nombre, h.Accion())
	}
	// Las de internet son las únicas que salen de la máquina: sin la variable,
	// el cliente no llega a ver la petición (SECURITY.md §4).
	if h.Categoria == CatInternet && !InternetPermitida() {
		return Resultado{}, fmt.Errorf("%w: las herramientas de internet están desactivadas; habilítalas con %s=1",
			ErrHerramientaNoPermitida, VarInternet)
	}

	args, err := r.decodificar(h, p.Argumentos)
	if err != nil {
		// La validación no es una avería: vuelve al modelo con el motivo y con
		// lo que se esperaba, para que corrija en la misma pasada.
		return Resultado{Error: err.Error() + "\n" + esperado(h)}, nil
	}

	limite := r.LimiteTokens
	if limite <= 0 {
		limite = LimiteTokensSalida
	}
	resumen := TemaDe(h, args)
	if r.Eventos != nil {
		r.Eventos.HerramientaInvocada(h.Nombre, p.Agente, VerboDe(h), resumen)
	}
	if r.Hooks.AntesDeEjecutar != nil {
		r.Hooks.AntesDeEjecutar(h.Nombre, args, nil)
	}

	c := Contexto{
		Ctx:      ctx,
		SesionID: p.SesionID,
		Agente:   p.Agente,
		Permisos: p.Permisos,
		Ask:      r.Ask,
		Meta:     r.Meta,
	}
	var res Resultado
	if h.Ejecutar == nil {
		res = Resultado{Error: "la herramienta `" + h.Nombre + "` no está disponible en este momento"}
	} else {
		res, err = h.Ejecutar(ctx, args, c)
	}
	if err != nil {
		if r.Hooks.DespuesDeEjecutar != nil {
			r.Hooks.DespuesDeEjecutar(h.Nombre, res, err, nil)
		}
		if r.Eventos != nil {
			r.Eventos.HerramientaResultado(h.Nombre, false, err.Error(), false, MedidaDe(h, res))
		}
		return res, err
	}

	// El recorte es universal y no silencioso. Si el handler ya recortó por su
	// cuenta —la terminal tiene su propio límite—, se respeta su marca.
	if !res.Truncado {
		if salida, cortado := Recortar(res.Salida, limite); cortado {
			res.Salida = salida
			res.Truncado = true
		}
	}
	if r.Hooks.DespuesDeEjecutar != nil {
		r.Hooks.DespuesDeEjecutar(h.Nombre, res, nil, nil)
	}
	if r.Eventos != nil {
		r.Eventos.HerramientaResultado(h.Nombre, res.Error == "", res.Error, res.Truncado, MedidaDe(h, res))
	}
	return res, nil
}

// decodificar convierte los argumentos al tipo de contrato de la herramienta y
// los valida. Para una herramienta del usuario el contrato es un objeto
// genérico: se comprueba que lo que llega es un objeto, nada más.
func (r *Registro) decodificar(h Herramienta, crudo any) (any, error) {
	switch v := crudo.(type) {
	case nil:
		if h.Categoria == CatUsuario {
			return map[string]any{}, nil
		}
		p, ok := NuevaPeticion(h.Nombre)
		if !ok {
			return nil, fmt.Errorf("la herramienta `%s` no tiene contrato de argumentos", h.Nombre)
		}
		return p, nil
	case json.RawMessage:
		return r.decodificarCrudo(h, []byte(v))
	case []byte:
		return r.decodificarCrudo(h, v)
	case string:
		return r.decodificarCrudo(h, []byte(v))
	default:
		if h.Categoria == CatUsuario {
			return v, nil
		}
		if err := Validar(h.Nombre, v); err != nil {
			return nil, err
		}
		return v, nil
	}
}

func (r *Registro) decodificarCrudo(h Herramienta, datos []byte) (any, error) {
	if len(datos) == 0 {
		datos = []byte("{}")
	}
	if h.Categoria == CatUsuario {
		return DecodificarUsuario(datos)
	}
	return Decodificar(h.Nombre, datos)
}

// esperado describe lo que una herramienta espera, para el mensaje corregible.
func esperado(h Herramienta) string {
	var partes []string
	if h.Esquema != nil {
		claves := make([]string, 0, len(h.Esquema.Properties))
		for k := range h.Esquema.Properties {
			claves = append(claves, k)
		}
		sort.Strings(claves)
		obligatorios := map[string]bool{}
		for _, req := range h.Esquema.Required {
			obligatorios[req] = true
		}
		for _, k := range claves {
			if obligatorios[k] {
				partes = append(partes, k+" (obligatorio)")
			} else {
				partes = append(partes, k+" (opcional)")
			}
		}
	}
	if len(partes) == 0 {
		return "`" + h.Nombre + "` no espera argumentos"
	}
	return "`" + h.Nombre + "` espera: " + strings.Join(partes, ", ")
}

// VerboDe devuelve la etiqueta corta con la que la TUI nombra la herramienta.
// Una herramienta sin verbo declarado —una del usuario, por ejemplo— cae en su
// propio nombre: nunca se queda sin etiqueta.
func VerboDe(h Herramienta) string {
	if h.Verbo != "" {
		return h.Verbo
	}
	return h.Nombre
}

// TemaDe devuelve el objetivo de la herramienta tal como se enseña en la línea:
// el valor del campo que su catálogo marca como tema (la ruta, el patrón, el
// comando), colapsado y recortado. Es SOLO ese campo: el resto de argumentos
// —cuerpos de archivo, credenciales— no se expone. Sin tema declarado o sin
// valor, devuelve "".
func TemaDe(h Herramienta, args any) string {
	if h.Tema == "" || args == nil {
		return ""
	}
	v := reflect.ValueOf(args)
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return ""
	}
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" {
			continue
		}
		if strings.Split(f.Tag.Get("json"), ",")[0] != h.Tema {
			continue
		}
		if v.Field(i).Kind() == reflect.String {
			return abreviar(v.Field(i).String(), 60)
		}
		return ""
	}
	return ""
}

// MedidaDe mide el resultado en la unidad declarada por el catálogo («70
// líneas», «3 coincidencias»). Sin unidad declarada —una escritura, por
// ejemplo— o sin salida, no hay medida.
func MedidaDe(h Herramienta, res Resultado) string {
	if h.Unidad == "" || strings.TrimSpace(res.Salida) == "" {
		return ""
	}
	n := contarLineas(res.Salida)
	if n == 0 {
		return ""
	}
	return strconv.Itoa(n) + " " + pluralizar(h.Unidad, n)
}

// abreviar colapsa los espacios y recorta a `max` runas, marcando el corte.
func abreviar(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	if max <= 0 {
		return s
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}

// contarLineas cuenta las líneas con contenido de una salida.
func contarLineas(s string) int {
	n := 0
	for _, l := range strings.Split(s, "\n") {
		if strings.TrimSpace(l) != "" {
			n++
		}
	}
	return n
}

// pluralizar ajusta la unidad al número: singular si es uno, plural si no.
func pluralizar(unidad string, n int) string {
	if n == 1 {
		return unidad
	}
	return unidad + "s"
}
