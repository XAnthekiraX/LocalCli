# DOCS VALIDATION REPORT — LocalCli

## Summary

| Severity | Count |
|----------|-------|
| CRITICAL | 0 |
| ERROR | 0 |
| WARNING | 3 |
| INFO | 2 |
| OPTIMIZATION | 1 |
| Total | 6 |

- Layout detectado: `by-layer`
- Documentos analizados: 45 (17 specs + 12 backend + 12 database + 4 frontend, más IDEA y PROJECT)
- Enlaces wiki verificados: 528, **0 rotos** (verificación programática sobre todo `ai/docs/`)
- Idioma: prosa en español en todos los documentos; código e identificadores en inglés (permitido)
- Reporte generado: 2026-09-25T12:00:00Z
- Veredicto: **CONSISTENT** (0 CRITICAL, 0 ERROR)

## Findings

### WARNING

#### [W01] Prosa en inglés en SPEC-TOOLS
- Layer: backend
- Files: `specs/SPEC-TOOLS.md`
- Anchor: "El relevo entre agentes"
- Category: 8 (idioma)
- Description: "`plan` no escribe. Termina su trabajo proposing" — verbo en inglés dentro de la prosa en español.
- Fix: traducir a español ("termina su trabajo proponiendo").

#### [W02] Palabras unidas en INTEGRATIONS
- Layer: backend
- Files: `backend/04-infrastructure/INTEGRATIONS.md`
- Anchor: "### Ollama", viñeta Streaming
- Category: 8 (idioma / prosa)
- Description: "que es elrequisito de la interfaz" — dos palabras unidas por error tipográfico.
- Fix: "el requisito".

#### [W03] `messages.created_at` ausente en TABLES
- Layer: database
- Files: `database/01-schema/TABLES.md`
- Anchor: "### `messages`", tabla de columnas
- Category: coherencia de contenido
- Description: SCHEMA.md (§3) y CONSTRAINTS.md (NOT NULL) declaran `messages.created_at` como obligatoria, pero la tabla de columnas de TABLES.md no tiene la fila correspondiente. Divergencia de completitud entre los tres documentos del mismo esquema. (La tabla `reasoning` sí incluye su `created_at`.)
- Fix: añadir la fila `created_at` a la tabla de `messages` en TABLES.md.

### INFO

#### [I01] Palabra duplicada en SPEC-COLA-TAREAS
- Layer: specs
- Files: `specs/SPEC-COLA-TAREAS.md`
- Anchor: "## Actores"
- Category: prosa
- Description: "pausa, reanuda, cancela o cancela una tarea" — "cancela" duplicado.
- Fix: "pausa, reanuda o cancela una tarea".

#### [I02] Marcas de tiempo abreviadas en el ejemplo de TABLES
- Layer: database
- Files: `database/01-schema/TABLES.md`
- Anchor: "## 5. Ejemplo conceptual"
- Category: convención
- Description: El ejemplo usa horas abreviadas ("09:00") frente a la convención ISO 8601 UTC declarada en SCHEMA §1. Es un ejemplo ilustrativo, no una definición.
- Fix: opcional; puede dejarse si se entiende como abreviación de legibilidad.

### OPTIMIZATION

#### [O01] Concepto "cola derivada de los archivos" descrito en varios archivos
- Layer: cross
- Files: `backend/01-domain/DOMAIN.md`, `backend/01-domain/BUSINESS_RULES.md`, `database/DATABASE.md`, `database/02-rules/DATA_FLOW.md`, `PROJECT.md`, `specs/SPEC-COLA-TAREAS.md`
- Anchor: "La cola es el TODO de esta ejecución" / "Cola derivada de los archivos de tarea"
- Category: 8.1 (información fragmentada)
- Description: El concepto aparece desarrollado en más de un documento. La mayor parte ya lo referencia en lugar de redefinirlo, y las descripciones existentes cubren ángulos distintos (comportamiento en BUSINESS_RULES, datos en DATA_FLOW, decisión en DECISIONS).
- Fix: mantener la definición canónica en [[database/02-rules/DATA_FLOW]] y [[backend/DECISIONS]]; los demás documentos referencian sin volver a explicar el mecanismo.
- Optimización: "referenciar [[database/02-rules/DATA_FLOW]] en lugar de duplicar"

## Accepted Exceptions

Desviaciones estructurales con justificación escrita; no cuentan como hallazgos:

1. **Archivos raíz por capa: `BACKEND.md`, `DATABASE.md`, `FRONTEND.md`** (en lugar de `ARCHITECTURE.md`). Es el nombre que fija la skill `project-planner` y las documentation-architects; cada raíz centraliza su capa y funciona como mapa de navegación. El STANDARD incluye ambas variantes; el proyecto sigue la de SKILL.md.
2. **`backend/02-interfaces/` en lugar de `02-api/`.** LocalCli no expone API HTTP: sus superficies son comandos de CLI, herramientas del agente y eventos (justificado en INTERFACES-GENERAL §1). Se conserva el prefijo numérico.
3. **Frontend con estructura reducida** (`FRONTEND.md`, `01-domain/DOMAIN.md`, `02-interfaces/INTERFACES.md`, `05-quality/TESTING.md`) frente a las nueve carpetas canónicas. No aplican user-flow, auth ni api-dependencies a una TUI de presentación pura; estructura espejo del backend, aprobada por el usuario en Fase 4.
4. **`specs/`, `IDEA.md` y `PROJECT.md` en la raíz de `ai/docs/`.** Pertenecen al flujo de planificación (`project-planner`), no a una capa; las estructuras canónicas cubren solo backend/frontend/database.
5. **I02** se acepta como abreviación de legibilidad en un ejemplo conceptual.
