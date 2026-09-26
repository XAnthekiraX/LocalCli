> T-F011 — Pruebas y validaciones: unitarias de render, de componente con arnés Bubble Tea y verificación de criterios de aceptación.
> Acción de esta descomposición: `crear`.

## Referencias

- [[frontend/05-quality/TESTING]] — niveles, reglas para nuevos tests (dorados, sin `sleep`, una prueba una regla) y checklist final.
- [[specs/SPEC-INTERFAZ]] — 17 criterios de aceptación de la disposición y bienvenida.
- [[specs/SPEC-INTERFAZ-ATAJOS]] — 6 criterios de aceptación de atajos y aprobaciones.
- [[frontend/FRONTEND]] — límites de la capa que las pruebas no deben violar.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-F011-01 | crear | Suite unitaria de funciones puras: recorte, formatos de razonamiento/intercambio, contador, recorte por alto y mapa de teclas | completada | `internal/tui/styles_test.go`, `notify_test.go`, `chat_test.go`, `keys_test.go` | `go test ./internal/tui/ -short` en verde |
| T-F011-02 | crear | Arnés de pruebas de componente (programa Bubble Tea in-process con msgs sintéticos, sin base ni red) reutilizable | completada | `internal/tui/testing_test.go` | Test propio del arnés: inyecta un msg y lee la view |
| T-F011-03 | crear | Tests de componente por recorrido clave: bienvenida→envío→chat, cambio de sesión en vivo, razonamiento ocultable, panel plegable | completada | `internal/tui/app_test.go` | `go test ./internal/tui/ -run Recorrido` en verde |
| T-F011-04 | crear | Salidas doradas nuevas (vistas de panel, ayuda, selector, aprobaciones) generadas y revisadas a la vista | completada | `internal/tui/testdata/*.golden`, `golden_test.go` | Test golden en verde; regeneración consciente con `-update` |
| T-F011-05 | verificar | Pasada de verificación de los 23 criterios de aceptación de SPEC-INTERFAZ y SPEC-INTERFAZ-ATAJOS contra las pruebas existentes | completada | `ai/tasks/frontend/011-task-tests.md` | Los 23 criterios mapeados a su prueba; ninguno bloqueado |
| T-F011-06 | crear | Gates finales: `gofmt -l`, `go vet ./internal/tui/` y `go test ./...` | completada | — | Los tres comandos salen sin errores |

## Criterios de aceptación (T-F011-05)

Los 23 criterios de [[specs/SPEC-INTERFAZ]] (17) y [[specs/SPEC-INTERFAZ-ATAJOS]] (6), cada uno con la prueba que lo comprueba. Ninguno bloqueado.

### SPEC-INTERFAZ

| # | Criterio | Prueba |
|---|----------|--------|
| 1 | Con el panel cerrado, el chat ocupa todo el ancho | `TestConElPanelCerradoElChatOcupaTodoElAncho` |
| 2 | El panel se abre y se cierra sin interrumpir el trabajo | `TestElPanelSeAbreYCierraSinTocarLaEntrada` |
| 3 | El panel muestra los nueve datos definidos | `TestElPanelMuestraLosNueveDatos` |
| 4 | El panel muestra los datos de la sesión activa | `TestElEstadoDeOtraSesiónNoCambiaElPanel` |
| 5 | El razonamiento se muestra en vivo, arriba de la respuesta | `TestLaRespuestaNoSeMezclaConElRazonamiento` |
| 6 | El razonamiento se distingue y se puede ocultar | `TestRenderRazonamientoDistinguibleYOcultable`, `TestOcultarElRazonamientoNoBorraNiDetieneLaAcumulación` |
| 7 | Un atajo abre el selector con nombre y estado | `TestElSelectorListaYCambiaDeSesión` |
| 8 | Elegir una sesión cambia el chat sin detener lo demás | `TestAlCambiarDeSesiónLlegaElHistorialDeEsaSesión`, `TestElegirSesiónNoCancelaLoQueCorre` |
| 9 | Con el panel cerrado se ven las aprobaciones pendientes | `TestElAvisoSeVeConElPanelCerradoYNoConElAbierto` |
| 10 | Nombre y versión de LocalCli aparecen en el panel | `TestElPanelMuestraLosNueveDatos` |
| 11 | Cambiar de sesión no detiene ninguna ejecución | `TestElegirSesiónNoCancelaLoQueCorre` |
| 12 | Al ejecutar se ve la bienvenida completa | `TestLaBienvenidaSoloTieneLogotipoNombreYEntrada` |
| 13 | La primera petición es el primer mensaje del chat | `TestEnviarDesdeLaBienvenidaResuelveLaSesiónYEnvíaUnaVez` |
| 14 | La transición no repite ni pide confirmación | `TestEnviarEnLaBienvenidaAbreLaInterfazUnaVez` |
| 15 | Desde la bienvenida solo escribir, enviar y salir | `TestEnLaBienvenidaLosAtajosDeLaPrincipalNoExisten` |
| 16 | La bienvenida se muestra aunque Ollama no esté | `TestElModeloArrancaYSePintaSinPánico` |
| 17 | El logotipo coincide byte a byte con el dorado | `TestElLogotipoCoincideConLaSalidaDorada`, `TestElLogotipoSePintaTalCualSinReescalarNiCentrar` |

### SPEC-INTERFAZ-ATAJOS

| # | Criterio | Prueba |
|---|----------|--------|
| 18 | Se listan los atajos y la acción de cada uno | `TestLaAyudaListaLosAtajosYSeCierraSinEfectos` |
| 19 | Se puede cambiar un atajo y el cambio se guarda | `TestSaveLoadRoundTrip` |
| 20 | El panel muestra las pendientes de todas las sesiones | `TestDosSesionesEsperanComoDosLíneasIndependientes` |
| 21 | Cada aprobación se resuelve de forma independiente | `TestAprobarYDeclinarResuelvenSoloLaLíneaSeleccionada` |
| 22 | El panel no interrumpe el trabajo de ninguna sesión | `TestAbrirYCerrarElPanelDeAprobacionesNoDetieneNada` |
| 23 | Una aprobación que ya no aplica se marca obsoleta | `TestLaSesiónTerminadaMarcaSuLíneaComoObsoleta` |

Dependencias: T-F011-02 depende de T-F011-01; T-F011-03 depende de T-F011-02; T-F011-04 depende de T-F011-03; T-F011-05 depende de T-F011-04; T-F011-06 depende de todas. Requiere T-F010 completada.
