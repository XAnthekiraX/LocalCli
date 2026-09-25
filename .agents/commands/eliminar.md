---
description: Eliminar una funcionalidad existente del proyecto. Analiza impacto, actualiza documentación y crea tareas de eliminación con acción eliminar.
---

# Comando: Eliminar Funcionalidad

**Entrada del usuario:** `$ARGUMENTS`

Eres un orquestador que analiza la eliminación solicitada, actualiza la documentación y define las tareas necesarias para retirar la funcionalidad de forma segura.

---

## Fase 0: Contexto del Proyecto

1. Lee `ai/docs/PROJECT.md` si existe. Si no existe, informa al usuario que primero debe ejecutar `/project-planner` o crear el proyecto.
2. Identifica las tecnologías, arquitectura y stack del proyecto.
3. Busca las especificaciones existentes en `ai/docs/specs/`.
4. Identifica la documentación afectada en `ai/docs/database/`, `ai/docs/backend/` y `ai/docs/frontend/`.
5. Identifica las tareas principales de `MAIN-TASKS.md` y sus archivos `NNN-task-<nombre>.md`.

---

## Fase 1: Análisis de Impacto

1. Analiza `$ARGUMENTS`.
2. Identifica las especificaciones y funcionalidades que se eliminarán.
3. Lee esas especificaciones y la documentación relacionada.
4. Busca referencias en todo el proyecto:
   - Dependencias de funcionalidades.
   - Endpoints y llamadas que la consumen.
   - Componentes y rutas que la usan.
   - Datos, relaciones y migraciones relacionadas.
5. Identifica el alcance de la eliminación:
   - Entidades, campos y relaciones afectados.
   - Endpoints, servicios, componentes, estados y rutas afectados.
   - Datos que se perderán o deberán migrarse.
   - Dependencias que deben retirarse.
6. Presenta al usuario las advertencias, funcionalidades afectadas, datos, riesgos y conflictos.
7. Espera confirmación explícita antes de continuar.

---

## Fase 2: Plan de Eliminación

Después de la confirmación, genera un plan detallado para:

### Base de datos
- Tablas, columnas, relaciones, índices y restricciones que se eliminan.
- Migraciones necesarias.
- Datos que deben preservarse o migrarse.

### Backend
- Endpoints, servicios, repositorios, DTOs, validaciones y reglas que se eliminan.
- Dependencias que pueden retirarse.

### Frontend
- Componentes, páginas, estados, flujos, llamadas API y rutas que se eliminan.
- Dependencias que pueden retirarse.

Presenta el plan y espera aprobación.

---

## Fase 3: Documentación

Después de aprobar el plan, actualiza la documentación antes de crear las tareas. La documentación es la fuente de verdad.

- Elimina o actualiza la especificación afectada según corresponda.
- Actualiza los documentos de database, backend y frontend afectados.
- Actualiza referencias cruzadas y no dejes enlaces rotos.
- Actualiza `ai/docs/PROJECT.md` si la funcionalidad formaba parte de su alcance.
- Actualiza un archivo a la vez y pide aprobación antes del siguiente.

---

## Fase 4: Tareas de eliminación

Con la documentación actualizada, modifica `ai/tasks/backend/MAIN-TASKS.md` y/o `ai/tasks/frontend/MAIN-TASKS.md`.

### Reglas de estructura
- `MAIN-TASKS.md` contiene solo tareas principales, acción, estado, dependencias y referencia al archivo de detalle.
- No agregues tablas, listas ni detalles de subtareas en `MAIN-TASKS.md`.
- La descomposición vive únicamente en `ai/tasks/<capa>/NNN-task-<nombre>.md`.
- Cada tarea principal debe referenciar su archivo de detalle.
- Si la eliminación afecta una tarea existente, actualiza esa tarea en lugar de crear otra equivalente.

### Acción de las subtareas
- Todas las subtareas nuevas o modificadas por este comando deben llevar `Acción: eliminar`, incluso si se agrega una fila a un archivo de detalle existente.
- Nunca uses `Acción: crear` ni `Acción: actualizar` para una subtarea producida por `/eliminar`.
- Las subtareas no afectadas conservan su acción y estado.
- No modifiques tareas ya completadas salvo petición explícita.
- `Acción: eliminar` describe el efecto de la implementación; no significa eliminar físicamente el archivo de planificación.

### Backend
- Si el usuario indica `T-B00X`, actualiza esa fila y `ai/tasks/backend/NNN-task-<nombre>.md`.
- Si no hay tarea objetivo, crea una tarea principal y su archivo de detalle cuando el cambio no pueda pertenecer a una tarea existente.
- Si creas una nueva tarea principal por este comando, su acción debe ser `eliminar`.

### Frontend
- Si el usuario indica `T-F00X`, actualiza esa fila y `ai/tasks/frontend/NNN-task-<nombre>.md`.
- Si no hay tarea objetivo, crea una tarea principal y su archivo de detalle cuando el cambio no pueda pertenecer a una tarea existente.
- Si creas una nueva tarea principal por este comando, su acción debe ser `eliminar`.

---

## Reglas Duras

1. Nunca ejecutes código. Solo documenta y define tareas.
2. Nunca elimines funcionalidades sin confirmación explícita.
3. Siempre actualiza la documentación antes de crear o actualizar tareas.
4. Siempre pide aprobación en cada fase.
5. Siempre usa la documentación como fuente de verdad.
6. Nunca dejes referencias rotas en la documentación.
7. Nunca saltes fases.
8. Nunca modifiques tareas ya completadas sin autorización explícita.
9. Advierte sobre dependencias antes de eliminar.
10. Preserva los datos que puedan ser necesarios.
11. Toda subtarea nueva o modificada por este comando debe decir `Acción: eliminar`.

---

## Output Final

```
⚠️ Funcionalidad "[nombre]" marcada para eliminación.

📁 Documentación actualizada:
  - ai/docs/specs/NNN-[nombre]/SPEC.md (eliminado o actualizado)
  - ai/docs/database/... (si aplica)
  - ai/docs/backend/... (si aplica)
  - ai/docs/frontend/... (si aplica)

📋 Tareas creadas o actualizadas:
  - ai/tasks/backend/MAIN-TASKS.md (si aplica)
  - ai/tasks/frontend/MAIN-TASKS.md (si aplica)
  - Archivos NNN-task-*.md con acción eliminar

⚠️ Dependencias afectadas:
  - [lista de funcionalidades afectadas]

Para implementar la eliminación, ejecuta:
  - /ejecutar-backend
  - /ejecutar-frontend
```
