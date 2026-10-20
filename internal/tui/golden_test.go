package tui

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"localcli/internal/tui/components/chat"
	"localcli/internal/tui/components/input"
	"localcli/internal/tui/components/modals"
	"localcli/internal/tui/components/panel"
	"localcli/internal/tui/render"
)

// Salidas doradas de las vistas (TESTING §3): «las vistas se comparan contra
// salidas doradas en lo que a formato respecta: un cambio de estilo se revisa a
// la vista, no a ciegas». Las doradas se comparan sin códigos de color, para que
// se puedan revisar a simple vista, y se regeneran a conciencia con
// `go test ./internal/tui/ -update`.
//
// Todo lo que entra en una dorada es fijo: geometría, datos y tiempos. Ni el
// reloj ni el puerto participan, así que la misma máquina y otra cualquiera
// obtienen el mismo archivo.
var actualizarDoradas = flag.Bool("update", false, "regenera las salidas doradas de testdata/")

// dorada compara el marco con su archivo de testdata, sin colores. Con `-update`
// lo reescribe y lo anuncia: regenerar es una decisión, no un accidente.
func dorada(t *testing.T, nombre, marco string) {
	t.Helper()
	got := render.SinANSI(marco) + "\n"
	ruta := filepath.Join("testdata", nombre)
	if *actualizarDoradas {
		if err := os.WriteFile(ruta, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("dorada %s regenerada: revísala a la vista antes de aprobarla", ruta)
		return
	}
	quiero, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatalf("no se pudo leer la dorada %s: %v (genérala con -update)", ruta, err)
	}
	if got == string(quiero) {
		return
	}
	t.Errorf("la vista cambió frente a %s (regénérala con -update si el cambio es intencionado):\n%s",
		ruta, primerasDiferentes(string(quiero), got))
}

// primerasDiferentes muestra las primeras filas que no coinciden, con el número
// de fila delante: sin esto, un cambio de un espacio obliga a buscar a ojo.
func primerasDiferentes(quiero, got string) string {
	a, b := strings.Split(quiero, "\n"), strings.Split(got, "\n")
	var salida strings.Builder
	for i := 0; i < len(a) || i < len(b); i++ {
		var enQuiero, enGot string
		if i < len(a) {
			enQuiero = a[i]
		}
		if i < len(b) {
			enGot = b[i]
		}
		if enQuiero == enGot {
			continue
		}
		salida.WriteString("\n  fila " + itoa(i) + " dorada: " + enQuiero)
		salida.WriteString("\n  fila " + itoa(i) + " render: " + enGot)
		if salida.Len() > 2000 {
			break
		}
	}
	return salida.String()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digitos []byte
	for n > 0 {
		digitos = append([]byte{byte('0' + n%10)}, digitos...)
		n /= 10
	}
	return string(digitos)
}

// appDorada deja la app con geometría fija, sin el puerto ni el chat de muestra:
// lo que se pinta es lo que la prueba pone.
func appDorada(ancho, alto int, vista Vista) *App {
	app := Nueva(nil)
	app.ancho, app.alto = ancho, alto
	app.vista = vista
	app.agente, app.modelo, app.motor = AgenteBuild, "qwen3:8b", "llama.cpp"
	app.entrada.Agente, app.entrada.Modelo, app.entrada.Motor = app.agente, app.modelo, app.motor
	app.entrada.Capacidades = input.Capacidades{
		Herramientas: input.Soportada,
		Vision:       input.NoSoportada,
		Razonamiento: input.Soportada,
	}
	app.sesion = &Sesion{ID: "s1", Nombre: "arreglar el login", Estado: EstadoTrabajando}
	return app
}

// datosDeDorada son los nueve datos del panel, congelados.
func datosDeDorada() panel.Datos {
	return panel.Datos{
		Sesion:             "arreglar el login",
		Estado:             EstadoTrabajando,
		Agente:             AgenteBuild,
		ContextoTokens:     1234,
		TokensEstimados:    true,
		LimiteTokens:       2000,
		ElementoActual:     "T-F009",
		ElementosRestantes: 3,
		Capa:               "frontend",
		TareasGrandes:      2,
		Aprobaciones:       1,
		Ruta:               "/tmp/proyecto",
		GitRama:            "refact/logic",
		GitCambios:         2,
		Proyecto:           render.Nombre,
		Version:            render.Version,
		Tareas: []panel.Tarea{
			{Contenido: "escribir las doradas", Estado: panel.EnProgreso},
			{Contenido: "unitarias de render", Estado: panel.Completada},
			{Contenido: "probar los modales", Estado: panel.Cancelada},
			{Contenido: "revisar el marco", Estado: panel.Pendiente},
		},
	}
}

// hiloDeDorada llena el chat con un intercambio completo —texto, herramienta,
// fallo recortado y aviso—, con tiempos fijos.
func hiloDeDorada(app *App) {
	app.chat.AñadirSegmento(chat.Segmento{Hablante: chat.Usuario, Texto: "arregla el login"})
	app.chat.AñadirSegmento(chat.Segmento{
		Hablante: chat.Agente,
		Texto:    "Voy a revisar el flujo de autenticación y después las pruebas.",
		Duracion: 1500 * time.Millisecond,
	})
	app.chat.AnotarEtapa("Diagnosticar")
	app.chat.AnotarInvocacion("leer", "LEER", "internal/tui/app.go")
	app.chat.CerrarHerramienta("leer", true, false, "70 líneas", "", 400*time.Millisecond, "", "")
	app.chat.AnotarInvocacion("ejec", "EJEC", "go test ./...")
	app.chat.CerrarHerramienta("ejec", false, true, "42 líneas", "el comando falló", 1200*time.Millisecond, "", "")
	app.chat.AnotarAviso("cambio aplicado en internal/tui/app.go")
}

