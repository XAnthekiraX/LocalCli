> T-F003 — welcome: pantalla de bienvenida con logotipo ASCII dorado, nombre con versión y primera petición.
> Acción de esta descomposición: `crear`.

## Referencias

- [[frontend/01-domain/DOMAIN]] — §3 reglas de la bienvenida (primera vista, única salida Ctrl+C, sin dependencias externas).
- [[specs/SPEC-INTERFAZ]] — arte canónico del logotipo (6×53), nombre con versión y línea «En qué te ayudo hoy».
- [[frontend/02-interfaces/INTERFACES]] — §2: enviar mensaje y crear/retomar sesión desde la bienvenida.
- [[frontend/05-quality/TESTING]] — comparación byte a byte contra `internal/tui/testdata/logo.txt`.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-F003-01 | crear | Modelo `welcome` con foco en la entrada y estado mínimo (texto escrito) | completada | `internal/tui/welcome.go` | `go build ./internal/tui/` sin errores |
| T-F003-02 | crear | Render del logotipo tal cual (sin reescalar ni centrar) leído del dorado `testdata/logo.txt` embebido | completada | `internal/tui/welcome.go` | Test golden: salida idéntica byte a byte a `logo.txt` |
| T-F003-03 | crear | Composición de la vista: arte + «LocalCli · v0.1» + línea de entrada, fuera del arte dorado | completada | `internal/tui/welcome.go` | Test de componente: vista contiene nombre con versión y prompt |
| T-F003-04 | crear | Update de teclado de la bienvenida: escribir/enviar y solo `Ctrl+C` como salida; sin panel/selector/aprobaciones | completada | `internal/tui/welcome.go` | Test de componente: otras teclas no producen acciones de zona principal |
| T-F003-05 | crear | Emisión de la petición compuesta (cmd de crear/retomar sesión + enviar mensaje) sin repetir ni confirmar | completada | `internal/tui/welcome.go` | Test: al enviar se generan exactamente los dos comandos, una sola vez |

Dependencias: T-F003-02..05 dependen de T-F003-01; T-F003-05 depende de T-F003-03.
