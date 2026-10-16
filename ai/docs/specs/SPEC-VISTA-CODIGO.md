---
title: SPEC — Vista de código real
tags: [specs, requisito]
depende_de:
  - "[[IDEA]]"
  - "[[specs/SPEC-INTERFAZ]]"
relacionado:
  - "[[specs/SPEC-INTERFAZ-ATAJOS]]"
  - "[[frontend/01-domain/DOMAIN]]"
  - "[[frontend/02-interfaces/INTERFACES]]"
---
# SPEC — Vista de código real

Prioridad: P2 (interfaz)

## Propósito

Hacer legible lo que el agente entrega. Hoy el texto de la respuesta se pinta plano, de
modo que un bloque de código llega crudo —con sus fences ```` ``` ```` y sin ninguna
estructura— y hay que arrastrar el ratón para copiarlo. Esta spec define cómo el chat
reconoce el Markdown de la respuesta, cómo pinta un bloque de código como una superficie
propia y cómo ofrece un botón para copiarlo de un clic.

## Alcance

Incluye:

- El reconocimiento de un **subconjunto** de Markdown dentro del texto de cada respuesta:
  bloques de código, títulos, negrita, cursiva, código inline, listas, citas y reglas
  horizontales. El resto del texto se pinta literal.
- El pintado de un **bloque de código** como superficie propia, con su cabecera y su
  lenguaje.
- Un **botón de copiar** en la cabecera del bloque, que copia su contenido al portapapeles
  con un clic, **sin arrastre**.
- La envoltura de las líneas de código que exceden el ancho, para no perder código.

No incluye:

- Resaltado de sintaxis por tokens (colorear palabras clave, cadenas, comentarios).
- Renderizado de tablas, imágenes ni HTML embebido.
- Botón de copiar por teclado, ni navegación entre bloques: la copia es por clic en la
  cabecera.
- Cambios en cómo se guarda la respuesta: el texto se guarda crudo, tal como lo entrega el
  modelo; el Markdown se interpreta **solo al pintar**.

El resultado se ve en la zona del chat ([[specs/SPEC-INTERFAZ]] §Zonas 1); el detalle de
componentes y presentación vive en [[frontend/01-domain/DOMAIN]] y
[[frontend/02-interfaces/INTERFACES]].

## Actores

- **Usuario**: lee el código y, cuando lo necesita, pulsa su botón de copiar.
- **Sistema**: reconoce el Markdown de la respuesta, pinta el bloque y resuelve el clic.

## Flujo principal

1. El agente responde con texto que puede contener Markdown, incluidos bloques de código
   entre fences ```` ``` ````.
2. Al pintar cada segmento del chat, el sistema interpreta el subconjunto de Markdown:
   - Un bloque de código se pinta como superficie propia con su cabecera (el lenguaje
     declarado, o una etiqueta genérica si no lo hay) y su botón `copiar`.
   - El Markdown en línea (títulos, negrita, cursiva, código inline, listas, citas, reglas)
     se pinta con su estilo; el resto del texto, literal.
3. El usuario pulsa el botón de la cabecera de un bloque.
4. El sistema copia al portapapeles el **cuerpo del bloque** (sin los fences) y muestra el
   aviso transitorio `[Copiado]`.

## Flujos alternativos

- **Bloque sin cerrar.** Mientras el modelo genera, un fence abierto sin su cierre se pinta
  como bloque en curso; al llegar el cierre, se completa. El botón funciona igual sobre lo
  que ya hay.
- **Lenguaje desconocido o ausente.** La cabecera muestra una etiqueta genérica; el
  comportamiento del bloque no cambia.
- **Línea de código más ancha que el bloque.** Se envuelve en varias filas; el código no se
  recorta ni se pierde.
- **Markdown mal formado.** Lo que no encaje en el subconjunto se pinta literal: nunca se
  pierde texto.

## Reglas de negocio

- El Markdown **no se interpreta dentro de un bloque de código**: el cuerpo de un fence es
  literal, salvo su envoltura al ancho.
- El **código inline** y el cuerpo de un bloque **no** aplican negrita, cursiva ni listas.
- Los fences de apertura y cierre **no se muestran** al usuario: se reemplazan por la
  superficie del bloque.
- El botón copia el **cuerpo exacto** del bloque, tal como lo entregó el modelo, sin los
  fences, sin la cabecera y sin las columnas de la superficie.
- El clic sobre el botón **no inicia una selección**: es una acción, como los botones del
  panel de aprobaciones.
- El copiado usa el mismo mecanismo y el mismo aviso `[Copiado]` que la selección con el
  ratón; si el portapapeles del sistema no está disponible, cae en OSC 52.
- El texto guardado y enviado al modelo **no cambia**: la interpretación es solo de pintado.

## Subconjunto de Markdown

| Construcción                     | Entrada                                | Cómo se pinta                                          |
| -------------------------------- | -------------------------------------- | ------------------------------------------------------ |
| Bloque de código                 | ```` ```lenguaje ```` … ```` ``` ````  | Superficie propia con cabecera (lenguaje + botón)      |
| Título                           | `#`, `##`, `###` al inicio de línea    | En negrita, sin mostrar el `#`                         |
| Negrita                          | `**texto**`                            | En negrita                                             |
| Cursiva                          | `*texto*` o `_texto_`                  | En cursiva                                            |
| Código inline                    | `` `texto` ``                          | Estilo propio de código, en la misma línea             |
| Lista sin orden                  | `- ` o `* ` al inicio                  | Viñeta `•`                                             |
| Lista ordenada                   | `1. ` al inicio                        | Conserva el número                                     |
| Cita                             | `> ` al inicio                         | Prefijo `│` atenuado                                   |
| Regla horizontal                 | `---` en su propia línea               | Línea de `─`                                           |

## Criterios de aceptación

- [ ] Un bloque de código entre fences se pinta como superficie propia, con su cabecera y su
      lenguaje, sin mostrar los fences.
- [ ] La cabecera del bloque tiene un botón de copiar visible.
- [ ] Un clic sobre el botón copia el cuerpo del bloque al portapapeles y muestra el aviso
      `[Copiado]`; no hace falta arrastrar para copiar.
- [ ] Lo copiado es el código exacto, sin fences, sin cabecera y sin las columnas de la
      superficie.
- [ ] Un bloque sin cerrar (generación en curso) se pinta como bloque y su botón funciona
      sobre lo que ya hay.
- [ ] Las líneas de código más anchas que el bloque se envuelven; no se recorta ni se pierde
      código.
- [ ] El Markdown dentro de un bloque de código no se interpreta.
- [ ] Títulos, negrita, cursiva, código inline, listas, citas y reglas se pintan con su
      estilo; lo que no encaja en el subconjunto se pinta literal.
- [ ] El clic sobre el botón no inicia una selección de texto.
- [ ] El texto guardado y el que se envía al modelo no cambian por esta feature.

## Requisitos no funcionales

- Interpretar el Markdown y pintar el bloque no debe retrasar perceptiblemente la respuesta
  ni la generación token a token.
- La feature no añade dependencias: se implementa sobre la capa de render existente.
- El bloque se reajusta al ancho disponible al redimensionar la terminal.

## Dependencias funcionales

- [[specs/SPEC-INTERFAZ]]

## Supuestos

- El modelo entrega Markdown con fences de tres backticks, como es habitual.
- La respuesta se guarda cruda; la interpretación es responsabilidad de la vista.

## Referencias

- [[IDEA]]
