# DOMAIN — Componentes de la TUI

Cada componente tiene una responsabilidad y un límite. Ninguno contiene reglas de negocio: si algo hay que decidir, la decisión pertenece al motor y el componente solo la pinta o la pregunta al usuario.

## 1. Componentes

| Componente | Responsabilidad | Lo que no hace |
|---|---|---|
| `app` | Modelo raíz: reparte los eventos entre componentes, mantiene qué vista está activa (bienvenida o principal) y decide la transición | No conoce el detalle de cada componente; solo enruta |
| `welcome` | Pantalla de bienvenida: logotipo ASCII con nombre y versión, y una línea de entrada para la primera petición | No crea sesiones ni valida nada: envía lo escrito como petición y la vista cambia a la principal |
| `chat` | Muestra el historial de la sesión activa: razonamiento y respuesta de cada intercambio | No gestiona sesiones; pinta lo que le llega de la activa |
| `input` | Una línea de texto; compone y envía la petición hacia la sesión activa | No valida reglas de negocio |
| `panel` | Los nueve datos del panel, plegable a la derecha | Es de lectura: nada se escribe desde él |
| `sessions` | Selector momentáneo con nombre y estado de cada sesión del proyecto | No mantiene una lista permanente: aparece y desaparece |
| `approvals` | Panel de aprobaciones pendientes de todas las sesiones, resolvibles una a una | No decide: envía la decisión del usuario |
| `notify` | Línea discreta con el número de aprobaciones pendientes, visible con el panel cerrado | Es el único dato que se muestra fuera del panel |
| `keys` | Mapa de teclas con sus valores por defecto y su reasignación | Un atajo no cambia ninguna regla de permiso |

## 2. Reglas de presentación

- El razonamiento se muestra en vivo, arriba de la respuesta, distinguible visualmente y ocultable sin detener la generación.
- El panel refleja siempre los datos de la sesión activa, no los de otra.
- Con el panel cerrado, el contador de aprobaciones pendientes sigue visible y se actualiza con cada evento.
- Los estados que se pintan son los de [[database/01-schema/ENUMS]]; la TUI no inventa estados ni transiciones.
- La estimación de tokens se marca como estimación cuando lo es. Ver [[specs/SPEC-PANEL-CONTEXTO]].
- Cambiar de sesión no interrumpe ninguna ejecución: solo cambia lo que se pinta.

## 3. Reglas de la bienvenida

- Es la primera vista al ejecutar `localcli`. Solo hay logotipo, nombre con versión y una línea de entrada; sin paneles, selector ni aprobaciones.
- Lo escrito es la primera petición: se envía a la sesión activa (se retoma si existe, se crea si no) y la vista cambia a la principal, donde aparece como primer mensaje del chat. La transición no repite la petición ni pide confirmación.
- Se pinta sin esperar a Ollama ni a la base: no depende de nada externo. La única salida desde ella es `Ctrl+C`.
- El logotipo es un arte ASCII fijo de la aplicación, no contenido de sesión: vive en el código de la TUI y no entra al contexto del modelo.

## 4. Lo que el frontend nunca hace

- No llama a Ollama, no abre la base para escribir, no toca archivos del proyecto.
- No comprueba permisos ni aprueba nada por su cuenta: pinta la petición y envía la decisión del usuario.
- No calcula el contexto ni interpreta el TODO: recibe los datos hechos por el motor.

## Referencias

- [[frontend/FRONTEND]] — mapa de la capa.
- [[specs/SPEC-INTERFAZ]] — qué dato vive en cada zona, incluida la bienvenida.
- [[backend/04-infrastructure/EVENTS]] — los eventos que consumen estos componentes.
