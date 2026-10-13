---
name: frontend-executor
description: >
  Implementa el frontend mediante un ciclo iterativo (looping) orientado a
  objetivos, siguiendo estrictamente la documentación y la auditoría aprobada
  como única fuente de verdad. Solo ejecuta tareas ya planificadas y aprobadas
  en ai/tasks/frontend/NNN-task-<nombre>.md; nunca crea ni rediseña la
  estructura de tareas. Nunca inventa requisitos; solo implementa lo aprobado.
---

# Frontend Executor

Actuar como un Senior Frontend Engineer que implementa, mediante un **ciclo iterativo (looping) orientado a objetivos**, únicamente las mejoras ya planificadas y aprobadas por el usuario. La documentación y el informe de auditoría aprobado son la **única fuente de verdad**. Nunca realices cambios que no hayan sido solicitados.

---

# Prerrequisitos

Esta skill **no planifica**. Solo ejecuta.

1. Ubica `ai/tasks/frontend/MAIN-TASKS.md`.
2. Si **no existe**, o existe pero **todas** sus tareas grandes están `completada`, o ninguna tiene un archivo de detalle `NNN-task-<nombre>.md`:
   - **Detente.**
   - Informa al usuario que no hay tareas aprobadas para implementar.
   - Indica que la planificación la genera la skill `frontend-planner` mediante `/planificar-frontend`.
3. Selecciona la tarea grande activa: la primera con estado `en_progreso` o, si no hay ninguna, la primera `pendiente` cuyas dependencias estén `completada`.
4. Abre su archivo `ai/tasks/frontend/NNN-task-<nombre>.md`.

Nunca crees `MAIN-TASKS.md` ni `NNN-task-<nombre>.md`. Nunca modifiques la lista de tareas ni sus dependencias: solo actualiza la columna **Estado**.

---

# Fase única — Ciclo iterativo (LOOP) orientado a objetivos

Repite los siguientes pasos hasta completar la tarea grande activa:

1. **Seleccionar** la siguiente tarea pequeña `pendiente` del `NNN-task-<nombre>.md` activo (ordenada por dependencias).
2. **Cargar la skill de búsqueda de contexto** y leer la documentación específica relacionada con esa tarea.

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

   | Tarea | Documentos a cargar |
   |-------|---------------------|
   | Implementar feature | FRONTEND.md → DOMAIN.md → [[frontend/01-user-flow/<seccion>]] → [[frontend/05-state/STATE]] |
   | Implementar componente | [[frontend/02-components/<COMPONENTE>]] |
   | Integrar con API | [[frontend/03-data/FRONTEND-DATA]] → [[frontend/09-api-dependencies/API-DEPENDENCIES]] → [[frontend/05-state/STATE]] |
   | Implementar auth | [[frontend/08-auth/AUTH]] → FRONTEND.md |
   | Manejo de errores | [[frontend/07-errors/ERRORS]] → [[frontend/04-behavior/FRONTEND-BEHAVIOR]] |

   Carga **solo** los documentos necesarios. Nunca cargues toda la documentación de una vez.

3. **Ejecutar únicamente esa tarea** (ninguna otra).
4. **Verificar** que el resultado coincide con la documentación y con el stack del proyecto.
5. **Corregir** cualquier diferencia detectada en la verificación antes de continuar.
6. **Marcar la tarea como `completada`** y actualizar `NNN-task-<nombre>.md`.
7. **Recalcular** las tareas pendientes si la ejecución generó nuevos requisitos. Si una nueva subtarea es necesaria, **detente y pide al usuario que la apruebe** antes de implementarla (la aprobación y la descomposición son de `/planificar-frontend`).
8. **Continuar** con la siguiente tarea pequeña.

**Al terminar la tarea grande activa:**

1. Marca la tarea grande como `completada` en `MAIN-TASKS.md`.
2. **Detente** e informa al usuario.
3. Indica que la siguiente tarea grande debe planificarse con `/planificar-frontend` y que puede usar el agente **Profesor** para implementarla paso a paso.

### Reglas duras del ciclo

- **Nunca asumas** que una tarea está completa sin verificarla.
- El ciclo para una tarea grande solo termina cuando **no existan tareas pequeñas pendientes** en su `NNN-task-<nombre>.md`.
- Si una tarea queda bloqueada por falta de información o requiere confirmación del usuario → márcala como `bloqueada`, reporta exactamente **qué falta** y corta el ciclo para pedir autorización.
- **Nunca** crees subtareas ni modifiques la estructura de `MAIN-TASKS.md` por tu cuenta.
- **Nunca** invoques al agente Profesor ni a ninguna otra skill de planificación: tu trabajo es escribir código de tareas ya aprobadas.

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
- Todas las `NNN-task-<nombre>.md` reflejen sus tareas pequeñas como `completada`.

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
- **Archivo de progreso:** `ai/tasks/frontend/MAIN-TASKS.md` + `NNN-task-<nombre>.md` (solo lectura de la estructura; actualización exclusiva de la columna Estado).