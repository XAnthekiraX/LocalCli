package tools

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestEsquemaDerivadoObligatoriosYOpcionales(t *testing.T) {
	h := Herramienta{TipoPeticion: reflect.TypeOf(PeticionLeerArchivo{})}
	e := h.Esquema()
	if e.Tipo != "object" {
		t.Fatalf("tipo = %q", e.Tipo)
	}
	if len(e.Requeridos) != 1 || e.Requeridos[0] != "ruta" {
		t.Fatalf("solo `ruta` debería ser obligatorio: %v", e.Requeridos)
	}
	for _, opcional := range []string{"inicio_linea", "fin_linea", "motivo"} {
		if _, ok := e.Propiedades[opcional]; !ok {
			t.Fatalf("`%s` debería estar en properties", opcional)
		}
	}
	if e.Propiedades["inicio_linea"].Tipo != "integer" {
		t.Fatalf("inicio_linea debería ser integer: %+v", e.Propiedades["inicio_linea"])
	}
}

func TestEsquemaAnidaElTipoDelElemento(t *testing.T) {
	h := Herramienta{TipoPeticion: reflect.TypeOf(PeticionEditarArchivo{})}
	items := h.Esquema().Propiedades["reemplazos"].Items
	if items == nil || items.Tipo != "object" {
		t.Fatalf("reemplazos debería ser un arreglo de objetos: %+v", items)
	}
	for _, campo := range []string{"buscar", "reemplazo"} {
		if _, ok := items.Propiedades[campo]; !ok {
			t.Fatalf("el objeto anidado debería exponer `%s`", campo)
		}
	}
	if len(items.Requeridos) != 2 {
		t.Fatalf("buscar y reemplazo deberían ser obligatorios: %v", items.Requeridos)
	}
}

func TestEsquemaGenericoParaHerramientaDelUsuario(t *testing.T) {
	h := Herramienta{Nombre: "contar_lineas"}
	e := h.Esquema()
	if e.Tipo != "object" || len(e.Propiedades) != 0 {
		t.Fatalf("el esquema del usuario es un objeto genérico: %+v", e)
	}
}

func TestCatalogoTieneLasDieciseis(t *testing.T) {
	catalogo := Catalogo()
	if len(catalogo) != 16 {
		t.Fatalf("el catálogo incluido debería tener 16 herramientas: %d", len(catalogo))
	}
	if !NombresIncluidos()["ejecutar_flujo"] {
		t.Fatal("ejecutar_flujo debería estar en el catálogo")
	}
	if _, ok := NuevoRegistro(catalogo).Buscar("editar_archivo"); !ok {
		t.Fatal("editar_archivo debería estar")
	}
}

func TestRepartirDejaPlanSoloLectura(t *testing.T) {
	catalogo := Catalogo()
	plan := Repartir([]string{"read"}, catalogo)
	for _, h := range plan {
		if h.Permiso == PermisoWrite || h.Permiso == PermisoEdit {
			t.Fatalf("plan no debería recibir herramientas de escritura: %+v", h)
		}
	}
	if len(Repartir([]string{"read", "write", "edit"}, catalogo)) != len(catalogo)-1 {
		t.Fatal("build debería recibir todo menos la de permiso derivado")
	}
}

func TestElPermisoSeCompruebaAntesDeValidar(t *testing.T) {
	r := NuevoRegistro([]Herramienta{{
		Nombre: "crear_archivo", Permiso: PermisoWrite, TipoPeticion: reflect.TypeOf(PeticionCrearArchivo{}),
		Ejecutar: func(context.Context, any, Contexto) (Resultado, error) { return Resultado{}, nil },
	}})
	// Sin permiso y con argumentos inválidos, gana la falta de permiso.
	res := r.Ejecutar(context.Background(), Peticion{Nombre: "crear_archivo", Permisos: []string{"read"}})
	if !strings.HasPrefix(res.Error, CodToolNotAllowed) {
		t.Fatalf("debería denegarse por permiso antes que por validación: %q", res.Error)
	}
}

