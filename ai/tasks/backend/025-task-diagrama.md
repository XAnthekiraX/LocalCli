# 025-task-diagrama.md

> T-B025 — Herramienta `diagram`: parser, IR, layout, render, handler, integración al catálogo y emisión del bloque en `herramienta_resultado`.
> Acción de esta descomposición: `crear`; T-B025-11 se añade con `actualizar` al fijar el contrato de entrada.

## Referencias

- [[specs/SPEC-DIAGRAMA]] — propósito, alcance, contrato de entrada `{code}`, sintaxis admitida (flowchart TD/LR), §Ejemplo completo, IR, layout, render Unicode, errores corregibles, límites adaptativos y presentación en la conversación.
- [[backend/02-interfaces/TOOLS]] §1 «Diagramas», §2 reparto (`read` para ambos agentes), §8 capa universal.
- [[backend/02-interfaces/dto/TOOLS-DTO]] §3 Request (`diagram.code`), §4 Response (`diagram` devuelve string).
- [[backend/04-infrastructure/EVENTS]] §3 `herramienta_resultado` — extensión mínima con campo opcional `bloque` solo para `diagram`.
- [[backend/05-quality/ERRORS]] §3 Códigos internos (`E_BAD_ARGS`, `E_PERM_DENIED`, `E_TOOL_EXEC`, etc.), §5 Errores por operación.
- [[backend/05-quality/VALIDATION]] — validación de argumentos antes de procesar.
- [[backend/03-security/SECURITY]] — entrada no confiable, sin ejecución de código Mermaid, robustez ante panics.
- [[backend/01-domain/DOMAIN]] — límites entre `tools`, `agent`, `tui`.
- [[frontend/02-interfaces/INTERFACES]] §1.1 — la línea de herramienta y el bloque de diagrama (adicional, no sustituto).

## Contexto

La herramienta `diagram` convierte un subconjunto de Mermaid flowchart en un diagrama Unicode (no ASCII puro) mostrado como bloque visual en la TUI, sin navegador/SVG/imagen. Su permiso es `read` para `plan` y `build`.

No existen aún archivos de implementación para esta herramienta; `internal/tools/` solo contiene `CategoriaDiagramas` en `modelo.go`. Hay que crear el handler, parser, IR, layout, renderizador y cablearlo al catálogo, siguiendo las reglas de la capa universal (permiso antes que validación, medición de duración, recorte si aplica, emisión de `herramienta_invocada`/`herramienta_resultado`).

La SPEC indica que, si el payload actual de `herramienta_resultado` no basta, se extiende mínimamente para transportar el bloque: hay que añadir ese campo opcional `bloque` para `diagram`, sin alterar el formato de los demás mensajes.

## Contrato de entrada

La llamada de herramienta es un JSON con un único argumento `code`:

```json
{
  "code": "flowchart TD\n  A[Inicio] --> B{¿Sigo?}\n  B -->|Sí| C[Fin]"
}
```

`code` es un **string con texto Mermaid**, no un objeto de grafo. No hay parámetros `nodes`/`edges` ni campos separados para orientación o título: la orientación sale de la cabecera (`TD`/`LR`) y el grafo se deduce del código. El handler valida ese string, lo pasa al parser y devuelve el bloque renderizado en `Salida`; el mismo texto viaja en `bloque` para la TUI y se persiste con la línea. El ejemplo end-to-end está en [[specs/SPEC-DIAGRAMA]] §Ejemplo completo.

## Tareas pequeñas

