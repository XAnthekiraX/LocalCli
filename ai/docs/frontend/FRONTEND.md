---
title: LocalCli — capa de interfaz
tags: [frontend, indice]
depende_de:
    - "[[PROJECT]]"
    - "[[backend/BACKEND]]"
    - "[[specs/SPEC-INTERFAZ]]"
relacionado:
    - "[[specs/SPEC-INTERFAZ-ATAJOS]]"
    - "[[database/DATABASE]]"
    - "[[backend/01-domain/DOMAIN]]"
    - "[[backend/04-infrastructure/EVENTS]]"
    - "[[database/03-operations/QUERIES]]"
    - "[[frontend/01-domain/DOMAIN]]"
    - "[[frontend/02-interfaces/INTERFACES]]"
    - "[[frontend/05-quality/TESTING]]"
---

# FRONTEND — LocalCli

La capa de interfaz: una TUI en terminal construida con Bubble Tea y Lip Gloss. Es el único consumidor del motor de inferencia y su único trabajo es pintar lo que llega y enviar lo que pulsas. No ejecuta nada por su cuenta: no habla con el motor de inferencia, no escribe en SQLite y no toca archivos.

Este documento es el mapa de navegación del frontend. El orquestador está en [[backend/BACKEND]]; los datos, en [[database/DATABASE]].

## 1. Visión General

- **Arquitectura Elm de Bubble Tea:** un modelo de estado, un bucle de eventos (`update`) y una función de pintado (`view`). Un solo bucle; sin goroutines propias más allá de las suscripciones de eventos. Dos vistas que ese bucle pinta: la bienvenida y la interfaz principal.
- **Presentación pura.** El frontend no decide nada de negocio: no comprueba permisos, no recorta contexto, no calcula estados. Recibe eventos y manda peticiones.
- Lo que ve llega por dos caminos: los eventos de [[backend/04-infrastructure/EVENTS]] y las consultas de lectura de [[database/03-operations/QUERIES]]. Lo que hace lo pide a `session`, que es la única puerta del orquestador.

## 2. Estructura del Proyecto

```
internal/tui/
  app.go         modelo raíz, enrutado de eventos y suscripciones, y composición de la vista
                 principal (la línea de actividad se pinta fija encima de la caja de entrada)
  teclado.go     resolución de una pulsación: modales, paleta, edición de las cajas y despacho de acciones
  peticiones.go  enviar lo escrito, crear sesión en la bienvenida y lanzar/cancelar turnos
  eventos.go     aplicación de los eventos del orquestador al estado de vista
 comandos.go    paleta de comandos de flujo: catálogo del orquestador, filtro y render; superficie
                  propia (#B1BAF9) alineada y del ancho de la caja de entrada; **overlay que
                  muestra como máximo 5 elementos y scroll vertical (▲/▼)**
  approvals.go   panel de aprobaciones pendientes de todas las sesiones
  notify.go      línea de aviso de aprobaciones pendientes con el panel cerrado
  selection.go   selección con el ratón, limitada a la zona donde empezó el arrastre y anclada al texto:
                 en el chat se ve en coordenadas de contenido y se prolonga fuera del viewport al llegar
                 a un borde (desplazamiento automático); copia al portapapeles
  botones.go     hit-test de los botones clicables del chat: el botón de copiar de la cabecera de un bloque
                 de código copia su cuerpo al portapapeles (SPEC-VISTA-CODIGO), como los botones de aprobación
  keymap.go      mapa central de teclas: acciones, bindings múltiples y tecla líder
  keyresolver.go KeyResolver: líder ctrl+x, timeout, contexto y acciones
  keys.go        persistencia del mapa de teclas en ~/.config/localcli/keys.json
  estilos.go     paleta del usuario: lee/siembra ~/.config/localcli/styles.json y la aplica
  config.go      preferencias del usuario: último agente, presupuesto de historial y razonamiento
  logo.go        logotipos canónicos de la bienvenida (completo y compacto)
  wire.go        Puerto: la frontera de la tui con el orquestador (eventos y peticiones)
  arranque.go    arranque del programa Bubble Tea (alt screen, ratón)
  doc.go         documentación del paquete
  render/        capa común de render: estilos Lip Gloss —incluido el estilo del bloque de código y su botón—
                 sobre una paleta de tokens editable (colores de texto y fondos), normalización del marco
                 y envoltura de texto
  components/
    adjuntos/    motor compartido de tokens de archivo, carpeta o texto pegado ([nombre.ext],
                 [CARPETA N elementos], [PEGADO N líneas]) y su expansión al valor real
    welcome/     pantalla de bienvenida estilo opencode: logotipo sobre el fondo, caja de entrada con
                 línea de estado —y pegado tokenizado— y pistas de teclado
    chat/        historial de la sesión activa: respuesta —con su Markdown interpretado al pintar, bloques
                 de código y botón de copiar (SPEC-VISTA-CODIGO)—, líneas compactas de herramienta con sus
                 bloques visuales (diagrama y diferencia de un cambio), razonamiento revelable y la línea de
                 actividad en vivo
    input/       caja de entrada con fondo: línea de texto —con pegado tokenizado— y pie con el agente y el modelo
    panel/       sidebar de datos plegable: título, contexto, TODO, tareas, estado y pie
    modals/      modales de modelos (Ctrl+X m), sesiones (Ctrl+X l) y atajos (Ctrl+P)
  testdata/      salidas doradas: logo.txt (logotipo canónico de la bienvenida) y *.golden
```