func TestUnCampoAusenteEsCorregible(t *testing.T) {
	var ejecutado bool
	r := NuevoRegistro([]Herramienta{{
		Nombre: "leer_archivo", Permiso: PermisoRead, TipoPeticion: reflect.TypeOf(PeticionLeerArchivo{}),
		Ejecutar: func(context.Context, any, Contexto) (Resultado, error) { ejecutado = true; return Resultado{}, nil },
	}})
	res := r.Ejecutar(context.Background(), Peticion{Nombre: "leer_archivo", Permisos: []string{"read"}, Argumentos: []byte(`{}`)})
	if !strings.HasPrefix(res.Error, CodBadArgs) || !strings.Contains(res.Error, "ruta") {
		t.Fatalf("debería faltar `ruta`: %q", res.Error)
	}
	if ejecutado {
		t.Fatal("no debería ejecutarse el handler")
	}
}

func TestElRecorteRespetaElDelHandler(t *testing.T) {
	grande := strings.Repeat("x", TopeSalidaBytes+100)
	r := NuevoRegistro([]Herramienta{{
		Nombre: "ejecutar_comando", Permiso: PermisoRead, Unidad: UnidadLinea,
		TipoPeticion: reflect.TypeOf(PeticionEjecutarComando{}), Verbo: "EJEC",
		Ejecutar: func(context.Context, any, Contexto) (Resultado, error) {
			return Resultado{Salida: grande}, nil
		},
	}})
	res := r.Ejecutar(context.Background(), Peticion{Nombre: "ejecutar_comando", Permisos: []string{"read"}, Argumentos: []byte(`{"comando":"x"}`)})
	if !res.Truncado {
		t.Fatal("una salida enorme debería recortarse")
	}

	r2 := NuevoRegistro([]Herramienta{{
		Nombre: "ejecutar_comando", Permiso: PermisoRead,
		TipoPeticion: reflect.TypeOf(PeticionEjecutarComando{}),
		Ejecutar: func(context.Context, any, Contexto) (Resultado, error) {
			return Resultado{Salida: grande, Truncado: true}, nil
		},
	}})
	res2 := r2.Ejecutar(context.Background(), Peticion{Nombre: "ejecutar_comando", Permisos: []string{"read"}, Argumentos: []byte(`{"comando":"x"}`)})
	if len(res2.Salida) != len(grande) {
		t.Fatal("una salida ya recortada por el handler no debería volver a cortarse")
	}
}

func TestTemaVisibleYFinal(t *testing.T) {
	h := Herramienta{Tema: "ruta"}
	args := PeticionLeerArchivo{Ruta: "main.go"}
	if tema := TemaVisible(h, args); tema != "main.go" {
		t.Fatalf("tema = %q", tema)
	}
	if tema := TemaFinal(h, "main.go", map[string]any{"rango": "1-70"}); tema != "main.go:1-70" {
		t.Fatalf("tema parcial = %q", tema)
	}
}

func TestMedirCuentaUnidades(t *testing.T) {
	if m := Medir(UnidadLinea, "a\nb\n\nc\n"); m != "3 líneas" {
		t.Fatalf("medida = %q", m)
	}
	if m := Medir(UnidadCoincidencia, "solo"); m != "1 coincidencia" {
		t.Fatalf("medida = %q", m)
	}
}

func TestEventosDeHerramienta(t *testing.T) {
	var invocada EventoInvocacion
	var resultado EventoResultado
	r := NuevoRegistro([]Herramienta{{
		Nombre: "leer_archivo", Permiso: PermisoRead, Verbo: "LEER", Tema: "ruta", Unidad: UnidadLinea,
		TipoPeticion: reflect.TypeOf(PeticionLeerArchivo{}),
		Ejecutar: func(_ context.Context, _ any, _ Contexto) (Resultado, error) {
			return Resultado{Salida: "hola\n"}, nil
		},
	}})
	r.Eventos = Eventos{
		Invocada:  func(e EventoInvocacion) { invocada = e },
		Resultado: func(e EventoResultado) { resultado = e },
	}
	r.Ejecutar(context.Background(), Peticion{Sesion: "s1", Agente: "plan", Nombre: "leer_archivo", Permisos: []string{"read"}, Argumentos: []byte(`{"ruta":"a.go"}`)})
	if invocada.Verbo != "LEER" || invocada.Tema != "a.go" {
		t.Fatalf("invocación = %+v", invocada)
	}
	if !resultado.OK || resultado.Medida != "1 línea" {
		t.Fatalf("resultado = %+v", resultado)
	}
}

