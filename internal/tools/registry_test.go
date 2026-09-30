package tools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// registroStub arma el registro del catálogo cerrado con un handler propio por
// herramienta que devuelve `marca`. Cuenta las llamadas para poder comprobar
// que una petición rechazada NO llega al handler.
func registroStub(marca string, llamado *int) *Registro {
	var impls []Herramienta
	for _, nombre := range NombresCatalogo() {
		n := nombre
		h, _ := NuevaHerramienta(n, func(ctx context.Context, args any, c Contexto) (Resultado, error) {
			if llamado != nil {
				*llamado++
			}
			return Resultado{Salida: marca + ":" + n}, nil
		})
		impls = append(impls, h)
	}
	r, err := NuevoRegistro(impls)
	if err != nil {
		panic(err)
	}
	return r
}

// TestRegistroExigeHerramientas — montar el registro sin ninguna herramienta es
// un error de cableado, no algo que deba reventar en tiempo de ejecución.
func TestRegistroExigeHerramientas(t *testing.T) {
	if _, err := NuevoRegistro(nil); err == nil {
		t.Fatal("quiero error por registro vacío")
	}
	impl, _ := NuevaHerramienta("leer_archivo", nil)
	if _, err := NuevoRegistro([]Herramienta{impl, impl}); err == nil {
		t.Fatal("quiero error por herramienta declarada dos veces")
	}
}

// TestEjecutarLlegaAlHandlerDeCadaHerramienta — el handler propio sustituye al
// enrutado por categorías: cada nombre ejecuta el suyo.
func TestEjecutarLlegaAlHandlerDeCadaHerramienta(t *testing.T) {
	r := registroStub("ok", nil)
	res, err := r.Ejecutar(context.Background(), Peticion{
		Agente:      AgenteBuild,
		Permisos:    AccionesDeBuild(),
		Herramienta: "listar_carpeta",
		Argumentos:  json.RawMessage(`{"ruta":"."}`),
	})
	if err != nil {
		t.Fatalf("Ejecutar: %v", err)
	}
	if res.Salida != "ok:listar_carpeta" {
		t.Errorf("salida = %q, quiero la del handler propio", res.Salida)
	}
}

// TestPermisoAntesDeValidarYDeEjecutar — el permiso va primero: `plan` pidiendo
// escritura es E_TOOL_NOT_ALLOWED y no se valida ni se ejecuta nada.
func TestPermisoAntesDeValidarYDeEjecutar(t *testing.T) {
	var llamado int
	r := registroStub("x", &llamado)
	_, err := r.Ejecutar(context.Background(), Peticion{
		Agente:      AgentePlan,
		Permisos:    AccionesDePlan(),
		Herramienta: "crear_archivo",
		// Argumentos INCOMPLETOS a propósito: el permiso va antes.
		Argumentos: json.RawMessage(`{}`),
	})
	if !errors.Is(err, ErrHerramientaNoPermitida) {
		t.Fatalf("err = %v, quiero E_TOOL_NOT_ALLOWED", err)
	}
	if llamado != 0 {
		t.Errorf("el handler se llamó sin permiso")
	}
}

