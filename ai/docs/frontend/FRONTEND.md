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

La capa de interfaz: una TUI en terminal construida con Bubble Tea y Lip Gloss. Es el único consumidor del motor y su único trabajo es pintar lo que llega y enviar lo que pulsas. No ejecuta nada por su cuenta: no habla con Ollama, no escribe en SQLite y no toca archivos.

Este documento es el mapa de navegación del frontend. El motor está en [[backend/BACKEND]]; los datos, en [[database/DATABASE]].

## 1. Visión General

- **Arquitectura Elm de Bubble Tea:** un modelo de estado, un bucle de eventos (`update`) y una función de pintado (`view`). Un solo bucle; sin goroutines propias más allá de las suscripciones de eventos. Dos vistas que ese bucle pinta: la bienvenida y la interfaz principal.
- **Presentación pura.** El frontend no decide nada de negocio: no comprueba permisos, no recorta contexto, no calcula estados. Recibe eventos y manda peticiones.
- Lo que ve llega por dos caminos: los eventos de [[backend/04-infrastructure/EVENTS]] y las consultas de lectura de [[database/03-operations/QUERIES]]. Lo que hace lo pide a `session`, que es la única puerta del motor.

## 2. Estructura del Proyecto

```
internal/tui/
  app.go        modelo raíz, enrutado de eventos y suscripciones
  welcome.go    pantalla de bienvenida estilo opencode: logotipo sobre el fondo, caja de entrada con línea de estado y pistas de teclado
  chat.go       historial de la sesión activa: respuesta, líneas compactas de herramienta y razonamiento revelable
  input.go      caja de entrada con fondo: línea de texto y pie con el agente y el modelo
  comandos.go   paleta de comandos de flujo: catálogo del motor, filtro y render sobre el input
  panel.go      sidebar de datos plegable: título, contexto, TODO, tareas, estado y pie
               (git, ruta y firma del harness)
  reasoning.go  estado del razonamiento en vivo (texto revelado u oculto)
  approvals.go  panel de aprobaciones pendientes de todas las sesiones
  notify.go     línea de aviso de aprobaciones pendientes con el panel cerrado
  modelsmodal.go modal de modelos (Ctrl+X m): lista de Ollama cargada bajo demanda
  sessionsmodal.go modal de sesiones (Ctrl+X l): al aplicar abre esa sesión
  keysmodal.go  modal de atajos (Ctrl+P): listado de acción + tecla
  keymap.go     mapa central de teclas: acciones, bindings múltiples y tecla líder
  keyresolver.go KeyResolver: líder ctrl+x, timeout, contexto y acciones
  keys.go       persistencia del mapa de teclas en ~/.config/localcli/keys.json
  config.go     preferencias del usuario: último modelo y último agente
  adjuntos.go   tokens de archivo, carpeta o texto pegados o arrastrados ([nombre.ext], [CARPETA N elementos], [PEGADO N líneas]) y su expansión al valor real
  selection.go  selección con el ratón y copia al portapapeles
  styles.go     estilos Lip Gloss y render de bloques
  marco.go      capa común de render: normaliza el marco a Ancho×Alto celdas con fondo (superficie rectangular continua)
  wire.go       Puerto: la frontera de la tui con el motor (eventos y peticiones)
  doc.go        documentación del paquete
  testdata/     salidas doradas: logo.txt (logotipo canónico de la bienvenida) y *.golden
```

## 3. Contratos

- **Entradas:** teclado y eventos del motor. Ver [[frontend/02-interfaces/INTERFACES]].
- **Salidas:** peticiones a `session` (enviar mensaje, cambiar de sesión, aprobar, declinar, cancelar flujo).
- **Lecturas:** solo las consultas definidas en [[database/03-operations/QUERIES]]. Nunca escribe en la base: el estado lo persiste el motor.
- **Configuración propia:** el mapa de teclas vive en `~/.config/localcli/keys.json` y las preferencias del usuario —último modelo y último agente— en `~/.config/localcli/config.json`, fuera del proyecto, porque son preferencia del usuario y no contenido del proyecto.

## 4. Decisiones

| Decisión                                                | Por qué                                                                                                                                          | Alternativa descartada                                                                     |
| ------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------ |
| Presentación pura, sin lógica de negocio                | Todo lo que la TUI decidiera habría que auditarlo dos veces: en el módulo y en la pantalla                                                       | Repartir decisiones entre pantalla y motor                                                 |
| Estado de vista en memoria, nada persistido por la TUI  | Lo que vale está en SQLite o en archivos; lo que muere con el proceso es solo vista (scroll, plegado, foco)                                      | Persistir el estado de vista, que añadiría una cuarta fuente de verdad                     |
| Mapa de teclas en `~/.config/localcli/keys.json`        | El atajo es del usuario, no del proyecto; ubicación XDG estándar, editable a mano o desde la ayuda                                               | Guardarlo dentro del proyecto, que mezclaría preferencia personal con contenido versionado |
| Preferencias en `~/.config/localcli/config.json`        | El último modelo y el último agente son preferencia del usuario, no estado del proyecto                                                          | Persistirlos en SQLite, que los ataría a un solo proyecto                                  |
| Ratón capturado en la TUI                               | Permite seleccionar y copiar texto con el ratón y desplazar con la rueda                                                                         | No capturarlo, que deja a la app sin poder copiar al instante                              |
| Paleta de comandos alimentada por el catálogo del motor | Un solo origen de verdad: los flujos oficiales y los propios de `.localcli/flows/*.json` se descubren sin mantener una segunda lista en la vista | Cablear los comandos en la TUI, que se desincronizaría de los flujos reales                |

## Mapa de Navegación

Guía de lectura para la IA según la tarea:

- Componentes y límites → [[frontend/01-domain/DOMAIN]]
- Eventos, teclas y peticiones → [[frontend/02-interfaces/INTERFACES]]
- Pruebas → [[frontend/05-quality/TESTING]]
- Disposición y zonas funcionales → [[specs/SPEC-INTERFAZ]]
- Atajos y panel de aprobaciones → [[specs/SPEC-INTERFAZ-ATAJOS]]

## Referencias

- [[PROJECT]] — stack aprobado (Bubble Tea + Lip Gloss).
- [[backend/BACKEND]] — el motor que esta capa pinta.
- [[backend/01-domain/DOMAIN]] — límites de `tui` como módulo del motor.