// La bienvenida se pinta centrada: logotipo, caja con su línea de modelo, pistas
// y versión (SPEC-INTERFAZ §Pantalla de bienvenida).
func TestLaBienvenidaCoincideConSuDorada(t *testing.T) {
	app := appDorada(100, 30, VistaBienvenida)
	dorada(t, "bienvenida.golden", app.View())
}

// La vista principal se pinta con el chat, la caja de entrada al pie y el sidebar
// de datos a la derecha (SPEC-INTERFAZ §Disposición).
func TestLaVistaPrincipalCoincideConSuDorada(t *testing.T) {
	app := appDorada(100, 24, VistaPrincipal)
	app.panel.Datos = datosDeDorada()
	hiloDeDorada(app)
	app.entrada.Escribir("y añade una prueba")
	app.entrada.Tokens = 1234
	dorada(t, "principal.golden", app.View())
}

// La caja de entrada crece en filas al desbordar el ancho, sin recortar lo
// escrito (SPEC-INTERFAZ §Zonas 2).
func TestLaCajaDeEntradaCoincideConSuDorada(t *testing.T) {
	app := appDorada(60, 24, VistaPrincipal)
	app.entrada.Escribir("una petición bastante larga que desborda el ancho de la caja y salta de renglón")
	dorada(t, "entrada.golden", app.entrada.View(60))
}

// El panel lateral pinta sus bloques con los nueve datos y el pie de tres filas
// (SPEC-INTERFAZ §Zonas 3).
func TestElPanelCoincideConSuDorada(t *testing.T) {
	app := appDorada(100, 24, VistaPrincipal)
	app.panel.Datos = datosDeDorada()
	dorada(t, "panel.golden", app.panel.Render(panel.Ancho, 24))
}

// El modal de modelos lista la oferta del motor, con el resaltado en el modelo en
// uso y la marca de los que no declaran herramientas (SPEC-INTERFAZ §Modales).
func TestElModalDeModelosCoincideConSuDorada(t *testing.T) {
	app := appDorada(100, 30, VistaPrincipal)
	app.modelos = modals.Modelos{}
	app.modelos.Abrir("qwen2.5")
	app.modelos.Fijar([]modals.Modelo{
		{Nombre: "qwen2.5", Motor: "Ollama local"},
		{Nombre: "llama3.2", Motor: "Ollama local", SinHerramientas: true},
		{Nombre: "mistral", Motor: "llama.cpp 8080"},
	}, nil)
	dorada(t, "modelos.golden", app.modelos.Render(100, 30))
}

// El modal de sesiones lista las del proyecto con su estado y marca la activa
// (SPEC-INTERFAZ §Cambiar de sesión).
func TestElModalDeSesionesCoincideConSuDorada(t *testing.T) {
	app := appDorada(100, 30, VistaPrincipal)
	app.sesiones = modals.Sesiones{}
	app.sesiones.Abrir("s2")
	app.sesiones.Fijar([]modals.Sesion{
		{ID: "s1", Nombre: "primera", Estado: EstadoInactiva},
		{ID: "s2", Nombre: "segunda", Estado: EstadoTrabajando},
		{ID: "s3", Nombre: "tercera", Estado: EstadoEsperandoPermiso},
	}, nil)
	dorada(t, "sesiones.golden", app.sesiones.Render(100, 30))
}

// El modal de atajos lista la acción y su tecla agrupadas por categorías, con la
// líder ya expandida y sin nada que aplicar (SPEC-INTERFAZ §Modales).
func TestLosAtajosCoincidenConSuDorada(t *testing.T) {
	app := appDorada(100, 40, VistaPrincipal)
	app.atajos = modals.Atajos{}
	app.atajos.Abrir(FilasDeAtajos(app.teclaRes.Kmap.Entradas(), app.teclaRes.Kmap.Lider()))
	dorada(t, "atajos.golden", app.atajos.Render(100, 40))
}

// La vista principal interpreta el Markdown de la respuesta —título, negrita,
// código inline y lista— y pinta cada bloque de código con su cabecera, su
// superficie y su botón de copiar (SPEC-VISTA-CODIGO).
func TestLaVistaDeCodigoCoincideConSuDorada(t *testing.T) {
	app := appDorada(100, 24, VistaPrincipal)
	app.panel.Datos = datosDeDorada()
	app.chat.AñadirSegmento(chat.Segmento{Hablante: chat.Usuario, Texto: "¿cómo compilo?"})
	app.chat.AñadirSegmento(chat.Segmento{
		Hablante: chat.Agente,
		Texto: "## Compilar\n\nUsa **go build** y después `make test`:\n\n" +
			"```bash\ngo build -o bin/app ./cmd/app\nmake test\n```\n\n- primer paso\n- segundo paso\n",
		Duracion: 1200 * time.Millisecond,
	})
	dorada(t, "codigo.golden", app.View())
}