| ID         | Acción | Tarea                                                                 | Estado    | Archivos                                                                                   | Verificación                                                                             |
| ---------- | ------ | --------------------------------------------------------------------- | --------- | ------------------------------------------------------------------------------------------ | ---------------------------------------------------------------------------------------- |
| T-B025-01  | crear  | Tipo de petición y schema: `PeticionDiagram` con `Code string \`json:"code" desc:"Código Mermaid flowchart (TD/LR, formas básicas, conexiones etiquetadas)"\``. Añadir a `peticiones.go`. | completada | `internal/tools/peticiones.go`                                                             | Derivación por reflexión expone `code` obligatorio.                                      |
| T-B025-02  | crear  | Añadir herramienta `diagram` al catálogo cerrado (`Catalogo()`), con categoría `CategoriaDiagramas`, permiso `read`, modo `lee`, verbo `DIAGRAMAR`, tema `code` (o `code` primer trozo), tipo de petición reflect.TypeOf(PeticionDiagram{}). | completada | `internal/tools/catalogo.go`, `internal/tools/modelo.go` (asegurar `CategoriaDiagramas` usada correctamente) | `plan` y `build` reciben `diagram`; `diagram` es `read`.                                 |
| T-B025-03  | crear  | IR: estructuras `Nodo` (ID, Texto, Forma), `Conexion` (Origen, Destino, Etiqueta), `Diagrama` (Nodos, Conexiones, Orientacion, Posiciones, Dimensiones). Separada de parser y render. | completada | `internal/tools/diagram/ir.go` (nuevo paquete)                                             | Testable: construir IR desde datos conocidos.                                             |
| T-B025-04  | crear  | Parser del subconjunto: reconocer cabecera `flowchart TD/LR` (default TD), nodos `[ ]`, `{ }`, `(( ))`, conexiones `-->`, etiquetas `-->\|texto\|`, IDs duplicados contradictorios, nodos sin forma (implícitos, con aviso) y directivas cosméticas (`style`, `classDef`, `class`, `linkStyle`, `%%`, ignoradas con aviso). Errores corregibles con línea/posición. | completada | `internal/tools/diagram/parser.go`                                                         | Casos: A[Uno]->B{Pregunta}? Sí\|C, IDs repetidos contradictorios generan error; `A --> B` crea B con aviso; `style` no rompe. |
| T-B025-05  | crear  | Layout por niveles (vertical TD, horizontal LR), ramas y destinos compartidos, ciclos aceptados con arista de vuelta por el margen, compacta al espacio disponible. Un nivel que no cabe a lo ancho se reparte en una lista vertical y sus aristas se enrutan por un bus; no elimina información semántica y, si no cabe, devuelve error explícito. | completada | `internal/tools/diagram/layout.go`                                                         | TD y LR, destinos compartidos, abanico de 20 hermanos sin desbordar el ancho; ciclo de vida con iteración se dibuja sin buclear. |
| T-B025-06  | crear  | Render Unicode con ancho de columna real (`runewidth`) de la IR posicionada: bordes, flechas, formas, textos. Respeta límites de dimensiones, nunca asume 1 columna por carácter. | completada | `internal/tools/diagram/render.go`                                                         | Salida legible, ancho real calculado; textos largos recortados con aviso (no silencioso). |
| T-B025-07  | crear  | Handler `diagramHandler`: valida args, llama parser→layout→render, devuelve `Resultado{Salida: string(renderizado), Meta: ...}`. La medida es "X líneas" o "medida adecuada" (definir unidad: líneas del bloque). No expone código de entrada. | completada | `internal/tools/diagram/handler.go`, cableado en `internal/tools/registro` o integrar en catálogo vía `Ejecutar` | Sigue orden capa universal (permiso→validación); duración medida solo handler.             |
| T-B025-08  | crear  | Integración al ciclo: el handler debe producir el bloque para TUI. Ampliar `herramienta_resultado` para transportar `Bloque string` opcional cuando herramienta es `diagram`; en otros casos vacío. Actualizar `eventos.go` (emisión) y payloads/eventos backend. | completada | `internal/tools/eventos.go`, `internal/session/orquestador.go` o capa que emite evento según payload existente | Solo `diagram` emite bloque; otros mensajes no se alteran.                                |
| T-B025-09  | crear  | Límites: código ~8KB, nodos ~50, conexiones ~100, etiquetas recortadas con aviso. Aplicar validaciones explícitas; errores corregibles con ejemplo válido. | completada | `internal/tools/diagram/limites.go` o dentro de parser/layout                              | Excede límites → error corregible.                                                         |
| T-B025-10  | crear  | Pruebas unitarias: parser (subconjunto), layout (TD/LR, compartidos), render (ancho real, Unicode), límites y errores; `tools_test.go` sigue pasando (catálogo 16→17). | completada | `internal/tools/diagram/*_test.go`, actualizar tests existentes si necesario               | Todas pasan; `TestCatalogoTieneLasDiecisiete` espera 17 herramientas. |
| T-B025-11  | actualizar | Prueba de contrato end-to-end: la llamada `{"code": "<mermaid>"}` del §Ejemplo completo produce el bloque esperado, sin `nodes`/`edges`; el resultado al modelo es el bloque en texto y el evento lleva `bloque`. | completada | `internal/tools/diagram/*_test.go`, `internal/tools/*_test.go` | El ejemplo de la spec se reproduce; una llamada sin `code` devuelve `E_BAD_ARGS`. |