// TestValidacionVuelveCorregible — un payload incompleto NO corta el turno:
// vuelve como Resultado.Error con el motivo y con lo que se esperaba.
func TestValidacionVuelveCorregible(t *testing.T) {
	r := registroStub("x", nil)
	res, err := r.Ejecutar(context.Background(), Peticion{
		Agente:      AgenteBuild,
		Permisos:    AccionesDeBuild(),
		Herramienta: "leer_archivo",
		Argumentos:  json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("una validación fallida no es un error duro: %v", err)
	}
	if res.Error == "" {
		t.Fatal("quiero un Resultado.Error corregible")
	}
	if !strings.Contains(res.Error, "ruta") {
		t.Errorf("el motivo debe nombrar el campo que falta: %q", res.Error)
	}
}

// TestHerramientaFueraDelCatalogoNoSeEjecuta — una herramienta fuera del
// catálogo no se ejecuta; el modelo recibe el motivo.
func TestHerramientaFueraDelCatalogoNoSeEjecuta(t *testing.T) {
	r := registroStub("x", nil)
	res, err := r.Ejecutar(context.Background(), Peticion{
		Agente:      AgenteBuild,
		Permisos:    AccionesDeBuild(),
		Herramienta: "borrar_todo",
		Argumentos:  json.RawMessage(`{}`),
	})
	if err != nil || res.Error == "" {
		t.Fatalf("res=%+v err=%v, quiero un resultado con el motivo", res, err)
	}
}

// TestRecorteUniversal — TODA salida se recorta, no solo la de la terminal, y
// el recorte se marca para que el modelo lo sepa.
func TestRecorteUniversal(t *testing.T) {
	impls := []Herramienta{}
	for _, nombre := range NombresCatalogo() {
		h, _ := NuevaHerramienta(nombre, func(ctx context.Context, args any, c Contexto) (Resultado, error) {
			return Resultado{Salida: strings.Repeat("x", 100000)}, nil
		})
		impls = append(impls, h)
	}
	r, err := NuevoRegistro(impls)
	if err != nil {
		t.Fatal(err)
	}
	res, err := r.Ejecutar(context.Background(), Peticion{
		Agente:      AgenteBuild,
		Permisos:    AccionesDeBuild(),
		Herramienta: "leer_archivo",
		Argumentos:  json.RawMessage(`{"ruta":"a.md"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Truncado {
		t.Error("una salida enorme debe recortarse y marcarse")
	}
	if EstimarTokens(res.Salida) > r.LimiteTokens {
		t.Errorf("la salida recortada todavía excede el límite: %d", EstimarTokens(res.Salida))
	}
}

// TestEventosDeTodaEjecucion — `herramienta_invocada` y `herramienta_resultado`
// salen de la capa universal, sin instrumentar cada handler.
func TestEventosDeTodaEjecucion(t *testing.T) {
	pub := &publicadorGrabador{}
	r := registroStub("ok", nil)
	r.Eventos = pub
	_, err := r.Ejecutar(context.Background(), Peticion{
		Agente:      AgenteBuild,
		SesionID:    "s1",
		Permisos:    AccionesDeBuild(),
		Herramienta: "leer_archivo",
		Argumentos:  json.RawMessage(`{"ruta":"a.md"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if pub.invocada != 1 || pub.resultado != 1 {
		t.Fatalf("eventos: invocada=%d resultado=%d, quiero 1 y 1", pub.invocada, pub.resultado)
	}
	if pub.sesion != "s1" {
		t.Errorf("el evento debe decir de qué sesión es: %q", pub.sesion)
	}
	if pub.agente != AgenteBuild {
		t.Errorf("el evento debe decir qué agente la pidió: %q", pub.agente)
	}
	if pub.verbo != "LEER" {
		t.Errorf("la invocación lleva el verbo de pantalla: %q", pub.verbo)
	}
	// El evento lleva SOLO el tema (la ruta), no el resto de argumentos.
	if pub.tema != "a.md" {
		t.Errorf("el tema debe ser la ruta de la petición: %q", pub.tema)
	}
	if pub.medida != "1 línea" {
		t.Errorf("el resultado se mide en su unidad: %q", pub.medida)
	}
}

type publicadorGrabador struct {
	invocada, resultado   int
	sesion, agente, verbo string
	tema                  string
	medida                string
}

func (p *publicadorGrabador) HerramientaInvocada(sesion, nombre, agente, verbo, tema string) {
	p.invocada++
	p.sesion = sesion
	p.agente = agente
	p.verbo = verbo
	p.tema = tema
}

func (p *publicadorGrabador) HerramientaResultado(sesion, nombre string, ok bool, err string, truncado bool, medida string) {
	p.resultado++
	p.medida = medida
}
