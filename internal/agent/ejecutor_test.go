package agent

import (
	"context"
	"strings"
	"testing"

	"localcli/internal/llm"
)

// motorGuion devuelve respuestas prefijadas, una por ronda, y guarda lo pedido.
type motorGuion struct {
	respuestas []Respuesta
	pedidos    []Peticion
	i          int
}

func (m *motorGuion) Ronda(_ context.Context, p Peticion, alDelta func(Delta) error) (Respuesta, error) {
	m.pedidos = append(m.pedidos, p)
	var r Respuesta
	if m.i < len(m.respuestas) {
		r = m.respuestas[m.i]
	}
	m.i++
	if r.Texto != "" && alDelta != nil {
		if err := alDelta(Delta{Texto: r.Texto}); err != nil {
			return r, err
		}
	}
	return r, nil
}

// ejecutorGuion registra el orden de ejecución y deja forzar resultados.
type ejecutorGuion struct {
	orden  []string
	fallos map[string]Resultado
}

func (e *ejecutorGuion) Ejecutar(_ context.Context, _ string, l Llamada) (Resultado, error) {
	e.orden = append(e.orden, l.Nombre)
	if r, ok := e.fallos[l.Nombre]; ok {
		return r, nil
	}
	return Resultado{Salida: "salida de " + l.Nombre}, nil
}

func TestSinCapacidadDeHerramientasCaeAModoConversacion(t *testing.T) {
	for _, capacidad := range []llm.Capacidad{llm.NoSoportada, llm.Desconocida} {
		motor := &motorGuion{respuestas: []Respuesta{{Texto: "solo converso"}}}
		var avisos int
		e := &Ejecutor{
			Agente:    Agente{Nombre: "plan"},
			Cerrado:   catalogoDePrueba(),
			Motor:     motor,
			Capacidad: capacidad,
			Avisar:    func(string, string) { avisos++ },
		}

		respuesta, err := e.Turno(context.Background(), nil, nil)
		if err != nil || respuesta.Texto != "solo converso" {
			t.Fatalf("capacidad %d: turno = %+v, %v", capacidad, respuesta, err)
		}
		if len(motor.pedidos) != 1 || len(motor.pedidos[0].Herramientas) != 0 {
			t.Fatalf("no debería ofrecer herramientas en conversación: %+v", motor.pedidos)
		}
		if avisos == 0 {
			t.Fatal("debería avisar del modo conversación")
		}
	}
}

func TestElTurnoEjecutaEnOrdenYVuelveAlModelo(t *testing.T) {
	motor := &motorGuion{respuestas: []Respuesta{
		{Llamadas: []Llamada{{ID: "1", Nombre: "leer_archivo"}, {ID: "2", Nombre: "crear_archivo"}}},
		{Texto: "hecho", Uso: Uso{Entrada: 9, Salida: 4}},
	}}
	exec := &ejecutorGuion{}
	e := &Ejecutor{
		Agente:       Agente{Nombre: "build", Permissions: map[string]string{"read": EfectoAllow, "write": EfectoAllow}},
		Cerrado:      catalogoDePrueba(),
		Motor:        motor,
		Herramientas: exec,
		Capacidad:    llm.Soportada,
	}

	respuesta, err := e.Turno(context.Background(), []Mensaje{{Rol: RolUsuario, Contenido: "lee y crea"}}, nil)
	if err != nil || respuesta.Texto != "hecho" {
		t.Fatalf("turno = %+v, %v", respuesta, err)
	}
	if respuesta.Uso != (Uso{Entrada: 9, Salida: 4}) {
		t.Fatalf("el uso de la última ronda debería volver: %+v", respuesta.Uso)
	}
	if len(motor.pedidos) != 2 {
		t.Fatalf("leer y escribir debería costar al menos dos rondas: %d", len(motor.pedidos))
	}
	if strings.Join(exec.orden, ",") != "leer_archivo,crear_archivo" {
		t.Fatalf("las herramientas deberían ejecutarse en el orden pedido: %v", exec.orden)
	}
	ultima := motor.pedidos[1].Mensajes
	fin := ultima[len(ultima)-1]
	if fin.Rol != RolHerramienta || fin.IDLlamada != "2" {
		t.Fatalf("el resultado debería volver como mensaje de herramienta con su id: %+v", fin)
	}
}

