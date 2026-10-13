package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"localcli/internal/tui/render"
)

// --- T-F007-01: el modelo del panel ------------------------------------------

// El modelo guarda cada pendiente con su sesión y su acción, y cuál está
// resaltado: la primera línea, mientras no se mueva el resaltado.
func TestElModeloGuardaCadaPendienteConSuSesionYAccion(t *testing.T) {
	ap := NuevaAprobaciones()
	ap.Anotar(Aprobacion{ID: "a1", Sesion: "s1", Accion: "crear archivo"})
	ap.Anotar(Aprobacion{ID: "a2", Sesion: "s2", Accion: "borrar carpeta"})

	if len(ap.Items) != 2 {
		t.Fatalf("dos pendientes, dos líneas: %+v", ap.Items)
	}
	if ap.Items[0].Sesion != "s1" || ap.Items[0].Accion != "crear archivo" {
		t.Fatalf("la primera línea no guarda su sesión y su acción: %+v", ap.Items[0])
	}
	if !ap.Abierto {
		t.Fatal("con pendientes el panel se muestra")
	}
	if ap.Indice != 0 {
		t.Fatalf("el resaltado arranca en la primera línea: %d", ap.Indice)
	}
}

// --- T-F007-02: el formato de una línea --------------------------------------

// anchoFolgado es un ancho de sobra: las pruebas de formato y clic comprueban el
// texto completo, sin el ajuste a una terminal estrecha.
const anchoFolgado = 200

// La línea es `sesión | acción propuesta | aprobar | declinar`, sin reservar
// hueco para la tercera opción que aún está por definir.
func TestElFormatoDeLineaEsElDocumentado(t *testing.T) {
	ap := Aprobaciones{Abierto: true, Items: []Aprobacion{{ID: "a1", Sesion: "api", Accion: "crear archivo"}}}

	lineas := ap.Lineas(anchoFolgado)
	if len(lineas) != 1 {
		t.Fatalf("una línea: %v", lineas)
	}
	quiere := "› api | crear archivo | aprobar | declinar"
	if lineas[0] != quiere {
		t.Errorf("línea = %q, quiero %q", lineas[0], quiere)
	}
}

// La línea va en corto: la sesión por su identificador abreviado y la carpeta del
// usuario abreviada a `~`, para que quepa de un vistazo.
func TestLaLineaAbreviaLaSesionYLaCasa(t *testing.T) {
	largo := "24afd397-ccdd-45df-a8df-90bf3ad0974a"
	fila := filaAprobacion(Aprobacion{ID: "a1", Sesion: largo, Accion: "crear archivo"}, "", "", anchoFolgado)
	if fila != "24afd397 | crear archivo | aprobar | declinar" {
		t.Fatalf("la sesión va abreviada a ocho: %q", fila)
	}

	fila = filaAprobacion(Aprobacion{ID: "a1", Sesion: "s1", Accion: "leer /home/x/foto.png"}, "/home/x", "", anchoFolgado)
	if !strings.Contains(fila, "leer ~/foto.png") {
		t.Errorf("la carpeta del usuario se abrevia a ~: %q", fila)
	}
}

// --- T-F007-03: el render del panel ------------------------------------------

// Con pendientes se pinta la lista; sin pendientes el panel desaparece sin dejar
// hueco.
func TestElPanelPintaLaListaYDesapareceSinPendientes(t *testing.T) {
	var ap Aprobaciones
	if v := ap.Render(anchoFolgado); v != "" {
		t.Fatalf("sin pendientes el panel no se pinta: %q", v)
	}

	ap.Anotar(Aprobacion{ID: "a1", Sesion: "s1", Accion: "crear archivo"})
	v := render.SinANSI(ap.Render(anchoFolgado))
	if !strings.Contains(v, "s1 | crear archivo | aprobar | declinar") {
		t.Fatalf("con pendientes se pinta la línea:\n%s", v)
	}

	ap.Retirar("a1")
	if v := ap.Render(anchoFolgado); v != "" {
		t.Fatalf("al resolverse la última, el panel desaparece: %q", v)
	}
}

// --- T-F007-04: alta y baja por evento ---------------------------------------

