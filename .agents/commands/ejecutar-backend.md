---
description: Ejecutar las tareas de implementación del backend ya planificadas y aprobadas, siguiendo la documentación del proyecto.
---

# Comando: Ejecutar Backend

**Entrada del usuario:** `$ARGUMENTS`

Eres un orquestador que invoca la skill `backend-executor` para **implementar** las tareas ya planificadas del backend.

---

## Instrucciones

1. Carga la skill `backend-executor` usando la herramienta `skill`.
2. Sigue las instrucciones de la skill para implementar las tareas pendientes.
3. La skill leerá la documentación de `ai/docs/backend/` y las tareas de `ai/tasks/backend/MAIN-TASKS.md` + `NNN-task-<nombre>.md`.
4. Implementa las subtareas una por una, siguiendo el ciclo iterativo definido en la skill.

---

## Reglas

1. **Nunca** inventes requisitos. Solo implementa lo documentado.
2. **Nunca** saltes tareas sin completar las dependencias.
3. **Nunca** crees ni modifiques `MAIN-TASKS.md` ni `NNN-task-<nombre>.md`: solo actualiza la columna **Estado**.
4. **Siempre** pide aprobación antes de continuar con la siguiente tarea grande.
5. Si no existen tareas planificadas, **detente** e indica ejecutar `/planificar-backend`. No planifiques aquí.
6. Si el usuario quiere implementar paso a paso con guía, indícale que cambie al agente **Profesor**.