> T-F033 — indicador en vivo, herramientas compactas y contador de tokens: en lugar de volcar el razonamiento en crudo, la TUI muestra un indicador animado (`[⠋ Pensando]`, `[⠋ Usando herramienta: X]`, `[⠋ Generando]`) con el tiempo transcurrido; el texto del razonamiento se revela con `Ctrl+R`. Las líneas de herramienta pasan a formato compacto (`[→ herramienta: X]`, `[✓/✗ …]`). El consumo del turno se muestra bajo la entrada (`tokens: 54k`).
> Acción de esta descomposición: `actualizar` (sobre T-F002 styles, T-F005 chat y T-F010 app; no se modifican sus entradas porque el cambio se registra aquí).

## Referencias

- [[specs/SPEC-INTERFAZ]] — §Zonas 1 y 2 (indicador en vivo y línea de tokens), §Razonamiento del modelo, §Reglas de negocio y §Criterios de aceptación.
- [[specs/SPEC-PANEL-CONTEXTO]] — significados del conteo de tokens y de la visualización del razonamiento.
- [[frontend/01-domain/DOMAIN]] — §1 (`chat`), §2 (reglas de presentación).
- [[frontend/05-quality/TESTING]] — §2: qué se prueba del indicador y del conteo; §3: funciones puras y doradas.
- [[backend/04-infrastructure/EVENTS]] — `token`, `herramienta_invocada`/`resultado` y `tokens_turno`.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-F033-01 | actualizar | `styles`: frames del indicador (`glifoActividad`), `renderActividad`, `formatearTokens` y la nueva semántica de `renderReasoning` (solo con el texto revelado) | completada | `internal/tui/styles.go` | Tests `TestGlifoActividadRecorreElCiclo`, `TestRenderActividadComponeElIndicador` y `TestFormatearTokensAcortaElConteoEnLaUnidadLegible` en verde |
| T-F033-02 | actualizar | `reasoning`: el texto arranca no revelado; el indicador no depende de la bandera | completada | `internal/tui/reasoning.go` | Tests de razonamiento de `tui_test.go` y `chat_test.go` en verde |
| T-F033-03 | actualizar | `chat`: líneas de herramienta compactas (`[→ …]`, `[✓/✗ …]`) y bandera `MostrarRazonamiento` para el historial | completada | `internal/tui/chat.go` | Test `TestLosEventosDeHerramientaSePintan` en verde |
| T-F033-04 | actualizar | `app`: frame de animación y latido rápido, `lineaDeActividad` y `lineaDeTokens`, sincronía del toggle con el historial y reinicio del consumo por turno | completada | `internal/tui/app.go`, `internal/tui/welcome.go` | Tests `TestElIndicadorDicePensandoYGenerando`, `TestLaLineaDeTokensMuestraElConsumoDelTurno` y `TestUnTurnoNuevoReiniciaElConsumo` en verde |
| T-F033-05 | actualizar | `wire`: consumo vivo por fragmento, herramienta en curso para el indicador y corrección con el valor exacto de `tokens_turno` | completada | `internal/tui/wire.go` | Test `TestElConsumoVivoSeCorrigeConElValorExactoDelTurno` en verde |

Dependencias: T-F033-02 y T-F033-03 dependen de T-F033-01; T-F033-04 depende de T-F033-01 y T-F033-02; T-F033-05 depende de T-F033-04.

## Notas

- El indicador reutiliza la cadena de latido existente (`tickMsg`/`latido`) con un intervalo de ~120 ms, sin añadir `bubbles/spinner`: un solo reloj de animación que se detiene solo al cerrar el turno.
- El conteo vivo de tokens es una aproximación (cuenta fragmentos de `token`, de razonamiento y de respuesta); `tokens_turno` lo corrige con el valor exacto al cerrarse el turno (SPEC-PANEL-CONTEXTO).
- El texto del razonamiento no se pierde al no revelarlo: se sigue acumulando y `Ctrl+R` lo recupera entero, en vivo y en el historial (misma bandera).
- Los frames del indicador viven en `framesActividad` (styles.go), un único sitio editable.
