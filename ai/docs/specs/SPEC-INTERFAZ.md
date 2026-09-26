---
title: SPEC — Interfaz
tags: [specs, requisito]
depende_de:
  - "[[IDEA]]"
  - "[[specs/SPEC-INTERFAZ-ATAJOS]]"
  - "[[specs/SPEC-PANEL-CONTEXTO]]"
  - "[[specs/SPEC-SESIONES]]"
  - "[[specs/SPEC-COLA-TAREAS]]"
  - "[[specs/SPEC-TOOLS]]"
relacionado:
  - "[[specs/SPEC-ARCHIVOS]]"
---
# SPEC — Interfaz

Prioridad: P0 (núcleo)

## Propósito

Definir dónde vive cada cosa en la pantalla: qué se ve, dónde está y cómo se recorre.

## Alcance

Incluye la disposición de la pantalla, las zonas, los datos que muestra cada una, la pantalla de bienvenida y cómo se cambia de sesión.
No incluye los atajos de teclado, que están en [[specs/SPEC-INTERFAZ-ATAJOS]] ni el mecanismo de resolución de teclas y la tecla líder, que está en [[specs/SPEC-KEYBINDS]], ni el significado y el cálculo de los números de contexto, que están en [[specs/SPEC-PANEL-CONTEXTO]], ni las reglas de permiso, que están en [[specs/SPEC-ARCHIVOS]].

## Actores

- **Usuario**: lee, escribe y navega.

## Disposición

El chat ocupa la pantalla completa. El panel de datos está plegado a la derecha y se abre y se cierra.

```
┌──────────────────────────────────┬──────────────┐
│                                  │  PANEL       │
│  CHAT                            │  DE DATOS    │
│                                  │              │
│   razonamiento del modelo        │  sesión      │
│   respuesta                      │  contexto    │
│   ...                            │  TODO        │
│                                  │  ruta        │
│                                  │  git         │
│                                  │  capa y cola │
│                                  │  aprobaciones│
│                                  │  agente      │
│                                  │  proyecto    │
├──────────────────────────────────┤  LocalCli    │
│  entrada de texto                │              │
└──────────────────────────────────┴──────────────┘

Con el panel cerrado, el chat ocupa todo el ancho.
```

El panel es de lectura. No se escribe nada desde ahí.

## Pantalla de bienvenida

Es la primera vista al ejecutar `localcli`, antes de que exista conversación: un logotipo en ASCII con el nombre de la herramienta y una única línea de entrada, como en opencode.

```
██╗      ██████╗  ██████╗  █████╗  ██╗ ██████╗ ██╗   ██╗
██║     ██╔═══██╗██╔════╝ ██╔══██╗ ██║██╔════╝ ██║   ██║
██║     ██║   ██║██║      ███████║ ██║██║      ██║   ██║
██║     ██║   ██║██║      ██╔══██╗ ██║██║      ██║   ██║
███████╗╚██████╔╝╚██████╗ ██║  ██║ ██║╚██████╗ ████║ ██║
╚══════╝ ╚═════╝  ╚═════╝ ╚═╝  ╚═╝ ╚═╝ ╚═════╝ ╚═══╝ ╚═╝

                   LocalCli · v0.1

             modelo: llama3.2  (Ctrl+X m cambiar)

             [plan] > En qué te ayudo hoy: █╚
```

A la izquierda del input figura el agente activo (`plan` o `build`); `Tab` lo alterna.

### Arte canónico del logotipo

El logotipo es fijo y esta es su definición, que sirve de salida dorada para las pruebas:

