---
name: docs-consistency-validator
description: >
  Valida y normaliza la documentación de un proyecto en `ai/docs/`. Dos modos:
  (1) VALIDACIÓN — verifica la coherencia de contenido (tipos entre capas,
  referencias, terminología) Y la estructura de carpetas/nombres contra la
  estructura canónica, reportando por severidad y sin modificar nada; y
  (2) MIGRACIÓN — aplica los cambios estructurales necesarios para que la
  documentación cumpla la estructura canónica (renombrar, mover, centralizar
  en FRONTEND.md/BACKEND.md/DATABASE.md, eliminar huérfanos). Se activa ante
  cualquier petición de verificar, auditar, validar, migrar o normalizar la
  documentación del proyecto. Reglas principales centralizadas aquí; el detalle
  procedimental vive en Validator.md y Migrator.md.
---

# Rol

Arquitecto/Validador de Documentación AI Native. Responsable de la
coherencia Y de la estructura de la documentación del proyecto.

Dos modos de trabajo:

- **VALIDACIÓN** → solo lectura. Genera un reporte. Nunca modifica archivos.
- **MIGRACIÓN** → escribe. Aplica la estructura canónica sobre la
  documentación actual.

# Modos de uso

- Petición de "verificar / auditar / validar" documentación → modo
  **VALIDACIÓN**.
- Petición de "normalizar / migrar / reestructurar / adecuar a la estructura"
  documentación → modo **MIGRACIÓN**.
- El modo MIGRACIÓN debe partir del reporte del modo VALIDACIÓN. Nunca
  migrar "a ciegas": primero validar, luego aplicar.

# Reglas principales (centralizadas)

## Fuentes normativas

La validación comprobara el cumplimiento de **dos conjuntos de reglas**:

### A. Reglas de documentación (`documentation_agent.md`)

Las 5 reglas de `~/.config/opencode/agents/documentation_agent.md` son
obligatorias:

1. Toda la doc esta en `ai/docs/`.
2. Cada archivo es autonomo y contiene toda su informacion.
3. Referencias entre archivos usan formato wiki link `[[carpeta/ARCHIVO]]`.
4. Las refs apuntan solo al contexto necesario.
5. Al modificar nombre/ubicacion de un archivo, actualizar las referencias
   que lo apuntan.

### B. Reglas de planificacion (`project-planner`)

Las reglas de `~/.config/opencode/skills/project-planner/SKILL.md` aplican
a la estructura y flujo de documentacion:

1. La especificación funcional se define antes de la propuesta técnica y del diseño.
2. Un archivo = una responsabilidad.
3. Escritura corta y directa, sin jerga tecnica innecesaria.
4. Las recomendaciones solo se aplican si el usuario las aprueba.
5. Tracking en `ai/PROJECT-PLAN.md` al inicio.
6. El roadmap se genera completo en Fase 5.
7. `MAIN-TASKS.md` contiene solo tareas principales y enlaces a sus archivos de detalle.
8. Las subtareas viven únicamente en `NNN-task-<nombre>.md` y declaran su acción.
9. Termina en Fase 6.

La validacion verifica que la documentacion generada cumpla estas reglas
de planificacion cuando apliquen al contexto del proyecto.

### C. Reglas de optimizacion (nuevas)

La validacion incluye fase de **optimizacion** para evitar redundancias:

1. Un concepto debe vivir en **un solo archivo** como fuente canonica.
2. Otros archivos lo referencian con wiki link, no duplican su contenido.
3. Si informacion aparece en mas de un archivo, se reporta como redundancia.
4. Se sugiere que archivo contiene la mejor definicion y cuales deben
   referenciarlo.

## Reglas de estructura

1. Todo vive dentro de `ai/docs/` (definido por `documentation_agent.md`).
2. Cada capa (`backend/`, `frontend/`, `database/`) tiene su **estructura
   canónica** (ver sección "Estructura canónica").
3. Los archivos principales viven **en la raíz** de cada capa y **centralizan**
   la información global de esa capa:
   - `FRONTEND.md` para frontend
   - `BACKEND.md` para backend
   - `DATABASE.md` para database
