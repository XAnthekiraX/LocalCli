package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// runnerGrabador es un doble de `Runner`: recuerda el equipo y la entrada con
// la que se le llamó.
type runnerGrabador struct {
	equipo  []string
	entrada []byte
	llamado int
}

func (r *runnerGrabador) EjecutarEquipo(ctx context.Context, equipo []string, entrada []byte, timeout time.Duration) (RespuestaEjecutarComando, error) {
	r.llamado++
	r.equipo = append([]string(nil), equipo...)
	r.entrada = append([]byte(nil), entrada...)
	return RespuestaEjecutarComando{Salida: "ok", Termino: true}, nil
}

func escribirJSON(t *testing.T, dir, nombre, contenido string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, nombre), []byte(contenido), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestCargaHerramientasDelUsuario — T-B024-08: un JSON válido se carga y uno
// inválido se salta sin romper el arranque.
func TestCargaHerramientasDelUsuario(t *testing.T) {
	dir := t.TempDir()
	escribirJSON(t, dir, "contar_lineas.json",
		`{"nombre":"contar_lineas","descripcion":"Cuenta líneas.","modo":"lee","equipo":["wc","-l"]}`)
	escribirJSON(t, dir, "roto.json", `{no es json`)
	escribirJSON(t, dir, "incompleta.json", `{"nombre":"sin_equipo","descripcion":"x"}`)

	validas, avisos := CargarHerramientasUsuario(dir)
	if len(validas) != 1 {
		t.Fatalf("válidas = %d, quiero 1: %+v", len(validas), validas)
	}
	if validas[0].Nombre != "contar_lineas" {
		t.Errorf("nombre = %q", validas[0].Nombre)
	}
	if len(avisos) != 2 {
		t.Errorf("avisos = %d, quiero 2 (JSON roto y campo obligatorio ausente): %v", len(avisos), avisos)
	}
}

// TestModoEscribeSeRechaza — declarar `escribe` no se admite: la única vía de
// escritura son las herramientas de archivo.
func TestModoEscribeSeRechaza(t *testing.T) {
	dir := t.TempDir()
	escribirJSON(t, dir, "escribe.json",
		`{"nombre":"escribe_algo","descripcion":"x","modo":"escribe","equipo":["rm","-rf","."]}`)
	validas, avisos := CargarHerramientasUsuario(dir)
	if len(validas) != 0 {
		t.Fatalf("`modo: escribe` no puede cargarse: %+v", validas)
	}
	if len(avisos) != 1 || !strings.Contains(avisos[0], "escribe") {
		t.Errorf("el aviso debe explicar el motivo: %v", avisos)
	}
}

// TestNombreQueChocaNoSeCarga — no hay namespacing: un nombre que coincide con
// una incluida no se carga, y se avisa.
func TestNombreQueChocaNoSeCarga(t *testing.T) {
	dir := t.TempDir()
	escribirJSON(t, dir, "colision.json",
		`{"nombre":"leer_archivo","descripcion":"x","equipo":["cat"]}`)
	validas, avisos := CargarHerramientasUsuario(dir)
	if len(validas) != 0 {
		t.Fatalf("el nombre que choca no puede cargarse: %+v", validas)
	}
	if len(avisos) != 1 || !strings.Contains(avisos[0], "incluida") {
		t.Errorf("aviso inesperado: %v", avisos)
	}
}

// TestHerramientaDelUsuarioSiemprePideAprobacion — aunque el equipo sea un
// comando inocente, pide permiso siempre; y si se declina, no se ejecuta.
func TestHerramientaDelUsuarioSiemprePideAprobacion(t *testing.T) {
	runner := &runnerGrabador{}
	h := NuevaHerramientaUsuario(HerramientaUsuario{
		Nombre: "contar_lineas", Descripcion: "x", Modo: ModoLee, Equipo: []string{"wc", "-l"},
	}, runner)

	pedidas := 0
	c := Contexto{Ask: func(ctx context.Context, s Solicitud) (Decision, error) {
		pedidas++
		return Decision{Aprobada: false}, nil
	}}
	res, err := h.Ejecutar(context.Background(), map[string]any{"ruta": "a.md"}, c)
	if err != nil {
		t.Fatalf("Ejecutar: %v", err)
	}
	if pedidas != 1 {
		t.Errorf("aprobaciones pedidas = %d, quiero 1", pedidas)
	}
	if runner.llamado != 0 {
		t.Error("declinada, la herramienta no puede ejecutarse")
	}
	if res.Error == "" {
		t.Error("quiero un resultado explicando la declinación")
	}
}

// TestHerramientaDelUsuarioSeEjecutaConSuEquipo — aprobada, corre con
// exactamente el equipo declarado y recibe los argumentos por la entrada
// estándar, nunca en la línea de comandos.
func TestHerramientaDelUsuarioSeEjecutaConSuEquipo(t *testing.T) {
	runner := &runnerGrabador{}
	h := NuevaHerramientaUsuario(HerramientaUsuario{
		Nombre: "contar_lineas", Descripcion: "x", Modo: ModoLee, Equipo: []string{"wc", "-l"},
	}, runner)

	c := Contexto{Ask: func(ctx context.Context, s Solicitud) (Decision, error) {
		return Decision{Aprobada: true}, nil
	}}
	res, err := h.Ejecutar(context.Background(), map[string]any{"ruta": "a.md"}, c)
	if err != nil {
		t.Fatalf("Ejecutar: %v", err)
	}
	if res.Salida != "ok" {
		t.Errorf("salida = %q", res.Salida)
	}
	if len(runner.equipo) != 2 || runner.equipo[0] != "wc" || runner.equipo[1] != "-l" {
		t.Errorf("equipo = %v, quiero exactamente el declarado", runner.equipo)
	}
	for _, a := range runner.equipo {
		if strings.Contains(a, "a.md") {
			t.Error("los argumentos del modelo no pueden llegar a la línea de comandos")
		}
	}
	if !strings.Contains(string(runner.entrada), "a.md") {
		t.Errorf("los argumentos deben viajar por la entrada estándar: %q", runner.entrada)
	}
}

// TestHerramientaDelUsuarioEsDeLecturaYNoUsaEsquemaPropio — su esquema es un
// objeto genérico: el harness no conoce el ejecutable (TOOLS-DTO.md §3).
func TestHerramientaDelUsuarioEsDeLecturaYNoUsaEsquemaPropio(t *testing.T) {
	h := NuevaHerramientaUsuario(HerramientaUsuario{Nombre: "x", Descripcion: "d", Equipo: []string{"true"}}, &runnerGrabador{})
	if h.Modo != Lee || h.SoloBuild() {
		t.Errorf("una herramienta del usuario es de lectura: %+v", h)
	}
	if h.Accion() != AccionLeer {
		t.Errorf("acción = %s, quiero leer", h.Accion())
	}
	if h.Esquema == nil || h.Esquema.Type != "object" || len(h.Esquema.Properties) != 0 {
		t.Errorf("esquema = %+v, quiero un objeto sin campos", h.Esquema)
	}
}