- Texto `LocalCli` en bloques de 6 filas × 53 columnas, con los caracteres de bloque `█ ╗ ╝ ║ ╔ ═ ╚` en UTF-8.
- La copia canónica byte a byte vive en `internal/tui/testdata/logo.txt`: seis líneas de exactamente 53 columnas, sin espacios finales. El bloque de arriba es su render fiel; la comparación de la prueba va contra el archivo.
- Debajo del arte van el nombre con versión, la línea con el modelo elegido y la línea de entrada, compuestos alrededor pero fuera del arte: no forman parte de la salida dorada.
- El arte no se reescala: las 6 filas × 53 columnas se pintan íntegras, sin recorte. Lo que sí hace la vista es centrar el bloque completo (logotipo + nombre con versión + línea de modelo + línea de entrada) dentro de la terminal, en horizontal y en vertical, según el tamaño reportado por la ventana. Si la terminal es más pequeña que el bloque, se alinea arriba a la izquierda sin recortar el arte.
- No entra al contexto del modelo ni a ningún historial: es arte de la aplicación.

- Solo hay logotipo, nombre con su versión, una línea que muestra el modelo en uso y una línea de entrada. Los modelos NO se listan en la bienvenida: se consultan y eligen desde un modal. Sin paneles ni datos.
- **Línea de modelo**: bajo el nombre con versión se muestra solo el modelo en uso (el detectado en el arranque o el último elegido en el modal) y el recordatorio del atajo para cambiarlo (`Ctrl+X m`). No hay lista de modelos visible en la bienvenida.
- **Modal de modelos**: la secuencia líder `Ctrl+X` seguida de `m` abre un modal centrado con la lista de modelos locales que reporta Ollama, con uno resaltado. `↑`/`↓` mueven la selección, `Enter` aplica el elegido y cierra el modal, `Esc` cierra sin cambiar nada. La lista se pide a Ollama al abrir el modal (no en el arranque); si Ollama no responde, el modal muestra el aviso «sin modelos» y se puede cerrar con `Esc` sin bloquear nada. Mientras el modal está abierto, las teclas son solo del modal: la escritura de la bienvenida no las recibe. Lo elegido pasa al motor como modelo de la sesión y se refleja en la línea de modelo de la bienvenida. La elección del usuario prevalece sobre la autodetección del arranque ([[specs/SPEC-OLLAMA-PERFIL]]: el modelo lo elige el usuario). El mecanismo de la secuencia está en [[specs/SPEC-KEYBINDS]].
- Lo que se escribe es la **primera petición de la sesión**: se envía tal cual, igual que se enviaría desde el chat.
- Al enviarla, la vista cambia a la interfaz principal y la petición aparece como primer mensaje del chat. La transición no repite la petición ni pide confirmación.
- La sesión que la recibe es la sesión activa del proyecto: se retoma si existe o se crea una nueva, con la misma regla que el resto de la aplicación. Ver [[specs/SPEC-SESIONES]].
- Desde la bienvenida no se lanza ningún flujo ni se abre el panel de contexto: hay logotipo, línea de modelo y línea de entrada; con `Ctrl+X m` se abre el modal de modelos y la salida es `Ctrl+C`.

## Zonas

### 1. Chat

- Ocupa el resto del ancho.
- Muestra el historial de la sesión activa.
- Cada intercambio muestra el razonamiento del modelo y su respuesta.
- Muestra las propuestas pendientes de aprobación.

### 2. Entrada de texto

- Una sola línea para escribir.
- Escribe hacia la sesión activa.
- **Indicador de agente a la izquierda del input**: justo al lado izquierdo de la línea de entrada se muestra el agente activo, `plan` o `build` (p. ej. `[plan] > █`). Cambia al instante con `Tab`, que alterna entre los agentes disponibles. El indicador es visible tanto en la interfaz principal como en la bienvenida.
- Cambia de comportamiento según el agente activo: con `plan` envía peticiones, con `build` revisa o cancela.

### 3. Panel de datos

| Dato | Qué muestra |
|---|---|
| Sesión | Nombre de la sesión activa |
| Contexto | Tokens usados y porcentaje ocupado |
| TODO | Elemento del TODO en curso y cuántos le quedan |
| Ruta | Carpeta actual del proyecto |
| Git | Rama activa y si hay cambios sin confirmar |
| Capa y cola | Capa en la que se está trabajando y cuántas tareas grandes le quedan |
| Aprobaciones | Cuántas hay esperando tu decisión |
| Agente | `plan` o `build` |
| Proyecto | Nombre y versión de LocalCli |

