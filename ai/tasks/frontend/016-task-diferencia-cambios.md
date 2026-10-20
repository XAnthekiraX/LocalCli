# 016-task-diferencia-cambios.md

> T-F016 — Bloque de diferencia: pintar el antes y el después del cambio bajo la línea de herramienta, con su tipo, su color y su recuperación.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[frontend/02-interfaces/INTERFACES]] §1.1 «Los bloques de la línea de herramienta», apartado `diferencia`.
- [[frontend/01-domain/DOMAIN]] — el componente `chat` y sus líneas.
- [[backend/04-infrastructure/EVENTS]] §3 `herramienta_resultado` — `bloque` y `bloque_tipo`.
- [[backend/02-interfaces/TOOLS]] §La diferencia que se ve.
- [[specs/SPEC-TOOLS]] §La diferencia a la vista y [[specs/SPEC-ARCHIVOS]].
- [[specs/SPEC-INTERFAZ]] — el chat y las líneas del sistema.
- [[database/02-rules/DATA_FLOW]] §Creación — la línea se recupera con su bloque.

## Contexto

El bloque visual bajo la línea de herramienta ya existe (T-F015): `Linea.Bloque` se pinta
debajo, separado por una fila en blanco, y la línea conserva su texto. Esta tarea añade
el segundo tipo de bloque, `diferencia`, que llega en `herramienta_resultado` con
`bloque_tipo` y se pinta con el color de cada marca:

```
  ✓ EDITAR [internal/session/historial.go] · 1 reemplazo · 0.02 s

  @@ -248,7 +248,7 @@
     if corte >= len(conv) {
  -      return 0
  +      return corte
     }
```

La TUI **no calcula el diff**: lo recibe ya renderizado y recortado del backend, y solo
decide cómo pintarlo. `diagrama` se sigue pintando como hasta ahora.

## Tareas pequeñas

| ID        | Acción     | Tarea                                                                                                                                                                                    | Estado    | Archivos                                                                   | Verificación                                                                                   |
| --------- | ---------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------- | -------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| T-F016-01 | actualizar | Llevar `bloque_tipo` desde el evento y desde el historial hasta la línea: `Linea.BloqueTipo`, `LineaHistorial.BloqueTipo`, `CerrarHerramienta` con el tipo y el montaje del historial en `app` | completada | `internal/tui/wire.go`, `internal/tui/eventos.go`, `internal/tui/app.go`, `internal/tui/components/chat/{chat,lineas}.go` | Un resultado con `diferencia` llega a la línea con su tipo; `diagrama` conserva el suyo    |
| T-F016-02 | actualizar | Pintar el bloque `diferencia` como diff: líneas `-` en rojo, `+` en verde, `@@` y contexto atenuados; envolver las líneas que exceden el ancho sin perder texto y sin romper el marco        | completada | `internal/tui/components/chat/vista.go`, `internal/tui/render/estilos.go`   | El bloque se ve bajo la línea, con sus colores y el texto completo; un ancho estrecho no desborda |
| T-F016-03 | actualizar | Mantener el bloque dentro de la línea para la selección con ratón y el scroll; sin zonas nuevas ni botones; el aviso `… N líneas más` se pinta como una línea más del bloque                   | completada | `internal/tui/components/chat/vista.go`, `internal/tui/selection.go`        | Arrastrar sobre el bloque lo copia como parte de la línea; la rueda y los bordes siguen funcionando |
| T-F016-04 | actualizar | Recuperar el bloque al retomar la sesión y probar la estructura con salidas doradas: el bloque no sustituye la línea, un cambio fallido no deja bloque y el recorte se conserva                | completada | `internal/tui/components/chat/*_test.go`, `internal/tui/*_test.go`, `internal/tui/testdata/*.golden` | El hilo recuperado se lee igual que en vivo; los dorados existentes no se rompen               |

**Dependencias:** `01` → `02` → `03`; `04` valida la integración. T-F016 depende de
T-B027 (el bloque y su persistencia) y de T-F015 (el bloque bajo la línea).

**Fuera de alcance:** plegar o expandir el diff con una tecla, un botón de copiar para el
bloque —la selección con ratón ya lo copia— y mostrar la diferencia en la aprobación de
rutas fuera del proyecto.

## Notas

- El bloque llega recortado desde el backend (40 líneas y `… N líneas más`): la TUI no
  vuelve a recortarlo ni lo recalcula.
- `diagrama` y `diferencia` comparten el mismo mecanismo de bloque; lo único que cambia
  es el estilo de las líneas.
- Si `bloque` viene vacío, no se pinta nada: es el caso de todas las herramientas que no
  producen bloque.
