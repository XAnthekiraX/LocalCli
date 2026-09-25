---
description: Actualizar una funcionalidad existente del proyecto. Analiza impacto, actualiza documentación y actualiza la tarea principal y sus subtareas con acción actualizar.
---

# Comando: Actualizar Funcionalidad Existente

**Entrada del usuario:** `$ARGUMENTS`

Eres un orquestador que integra las skills de documentación y ejecución en un flujo simplificado. Tu objetivo es tomar la actualización solicitada por el usuario, analizar el impacto, actualizar la documentación y actualizar las tareas existentes.

---

## Fase 0: Contexto del Proyecto

1. Lee `ai/docs/PROJECT.md` si existe. Si no existe, informa al usuario que primero debe ejecutar `/project-planner` o crear el proyecto.
2. Identifica las tecnologías, arquitectura y stack del proyecto.
3. Busca todas las SPECs existentes en `ai/docs/specs/` para entender qué funcionalidades ya existen.
4. Identifica qué documentación existe en `ai/docs/database/`, `ai/docs/backend/`, `ai/docs/frontend/`.
5. Identifica las tareas principales de `MAIN-TASKS.md` y sus archivos `NNN-task-<nombre>.md`.

---

## Fase 1: Análisis de Impacto

1. Analiza la entrada del usuario: `$ARGUMENTS`.
2. Determina si el usuario indicó una tarea existente:
   - Busca un ID tipo T-F00X o T-B00X (o variantes como TF007, TB007) en la entrada.
   - Si existe, mapea a la tarea grande correspondiente en `ai/tasks/<layer>/MAIN-TASKS.md` y al archivo de detalle `ai/tasks/<layer>/NNN-task-<nombre>.md`.
   - Si no existe, identifica la tarea existente por nombre o continúa sin asumir una tarea objetivo.
3. Identifica qué SPEC(s) existente(s) se ven afectados.
4. Lee los SPECs afectados para entender la funcionalidad actual.
5. Lee la documentación relevante de DB, backend y frontend.
6. Identifica exactamente qué cambia:
   - ¿Qué entidades/campos se modifican?
   - ¿Qué endpoints cambian?
   - ¿Qué componentes se ven afectados?
   - ¿Qué flujos de usuario cambian?
7. Presenta el análisis de impacto al usuario con un "diff conceptual":
   - Qué documentos se actualizarán
   - Qué archivos de código se modificarán
   - Qué dependencias se rompen o afectan
8. Si hay tarea objetivo, indica qué subtareas existentes se verán afectadas.
9. Espera confirmación del usuario antes de continuar.

---

## Fase 2: Plan de Actualización

Después de la confirmación, genera un plan detallado de cambios:

### Base de Datos
- ¿Qué tablas se modifican?
- ¿Qué columnas se agregan/eliminan/modifican?
- ¿Qué relaciones cambian?
- ¿Qué migraciones se necesitan?
- ¿Hay datos que migrar?

### Backend
- ¿Qué endpoints se modifican?
- ¿Qué services cambian?
- ¿Qué repositories/DAOs se actualizan?
- ¿Qué DTOs cambian?
- ¿Qué validaciones se modifican?
- ¿Qué reglas de negocio cambian?

### Frontend
- ¿Qué componentes se modifican?
- ¿Qué estados cambian?
- ¿Qué llamadas a API se actualizan?
- ¿Qué flujos de usuario cambian?
- ¿Qué dependencias nuevas se necesitan?

Presenta el plan al usuario y espera aprobación.

---

## Fase 3: Documentación

Después de la aprobación del plan, **actualiza la documentación** antes de actualizar las tareas. La documentación es la fuente de verdad.

### 3.1 Actualizar Especificación
1. Actualiza el `ai/docs/specs/NNN-<NombreFuncionalidad>/SPEC.md` afectado.
2. Agrega las nuevas funcionalidades/reglas/criterios.
3. Mantén consistencia con PROJECT.md.

### 3.2 Documentación de Base de Datos
Actualiza los documentos afectados en `ai/docs/database/`:
- `ai/docs/database/01-schema/TABLES.md`
- `ai/docs/database/01-schema/RELATIONSHIPS.md`
- `ai/docs/database/01-schema/ENUMS.md`
- `ai/docs/database/01-schema/CONSTRAINTS.md`
- `ai/docs/database/02-rules/BUSINESS_RULES.md`
- `ai/docs/database/03-operations/QUERIES.md`

