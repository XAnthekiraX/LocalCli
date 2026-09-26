// matriz_reglas_test.go — T-B015-01: una prueba por regla de negocio de
// BUSINESS_RULES.md.
//
// Fuente de verdad: ai/docs/backend/01-domain/BUSINESS_RULES.md §1 y TESTING.md
// §4 ("Toda regla de negocio de BUSINESS_RULES tiene al menos una prueba").
//
// Esta prueba es el índice verificable de esa promesa: recorre el listado de
// reglas → pruebas y falla si algún test de la matriz ya no existe. No repite
// lo que cada módulo prueba; ata el documento a la suite, de modo que añadir una
// regla sin su prueba deja la suite en rojo.
package tests

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// reglasPrueba es la matriz regla → prueba(s) que la sostienen. Cada entrada
// nombra la regla como está en BUSINESS_RULES.md y los Test* que la cubren
// (basta que exista uno). Los nombres van sin el prefijo Test para tolerar
// variantes con acentos.
type regla struct {
	Seccion string
	Regla   string
	Pruebas []string // nombres de tests, con o sin prefijo Test
}

func matrizDeReglas() []regla {
	return []regla{
		// --- §Sesiones ---
		{"Sesiones", "Una sesión pertenece a un solo proyecto; el chat de una carpeta nunca aparece en otra",
			[]string{"DosCarpetasNoCompartenSesiones"}},
		{"Sesiones", "El contenido de una sesión no se ve desde otra del mismo proyecto",
			[]string{"UnEventoDeOtraSesionNoCambiaElPanel"}},
		{"Sesiones", "Una sesión en segundo plano sigue trabajando aunque el usuario cambie de sesión o salga de la vista",
			[]string{"ElTrabajoSigueAunqueLaVistaEsteEnOtraSesion"}},
		{"Sesiones", "Cambiar de sesión no detiene nada",
			[]string{"ElTrabajoSigueAunqueLaVistaEsteEnOtraSesion", "ElSelectorListaYCambiaDeSesion"}},
		{"Sesiones", "Cada sesión expone siempre su estado",
			[]string{"NotificacionPorEstado", "NotificaAlTerminar"}},

		// --- §Notificaciones ---
		{"Notificaciones", "Al pasar a esperando permiso se manda una notificación",
			[]string{"AvisaDeLaEsperaDePermisoUnaVez"}},
		{"Notificaciones", "Al pasar a terminada se manda una notificación",
			[]string{"NotificaAlTerminar"}},
		{"Notificaciones", "La notificación se manda aunque no estés viendo esa sesión",
			[]string{"LaNotificacionLlegaAunqueNoSeVeaLaSesion"}},

		// --- §Motor de etapas ---
		{"Motor de etapas", "Las etapas se ejecutan en orden y cada una arranca cuando la anterior terminó",
			[]string{"ResolverSecuenciaDocumentada"}},
		{"Motor de etapas", "Si una etapa falla, el flujo se detiene",
			[]string{"EtapaFallidaEmiteEventoYDetiene"}},
		{"Motor de etapas", "Un flujo pausado por un permiso se retoma desde la misma etapa",
			[]string{"PausaPorPermisoDejaLaSesionEsperando"}},
		{"Motor de etapas", "El usuario puede cancelar y un flujo cancelado no deja etapas corriendo",
			[]string{"CancelacionSePropaga", "CancelarCortaElTrabajo"}},

		// --- §Detección de trabajo ordenado ---
		{"Detección", "Una petición con lista ordenada crea un TODO",
			[]string{"DetectaTrabajoOrdenado"}},
		{"Detección", "Una petición sin orden ni lista no crea TODO",
			[]string{"NoDetectaConversacionNormal"}},

		// --- §Cola ---
		{"Cola", "El orden del TODO es el orden de ejecución, y se respeta",
			[]string{"ReconstruirOrdenaPorDependencias", "OrdenTopologicoEstableSobreFixture"}},
		{"Cola", "Se ejecuta un elemento por iteración, nunca varios a la vez",
			[]string{"ConsumirColaUnElementoPorIteracion"}},
		{"Cola", "Un elemento bloqueado no detiene la cola si nada depende de él",
			[]string{"SiguienteDevuelveElPrimeroElegible", "BloqueadosPropagaElBloqueo"}},
		{"Cola", "Un elemento nuevo hace que la cola se re-derive",
			[]string{"RederrivarVeElElementoNuevo"}},
		{"Cola", "La cola refleja siempre el estado real del TODO",
			[]string{"MarcarActualizaLaFilaDelArchivo"}},
		{"Cola", "Al vaciarse, la cola se detiene sola y avisa",
			[]string{"ColaVacia", "SiguienteNilCuandoNadaPuedeArrancar"}},
		{"Cola", "Retomar no repite elementos ya completados",
			[]string{"ElegiblesRespetaCompletadas", "RangoSoloCuentaTareasGrandes"}},

		// --- §Nodo de contexto ---
		{"Contexto", "La decisión de qué es relevante la toma siempre el modelo",
			[]string{"ModeloOllamaSeleccionaPorHTTP", "InterpretarSeleccion"}},
		{"Contexto", "El contexto entregado nunca supera el límite",
			[]string{"RecortarDentroDelLimite", "PrepararRecortaYAuditaDescartes"}},
		{"Contexto", "Todo lo descartado queda registrado con su motivo",
			[]string{"PrepararRecortaYAuditaDescartes", "AuditarUnaFilaPorDocumento"}},
		{"Contexto", "Si el modelo pide un documento que no existe, la etapa se detiene",
			[]string{"PrepararDocumentoInexistenteDetiene"}},
		{"Contexto", "Si el modelo no puede decidir, se usa el objetivo declarado por la etapa",
			[]string{"PrepararFallbackSinModelo"}},

		// --- §Agentes ---
		{"Agentes", "plan solo tiene herramientas que leen",
			[]string{"PlanNoTieneHerramientasDeEscritura", "PlanNoPuedeEscribirNiConAprobadorAbierto"}},
		{"Agentes", "build tiene el catálogo completo",
			[]string{"BuildTieneElCatalogoCompleto", "UnAgenteBaseTieneSuCatalogo"}},

		// --- §Herramientas y terminal (reparto del permiso) ---
		{"Herramientas", "Una herramienta inventada se rechaza",
			[]string{"HerramientaInventadaNoExiste", "ValidarRechazaHerramientaDesconocida"}},
		{"Herramientas", "Los comandos de consulta se ejecutan sin preguntar; cualquier otro pide aprobación",
			[]string{"ComandoDeListaBlancaSeEjecutaSolo", "ComandoFueraDeListaSinAprobador"}},
		{"Herramientas", "La terminal no puede escribir en el proyecto",
			[]string{"LandlockImpideEscribirEnElProyecto", "ElProyectoSigueIntactoTrasLosIntentos"}},

		// --- Documentación (reglas del grafo, BUSINESS_RULES de docs) ---
		{"Documentación", "Las dependencias se declaran en el frontmatter, no en los enlaces del cuerpo",
			[]string{"GrafoTomaLasAristasDelFrontmatter", "UnEjemploDeWikiLinkNoEsUnaDependencia"}},
		{"Documentación", "Una etiqueta no cuenta como dependencia",
			[]string{"LasEtiquetasNoSonAristas"}},
	}
}

