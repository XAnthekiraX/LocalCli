---
title: SPEC — Keymap central, leader key y resolución de teclas
tags: [specs, requisito]
depende_de:
  - "[[IDEA]]"
  - "[[specs/SPEC-INTERFAZ]]"
  - "[[specs/SPEC-INTERFAZ-ATAJOS]]"
---
# SPEC — Keymap central, leader key y resolución de teclas

Prioridad: P1 (infraestructura de la interfaz)

## Propósito

Centralizar la traducción de pulsaciones de teclado a acciones de la aplicación. Los componentes no interpretan teclas: reciben acciones ya resueltas. Esto permite secuencias con tecla líder, múltiples atajos por acción y contextos de foco sin if dispersos en cada vista.

## Alcance

Incluye el identificador de acciones, el formato de los bindings, la tecla líder con su timeout, la resolución por contexto y el archivo de configuración.
No incluye las reglas de cuándo se pide permiso ([[specs/SPEC-ARCHIVOS]]) ni la disposición de pantalla ([[specs/SPEC-INTERFAZ]]). Las reglas generales de atajos y el panel de aprobaciones siguen en [[specs/SPEC-INTERFAZ-ATAJOS]]; esta spec define el mecanismo que las soporta.

## Actores

- **Usuario**: pulsa teclas; configura líder, timeout y bindings.
- **Componentes de la interfaz**: consumen acciones resueltas; nunca teclas crudas.

## Flujo principal

1. La librería de TUI entrega una pulsación al resolver de teclas.
2. El resolver determina el contexto activo (qué componente tiene el foco).
3. Si está esperando la segunda tecla de una secuencia líder, la combina; si completa un binding, emite la acción.
4. Si la pulsación coincide con un binding del contexto, emite la acción.
5. Si no coincide con nada y no hay secuencia en curso, la pulsación se entrega tal cual al componente con el foco (por ejemplo, para escribir texto).
6. El componente recibe la acción y reacciona. No sabe qué tecla la produjo.

## Modelo de datos

### Acción

Cada acción tiene un identificador estable, usado como clave en la configuración:

| ID | Qué hace | Atajo(s) por defecto | Contexto |
|---|---|---|---|
| `app_exit` | Salir de la aplicación | `ctrl+c` | global (siempre; también con modales abiertos) |
| `model_picker` | Abrir el modal de modelos | `<leader>m` | global |
| `session_picker` | Abrir el modal de sesiones del proyecto | `<leader>l` | global |
| `session_new` | Crear una sesión nueva y dejarla activa | `<leader>n` | vista principal |
| `session_delete` | Eliminar la sesión resaltada en el modal | `ctrl+d` | modal de sesiones |
| `command_palette` | Abrir el modal de atajos de teclado | `ctrl+p` | global |
| `agent_cycle` | Cambiar de agente (`plan` ↔ `build`) | `tab` | vista principal y bienvenida (nunca dentro de un modal) |
| `panel_toggle` | Abrir o cerrar el panel de datos | `ctrl+d` | vista |
| `reasoning_toggle` | Mostrar u ocultar el razonamiento | `ctrl+r` | vista |
| `approvals_toggle` | Abrir o cerrar el panel de aprobaciones | `ctrl+a` | vista |
| `cancel` | Cancelar el trabajo en curso | `ctrl+f` | vista |
| `send` | Enviar la petición | `enter` | input |
| `dismiss` | Cerrar cualquier modal abierto | `esc` | modal (y cancela la espera de líder) |
| `approve` / `decline` | Aprobar o declinar la línea seleccionada | `a` / `d` | panel de aprobaciones |
| `up` / `down` | Navegar la lista del modal | `↑` / `↓` | modal |
| `confirm` | Aplicar lo resaltado en el modal y cerrarlo | `enter` | modal |

`session_new` (`<leader>n`) crea una sesión nueva desde la vista principal y la deja activa; también se crea una enviando la primera petición desde la bienvenida ([[specs/SPEC-SESIONES]]). La ayuda clásica (`?`) queda sustituida por el modal de atajos (`command_palette`).

`ctrl+d` pertenece a dos acciones **en ámbitos distintos**: `panel_toggle` en la vista y `session_delete` en el modal de sesiones. No se pisan porque, con un modal abierto, la vista de abajo no recibe teclas; el duplicado solo se rechaza dentro del mismo ámbito.

Modal abierto ⇒ solo responden sus teclas (`up`/`down`/`confirm`/`dismiss`) más `app_exit`; el modal de sesiones añade `session_delete` (`ctrl+d`). Ninguna alcanza la vista de abajo. `tab` tampoco cicla agentes mientras hay un modal abierto.

La lista es abierta: añadir una acción añade su ID al mapa por defecto.

### Binding

- Una acción puede tener cero, uno o varios atajos. Cero significa deshabilitada.
- Un atajo es una tecla simple (`ctrl+p`, `enter`, `esc`) o una secuencia con prefijo de líder (`<leader>n`).
- La notación `<leader>` se expande con la tecla líder configurada.
- El mismo literal no puede estar asignado a dos acciones en un mismo contexto; el duplicado se rechaza al cargar y al guardar.

## Tecla líder

- Por defecto `ctrl+x`.
- Al pulsarla el resolver entra en estado `LEADER`: muestra un indicador discreto («lider ») en la barra de estado y espera la siguiente tecla.
- En estado `LEADER`, la siguiente tecla se combina: `<leader>m` → `model_picker`, `<leader>l` → `session_picker`, `<leader>n` → `session_new`. Esas tres son todas las secuencias con líder; el resto de acciones usa atajos directos (ver tabla).
- Timeout: si no llega otra tecla en el tiempo configurado (por defecto 2000 ms), el resolver vuelve a `NORMAL` sin emitir acción. La ventana de timeout arranca al pulsar la líder.
- Pulsar la líder otra vez mientras espera cancela la secuencia pendiente y vuelve a esperar desde cero.
- `esc` durante la secuencia cancela la espera y vuelve a `NORMAL`.
- Si la segunda tecla no forma ningún binding válido, la secuencia se descarta y se vuelve a `NORMAL`; ninguna tecla se pierde hacia el editor (la líder y su intento de secuencia no se escriben).