### 3.3 Documentación de Backend
Actualiza los documentos afectados en `ai/docs/backend/`:
- `ai/docs/backend/01-domain/DOMAIN.md`
- `ai/docs/backend/01-domain/BUSINESS_RULES.md`
- `ai/docs/backend/02-api/API.md`
- `ai/docs/backend/02-api/<recurso>.md`
- `ai/docs/backend/02-api/dto/<Recurso>DTO.md`

### 3.4 Documentación de Frontend
Actualiza los documentos afectados en `ai/docs/frontend/`:
- `ai/docs/frontend/DOMAIN.md`
- `ai/docs/frontend/01-user-flow/<seccion>.md`
- `ai/docs/frontend/02-components/<COMPONENTE>.md`
- `ai/docs/frontend/03-data/FRONTEND-DATA.md`
- `ai/docs/frontend/05-state/STATE.md`
- `ai/docs/frontend/09-api-dependencies/API-DEPENDENCIES.md`

### Reglas de documentación
- Actualiza un archivo a la vez y pide aprobación antes del siguiente.
- No dupliques información entre documentos.
- Referencia archivos con formato wiki link `[[archivo]]` cuando aplique.
- Mantén consistencia con la documentación existente.
- Conserva el historial de cambios cuando sea posible.

---

## Fase 4: Actualizar Tareas

Con la documentación actualizada, modifica `ai/tasks/backend/MAIN-TASKS.md` y/o `ai/tasks/frontend/MAIN-TASKS.md` según las capas afectadas.

### Reglas de estructura
- `MAIN-TASKS.md` contiene solo tareas principales, acción, estado, dependencias y referencia al archivo de detalle.
- No agregues tablas, listas ni detalles de subtareas en `MAIN-TASKS.md`.
- La descomposición vive únicamente en `ai/tasks/<capa>/NNN-task-<nombre>.md`.
- No crees una tarea grande nueva si el usuario indicó una tarea existente.
- Si no hay tarea objetivo, crea una nueva tarea principal solo cuando el cambio no pueda pertenecer a una tarea existente.
- Si creas una nueva tarea principal por este comando, su acción debe ser `actualizar`.

### Acción de la actualización
- Todas las subtareas nuevas o modificadas por este comando deben llevar `Acción: actualizar`.
- También deben llevar `Acción: actualizar` las subtareas que agregues a un archivo de detalle existente.
- Nunca uses `Acción: crear` para una subtarea producida por este comando.
- Las subtareas no afectadas conservan su acción y estado.
- No modifiques tareas ya completadas salvo petición explícita.

### Backend
- Si el usuario indica `T-B00X`, actualiza esa fila y `ai/tasks/backend/NNN-task-<nombre>.md`.
- Si no hay tarea objetivo, crea una fila nueva y su archivo de detalle cuando corresponda.

### Frontend
- Si el usuario indica `T-F00X`, actualiza esa fila y `ai/tasks/frontend/NNN-task-<nombre>.md`.
- Si no hay tarea objetivo, crea una fila nueva y su archivo de detalle cuando corresponda.

---

## Reglas Duras

1. **Nunca** ejecutes código. Solo documenta y actualiza tareas.
2. **Nunca** inventes requisitos. Solo lo que el usuario aprobó.
3. **Siempre** actualiza la documentación antes de actualizar tareas.
4. **Siempre** pide aprobación en cada fase.
5. **Siempre** usa la documentación como fuente de verdad.
6. **Nunca** asumas que algo está completo sin verificarlo.
7. **Nunca** saltes fases.
8. **Nunca** modifiques tareas ya completadas.
9. **Conserva** la estructura de documentación existente.
10. **Si el usuario indica una tarea existente**, no crees una tarea grande nueva.
11. Toda subtarea nueva o modificada por este comando debe decir `Acción: actualizar`.

---

## Output Final

Al terminar, informa al usuario:

```
✅ Funcionalidad "[nombre]" actualizada.

📁 Documentación actualizada:
  - ai/docs/specs/NNN-[nombre]/SPEC.md
  - ai/docs/database/... (si aplica)
  - ai/docs/backend/... (si aplica)
  - ai/docs/frontend/... (si aplica)

📋 Tareas actualizadas:
  - ai/tasks/backend/MAIN-TASKS.md (si aplica)
  - ai/tasks/frontend/MAIN-TASKS.md (si aplica)
  - Archivos NNN-task-*.md afectados

Acción de las subtareas nuevas o modificadas: `actualizar`.

Para implementar, ejecuta:
  - /ejecutar-backend
  - /ejecutar-frontend
```
