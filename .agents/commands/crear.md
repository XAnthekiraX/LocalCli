---
description: Agregar una nueva funcionalidad al proyecto. Crea especificación, actualiza documentación y agrega tareas principales y subtareas con acción crear.
---

# Comando: Crear Nueva Funcionalidad

**Entrada del usuario:** `$ARGUMENTS`

Agrega funcionalidades a un proyecto ya implementado. No crea el proyecto desde cero.

---

## Fase 0 — Contexto

1. Lee `ai/docs/PROJECT.md`. Si no existe, informa al usuario que primero debe ejecutar `/project-planner`.
2. Identifica qué capas se ven afectadas (DB, backend, frontend).
3. Identifica las tareas principales existentes y sus archivos de detalle.

---

## Fase 1 — Especificación

1. Analiza `$ARGUMENTS`.
2. Pregunta lo que falte (máx. 5 preguntas).
3. Crea `ai/docs/specs/NNN-<NombreFuncionalidad>/SPEC.md`.
4. Presenta al usuario para aprobación.

---

## Fase 2 — Documentación

Actualiza **solo los documentos afectados**:

### Database
- `ai/docs/database/01-schema/TABLES.md`
- `ai/docs/database/01-schema/RELATIONSHIPS.md`
- Otros si aplica.

### Backend
- `ai/docs/backend/01-domain/DOMAIN.md`
- `ai/docs/backend/02-api/API.md`
- `ai/docs/backend/02-api/<recurso>.md`
- Otros si aplica.

### Frontend
- `ai/docs/frontend/01-user-flow/<seccion>.md`
- `ai/docs/frontend/02-components/<COMPONENTE>.md`
- Otros si aplica.

**Reglas:**
- Un archivo a la vez, espera aprobación.
- No dupliques información.
- Solo actualiza, no crees documentación nueva.

---

## Fase 3 — Tareas

1. Crea o actualiza `ai/tasks/{backend,frontend}/MAIN-TASKS.md`.
2. Agrega una tarea principal por unidad funcional aprobada, con `Acción: crear` y referencia a su archivo `NNN-task-<nombre>.md`.
3. Crea o actualiza el archivo de detalle correspondiente.
4. Todas las subtareas nuevas o modificadas deben tener `Acción: crear`.
5. No agregues tablas, listas ni detalles de subtareas dentro de `MAIN-TASKS.md`.
6. Conserva las tareas existentes y no modifiques tareas completadas.

---

## Output

```
✅ Funcionalidad "[nombre]" preparada.

📁 Documentación actualizada:
  - ai/docs/specs/NNN-[nombre]/SPEC.md
  - (documentos modificados)

📋 Tareas agregadas:
  - ai/tasks/backend/MAIN-TASKS.md (si aplica)
  - ai/tasks/frontend/MAIN-TASKS.md (si aplica)
  - Archivos NNN-task-*.md con acción crear

Para implementar:
  - /ejecutar-backend
  - /ejecutar-frontend
```