func nombresDeTests(t *testing.T) map[string]bool {
	t.Helper()
	nombres := map[string]bool{}
	err := filepath.WalkDir("..", func(ruta string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			base := d.Name()
			if base == ".git" || base == ".localcli" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(ruta, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(token.NewFileSet(), ruta, nil, 0)
		if err != nil {
			return err
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || !strings.HasPrefix(fn.Name.Name, "Test") {
				continue
			}
			// Claves con acentos y sin ellos: los nombres de prueba llevan
			// tildes ("TestSesión") y la matriz las nombra sin tilde para
			// mantenerla legible.
			nombres[fn.Name.Name] = true
			nombres[sinTildes(strings.TrimPrefix(fn.Name.Name, "Test"))] = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("recorrido de tests: %v", err)
	}
	return nombres
}

// sinTildes normaliza los caracteres acentuados más comunes del castellano.
func sinTildes(s string) string {
	reemplazos := strings.NewReplacer(
		"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n",
		"Á", "A", "É", "E", "Í", "I", "Ó", "O", "Ú", "U", "Ü", "U", "Ñ", "N",
	)
	return reemplazos.Replace(s)
}

// TestTodaReglaDeNegocioTieneSuPrueba — TESTING.md §4 hecho comprobable: la
// matriz completa tiene que resolver contra tests que existen hoy.
func TestTodaReglaDeNegocioTieneSuPrueba(t *testing.T) {
	existentes := nombresDeTests(t)
	if len(existentes) < 50 {
		t.Fatalf("la suite debería tener cientos de pruebas, encontré %d", len(existentes))
	}
	for _, r := range matrizDeReglas() {
		cubierta := false
		for _, p := range r.Pruebas {
			if existentes[p] {
				cubierta = true
				break
			}
		}
		if !cubierta {
			t.Errorf("la regla %q (%s) no tiene ninguna de sus pruebas: %v (¿se renombró?)",
				r.Regla, r.Seccion, r.Pruebas)
		}
	}
}
