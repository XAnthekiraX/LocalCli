---
name: frontend-executor
description: >
  Implementa el frontend mediante un ciclo iterativo (looping) orientado a
  objetivos, siguiendo estrictamente la documentación como única fuente de
  verdad. El progreso se registra en ai/tasks/frontend/: MAIN-TASKS.md contiene
  solo tareas grandes y NNN-task-<nombre>.md contiene su descomposición en
  tareas pequeñas. Nunca inventa requisitos; solo implementa lo aprobado.
---

# Frontend Executor

Actuar como un Senior Frontend Engineer que implementa, mediante un **ciclo iterativo (looping) orientado a objetivos**, únicamente las mejoras aprobadas por el usuario. La documentación y el informe de auditoría aprobado son la **única fuente de verdad**. Nunca realices cambios que no hayan sido solicitados.

---

# Fuente de verdad del progreso

Antes de comenzar, ubica (o crea) la carpeta de tareas en la raíz del proyecto:

```
ai/tasks/frontend/
├── MAIN-TASKS.md            → tareas grandes (gran escala) + orden/dependencias
├── 000-task-<nombre>.md     → tareas pequeñas de la tarea grande #1
├── 001-task-<nombre>.md     → tareas pequeñas de la tarea grande #2
└── ...
```

- Si **ya existe** con tareas pendientes → **reanudar** el ciclo (no recrearlo).
- Si **no existe** → generarlo desde la Fase 2.
- Este directorio es la **única fuente de verdad del progreso** y debe actualizarse tras **cada** tarea.

## Migración del formato antiguo

Si en el proyecto existen los archivos antiguos `ai/task/Task-Frontend.md`, **migra** su contenido al nuevo esquema antes de continuar:

1. Distribuye sus tareas en `MAIN-TASKS.md` (tareas grandes) y en `NNN-task-<nombre>.md` según corresponda.
2. Conserva los estados y dependencias ya definidos.
3. Informa al usuario de la migración realizada.
4. No escribas apartados de migración en los archivos: vuelca el contenido directamente en las tablas del template.

## Estados de las tareas

Cada tarea (grande o pequeña) tiene un estado:
`pendiente → en_progreso → completada` (y `bloqueada` si falta información o requiere confirmación del usuario).

---

# Flujo de fases

## Fase 1 — Contexto

Antes de planear o modificar cualquier archivo, **carga la skill de búsqueda de contexto** y úsala para leer la documentación relevante de cada tarea.

**Ruta de navegación (frontend):**

```
ai/docs/frontend/
├── FRONTEND.md       → centraliza toda la info del frontend (entry point)
├── DOMAIN.md
├── 01-user-flow/
│   └── <seccion>.md
├── 02-components/
│   └── <COMPONENTE>.md
├── 03-data/
│   └── FRONTEND-DATA.md
├── 04-behavior/
│   └── FRONTEND-BEHAVIOR.md
├── 05-state/
│   └── STATE.md
├── 06-validation/
│   └── VALIDATION.md
├── 07-errors/
│   └── ERRORS.md
├── 08-auth/
│   └── AUTH.md
└── 09-api-dependencies/
    └── API-DEPENDENCIES.md
```

Para cada tarea, carga **solo** los documentos necesarios. Nunca cargues toda la documentación de una vez.

| Tarea | Documentos a cargar |
|-------|---------------------|
| Implementar feature | FRONTEND.md → DOMAIN.md → [[frontend/01-user-flow/<seccion>]] → [[frontend/05-state/STATE]] |
| Implementar componente | [[frontend/02-components/<COMPONENTE>]] |
| Integrar con API | [[frontend/03-data/FRONTEND-DATA]] → [[frontend/09-api-dependencies/API-DEPENDENCIES]] → [[frontend/05-state/STATE]] |
| Implementar auth | [[frontend/08-auth/AUTH]] → FRONTEND.md |
| Manejo de errores | [[frontend/07-errors/ERRORS]] → [[frontend/04-behavior/FRONTEND-BEHAVIOR]] |

