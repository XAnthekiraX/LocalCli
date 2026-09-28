> T-F035 — input: la línea de entrada (vista principal y bienvenida) salta de renglón al desbordar el ancho en vez de recortar el texto, y reajusta el reparto al redimensionar la terminal.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[specs/SPEC-INTERFAZ]] — §Zonas 2 (entrada de texto) y §Criterios de aceptación.
- [[frontend/01-domain/DOMAIN]] — ficha `input` y reglas de presentación.
- [[frontend/02-interfaces/INTERFACES]] — §5 (estados de espera).
- [[frontend/05-quality/TESTING]] — qué se prueba de la entrada.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-F035-01 | actualizar | `input`: la entrada pasa a un campo multilínea que envuelve por palabras, crece hasta un tope y descuenta el ancho del indicador del agente | completada | `internal/tui/input.go` | Tests `TestLaEntradaEnvuelveElTextoLargoEnVezDeRecortarlo`, `TestElSaltoDeLineaSeReajustaAlRedimensionar`, `TestElAltoDeLaEntradaNoPasaDelTope` |
| T-F035-02 | actualizar | `welcome`: la línea de entrada de la bienvenida envuelve el texto y ubica el cursor en la fila envuelta | completada | `internal/tui/welcome.go`, `internal/tui/styles.go` | Test `TestEnvolverConCursorReparteElTextoYUbicaElCursor` |
| T-F035-03 | actualizar | documentar el salto de línea en `SPEC-INTERFAZ`, `DOMAIN`, `INTERFACES` y `TESTING` | completada | `ai/docs/specs/SPEC-INTERFAZ.md`, `ai/docs/frontend/01-domain/DOMAIN.md`, `ai/docs/frontend/02-interfaces/INTERFACES.md`, `ai/docs/frontend/05-quality/TESTING.md` | Documentación coherente con el código |

Dependencias: T-F035-01 y T-F035-02 antes de T-F035-03.

## Notas

- La línea sigue siendo un solo párrafo: `Enter` envía y no inserta un salto. El `textarea` de bubbles se usa como motor de edición y envoltura.
- El alto se calcula con `filasEnvueltas`, que replica el soft-wrap del `textarea` (que no lo expone): si contara de menos, el campo recortaría la primera línea.
- Un tope de filas (`altoMáximoEntrada`) evita que la entrada se coma el chat; por encima, el texto se desplaza dentro de la ventana.
- Antes, el ancho del campo era el total sin descontar el indicador: la línea desbordaba la terminal por las columnas del indicador. Ahora se reparte ancho indicador + campo.
