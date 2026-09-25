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

Atajos por defecto, reasignables desde la ayuda y guardados en `~/.config/localcli/keys.json`. Propuesta inicial (los atajos de panel y selector no existen en la bienvenida; allí solo escriben, envían y salen):

| Atajo | Acción |
|---|---|
| `Ctrl+D` | Abrir o cerrar el panel de datos |
| `Ctrl+S` | Abrir el selector de sesiones |
| `Ctrl+R` | Mostrar u ocultar el razonamiento |
| `Ctrl+A` | Abrir el panel de aprobaciones |
| `Ctrl+F` | Cancelar el flujo en curso (pide confirmación) |
| `?` | Ayuda de atajos |
| `Ctrl+Q` | Salir |

En el panel de aprobaciones: `a` aprueba y `d` declina la línea seleccionada.

Reglas, según [[specs/SPEC-INTERFAZ-ATAJOS]]:

- Un atajo no puede quedar asignado a dos acciones; el duplicado se rechaza al guardar.
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