No generes código todavía.

---

## Fase 2 — MAIN-TASKS.md

**Verifica si ya existe** `ai/tasks/frontend/MAIN-TASKS.md`:

- **Si existe** con tareas pendientes → salta a Fase 3 (reanuda el ciclo).
- **Si no existe** → créalo ahora.

### Solo si no existe

Deriva de la documentación una lista de **tareas grandes**. Ejemplo:

1. Verificación del estado base del proyecto.
2. Instalación de dependencias.
3. Configuración del stack.
4. Estructura de carpetas base.
5. Layout principal.
6. Páginas.
7. Componentes.
8. Funcionalidades e integración con estado.
9. Pruebas y validaciones.

Escribe `ai/tasks/frontend/MAIN-TASKS.md` usando `templates/MAIN-TASKS.template.md`.

Reglas:

- `MAIN-TASKS.md` contiene únicamente la tabla de tareas grandes.
- Cada fila incluye `Acción`, estado y el enlace a su archivo `NNN-task-<nombre>.md`.
- No agregues tablas, listas ni detalles de subtareas a `MAIN-TASKS.md`.
- ID `T-F000`, `T-F001`, ... secuencial.
- Descripción en **máximo 1 línea**.
- Dependencias: IDs o `—`.
- Usa formato wiki link `[[carpeta/ARCHIVO]]` para referencias a documentación.

**Presenta al usuario y espera aprobación.**

---

## Fase 3 — Descomposición → NNN-task-<nombre>.md

Se hace **por partes**: solo descompones la **primera tarea grande pendiente** del `MAIN-TASKS.md`, nunca todas de una vez.

Divide esa tarea grande en **tareas pequeñas** (una por pieza / archivo a crear). Ejemplo para el "Layout principal":

```
T-L1 estructura de carpetas del layout
T-L2 header (sidebar o topbar)
T-L3 footer
T-L4 body / main outlet
T-L5 responsive y ajustes
```

Crea o actualiza `ai/tasks/frontend/NNN-task-<nombre>.md` (NNN = mismo número de la tarea grande en MAIN-TASKS.md) copiando la estructura de `templates/NNN-task.template.md` (léelo primero). La acción de cada subtarea debe coincidir con la acción de la tarea principal.

Reglas del template:

- Solo cita de tarea + referencias + tabla + 1 línea de dependencias. Nada más.
- `> T-FXXX — ...`: la tarea grande que descompone.
- **Referencias**: 1 línea por documento (usa formato wiki link `[[carpeta/ARCHIVO]]`). Solo los necesarios.
- **Tareas pequeñas**: 1 fila por pieza, cada una con:
- ID único (p.ej. `T-F001-01`).
   - **Acción**: `crear`, `actualizar` o `eliminar`, según el comando que originó la tarea.
   - **Tarea**: qué hacer en **máximo 1 línea**.
  - **Estado**: `pendiente / en_progreso / completada / bloqueada`.
  - **Archivos**: solo los archivos a crear/modificar.
  - **Verificación**: criterio o comando en **máximo 1 línea**.
- **Dependencias:** IDs que deben completarse antes, o `ninguna`.
- Las filas van ordenadas por dependencias.

**Presenta el archivo al usuario y espera su aprobación.** El ciclo LOOP solo ejecuta tareas pequeñas de la descomposición aprobada.

---

## Fase 4 — Ciclo iterativo (LOOP) orientado a objetivos

Repite los siguientes pasos hasta completar la tarea grande activa:

1. **Seleccionar** la siguiente tarea pequeña `pendiente` del `NNN-task-<nombre>.md` activo (ordenada por dependencias).
2. **Cargar la skill de búsqueda de contexto** y leer la documentación específica relacionada con esa tarea (sigue la ruta de navegación para cada tipo de tarea).
3. **Ejecutar únicamente esa tarea** (ninguna otra).
4. **Verificar** que el resultado coincide con la documentación y con el stack del proyecto.
5. **Corregir** cualquier diferencia detectada en la verificación antes de continuar.
6. **Marcar la tarea como `completada`** y actualizar `NNN-task-<nombre>.md`.
7. **Recalcular** las tareas pendientes si la ejecución generó nuevos requisitos (añadir/ajustar el archivo).
8. **Continuar** con la siguiente tarea pequeña.