// `peticion_aprobacion` añade la línea y `aprobacion_resuelta` la retira; el
// contador del panel de datos sigue contando las de todas las sesiones.
func TestLaPeticionAnadeYLaResolucionRetira(t *testing.T) {
	app := appConSesion()

	app.aplicarEvento(eventoDe("s2", EventoPeticionAprobacion, map[string]string{
		"aprobacion": "a1", "descripcion": "crear archivo",
	}))
	app.aplicarEvento(eventoDe("s3", EventoPeticionAprobacion, map[string]string{
		"aprobacion": "a2", "descripcion": "borrar carpeta",
	}))
	if len(app.aprobaciones.Items) != 2 || app.panel.Aprobaciones != 2 {
		t.Fatalf("dos pendientes de dos sesiones: %+v (%d)", app.aprobaciones.Items, app.panel.Aprobaciones)
	}

	app.aplicarEvento(eventoDe("s2", EventoAprobacionResuelta, map[string]string{"aprobacion": "a1"}))
	if len(app.aprobaciones.Items) != 1 || app.aprobaciones.Items[0].ID != "a2" {
		t.Fatalf("solo sale la línea resuelta: %+v", app.aprobaciones.Items)
	}
	if app.panel.Aprobaciones != 1 {
		t.Fatalf("el contador queda en una: %d", app.panel.Aprobaciones)
	}
}

// La sesión que termina mientras esperaba deja su línea obsoleta: se sigue
// viendo, pero ya no cuenta como pendiente.
func TestLaSesionTerminadaMarcaSuLineaObsoleta(t *testing.T) {
	app := appConSesion()

	app.aplicarEvento(eventoDe("s2", EventoPeticionAprobacion, map[string]string{"aprobacion": "a1"}))
	app.aplicarEvento(eventoDe("s2", EventoEstadoSesion, map[string]string{"estado": EstadoTerminada}))

	if len(app.aprobaciones.Items) != 1 || !app.aprobaciones.Items[0].Obsoleta {
		t.Fatalf("la sesión terminó mientras esperaba: la línea se marca obsoleta: %+v", app.aprobaciones.Items)
	}
	if app.panel.Aprobaciones != 0 {
		t.Fatalf("una obsoleta ya no espera decisión: %d", app.panel.Aprobaciones)
	}
}

// --- T-F007-08: el clic del ratón --------------------------------------------

// columnaDeOpción devuelve la columna donde empieza el botón `opcion` en la fila
// `indice`. La opción cierra la línea, así que se busca su última aparición: la
// acción puede contener la misma palabra y no es el botón.
func columnaDeOpción(t *testing.T, ap *Aprobaciones, indice int, opcion string) int {
	t.Helper()
	linea := ap.Lineas(anchoFolgado)[indice]
	i := strings.LastIndex(linea, opcion)
	if i < 0 {
		t.Fatalf("no encontré %q en %q", opcion, linea)
	}
	return render.Ancho(linea[:i])
}

// El clic sobre «aprobar» o «declinar» de una línea resuelve esa aprobación, y
// no hace falta que el panel tenga el foco.
func TestElClicSobreLasOpcionesResuelveEsaLinea(t *testing.T) {
	ap := Aprobaciones{Abierto: true, Items: []Aprobacion{
		{ID: "a1", Sesion: "s1", Accion: "crear archivo"},
		{ID: "a2", Sesion: "s2", Accion: "borrar carpeta"},
	}}

	col := columnaDeOpción(t, &ap, 1, "aprobar")
	it, decision, ok := ap.Clic(2, col, anchoFolgado)
	if !ok || it.ID != "a2" || decision != DecisionAprobar {
		t.Fatalf("el clic en «aprobar» de la segunda línea: %+v %v %v", it, decision, ok)
	}

	col = columnaDeOpción(t, &ap, 0, "declinar")
	it, decision, ok = ap.Clic(1, col, anchoFolgado)
	if !ok || it.ID != "a1" || decision != DecisionDeclinar {
		t.Fatalf("el clic en «declinar» de la primera línea: %+v %v %v", it, decision, ok)
	}
}

// Una acción que repite la palabra de una opción —«aprobar la etapa…», como
// titula el flujo al pausar— no desplaza la zona del clic: las opciones se
// localizan por el sufijo que cierra la línea.
func TestElClicNoSeConfundeConLaPalabraDeLaAccion(t *testing.T) {
	ap := Aprobaciones{Abierto: true, Items: []Aprobacion{
		{ID: "a1", Sesion: "s1", Accion: "aprobar la etapa resolver"},
	}}

	col := columnaDeOpción(t, &ap, 0, "aprobar")
	it, decision, ok := ap.Clic(1, col, anchoFolgado)
	if !ok || it.ID != "a1" || decision != DecisionAprobar {
		t.Fatalf("el clic en el botón «aprobar»: %+v %v %v", it, decision, ok)
	}

	col = columnaDeOpción(t, &ap, 0, "declinar")
	it, decision, ok = ap.Clic(1, col, anchoFolgado)
	if !ok || it.ID != "a1" || decision != DecisionDeclinar {
		t.Fatalf("el clic en el botón «declinar»: %+v %v %v", it, decision, ok)
	}
}

