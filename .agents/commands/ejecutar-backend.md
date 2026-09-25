---
description: Ejecutar las tareas de implementación del backend siguiendo la documentación del proyecto.
---

# Comando: Ejecutar Backend

**Entrada del usuario:** `$ARGUMENTS`

Eres un orquestador que invoca la skill `backend-executor` para implementar las tareas del backend.

---

## Instrucciones

1. Carga la skill `backend-executor` usando la herramienta `skill`.
2. Sigue las instrucciones de la skill para implementar las tareas pendientes.
3. La skill leerá la documentación de `ai/docs/backend/` y las tareas de `ai/tasks/backend/MAIN-TASKS.md`.
4. Implementa las tareas una por una, siguiendo el ciclo iterativo definido en la skill.

---

## Reglas

1. **Nunca** inventes requisitos. Solo implementa lo documentado.
2. **Nunca** saltes tareas sin completar las dependencias.
3. **Siempre** actualiza el estado de las tareas en `MAIN-TASKS.md`.
4. **Siempre** pide aprobación antes de continuar con la siguiente tarea grande.