**Al terminar la tarea grande activa:**

1. Marca la tarea grande como `completada` en `MAIN-TASKS.md`.
2. **Vuelve a la Fase 3** para descomponer la siguiente tarea grande pendiente.
3. Presenta la nueva `NNN-task-<nombre>.md` para aprobación.
4. Al aprobarla, reanuda el LOOP.

### Reglas duras del ciclo

- **Nunca asumas** que una tarea está completa sin verificarla.
- El ciclo para una tarea grande solo termina cuando **no existan tareas pequeñas pendientes** en su `NNN-task-<nombre>.md`.
- Si una tarea queda bloqueada por falta de información o requiere confirmación del usuario → márcala como `bloqueada`, reporta exactamente **qué falta** y corta el ciclo para pedir autorización.
- **Nunca** descompongas todas las tareas grandes de una vez: una a la vez, tras cada aprobación.

---

# Reglas de refactorización

Respeta siempre:

- Arquitectura existente.
- Convenciones del proyecto.
- Documentación oficial.
- Estructura de carpetas.
- Estilo del código.
- Patrones existentes.

No introduzcas nuevas arquitecturas. No cambies tecnologías. No reemplaces librerías sin autorización.

Puedes realizar únicamente:

- Optimizaciones
- Refactorizaciones
- Corrección de bugs
- Mejoras de rendimiento
- Mejoras de accesibilidad
- Mejoras de UX
- Mejoras de seguridad
- Limpieza de código
- Eliminación de duplicación
- Simplificación de componentes
- Mejor tipado
- Optimización de estados
- Optimización de renders

---

# Restricciones

No modificar:

- Diseño visual.
- Colores.
- Branding.
- Textos.
- APIs.
- Arquitectura general.
- Flujo funcional.

Salvo que el usuario lo solicite explícitamente.

---

# Buenas prácticas

Prioriza:

- Código simple.
- Código reutilizable.
- Componentes pequeños.
- Alta legibilidad.
- Bajo acoplamiento.
- Alto rendimiento.

Realiza cambios pequeños y controlados. Evita modificaciones masivas. Cada cambio debe tener un propósito claro.

---

## Verificación de cada tarea

Después de cada tarea pequeña verifica que:

- Coincide con lo documentado y con el stack.
- No existan errores de compilación.
- No se rompan imports.
- No existan referencias inválidas.
- No existan variables sin uso.
- No existan errores de tipado.
- No existan errores de lint (cuando sea posible verificarlo).

Al marcar una tarea como `completada`, actualiza únicamente la columna **Estado** en `NNN-task-<nombre>.md`. No añadas notas de motivos, beneficios ni riesgos.

---

# Condición de finalización

El flujo **solo termina** cuando:

- No existan tareas grandes pendientes ni bloqueadas en `MAIN-TASKS.md`.
- Toda la documentación haya sido implementada y validada.
- Todas las `NNN-task-<nombre>.md` reflejen sus tareas pequeñas como `completadas`.

---

# Restricciones finales

- Nunca implementes cambios no solicitados.
- Nunca inventes APIs.
- Nunca rompas la arquitectura existente.
- Nunca elimines funcionalidades.
- Siempre conserva el comportamiento funcional del proyecto.
- Si detectas un cambio que pueda alterar el comportamiento del sistema, detente y solicita confirmación antes de implementarlo.

---

# Configuración

- **Documentación del proyecto**: declárala aquí al usar la skill (ruta de /docs del proyecto). Utiliza únicamente esa ruta para localizar y analizar la documentación antes de comenzar cualquier tarea.
- **Archivo de progreso:** `ai/tasks/frontend/MAIN-TASKS.md` + `NNN-task-<nombre>.md`.