func TestUnFalloCorregibleNoCortaElTurno(t *testing.T) {
	motor := &motorGuion{respuestas: []Respuesta{
		{Llamadas: []Llamada{{ID: "1", Nombre: "leer_archivo"}}},
		{Texto: "quiero cerrar"},
		{Llamadas: []Llamada{{ID: "2", Nombre: "ejecutar_comando"}}},
		{Texto: "corregido"},
	}}
	exec := &ejecutorGuion{fallos: map[string]Resultado{"leer_archivo": {Error: "falta el campo ruta"}}}
	e := &Ejecutor{
		Agente:       Agente{Nombre: "plan", Permissions: map[string]string{"read": EfectoAllow}},
		Cerrado:      catalogoDePrueba(),
		Motor:        motor,
		Herramientas: exec,
		Capacidad:    llm.Soportada,
	}

	respuesta, err := e.Turno(context.Background(), nil, nil)
	if err != nil || respuesta.Texto != "corregido" {
		t.Fatalf("un fallo corregible no debería cortar el turno: %+v, %v", respuesta, err)
	}
	// El fallo vuelve al modelo en la ronda siguiente.
	segunda := motor.pedidos[1].Mensajes
	fin := segunda[len(segunda)-1]
	if fin.Rol != RolHerramienta || !strings.Contains(fin.Contenido, "falta el campo ruta") {
		t.Fatalf("el fallo debería volver al modelo: %+v", fin)
	}
	// Tras el intento de cerrar con texto, se le empuja a continuar.
	tercera := motor.pedidos[2].Mensajes
	ultimo := tercera[len(tercera)-1]
	if ultimo.Rol != RolSistema || !strings.Contains(ultimo.Contenido, "no cierres el turno") {
		t.Fatalf("debería inyectarse el empujón de continuación: %+v", ultimo)
	}
	// El acierto limpia el fallo y la respuesta en texto se acepta.
	if strings.Join(exec.orden, ",") != "leer_archivo,ejecutar_comando" {
		t.Fatalf("el modelo debería reintentar con otra herramienta: %v", exec.orden)
	}
}

func TestElUltimoEmpujonReintroduceLaPeticionDelUsuario(t *testing.T) {
	llamada := []Llamada{{ID: "1", Nombre: "leer_archivo"}}
	motor := &motorGuion{respuestas: []Respuesta{
		{Llamadas: llamada},
		{Texto: "no lo encuentro"},
		{Texto: "sigo sin encontrarlo"},
		{Texto: "¿qué hago?"},
		{Texto: "listo"},
	}}
	exec := &ejecutorGuion{fallos: map[string]Resultado{"leer_archivo": {Error: "el archivo no existe"}}}
	e := &Ejecutor{
		Agente:       Agente{Nombre: "plan", Permissions: map[string]string{"read": EfectoAllow}},
		Cerrado:      catalogoDePrueba(),
		Motor:        motor,
		Herramientas: exec,
		Capacidad:    llm.Soportada,
	}
	historial := []Mensaje{{Rol: RolUsuario, Contenido: "mira .agents/commands/actualizar.md"}}

	respuesta, err := e.Turno(context.Background(), historial, nil)
	if err != nil || respuesta.Texto != "listo" {
		t.Fatalf("el turno debería cerrar tras el último empujón: %+v, %v", respuesta, err)
	}
	// El primer empujón es genérico: no filtra la petición original.
	if ultimoDe(motor.pedidos[2].Mensajes) != instruccionContinuar {
		t.Fatalf("el primer empujón debería ser el genérico: %+v", motor.pedidos[2].Mensajes)
	}
	// El último sí, con la ruta tal cual la escribió el usuario.
	if !strings.Contains(ultimoDe(motor.pedidos[4].Mensajes), "mira .agents/commands/actualizar.md") {
		t.Fatal("el último empujón debería reintroducir la petición original")
	}
}

// ultimoDe devuelve el contenido del último mensaje de la lista.
func ultimoDe(mensajes []Mensaje) string {
	return mensajes[len(mensajes)-1].Contenido
}

