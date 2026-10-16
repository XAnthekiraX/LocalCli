# 014-task-vista-codigo.md

> T-F014 — Vista de código real: interpretar el Markdown de la respuesta del agente y
> ofrecer un botón para copiar cada bloque de código.
> Acción de esta descomposición: `crear`.

## Referencias

- [[specs/SPEC-VISTA-CODIGO]] — subconjunto de Markdown, reglas del bloque y del botón.
- [[specs/SPEC-INTERFAZ]] §Zonas 1 (Chat)
- [[frontend/FRONTEND]] §2 Estructura del Proyecto (`components/chat/`, `render/`, `botones.go`)
- [[frontend/01-domain/DOMAIN]] §1 Componentes (`chat`), §2 Reglas de presentación
- [[frontend/02-interfaces/INTERFACES]] §4 Teclado (Ratón)
- [[frontend/05-quality/TESTING]] §2 Qué debe probarse (Markdown y botón de copiar)

## Contexto

Hoy el texto de la respuesta se pinta plano: `contenidoSegmento` (`components/chat/vista.go`)
envuelve `Segmento.Texto` y lo pinta con `render.EstiloBlanco`, sin interpretar nada. Un bloque
de código entregado entre fences ```` ``` ```` llega crudo (fences incluidos) y para copiarlo hay
que arrastrar el ratón. Esta tarea interpreta el Markdown **solo al pintar** —el texto guardado y
el que se envía al modelo no cambian— y añade un botón clicable en la cabecera de cada bloque de
código que copia su cuerpo al portapapeles, reutilizando `copiarCmd` y el aviso `[Copiado]`.

No hay cambios de backend ni de base de datos: el Markdown vive en el texto ya almacenado.

## Tareas pequeñas

| ID         | Acción | Tarea                                                                                                                                                                                                    | Estado    | Archivos                                                                                  | Verificación                                                                                          |
| ---------- | ------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------- | ----------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------- |
| T-F014-01  | crear  | Parser + render del subconjunto Markdown (fences con lenguaje, títulos, negrita, cursiva, código inline, listas, citas, reglas): función pura que devuelve las líneas pintadas y los bloques de código con su geometría y su texto, respetando el ancho y envolviendo las líneas largas. Un fence sin cerrar se trata como bloque en curso y el Markdown dentro de un bloque no se interpreta. | completada | `internal/tui/components/chat/markdown.go`, `internal/tui/components/chat/markdown_test.go` | `go test ./internal/tui/components/chat/...`; el parser es puro y no depende del Model |
| T-F014-02  | crear  | Estilos del bloque de código en la capa de render: superficie propia (distinta del globo), etiqueta de lenguaje, cabecera y botón `copiar`, y estilo del código inline. Sin dependencias nuevas. | completada | `internal/tui/render/estilos.go`                                                          | `go build ./...`; el bloque se distingue del fondo del chat                                                  |
| T-F014-03  | crear  | Integrar el render en el segmento: `contenidoSegmento` usa el renderizador de Markdown en vez de la envoltura plana, y el `Model` registra por línea de contenido los botones de copia (línea, columnas, ancho y código), exponiendo `BotonEnVentana(fila, col)` en la traducción contenido↔ventana. | completada | `internal/tui/components/chat/vista.go`, `internal/tui/components/chat/chat.go`, `internal/tui/components/chat/traduccion.go` | El botón de un bloque es localizable en coordenadas de ventana                    |
| T-F014-04  | crear  | Resolver el clic: en el enrutado del ratón, un clic izquierdo sobre un botón de copia copia su código al portapapeles y muestra `[Copiado]`, sin iniciar una selección; el patrón sigue el de los botones de aprobación. | completada | `internal/tui/botones.go`, `internal/tui/selection.go`                                    | Un clic sobre el botón copia y no abre selección; `copiarFunc` permite el doble en pruebas            |
| T-F014-05  | crear  | Pruebas: unitarias del parser/render (fence cerrado y sin cerrar, lenguaje ausente, líneas largas, Markdown dentro del bloque literal, negrita/cursiva/inline/listas/citas/reglas) y a nivel `app` del clic-copia con el doble de portapapeles; actualizar las salidas doradas afectadas. | completada | `internal/tui/components/chat/markdown_test.go`, `internal/tui/botones_test.go`, `internal/tui/testdata/*.golden`, `internal/tui/golden_test.go` | `go test ./internal/tui/...` en verde                                    |

**Dependencias:** `01`–`02`→`03`; `03`→`04`; `01`–`04`→`05`.

## Notas

- La interpretación es **solo de pintado**: el texto guardado y el enviado al modelo no cambian
  ([[specs/SPEC-VISTA-CODIGO]] §Reglas de negocio).
- El botón copia el **cuerpo exacto** del bloque —sin fences, sin cabecera y sin las columnas de
  la superficie— y usa `copiarCmd`/`avisarCopiado` de `selection.go`, con OSC 52 como respaldo.
- Sin atajo de teclado: la copia es por clic en la cabecera.
- Estilo del bloque sin glifos de adorno: la superficie es el fondo, no un carácter
  ([[frontend/01-domain/DOMAIN]] §2).
- La copia desde el botón debe funcionar aunque la línea del botón esté fuera del viewport una
  vez replicada la traducción de coordenadas; si no, basta con la visible.