// Un clic fuera de las opciones —sobre la sesión, la acción o el título— no
// decide nada.
func TestElClicFueraDeLasOpcionesNoDecide(t *testing.T) {
	ap := Aprobaciones{Abierto: true, Items: []Aprobacion{{ID: "a1", Sesion: "s1", Accion: "crear archivo"}}}

	if _, _, ok := ap.Clic(0, 0, anchoFolgado); ok {
		t.Error("el título no decide")
	}
	if _, _, ok := ap.Clic(1, 0, anchoFolgado); ok {
		t.Error("la sesión y la acción no deciden")
	}
	if _, _, ok := ap.Clic(9, 0, anchoFolgado); ok {
		t.Error("una fila fuera del panel no decide")
	}
}

// Una línea obsoleta no se decide: su sesión ya no puede aplicar la propuesta.
func TestLaLineaObsoletaNoSeDecide(t *testing.T) {
	ap := Aprobaciones{Abierto: true, Items: []Aprobacion{
		{ID: "a1", Sesion: "s1", Accion: "crear archivo", Obsoleta: true},
	}}
	if fila := filaAprobacion(ap.Items[0], "", "", anchoFolgado); !strings.HasSuffix(fila, " | obsoleta") {
		t.Fatalf("la línea obsoleta se marca como tal: %q", fila)
	}
	if _, _, ok := ap.Clic(1, 0, anchoFolgado); ok {
		t.Error("sobre una obsoleta no se manda ninguna decisión")
	}
}

// Con una aprobación pendiente, `a` aprueba y `d` declina la línea resaltada: la
// decisión viaja al puerto y la línea se retira del panel.
func TestAprobarYDeclinarDesdeElPanel(t *testing.T) {
	p := &puertoFalso{}
	app := appDePrueba(p)
	app.vista = VistaPrincipal
	app.aplicarEvento(eventoDe("s2", EventoPeticionAprobacion, map[string]string{"aprobacion": "a1", "descripcion": "crear archivo"}))
	app.aplicarEvento(eventoDe("s1", EventoPeticionAprobacion, map[string]string{"aprobacion": "a2", "descripcion": "borrar carpeta"}))

	if !app.aprobaciones.Abierto {
		t.Fatal("con algo que decidir, el panel debería estar abierto")
	}
	if !strings.Contains(render.SinANSI(app.View()), "APROBACIONES") {
		t.Fatal("el panel de aprobaciones debería pintarse")
	}

	// Sin foco, la caja sigue escribiendo: `a` va al texto, no al panel.
	pulsar(app, teclaRuna("a"))
	if len(p.respondidas) != 0 {
		t.Fatalf("sin foco, `a` no debería decidir: %v", p.respondidas)
	}
	if !strings.Contains(app.entrada.Texto, "a") {
		t.Fatalf("sin foco, la caja debería escribir: %q", app.entrada.Texto)
	}

	// `Ctrl+A` da el foco: `a` aprueba la primera línea.
	pulsar(app, tea.KeyMsg{Type: tea.KeyCtrlA})
	pulsar(app, teclaRuna("a"))
	if len(p.respondidas) != 1 || p.respondidas[0] != "a1" {
		t.Fatalf("`a` debería aprobar la primera línea: %v", p.respondidas)
	}
	if app.panel.Aprobaciones != 1 {
		t.Fatalf("al aprobar una queda %d", app.panel.Aprobaciones)
	}

	pulsar(app, teclaRuna("d"))
	if len(p.respondidas) != 2 || p.respondidas[1] != "a2" {
		t.Fatalf("`d` debería declinar la que queda: %v", p.respondidas)
	}
	if app.aprobaciones.Abierto || app.focoAprobaciones {
		t.Fatal("sin nada que decidir, el panel se cierra y pierde el foco")
	}
}

// En una terminal estrecha la acción cede ante las opciones: la decisión nunca se
// sale del ancho de la columna y sigue siendo pulsable.
func TestLaLineaEstrechaConservaLasOpciones(t *testing.T) {
	ap := Aprobaciones{Abierto: true, Items: []Aprobacion{
		{ID: "a1", Sesion: "24afd397", Accion: "escribir internal/tui/app.go con un cambio bastante largo"},
	}}
	linea := ap.Lineas(40)[0]
	if w := render.Ancho(linea); w > 40 {
		t.Fatalf("la línea mide %d columnas: %q", w, linea)
	}
	if !strings.HasSuffix(linea, " | aprobar | declinar") {
		t.Fatalf("las opciones se conservan: %q", linea)
	}
	col := strings.LastIndex(linea, "aprobar")
	it, decision, ok := ap.Clic(1, render.Ancho(linea[:col]), 40)
	if !ok || it.ID != "a1" || decision != DecisionAprobar {
		t.Fatalf("el clic en «aprobar» a 40 columnas: %+v %v %v", it, decision, ok)
	}
}