## Estado explícito del resolver

```
NORMAL
   │
   │ ctrl+x (líder)
   ▼
LEADER ──── esc / timeout (2000 ms) ────► NORMAL
   │
   ├── m → model_picker    (modal de modelos)
   ├── l → session_picker  (modal de sesiones)
   └── n → session_new     (sesión nueva)
```

No hay más estados: la complejidad vive en el mapa de bindings, no en la máquina de estados.

## Resolución por contexto

Primero se determina qué componente tiene el foco; solo después se buscan los bindings:

```
Modal abierto          → bindings del modal (navegar, confirmar, dismiss)
   ↓ (no coincide)
Input activo           → el texto manda; solo pasan los bindings globales marcados como «siempre» (p. ej. send)
   ↓ (no coincide)
Vista principal        → bindings normales de la vista
   ↓ (no coincide)
Binding global         → líder y acciones globales
   ↓ (no coincide)
Se entrega la tecla cruda al componente con el foco
```

Reglas:

- Mientras un modal tiene el foco, sus flechas/enter/esc no llegan a la vista de abajo.
- Mientras el input tiene el foco y el usuario escribe, las letras sueltas no activan acciones (por eso los atajos de fábrica usan modificadores o la líder).
- La líder funciona desde cualquier contexto: es el escape hatch cuando un atajo choca con la escritura.

## Configuración

Vive en `~/.config/localcli/keys.json` (preferencia del usuario, fuera del proyecto):

```json
{
  "leader": "ctrl+x",
  "leader_timeout_ms": 2000,
  "keybinds": {
    "app_exit": ["ctrl+c"],
    "model_picker": ["<leader>m"],
    "session_picker": ["<leader>l"],
    "session_new": ["<leader>n"],
    "session_delete": ["ctrl+d"],
    "command_palette": ["ctrl+p"],
    "agent_cycle": ["tab"],
    "panel_toggle": ["ctrl+d"],
    "reasoning_toggle": ["ctrl+r"],
    "approvals_toggle": ["ctrl+a"],
    "cancel": ["ctrl+f"],
    "send": ["enter"],
    "dismiss": ["esc"]
  }
}
```

- Cada valor es una lista de literales; una lista vacía deshabilita la acción.
- Un archivo incompleto nunca deja una acción sin valor de fábrica: lo ausente se rellena con los defaults.
- Lo inválido (ID desconocido, literal vacío, duplicado en el mismo contexto) se rechaza al cargar; la aplicación arranca con los valores de fábrica y avisa.
- Reasignar desde la ayuda escribe el archivo y surte efecto sin reiniciar.

## Reglas de negocio

- Los componentes no comparan strings de tecla: solo consumen acciones.
- Un literal no puede pertenecer a dos acciones en el mismo contexto.
- La líder no es reasignable a una acción normal: siempre inicia secuencia.
- Ninguna pulsación produce dos acciones; la primera coincidencia gana.
- El timeout es configurable entre 500 y 10000 ms.
- Añadir un binding nuevo no requiere tocar el código de las vistas.

## Criterios de aceptación

- [ ] `ctrl+x` seguido de `m` abre el modal de modelos y `ctrl+x` seguido de `l` abre el modal de sesiones; `ctrl+x` sola no abre nada.
- [ ] `ctrl+p` abre el modal con la lista de atajos existentes (acción + tecla).
- [ ] `esc` cierra cualquier modal abierto desde cualquiera de los tres, sin cambiar nada.
- [ ] `tab` alterna el agente entre `plan` y `build` en bienvenida y vista principal; con un modal abierto no cicla.
- [ ] `ctrl+x n` crea una sesión nueva en la vista principal y la deja activa.
- [ ] `ctrl+d` con el modal de sesiones abierto elimina la sesión resaltada; si está trabajando, pide confirmación.
- [ ] Pasado el timeout sin segunda tecla, el resolver vuelve a `NORMAL` y el indicador desaparece.
- [ ] `esc` durante una secuencia líder la cancela sin emitir acción.
- [ ] Una acción admite varios atajos y admite deshabilitarse con lista vacía.
- [ ] Un duplicado en el mismo contexto se rechaza al cargar/guardar con mensaje claro.
- [ ] Con un modal abierto, sus teclas no alcanzan la vista de abajo.
- [ ] Con el input enfocado, escribir palabras no dispara acciones.
- [ ] La configuración persiste entre ejecuciones y los cambios surten efecto sin reiniciar.
- [ ] Ningún componente de la TUI contiene comparaciones directas contra teclas de acción.

## Requisitos no funcionales

- Resolver una pulsación debe ser inmediato (sin esperas visibles) salvo la propia ventana de la líder.
- El timeout se implementa con temporizador del framework, no con bloqueos.

## Dependencias funcionales

- [[specs/SPEC-INTERFAZ]] — usa los modales de modelos, sesiones y atajos.
- [[specs/SPEC-INTERFAZ-ATAJOS]] — reglas de reasignación y panel de aprobaciones.

## Supuestos

- Bubble Tea entrega las combinaciones Ctrl+letra como teclas únicas, lo que hace viable la líder sin ambigüedad.
- Las secuencias de tres teclas no están en el alcance inicial; solo líder + una tecla.

## Referencias

- [[IDEA]]
