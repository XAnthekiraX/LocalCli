---
description: Genera el AGENTS.md del proyecto como orquestador de contexto (overview, estructura raíz del proyecto, capas, navegación de arquitectura, flujo Spec-Driven y convención de referencias). No genera arquitectura interna.
---

# Comando: Generar AGENTS.md

**Entrada del usuario:** `$ARGUMENTS`

Eres un orquestador de contexto. Tu objetivo es generar (o actualizar) el `AGENTS.md` en la raíz del proyecto.

**Principio clave:** `AGENTS.md` es un orquestador de contexto, **no** un documento arquitectónico. Contiene la estructura global del proyecto a **nivel raíz** (capas, `ai/docs`, `ai/tasks`, configs) y dice al agente qué documentos consultar, cuándo y qué flujo seguir. La arquitectura interna de cada capa vive en `ai/docs/<capa>/<CAPA>.md`.

---

## Fase 0: Contexto del Proyecto

1. Si `AGENTS.md` ya existe en la raíz, léelo para conservar secciones custom al actualizar.
2. Detecta las capas documentadas con `find` (nunca asumas rutas):
   - `ai/docs/frontend/FRONTEND.md`
   - `ai/docs/backend/BACKEND.md`
   - `ai/docs/database/DATABASE.md`
3. Lee `ai/docs/PROJECT.md` si existe (nombre, propósito, estructura).
4. Si **ninguna** capa tiene su archivo principal (`FRONTEND.md`, `BACKEND.md`, `DATABASE.md`), detente e informa que primero debe ejecutarse `/project-planner` (o `/crear` para generar documentación).

---

## Fase 1: Project Overview

Solo información verdaderamente global:

- Nombre del proyecto
- Propósito general
- Capas existentes (solo las detectadas con documentación)
- Estructura general del repositorio (árbol raíz → Fase 2)

Nada de arquitectura interna.

---

## Fase 2: Estructura del proyecto (árbol raíz)

Genera el árbol de carpetas del proyecto a **nivel raíz** y preséntalo al usuario para aprobación.

### Origen del árbol

- **Si el proyecto ya existe:** detecta la estructura real con `find . -maxdepth 1` (carpetas y archivos raíz), excluyendo `node_modules`, `.git`, `dist`, `build` y similares. Incluye siempre `ai/docs/` y `ai/tasks/` si existen.
- **Si es un proyecto nuevo (greenfield):** construye el **árbol objetivo** a partir de las capas documentadas y la estructura canónica de `ai/docs/` y `ai/tasks/`. Indícalo como "estructura objetivo".

### Reglas del árbol

- Solo **nivel raíz**: carpetas de capas, `ai/docs/`, `ai/tasks/`, archivos de configuración raíz.
- **Nunca** bajes a archivos internos (`src/`, `components/`, `controllers/`, etc.): eso pertenece al `ARCHITECTURE.md` de cada capa.
- Marca con un comentario que el árbol profundo de cada capa vive en su `ARCHITECTURE.md`.

Ejemplo de la sección:

```markdown
## Estructura del proyecto

```
proyecto/
├── frontend/          → documentación: ai/docs/frontend/FRONTEND.md
├── backend/           → documentación: ai/docs/backend/BACKEND.md
├── database/          → documentación: ai/docs/database/DATABASE.md
├── ai/
│   ├── docs/          → documentación (PROJECT.md, specs, capas)
│   └── tasks/         → tareas de implementación (frontend/, backend/: MAIN-TASKS.md + NNN-task-*.md)
├── package.json
└── README.md
```

La estructura interna de cada capa se define en su propio FRONTEND.md, BACKEND.md o DATABASE.md.
```

Presenta el árbol al usuario y espera aprobación antes de continuar.

---

## Fase 3: Project Layers y Architecture Sources

Para cada capa detectada en Fase 0, agrega:

```
- <Capa>
  - Documentación: ai/docs/<capa>/<CAPA>.md
```

Solo referencias capas con su archivo principal existente (`FRONTEND.md`, `BACKEND.md`, `DATABASE.md`). No crees stubs.

---

## Fase 4: Architecture Navigation Rules

`AGENTS.md` no define la arquitectura de las capas. Antes de implementar:

- Frontend → leer `ai/docs/frontend/FRONTEND.md`
- Backend → leer `ai/docs/backend/BACKEND.md`
- Database → leer `ai/docs/database/DATABASE.md`
- Cross-layer → leer la documentación de **todas** las capas afectadas

Cada archivo principal (`FRONTEND.md`, `BACKEND.md`, `DATABASE.md`) es responsable de referenciar los archivos y módulos de su capa.