func TestElTopeDeEmpujonesSeRespeta(t *testing.T) {
	llamada := []Llamada{{ID: "1", Nombre: "leer_archivo"}}
	motor := &motorGuion{respuestas: []Respuesta{
		{Llamadas: llamada},
		{Texto: "no puedo"},
		{Texto: "de verdad no puedo"},
	}}
	exec := &ejecutorGuion{fallos: map[string]Resultado{"leer_archivo": {Error: "sin ruta"}}}
	e := &Ejecutor{
		Agente:       Agente{Nombre: "plan", Permissions: map[string]string{"read": EfectoAllow}},
		Cerrado:      catalogoDePrueba(),
		Motor:        motor,
		Herramientas: exec,
		Capacidad:    llm.Soportada,
		MaxEmpujones: 1,
	}

	respuesta, err := e.Turno(context.Background(), nil, nil)
	if err != nil || respuesta.Texto != "de verdad no puedo" {
		t.Fatalf("agotado el tope, se acepta la respuesta del modelo: %+v, %v", respuesta, err)
	}
	if len(motor.pedidos) != 3 {
		t.Fatalf("con un solo empujón deberían bastar tres rondas: %d", len(motor.pedidos))
	}
}

func TestAlAgotarLasRondasRedactaSinHerramientas(t *testing.T) {
	motor := &motorGuion{respuestas: []Respuesta{
		{Llamadas: []Llamada{{ID: "1", Nombre: "leer_archivo"}}},
		{Llamadas: []Llamada{{ID: "2", Nombre: "leer_archivo"}}},
		{Texto: "aquí va mi resultado"},
	}}
	e := &Ejecutor{
		Agente:       Agente{Nombre: "plan", Permissions: map[string]string{"read": EfectoAllow}},
		Cerrado:      catalogoDePrueba(),
		Motor:        motor,
		Herramientas: &ejecutorGuion{},
		Capacidad:    llm.Soportada,
		MaxRondas:    2,
	}

	respuesta, err := e.Turno(context.Background(), nil, nil)
	if err != nil || respuesta.Texto != "aquí va mi resultado" {
		t.Fatalf("el turno debería cerrar con la redacción: %+v, %v", respuesta, err)
	}
	ultima := motor.pedidos[len(motor.pedidos)-1]
	if len(ultima.Herramientas) != 0 {
		t.Fatalf("la redacción no debería ofrecer herramientas: %+v", ultima.Herramientas)
	}
}

func TestRedaccionQueInsisteEnHerramientasFalla(t *testing.T) {
	llamada := []Llamada{{ID: "1", Nombre: "leer_archivo"}}
	motor := &motorGuion{respuestas: []Respuesta{
		{Llamadas: llamada},
		{Llamadas: llamada},
		{Llamadas: llamada},
	}}
	var avisos int
	e := &Ejecutor{
		Agente:       Agente{Nombre: "plan", Permissions: map[string]string{"read": EfectoAllow}},
		Cerrado:      catalogoDePrueba(),
		Motor:        motor,
		Herramientas: &ejecutorGuion{},
		Capacidad:    llm.Soportada,
		MaxRondas:    1,
		Avisar:       func(string, string) { avisos++ },
	}

	respuesta, err := e.Turno(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("si el modelo no entrega texto, el turno debería fallar con el motivo")
	}
	if respuesta.Texto != "" {
		t.Fatalf("no debería devolverse texto: %+v", respuesta)
	}
	if avisos == 0 {
		t.Fatal("el fallo debería avisarse")
	}
}

func TestUnaLlamadaEscritaComoTextoSeEjecutaYNoSePinta(t *testing.T) {
	motor := &motorGuion{respuestas: []Respuesta{
		{Texto: "```json\n{\n  \"name\": \"ejecutar_comando\",\n  \"arguments\": {\n    \"comando\": \"go build ./...\"\n  }\n}\n```"},
		{Texto: "listo"},
	}}
	exec := &ejecutorGuion{}
	e := &Ejecutor{
		Agente:       Agente{Nombre: "build", Permissions: map[string]string{"read": EfectoAllow}},
		Cerrado:      catalogoDePrueba(),
		Motor:        motor,
		Herramientas: exec,
		Capacidad:    llm.Soportada,
	}

	var vistos []string
	respuesta, err := e.Turno(context.Background(), nil, func(d Delta) error {
		if !d.Razonamiento {
			vistos = append(vistos, d.Texto)
		}
		return nil
	})
	if err != nil || respuesta.Texto != "listo" {
		t.Fatalf("turno = %+v, %v", respuesta, err)
	}
	if strings.Join(exec.orden, ",") != "ejecutar_comando" {
		t.Fatalf("la llamada escrita debería ejecutarse: %v", exec.orden)
	}
	if texto := strings.Join(vistos, ""); texto != "listo" {
		t.Fatalf("el JSON de la llamada no debería pintarse: %q", texto)
	}

	// El mensaje del asistente lleva la llamada reconocida y su id, para que el
	// modelo vea su propia petición y el resultado en la ronda siguiente.
	segunda := motor.pedidos[1].Mensajes
	if len(segunda) < 3 {
		t.Fatalf("faltan el asistente y el resultado: %+v", segunda)
	}
	asistente, resultado := segunda[len(segunda)-2], segunda[len(segunda)-1]
	if asistente.Rol != RolAsistente || len(asistente.Llamadas) != 1 {
		t.Fatalf("el asistente debería llevar la llamada: %+v", asistente)
	}
	if asistente.Llamadas[0].Nombre != "ejecutar_comando" || asistente.Llamadas[0].ID == "" {
		t.Fatalf("la llamada debería llevar nombre e id: %+v", asistente.Llamadas[0])
	}
	if resultado.Rol != RolHerramienta || resultado.IDLlamada != asistente.Llamadas[0].ID {
		t.Fatalf("el resultado debería responder a esa llamada: %+v", resultado)
	}
}