func TestDefinicionesEntregaElEsquema(t *testing.T) {
	r := NuevoRegistro(Catalogo())
	defs := r.Definiciones()
	if len(defs) != 16 {
		t.Fatalf("definiciones = %d", len(defs))
	}
	var leer Definicion
	for _, d := range defs {
		if d.Nombre == "leer_archivo" {
			leer = d
		}
	}
	var esquema map[string]any
	if err := json.Unmarshal(leer.Esquema, &esquema); err != nil {
		t.Fatalf("el esquema debería ser JSON: %v", err)
	}
	if leer.Permiso != PermisoRead {
		t.Fatalf("permiso = %q", leer.Permiso)
	}
}

// El esquema de `crear_todo` expone las dos formas y el vocabulario cerrado de
// estado y prioridad, también en las subtareas.
func TestEsquemaDeLaListaDePasos(t *testing.T) {
	h := Herramienta{TipoPeticion: reflect.TypeOf(PeticionCrearTodo{})}
	e := h.Esquema()
	for _, campo := range []string{"contenido", "estado", "prioridad", "titulo", "subtareas"} {
		if _, ok := e.Propiedades[campo]; !ok {
			t.Fatalf("`%s` debería estar en properties", campo)
		}
	}
	if len(e.Requeridos) != 0 {
		t.Fatalf("las dos formas son excluyentes y se validan en el handler: %v", e.Requeridos)
	}
	estado := e.Propiedades["estado"]
	if len(estado.Enum) != 4 || estado.Enum[0] != "pendiente" {
		t.Fatalf("el enum de estado = %v", estado.Enum)
	}
	prioridad := e.Propiedades["prioridad"]
	if len(prioridad.Enum) != 3 || prioridad.Enum[1] != "media" {
		t.Fatalf("el enum de prioridad = %v", prioridad.Enum)
	}

	subtareas := e.Propiedades["subtareas"].Items
	if subtareas == nil || subtareas.Tipo != "object" {
		t.Fatalf("subtareas debería ser un arreglo de objetos: %+v", subtareas)
	}
	for _, campo := range []string{"contenido", "estado", "prioridad"} {
		if _, ok := subtareas.Propiedades[campo]; !ok {
			t.Fatalf("la subtarea debería exponer `%s`", campo)
		}
	}
	if _, anidada := subtareas.Propiedades["subtareas"]; anidada {
		t.Fatal("una subtarea no puede tener hijas: la jerarquía es de un solo nivel")
	}
	if len(subtareas.Requeridos) != 2 {
		t.Fatalf("contenido y estado deberían ser obligatorios en la subtarea: %v", subtareas.Requeridos)
	}
}

// `actualizar_todo` anida el objeto de cada elemento y admite sus subtareas.
func TestEsquemaDeActualizarTodoAnidaElElemento(t *testing.T) {
	h := Herramienta{TipoPeticion: reflect.TypeOf(PeticionActualizarTodo{})}
	items := h.Esquema().Propiedades["elementos"].Items
	if items == nil || items.Tipo != "object" {
		t.Fatalf("elementos debería ser un arreglo de objetos: %+v", items)
	}
	if _, ok := items.Propiedades["subtareas"]; !ok {
		t.Fatal("cada elemento debería admitir `subtareas`")
	}
	if items.Propiedades["estado"].Enum == nil {
		t.Fatal("el estado del elemento debería llevar su enum")
	}
}
