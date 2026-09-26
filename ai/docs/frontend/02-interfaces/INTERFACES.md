---
title: LocalCli — entradas y salidas de la TUI
tags: [frontend, interfaces]
depende_de:
  - "[[frontend/FRONTEND]]"
  - "[[backend/04-infrastructure/EVENTS]]"
  - "[[specs/SPEC-INTERFAZ]]"
relacionado:
  - "[[specs/SPEC-SESIONES]]"
  - "[[specs/SPEC-INTERFAZ-ATAJOS]]"
  - "[[database/03-operations/QUERIES]]"
---
# INTERFACES — Entradas y salidas de la TUI

La TUI no tiene red ni API: su frontera son dos entradas (teclado y eventos) y dos salidas (peticiones a `session` y lecturas a `store`).

## 1. Eventos que consume

De [[backend/04-infrastructure/EVENTS]] llega cada evento y así reacciona la pantalla:

| Evento | Qué hace la TUI |
|---|---|
| `token` | Añade el fragmento al bloque de razonamiento o a la respuesta, en vivo |
| `estado_sesion` | Actualiza el estado en el selector y en el panel |
| `notificacion` | Marca el aviso de esa sesión aunque no sea la activa |
| `peticion_aprobacion` | Añade la línea al panel de aprobaciones y actualiza el contador |
| `aprobacion_resuelta` | Retira o marca la línea y actualiza el contador |
| `etapa_iniciada` / `terminada` / `fallida` | Actualiza el estado de la sesión y el historial del chat |
| `flujo_pausado` / `reanudado` / `cancelado` | Refleja el estado del flujo |
| `cola_actualizada` / `elemento_bloqueado` | Actualiza la zona de capa y cola del panel |
| `cambio_aplicado` | Refresca el dato de git en el panel |
| `contexto_auditado` | Queda disponible para consulta; no se pinta por defecto |

## 2. Peticiones que envía

Todas van a `session`, la única puerta del motor:

| Petición | Cuándo |
|---|---|
| Enviar mensaje | El usuario escribe y confirma, tanto en la bienvenida (primera petición) como en el chat |
| Crear o retomar sesión | Al enviar desde la bienvenida: la sesión activa se retoma si existe o se crea nueva, y luego va el mensaje |
| Cambiar de sesión | Elige en el selector momentáneo |
| Aprobar / declinar | Resuelve una línea del panel de aprobaciones |
| Cancelar flujo | Lo pide con su atajo; si había flujo en marcha, `session` pregunta qué hacer, según [[specs/SPEC-SESIONES]] |

## 3. Lecturas a `store`

Solo lectura, con las consultas de [[database/03-operations/QUERIES]]: historial de la sesión activa, aprobaciones pendientes de todas las sesiones, auditoría de una etapa y datos del panel. Nunca escribe: si algo cambia, es el motor quien lo persiste y notifica.

## 4. Teclado

Atajos por defecto, reasignables desde la ayuda y guardados en `~/.config/localcli/keys.json`. El teclado pasa por un resolver central (`KeyResolver`, ver [[specs/SPEC-KEYBINDS]]): acciones con ID estable, múltiples bindings por acción, tecla líder `Ctrl+X` con timeout 2000 ms y resolución por contexto (modal → input → vista → global). Los componentes reciben acciones, nunca teclas:

| Atajo | Acción | Contexto |
|---|---|---|
| `Ctrl+C` | Salir (`app_exit`) | global, también con modales abiertos |
| `Ctrl+X m` | Abrir el modal de modelos (`model_picker`) | global |
| `Ctrl+X l` | Abrir el modal de sesiones; al elegir una con `Enter` se abre esa sesión (`session_picker`) | global |
| `Ctrl+P` | Abrir el modal con la lista de atajos existentes (`command_palette`) | global |
| `Tab` | Cambiar de agente: `plan` ↔ `build` (`agent_cycle`); el agente activo se pinta a la izquierda del input | vista y bienvenida (no con modal abierto) |
| `Esc` | Cerrar cualquier modal (`dismiss`) | modal |
| `↑` / `↓` | Navegar la lista del modal abierto | modal |
| `Enter` | Aplicar lo resaltado en el modal y cerrarlo | modal |
| `Enter` | Enviar la petición | input |
| `Ctrl+D` | Abrir o cerrar el panel de datos | vista |
| `Ctrl+R` | Mostrar u ocultar el razonamiento | vista |
| `Ctrl+A` | Abrir el panel de aprobaciones | vista |
| `Ctrl+F` | Cancelar el flujo en curso (pide confirmación) | vista |
| `a` / `d` | Aprobar / declinar la línea seleccionada | panel de aprobaciones |

No hay acción «sesión nueva»: las sesiones se crean enviando la primera petición desde la bienvenida. No hay ayuda por `?`: el listado de atajos es el modal de `Ctrl+P`.

Reglas, según [[specs/SPEC-INTERFAZ-ATAJOS]] y [[specs/SPEC-KEYBINDS]]:

- Un atajo no puede quedar asignado a dos acciones; el duplicado se rechaza al guardar.
- Una acción admite varios atajos y puede deshabilitarse con lista vacía en keys.json.
- Con un modal abierto, sus teclas (flechas/enter/esc) no llegan a la vista de abajo; con el input enfocado, las letras sueltas escriben y no activan acciones.
- Tras pulsar la líder sin segunda tecla, el estado vuelve a NORMAL al expirar el timeout.
- Reasignar no cambia reglas de permiso, solo la forma de invocar.
- El cambio se guarda sin reiniciar la aplicación.

## 5. Estados de espera

- Si la sesión activa está generando, la entrada sigue activa: escribir no bloquea ni cancela nada.
- Si el usuario cierra una sesión con un flujo en marcha, la TUI muestra la pregunta de qué hacer con el flujo; la decisión la aplica `session`.
- Mientras una sesión espera permiso, su estado se ve en selector, panel y, si procede, en la línea de aviso.

## Referencias

- [[frontend/FRONTEND]] — mapa de la capa.
- [[backend/04-infrastructure/EVENTS]] — productores y payloads.
- [[database/03-operations/QUERIES]] — las consultas de lectura que puede ejecutar.
- [[specs/SPEC-INTERFAZ-ATAJOS]] — reglas funcionales de los atajos.