func TestVariasLlamadasEscritasComoTextoSeEjecutanEnOrden(t *testing.T) {
	motor := &motorGuion{respuestas: []Respuesta{
		{Texto: "```json\n{\"name\":\"leer_archivo\",\"arguments\":{\"ruta\":\"main.go\"}}\n```\n```json\n{\"name\":\"ejecutar_comando\",\"arguments\":{\"comando\":\"go build ./...\"}}\n```"},
		{Texto: "listo"},
	}}
	exec := &ejecutorGuion{}
	e := &Ejecutor{
		Agente:       Agente{Nombre: "build", Permissions: map[string]string{"read": EfectoAllow}},
		Cerrado:      catalogoDePrueba(),
		Motor:        motor,
		Herramientas: exec,
		Capacidad:    llm.Soportada,
	}

	var vistos []string
	respuesta, err := e.Turno(context.Background(), nil, func(d Delta) error {
		if !d.Razonamiento {
			vistos = append(vistos, d.Texto)
		}
		return nil
	})
	if err != nil || respuesta.Texto != "listo" {
		t.Fatalf("turno = %+v, %v", respuesta, err)
	}
	if strings.Join(exec.orden, ",") != "leer_archivo,ejecutar_comando" {
		t.Fatalf("las dos llamadas escritas deberían ejecutarse en orden: %v", exec.orden)
	}
	if texto := strings.Join(vistos, ""); texto != "listo" {
		t.Fatalf("el JSON de las llamadas no debería pintarse: %q", texto)
	}

	segunda := motor.pedidos[1].Mensajes
	asistente := segunda[len(segunda)-3]
	if asistente.Rol != RolAsistente || len(asistente.Llamadas) != 2 {
		t.Fatalf("el asistente debería llevar las dos llamadas: %+v", asistente)
	}
	if asistente.Llamadas[0].ID == asistente.Llamadas[1].ID {
		t.Fatalf("cada llamada necesita su propio id: %+v", asistente.Llamadas)
	}
	for i, llamada := range asistente.Llamadas {
		resultado := segunda[len(segunda)-2+i]
		if resultado.Rol != RolHerramienta || resultado.IDLlamada != llamada.ID {
			t.Fatalf("el resultado %d debería responder a su llamada: %+v", i, resultado)
		}
	}
}

func TestUnaLlamadaDeTextoAUnaHerramientaNoConcedidaNoSeEjecuta(t *testing.T) {
	motor := &motorGuion{respuestas: []Respuesta{
		{Texto: `{"name":"crear_archivo","arguments":{"ruta":"x","contenido":"y"}}`},
	}}
	exec := &ejecutorGuion{}
	e := &Ejecutor{
		Agente:       Agente{Nombre: "plan", Permissions: map[string]string{"read": EfectoAllow}},
		Cerrado:      catalogoDePrueba(),
		Motor:        motor,
		Herramientas: exec,
		Capacidad:    llm.Soportada,
	}

	respuesta, err := e.Turno(context.Background(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(exec.orden) != 0 {
		t.Fatalf("no debería ejecutarse una herramienta no concedida: %v", exec.orden)
	}
	if !strings.Contains(respuesta.Texto, "crear_archivo") {
		t.Fatalf("el texto debería devolverse tal cual: %q", respuesta.Texto)
	}
}
