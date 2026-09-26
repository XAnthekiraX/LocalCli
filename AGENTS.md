# AGENTS.md — LocalCli

Orquestador de contexto para agentes de IA. Este archivo **no** define arquitectura:
dice qué documentos consultar, cuándo y qué flujo seguir. La arquitectura interna de
cada capa vive en su archivo principal bajo `ai/docs/<capa>/`.

## Project Overview

- **Nombre:** LocalCli
- **Propósito:** Harness de terminal en Go que planifica y ejecuta desarrollo de
  software con un modelo local de Ollama, entregando a cada etapa solo el contexto
  que necesita y pidiendo aprobación antes de aplicar cualquier cambio.
- **Capas documentadas:** Frontend (TUI), Backend, Database.
- **Stack:** Go, Bubble Tea + Lip Gloss, SQLite (`modernc.org/sqlite`, sin cgo),
  Landlock, Ollama por API HTTP. Un único binario.
- Detalle global del proyecto: [[PROJECT]]

## Estructura del proyecto

```
localcli/                       → proyecto Go (binario único)
├── main.go                     → punto de entrada
├── internal/                   → módulos internos del backend
│                                 (arquitectura: ai/docs/backend/BACKEND.md)
├── ai/
│   ├── docs/                   → documentación (PROJECT.md, specs/, frontend/, backend/, database/)
│   └── tasks/                  → tareas de implementación (backend/: MAIN-TASKS.md + NNN-task-*.md)
├── .agents/                    → comandos, skills y agents del harness
├── .localcli/                  → estado de ejecución (state.db), no versionado
├── go.mod / go.sum
└── AGENTS.md                   → este archivo
```

La estructura interna de cada capa se define en su propio `FRONTEND.md`,
`BACKEND.md` o `DATABASE.md`; aquí solo figura el nivel raíz.

## Project Layers

- **Frontend (TUI)**
  - Documentación: `ai/docs/frontend/FRONTEND.md`
- **Backend**
  - Documentación: `ai/docs/backend/BACKEND.md`
- **Database**
  - Documentación: `ai/docs/database/DATABASE.md`

## Architecture Navigation Rules

Antes de implementar o modificar código:

- Cambios de interfaz de terminal → leer `ai/docs/frontend/FRONTEND.md`
- Cambios de lógica, agentes, herramientas, flujos → leer `ai/docs/backend/BACKEND.md`
- Cambios de esquema, consultas o reglas de datos → leer `ai/docs/database/DATABASE.md`
- Cambios transversales → leer la documentación de **todas** las capas afectadas

Cada archivo principal es responsable de referenciar los documentos de su capa.

## Workflow Spec-Driven

```
Request
   ↓
Identify affected layers
   ↓
Read relevant layer doc (FRONTEND.md / BACKEND.md / DATABASE.md)
   ↓
SPEC (ai/docs/specs/)
   ↓
PLAN (por capas)
   ↓
TASKS (ai/tasks/<capa>/)
   ↓
Implement
   ↓
Validate
```

Mapeo con los comandos y skills del proyecto:

- `/project-planner` → crea `ai/docs/IDEA.md`, especificaciones, propuesta técnica y documentación por capas
- `/crear` → SPEC → PLAN por capas → actualiza documentación → crea tareas con acción `crear`
- `/actualizar` → análisis de impacto → actualiza SPEC y documentación → actualiza subtareas con acción `actualizar`
- `/eliminar` → análisis de impacto → actualiza documentación → actualiza subtareas con acción `eliminar`
- `/resolver` → ciclo de arreglo de lo roto ([[specs/SPEC-RESOLVER]])
- `/ejecutar-backend` y `/ejecutar-frontend` → implementan tareas (skills `backend-executor` y `frontend-executor`)
- Skills de documentación: `frontend-documentation-architect`, `backend-documentation-architect`, `database-documentation-architect`
- `docs-consistency-validator` → valida/normaliza la documentación contra la estructura canónica
- `context-obtainer` → sigue las referencias de este archivo → capa → documentos concretos

## Convención de referencias

- Formato: wiki link de Obsidian `[[carpeta/ARCHIVO]]`, relativo a `ai/docs/`.
- Las dependencias entre documentos se **declaran en el frontmatter** de cada
  archivo, no se deducen de los enlaces del cuerpo. Un enlace en el cuerpo es
  una mención, no una dependencia. La convención completa está en
  [[PROJECT]]: qué claves existen, cuándo va en `depende_de` y cuándo en
  `relacionado`.
- Si una modificación cambia el nombre o ubicación de un archivo referenciado, actualizarla.
- Las referencias apuntan solo al contexto necesario, no a archivos enteros: enlaza
  la sección que se cita, no el documento completo.

### Ejemplos

- `[[database/01-schema/TABLES]]` → tablas de la base de datos
- `[[backend/DECISIONS]]` → decisiones técnicas centralizadas del backend
- `[[frontend/FRONTEND]]` → arquitectura de la capa frontend
- `[[specs/SPEC-TOOLS]]` → catálogo de herramientas

## Especificaciones funcionales

17 specs aprobadas en `ai/docs/specs/` (núcleo P0, importante P1, interfaz P2 y
personalización P3). El índice completo está en [[PROJECT]].

## Tareas

- Backend: `ai/tasks/backend/MAIN-TASKS.md` + detalle en `NNN-task-<nombre>.md`
- Frontend: `ai/tasks/frontend/` (cuando exista documentación/tareas de esa capa)