## 3. Contratos

- **Entradas:** teclado y eventos del orquestador. Ver [[frontend/02-interfaces/INTERFACES]].
- **Salidas:** peticiones a `session` (enviar mensaje, cambiar de sesión, aprobar, declinar, cancelar flujo).
- **Lecturas:** solo las consultas definidas en [[database/03-operations/QUERIES]]. Nunca escribe en la base: el estado lo persiste el orquestador.
- **Configuración propia:** el mapa de teclas vive en `~/.config/localcli/keys.json`, las preferencias del usuario —último agente, presupuesto de historial y razonamiento— en `~/.config/localcli/config.json` y la paleta visual —colores de texto y fondos— en `~/.config/localcli/styles.json`, fuera del proyecto, porque son preferencia del usuario y no contenido del proyecto. El modelo activo no se guarda en `config.json`: lo resuelve el catálogo (`models.json`), única autoridad al arrancar.
- **Paleta editable:** `styles.json` es un mapa `clave: valor` de tokens de color (`color_*` de texto, `fondo_*` de fondo) a hex `#RRGGBB` o índice ANSI `0..255`; los valores ausentes conservan su valor de fábrica y un token desconocido o un color mal formado se rechazan enteros, arrancando con la paleta de fábrica. El diff tiene tokens propios (`color_diferencia_mas` en verde y `color_diferencia_menos` en rojo), de modo que no hereda el color del agente ni el del error. El archivo se siembra con los valores por defecto en el primer arranque si no existe.

## 4. Decisiones

| Decisión                                                | Por qué                                                                                                                                          | Alternativa descartada                                                                     |
| ------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------ |
| Presentación pura, sin lógica de negocio                | Todo lo que la TUI decidiera habría que auditarlo dos veces: en el módulo y en la pantalla                                                       | Repartir decisiones entre pantalla y orquestador                                          |
| Estado de vista en memoria, nada persistido por la TUI  | Lo que vale está en SQLite o en archivos; lo que muere con el proceso es solo vista (scroll, plegado, foco)                                      | Persistir el estado de vista, que añadiría una cuarta fuente de verdad                     |
| Mapa de teclas en `~/.config/localcli/keys.json`        | El atajo es del usuario, no del proyecto; ubicación XDG estándar, editable a mano o desde la ayuda                                               | Guardarlo dentro del proyecto, que mezclaría preferencia personal con contenido versionado |
| Paleta visual en `~/.config/localcli/styles.json`       | Los colores de texto y fondos son preferencia del usuario; un mapa `clave: valor` de tokens se edita a mano sin recompilar, y los estilos se reconstruyen al arrancar | Cablear los colores en el código, que obliga a recompilar para cada ajuste visual          |
| Preferencias en `~/.config/localcli/config.json`        | El último agente, el presupuesto de historial y el razonamiento son preferencia del usuario, no estado del proyecto; el modelo activo lo persiste el catálogo (`models.json`)                                                          | Persistirlos en SQLite, que los ataría a un solo proyecto                                  |
| Ratón capturado en la TUI                               | Permite seleccionar y copiar texto con el ratón y desplazar con la rueda; la selección se acota a su zona de origen, se ancla al texto y, en el chat, se prolonga fuera del viewport al llegar a un borde (el historial se desplaza solo mientras se arrastra) | No capturarlo, que deja a la app sin poder copiar al instante |
| Paleta de comandos alimentada por el catálogo del orquestador | Un solo origen de verdad: los flujos oficiales (sembrados en `~/.config/localcli/flows/`) y los propios de `<proyecto>/.localcli/flows/` se descubren sin mantener una segunda lista en la vista; **overlay con máximo 5 elementos y scroll** | Cablear los comandos en la TUI, que se desincronizaría de los flujos reales                |

## Mapa de Navegación

Guía de lectura para la IA según la tarea:

- Componentes y límites → [[frontend/01-domain/DOMAIN]]
- Eventos, teclas y peticiones → [[frontend/02-interfaces/INTERFACES]]
- Pruebas → [[frontend/05-quality/TESTING]]
- Disposición y zonas funcionales → [[specs/SPEC-INTERFAZ]]
- Atajos y panel de aprobaciones → [[specs/SPEC-INTERFAZ-ATAJOS]]

## Referencias

- [[PROJECT]] — stack aprobado (Bubble Tea + Lip Gloss).
- [[backend/BACKEND]] — el orquestador que esta capa pinta.
- [[backend/01-domain/DOMAIN]] — límites de `tui` como módulo del orquestador.