## Dependencias
`T-B025-01→02→07→08`; `T-B025-03→04→05→06→07`; `T-B025-09` cruza parser/layout. `T-B025-10` independiente por capa. `T-B025-11` cierra el contrato end-to-end y depende de `07`/`08`.

## Notas de implementación

- El paquete se creó limpio en `internal/tools/diagram/` (parser, IR, layout, render, límites y handler).
- Entrada no confiable: sin ejecución. Validar antes de procesar, tolerante a fallos, sin panics.
- El bloque que viaja a TUI es el texto renderizado ya calculado (líneas Unicode). El modelo recibe la salida por protocolo de herramientas (texto del diagrama) tal como devuelve el handler.
- Contrato de entrada: la llamada es `{"code": "<mermaid>"}`; el parser recibe un string, nunca un grafo JSON. El §Ejemplo completo de [[specs/SPEC-DIAGRAMA]] es el caso de aceptación.
- En `herramienta_resultado`, la regla "no lleva salida" sigue vigente salvo `bloque` opcional exclusivo para `diagram`. Esto es extensión mínima del payload (añadir campo opcional), documentado en `EVENTS.md`.
- Atender `tools_test.go`: `TestCatalogoTieneLasDiecisiete` espera 17 herramientas tras añadir `diagram`. **No eliminar las pruebas existentes**, ajustarlas coherentemente.
- El verbo en presentación es `DIAGRAMAR`, tema `code` (primera línea) como en TOOLS.md §1 tabla TUI.

## Cierre

Implementado en `internal/tools/diagram/` (`ir.go`, `parser.go`, `layout.go`,
`render.go`, `limites.go`, `handler.go`) y cableado en `cmd/localcli/main.go`
con `registro.Instalar("diagram", diagram.Handler)`. Cambios posteriores a la
primera versión, motivados por fallos reales con diagramas anchos:

- Un nivel cuyas cajas no caben a lo ancho se dispone como lista vertical; las
  aristas que no pueden bajar en línea recta salen a un bus en la columna 0 y
  entran por el carril o por la fila libre sobre la caja de destino. Así un
  abanico de veinte hermanos con etiquetas largas cabe en 100 columnas.
- Los identificadores sin forma (`A --> B`) se crean como rectángulos
  implícitos con el ID como texto y un aviso; una definición explícita manda.
- Los ciclos se aceptan: un DFS marca las aristas de vuelta, se excluyen del
  reparto por niveles y se dibujan por el canal del margen derecho (`TD`) o
  inferior (`LR`). Un ciclo de vida con iteraciones ya no devuelve error.
- Las directivas cosméticas (`style`, `classDef`, `class`, `linkStyle`, `%%`)
  se ignoran con aviso en vez de rechazar el diagrama.
- Límites finales: 8 KB de código, 50 nodos, 100 conexiones, 40 caracteres por
  texto, lienzo de 100×200.
- El tema visible de `diagram` es la primera línea del código (`flowchart TD`).
- El bloque del diagrama se centra en el ancho del hilo conservando la alineación
  interna de sus líneas; el diff de los cambios sigue alineado a la izquierda.

Verificación: `go build ./...`, `go vet ./...` y `go test ./...` pasan.

## Fuera de alcance

- Mermaid completo (secuencias, clases, Gantt, etc.).
- Un grafo JSON de nodos y conexiones como entrada: el diagrama se deduce del texto Mermaid de `code`.
- Render SVG/imagen/navegador.
- Parámetros `titulo`, `orientación` obligatorios (orientación sale de código).
- Cambios DB.
- Bucles de reintento autónomos.