---

## Fase 5: Workflow Spec-Driven (adaptado al proyecto)

```
Request
   ↓
Identify affected layers
   ↓
Read relevant ARCHITECTURE.md
   ↓
SPEC
   ↓
PLAN (por capas)
   ↓
TASKS
   ↓
Implement
   ↓
Validate
```

Mapeo con los comandos y skills del proyecto:

- `/project-planner` → crea `ai/docs/IDEA.md`, especificaciones funcionales, propuesta técnica, diseño, documentación por capas y roadmap
- `/crear` → SPEC → PLAN por capas → actualiza documentación → crea tareas principales y archivos de detalle con acción `crear`
- `/actualizar` → análisis de impacto → actualiza SPEC y documentación → actualiza tareas principales y subtareas con acción `actualizar`
- `/eliminar` → análisis de impacto → actualiza documentación → actualiza tareas principales y subtareas con acción `eliminar`
- `/ejecutar-backend` y `/ejecutar-frontend` → implementan las tareas (skills `backend-executor` y `frontend-executor`)
- Skills de documentación por capa (`frontend-documentation-architect`, `backend-documentation-architect`, `database-documentation-architect`) → generan la estructura canónica de cada capa
- `docs-consistency-validator` → valida/normaliza la documentación contra la estructura canónica
- `context-obtainer` → sigue las referencias de `AGENTS.md` → `FRONTEND.md`/`BACKEND.md`/`DATABASE.md` → archivos

---

## Fase 6: Convención de referencias

`AGENTS.md` debe declarar el contrato de referencias que usa todo el flujo:

- Formato: `[[carpeta/ARCHIVO]]` (misma capa) o `[[carpeta/ARCHIVO]]` (otra capa)
- Toda dependencia entre documentos se referencia con formato wiki link de Obsidian
- Las referencias apuntan solo al contexto necesario (no a archivos enteros)
- Si una modificación cambia el nombre o ubicación de un archivo referenciado, actualizarla

### Ejemplos de referencias

- `[[database/TABLES]]` → para referenciar tablas de la base de datos
- `[[backend/API-AUTH]]` → para referenciar endpoints de autenticación
- `[[frontend/FRONTEND]]` → para referenciar la arquitectura del frontend
- `[[backend/dto/AUTH-DTO]]` → para referenciar DTOs de autenticación

---

## Fase 7: Escribir AGENTS.md

- Crear en la raíz del proyecto: `AGENTS.md`.
- Si ya existe: actualiza las secciones estándar y **conserva** las secciones custom del usuario.
- Escribir en **español** (idioma canónico de la documentación del proyecto).
- Sin diagramas ASCII pesados: usa listas y código corto.
- Incluir la sección `## Estructura del proyecto` (aprobada en Fase 2) dentro de Project Overview.

---

## Fase 8: Verificación

1. Verifica con `find` que cada ruta referenciada en `AGENTS.md` existe.
2. Verifica que el árbol raíz coincida con la estructura real del proyecto (si existe).
3. Verifica que `AGENTS.md` no contenga arquitectura interna (sin rutas a `src/`, `components/`, etc.).
4. Informa al usuario si alguna capa detectada quedó fuera o si conviene ejecutar `docs-consistency-validator`.

---

## Reglas Duras

1. **Nunca** generes arquitectura interna en `AGENTS.md`.
2. **Nunca** referencies capas sin su archivo principal (`FRONTEND.md`, `BACKEND.md`, `DATABASE.md`).
3. **Nunca** bajes a archivos internos en el árbol raíz (eso vive en cada archivo principal de capa).
4. **Nunca** asumas rutas: usa `find`.
5. **Nunca** inventes capas, tecnologías o flujos.
6. **Siempre** respeta el formato wiki link `[[carpeta/ARCHIVO]]`.
7. **Siempre** conserva secciones custom si `AGENTS.md` ya existe.
8. **Siempre** pide aprobación del árbol antes de escribirlo.
9. **Siempre** escribe en español.

---

## Output Final

Al terminar, informa al usuario:

```
✅ AGENTS.md generado.

📄 Archivo:
  - AGENTS.md

🗂️ Capas referenciadas:
  - Frontend → ai/docs/frontend/FRONTEND.md
  - Backend → ai/docs/backend/BACKEND.md
  - Database → ai/docs/database/DATABASE.md

🌳 Estructura del proyecto (nivel raíz) incluida en Project Overview.

ℹ️ AGENTS.md es un orquestador de contexto. La arquitectura vive en cada ai/docs/<capa>/<CAPA>.md.
```