Los datos de contexto se calculan y se interpretan según [[specs/SPEC-PANEL-CONTEXTO]].

## Razonamiento del modelo

Se muestra **en vivo, arriba de la respuesta**, mientras el modelo genera.

- Se distingue visualmente de la respuesta para que nunca se confundan.
- Se puede ocultar para leer solo la respuesta.
- Ocultarlo no detiene la generación.

## Modales

Tres modales centrados comparten el mismo comportamiento: uno abierto a la vez, sus teclas capturan el teclado (`↑`/`↓` navegan, `Enter` aplica y cierra), `Esc` cierra sin cambios y `Ctrl+C` sigue saliendo de la aplicación.

| Modal | Cómo se abre | Contenido | Al aplicar |
|---|---|---|---|
| **Modelos** | `Ctrl+X m` | Lista de modelos locales de Ollama, cargada al abrir; aviso «sin modelos» si no responde | El modelo elegido pasa a ser el de la sesión y se ve en la línea de modelo |
| **Sesiones** | `Ctrl+X l` | Lista de las sesiones creadas anteriormente en el proyecto (nombre y estado, incluidas las de segundo plano) | Se abre esa sesión: el chat pasa a su historial sin detener lo que corre |
| **Atajos** | `Ctrl+P` | Tabla de los atajos existentes: acción y tecla(s) de cada uno, incluidas las secuencias con líder | No aplica nada: es solo lectura; `Esc` lo cierra |

No hay paleta de comandos ejecutables: `Ctrl+P` abre la lista de atajos. Ver [[specs/SPEC-SESIONES]] para el ciclo de vida de las sesiones.

## Cambiar de sesión

No hay una lista de sesiones siempre visible. El chat es para la sesión activa.

- `Ctrl+X l` abre el **modal de sesiones** con las sesiones del proyecto: nombre y estado de cada una.
- Al elegir una con `Enter`, el chat cambia a esa sesión.
- Las sesiones en segundo plano se ven desde cualquier otra en el modal, con su estado.

El modal aparece y desaparece. No ocupa espacio permanente.

## El aviso que no se puede ocultar

Con el panel cerrado, si alguna sesión está esperando tu aprobación, aparece una línea discreta indicando cuántas hay.

Es la única información que se muestra fuera del panel, porque es la única cuya ausencia detiene el trabajo: un flujo en segundo plano se queda parado hasta que decidas, y si no lo ves, no avanzas. El detalle de cada aprobación está en [[specs/SPEC-INTERFAZ-ATAJOS]].

## Reglas de negocio

- El panel de datos es de lectura. Nada se escribe desde él.
- El panel se abre y se cierra sin interrumpir ninguna sesión.
- Abrir el panel no ralentiza la generación de una respuesta.
- El razonamiento se muestra en vivo y se puede ocultar.
- El razonamiento nunca se mezcla visualmente con la respuesta final.
- No hay lista de sesiones permanente: se acceden con `Ctrl+X l`, que abre el modal de sesiones.
- Hay exactamente tres modales (modelos, sesiones, atajos); solo uno puede estar abierto a la vez y `Esc` cierra cualquiera.
- El agente activo (`plan` o `build`) se muestra siempre a la izquierda del input; `Tab` alterna entre los agentes disponibles, en bienvenida y en la vista principal. Con un modal abierto, `Tab` no cicla.
- Cambiar de sesión no detiene lo que está corriendo.
- El panel refleja los datos de la sesión activa, no de otra.
- Con el panel cerrado, el número de aprobaciones pendientes siempre se ve.
- El nombre y la versión de LocalCli se ven siempre que el panel esté abierto.
- La pantalla de bienvenida es la primera vista al ejecutar `localcli` y muestra logotipo, nombre con versión, la línea con el modelo en uso y una línea de entrada, todo centrado en la terminal. Los modelos solo se ven dentro del modal.
- Lo escrito en la bienvenida es la primera petición: se envía a la sesión activa y la vista cambia a la principal sin repetir ni confirmar.
- Desde la bienvenida no hay panel de contexto ni aprobaciones: solo abrir el modal de modelos (`Ctrl+X m`), escribir, enviar y salir.
- La bienvenida se pinta sin esperar a Ollama ni a la base: no depende de nada externo para mostrarse.