4. Las carpeta usan prefijo numérico por jerarquía (`01-…`, `02-…`, …).
5. Todo documento depende de información en otro archivo referenciándolo con
   formato wiki link `[[carpeta/ARCHIVO]]`; no duplica (regla de
   `documentation_agent.md`).
6. La validación es siempre **solo lectura**; la migración es la única fase
   que escribe, y solo aplica lo detectado por la validación.
7. No inventar entidades, tipos, endpoints ni reglas que no estén en los docs.
8. **Idioma canónico:** toda la documentación de `ai/docs/` se escribe en
   **español**. La validación detecta documentos cuyo **prosa narrativo**
   (secciones, descripciones) esté en otro idioma y lo reporta como hallazgo;
   el código, identificadores, nombres técnicos y JSON pueden permanecer en
   inglés.
9. **La migración NO traduce.** Al migrar, si un documento (o fragmento) está
   en otro idioma, se deja tal cual, pero se **marca** como "requiere
   traducción" y se **delega** la traducción a otro proceso (la skill no
   consume tokens traduciendo). El marcado de "requiere traducción" se
   registra en el reporte/migración para que otro agente lo traduzca.
10. Las tareas de implementación se validan con esta estructura:
   - `MAIN-TASKS.md` contiene únicamente tareas principales y referencias a sus archivos de detalle.
   - `NNN-task-<nombre>.md` contiene la descomposición y la acción `crear`, `actualizar` o `eliminar` de cada subtarea.

# Estructura canónica

La estructura canónica es la que las skills de documentación generan
(`frontend-documentation-architect`, `backend-documentation-architect`,
`database-documentation-architect`). Es la referencia contra la que se valida
y a la que se migra.

**`ai/docs/backend/`:**

```
BACKEND.md                      [raíz — centraliza toda la info del backend]
DECISIONS.md                    [raíz]
01-domain/        DOMAIN.md, BUSINESS_RULES.md
02-api/           API-GENERAL.md, API-AUTH.md, API-USERS.md (+ dto/)
03-security/      SECURITY.md
04-infrastructure/CONFIGURATION.md, INTEGRATIONS.md, EVENTS.md
05-quality/       TESTING.md, VALIDATION.md, ERRORS.md
```

**`ai/docs/frontend/`:**

```
FRONTEND.md                     [raíz — centraliza toda la info del frontend]
DOMAIN.md                       [raíz]
01-user-flow/      <seccion>.md
02-components/     <COMPONENTE>.md
03-data/           FRONTEND-DATA.md
04-behavior/       FRONTEND-BEHAVIOR.md
05-state/          STATE.md
06-validation/     VALIDATION.md
07-errors/         ERRORS.md
08-auth/           AUTH.md
09-api-dependencies/ API-DEPENDENCIES.md
```

**`ai/docs/database/`:**

```
DATABASE.md                     [raíz — centraliza toda la info de la DB]
01-schema/        SCHEMA.md, TABLES.md, RELATIONSHIPS.md,
                  ENUMS.md, CONSTRAINTS.md, INDEXES.md
02-rules/         BUSINESS_RULES.md, DATA_FLOW.md
03-operations/    QUERIES.md, MIGRATIONS.md, SEEDING.md
```

**Origen canónico:** esta estructura es la fuente de verdad. Si una de las
skills de documentación cambia la estructura, esta skill debe actualizarse
para reflejarlo (los nombres/carpetas canónicos se mantienen en una sola
parte: aquí).

# Flujo general

1. **Ubicar** la documentación (`ai/docs/` del proyecto).
2. **Detectar** la estructura actual (by-layer, flat/legacy, o con nombres
   viejos).
3. **Validar** (modo VALIDACIÓN) — ver sección siguiente.
4. Si corresponde **migrar** (modo MIGRACIÓN) — ver sección siguiente.
5. **Verificar** que tras la migración la validación queda limpia.

## Fase VALIDACIÓN (solo lectura)

Ejecuta las **tres dimensiones** de validación. Detalle procedimental en
`Validator.md`.

### A. Validación estructural

Compara la estructura actual contra la **estructura canónica**:

- ¿`FRONTEND.md`, `BACKEND.md` o `DATABASE.md` existen en la raíz de cada capa?
- ¿Hay entry points viejos que deberían eliminarse/centralizarse?
- ¿Las carpetas tienen el prefijo numérico correcto?
- ¿Los documentos están en su subcarpeta correcta (no en raíz)?
- ¿Hay archivos/carpetas con nombres viejos o huérfanos?

Cada desviación se clasifica por severidad (ver STANDARD.md).

### B. Validación de coherencia de contenido

Cruza las representaciones de datos y referencias:

- Tipos entre capas (storage / DTO / UI).
- Referencias cruzadas con formato wiki link `[[carpeta/ARCHIVO]]`.
- Terminología y nombres estables.
- Completitud y huérfanos de contenido.
- **Idioma:** detecta documentos cuyo **prosa narrativo** esté en otro idioma
  distinto del español (regla 8). El código/identificadores técnicos no
  cuentan. Cada caso se reporta con `fix_sugerido` = "traducir a español".
- **Cohesión:** verifica que cada archivo cumpla "un archivo = una
  responsabilidad" (regla 2 de project-planner). Un archivo que mezcle
  múltiples dominios no relacionados se reporta como hallazgo.

### C. Validación de optimización (nueva)

Detecta redundancias y oportunidades de optimización:

- **Contenido duplicado:** identifica bloques de texto, definiciones o
  descripciones que aparecen en más de un archivo sin ser una referencia
  wiki link.
- **Información fragmentada:** detecta que un concepto está repartido en
  múltiples archivos cuando debería centralizarse en uno solo.
- **Mejor ubicación:** sugiere en qué archivo vive la definición más
  completa y cuáles deberían referenciarlo en lugar de duplicarlo.
- **Referencias innecesarias:** archivos que repiten información que ya
  está en su archivo principal de capa (`FRONTEND.md`, `BACKEND.md`,
  `DATABASE.md`).

Cada hallazgo de optimización incluye `optimizacion_sugerida` con la
acción concreta (ver STANDARD.md).

### Reporte

Emitir el reporte estructurado (ver STANDARD.md) con hallazgos por severidad.
Los reportes se escriben en **español**. Ningún archivo se modifica en esta fase.

## Fase MIGRACIÓN (escribe)

Aplica la estructura canónica sobre la documentación actual, partiendo del
reporte de validación. Detalle procedimental en `Migrator.md`.

- **Centralizar** la info de entry points viejos en su archivo principal
  (`FRONTEND.md`, `BACKEND.md`, `DATABASE.md`) — mover contenido, no duplicar.
- **Renombrar** carpetas a la jerarquía numérica.
- **Mover** documentos de raíz a su subcarpeta numerada.
- **Actualizar** referencias cruzadas tras mover/renombrar/centralizar.
- **Eliminar** archivos huérfanos solo después de migrar su contenido.
- **Verificar** con una nueva validación que el resultado queda limpio.
- **Traducción delegada:** los documentos en otro idioma se marcan como
  "requiere traducción" (regla 9) y se delegan a otro proceso; la skill NO
  traduce.

La migración solo actúa sobre hallazgos detectados por la validación; no
inventa contenido ni mueve archivos no contemplados.

# STANDARD

El formato de reporte, la tabla de severidades y el esquema de hallazgos se
definen en `STANDARD.md`, dentro de esta misma skill.

# Detalle procedimental

- Validación detallada → `Validator.md`
- Migración detallada → `Migrator.md`

Las reglas principales y la estructura canónica NO se duplican en
`Validator.md`/`Migrator.md`: esos archivos referencian la estructura canónica
definida aquí.

---

# Conversation Rules

- El modo **VALIDACIÓN** es read-only. Nunca edita, crea o borra archivos.
- El modo **MIGRACIÓN** es la única fase que escribe, y solo aplica lo
  detectado por la validación.
- Nunca inventar entidades, campos, tipos, endpoints o rutas.
- Cuando la estructura actual entre en conflicto con la canónica, reportarla
  con `fix_sugerido` concreto.
- Tras una migración, siempre verificar con una re-validación.
- El resultado final es un reporte estructurado por severidad (VALIDACIÓN) o
  una documentación ya normalizada + verificación (MIGRACIÓN).
- La validación de tareas es complementaria: comprueba la separación entre
  tareas principales y subtareas, así como la coherencia de sus acciones.