## Criterios de aceptación

- [ ] Con el panel cerrado, el chat ocupa todo el ancho.
- [ ] El panel se abre y se cierra sin interrumpir el trabajo.
- [ ] El panel muestra los nueve datos definidos.
- [ ] El panel muestra los datos de la sesión activa, no los de otra.
- [ ] El razonamiento se muestra mientras se genera, arriba de la respuesta.
- [ ] El razonamiento se distingue visualmente de la respuesta y se puede ocultar.
- [ ] `Ctrl+X l` abre el modal de sesiones con nombre y estado de cada una; al seleccionar una con `Enter` se abre esa sesión y el chat muestra su historial.
- [ ] `Ctrl+P` abre el modal con la lista de atajos existentes (acción + tecla); es de solo lectura.
- [ ] `Esc` cierra cualquier modal sin cambiar nada; con ninguno abierto no hace nada visible.
- [ ] `Tab` alterna el agente entre `plan` y `build`; el indicador del agente aparece a la izquierda del input tanto en la bienvenida como en la vista principal y se actualiza al instante.
- [ ] Elegir una sesión cambia el chat a esa sesión sin detener lo demás.
- [ ] Con el panel cerrado se ve cuántas aprobaciones hay pendientes.
- [ ] El nombre y la versión de LocalCli aparecen en el panel.
- [ ] Cambiar de sesión no detiene ninguna ejecución.
- [ ] Al ejecutar `localcli` se ve la pantalla de bienvenida con logotipo, nombre, la línea del modelo en uso y una línea de entrada, sin lista de modelos visible.
- [ ] El bloque de la bienvenida (logotipo, nombre, línea de modelo y entrada) aparece centrado en horizontal y en vertical dentro de la terminal, sin recortar el arte.
- [ ] `Ctrl+X m` abre el modal de modelos con la lista de Ollama; `↑`/`↓` navegan, `Enter` aplica el resaltado y `Esc` cierra sin cambios. El modelo aplicado se usa para la primera petición y aparece en la línea de modelo.
- [ ] Sin Ollama disponible, la bienvenida se muestra igual (usa el modelo por defecto del arranque); al abrir el modal aparece el aviso «sin modelos» y se puede escribir y enviar sin él.
- [ ] La primera petición escrita en la bienvenida aparece como primer mensaje del chat al cambiar de vista.
- [ ] La transición de bienvenida a interfaz principal no repite la petición ni pide confirmación.
- [ ] Desde la bienvenida no hay panel de contexto ni aprobaciones; solo abrir el modal de modelos, escribir, enviar y salir.
- [ ] La bienvenida se muestra aunque Ollama no esté disponible.
- [ ] Con la líder pulsada (`Ctrl+X`) pero sin segunda tecla, no se abre ningún modal y tras el timeout el indicador de líder desaparece.
- [ ] El logotipo coincide byte a byte con la salida dorada de `internal/tui/testdata/logo.txt`.

## Requisitos no funcionales

- Mostrar el panel y el razonamiento no debe retrasar la respuesta del modelo.
- Las sesiones en segundo plano se siguen avanzando con el panel cerrado.
- La bienvenida se pinta al instante: no espera a conexiones externas.

## Dependencias funcionales

- [[specs/SPEC-INTERFAZ-ATAJOS]]
- [[specs/SPEC-KEYBINDS]]
- [[specs/SPEC-PANEL-CONTEXTO]]
- [[specs/SPEC-SESIONES]]
- [[specs/SPEC-COLA-TAREAS]]
- [[specs/SPEC-TOOLS]]

## Supuestos

- La librería de interfaz de terminal y la biblioteca de TUI concreta se eligen en FASE 2.
- El ancho mínimo de terminal y el comportamiento en terminales estrechas se fijan en FASE 2 y FASE 3.

## Referencias

- [[IDEA]]
