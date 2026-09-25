# Migrator — Migración de la estructura de documentación

Procedimiento detallado de la **Fase MIGRACIÓN** de la skill
`docs-consistency-validator`. Es la única fase que **escribe** sobre la
documentación. Parte del reporte de la fase VALIDACIÓN (ver `Validator.md`)
y referencia la estructura canónica definida en `SKILL.md`.

# Regla de oro

**Nunca migrar "a ciegas".** La migración solo aplica los hallazgos
detectados por la validación. Si no hay un reporte previo de validación,
ejecutar primero el modo VALIDACIÓN.

# Flujo de migración

## 1. Tomar como entrada el reporte de validación

Usar los hallazgos estructurales del reporte (`validation_report.md` o
`validation_report.json`). Agruparlos en operaciones:

- Centralizar entry points → `ARCHITECTURE.md`.
- Renombrar carpetas.
- Mover documentos.
- Eliminar huérfanos.
- Actualizar referencias.

## 2. Centralizar entry points viejos en ARCHITECTURE.md

Cuando existan `00-BACKEND.md`, `00-FRONTEND.md` o `DATABASE.md`:

1. Leer el contenido completo del entry point viejo.
2. Abrir el `ARCHITECTURE.md` de la capa (si no existe, crear el esqueleto a
   partir de la plantilla base de la skill de documentación de esa capa).
3. **Mover** (no duplicar) la información global del entry point al
   `ARCHITECTURE.md`, integrando en las secciones correspondientes.
4. Tras confirmar que la información quedó integrada, eliminar el entry point
   viejo.
5. Para `database/`, el mapa de navegación de `DATABASE.md` debe quedar en
   `ARCHITECTURE.md` (sección de navegación).

> Nota: nunca eliminar el archivo fuente antes de verificar que su información
> fue integrada correctamente en `ARCHITECTURE.md`.

## 3. Renombrar carpetas a la jerarquía numérica

Mapear carpetas viejas a su prefijo numérico según la estructura canónica:

| Capa | Carpeta vieja | Carpeta canónica |
|------|---------------|------------------|
| backend | `01-architecture/` | content mover a raíz / descomponer |
| backend | `02-domain/` | `01-domain/` |
| backend | `03-api/` | `02-api/` |
| backend | `04-security/` | `03-security/` |
| backend | `05-infrastructure/` | `04-infrastructure/` |
| backend | `06-quality/` | `05-quality/` |
| frontend | `user-flow/` | `01-user-flow/` |
| frontend | `components/` | `02-components/` |
| frontend | `FRONTEND-DATA.md` (raíz) | `03-data/FRONTEND-DATA.md` |
| frontend | `FRONTEND-BEHAVIOR.md` (raíz) | `04-behavior/FRONTEND-BEHAVIOR.md` |
| frontend | `STATE.md` (raíz) | `05-state/STATE.md` |
| frontend | `VALIDATION.md` (raíz) | `06-validation/VALIDATION.md` |
| frontend | `ERRORS.md` (raíz) | `07-errors/ERRORS.md` |
| frontend | `AUTH.md` (raíz) | `08-auth/AUTH.md` |
| frontend | `API-DEPENDENCIES.md` (raíz) | `09-api-dependencies/API-DEPENDENCIES.md` |
| database | `schema/` | `01-schema/` |
| database | `rules/` | `02-rules/` |
| database | `operations/` | `03-operations/` |

Cuando una carpeta vieja tenga un prefijo numérico pero en otro orden (p.ej.
backend `04-security/` → `03-security/`), simplemente renombrar el prefijo;
no mover archivos internos.

## 4. Mover documentos de raíz a su subcarpeta

- Mover cada documento en raíz que, según la estructura canónica, deba vivir
  en una subcarpeta numerada.
- La subcarpeta objetivo, si no existe, se crea antes de mover.

## 5. Actualizar referencias cruzadas

Tras mover, renombrar o centralizar, **toda referencia** a los archivos
afectados debe actualizarse al nuevo formato wiki link `[[nombre-archivo]]`
o `[[ruta/completa/Archivo]]`. Esto implementa la **regla 5 del subagente
de documentación** (`documentation_agent.md`): si una modificación cambia
el nombre o ubicación de un archivo referenciado, esa referencia debe
actualizarse.

1. Buscar referencias wiki link `[[...]]` a la ruta antigua en el resto de `ai/docs/`.
2. Actualizar el nombre o ruta dentro del wiki link.
3. Referencias a entry points eliminados (`DATABASE.md`, `00-BACKEND.md`,
   `00-FRONTEND.md`) redirigirlas a su `ARCHITECTURE.md`.
4. Asegurar que cada referencia respete el formato wiki link
   `[[nombre-archivo]]` o `[[ruta/completa/Archivo]]` (regla 3) y apunte
   solo al contexto necesario (regla 4).

## 6. Marcar documentos que requieren traducción

La migración **NO traduce** (ahorro de tokens). Si la validación detectó
documentos cuyo prosa no está en español:

- Dejarlos tal cual; no reescribir su contenido.
- **Marcarlos** como "requiere traducción" en el reporte de migración
  (lista de archivos afectados).
- **Delegar** la traducción a otro proceso/agente; esta skill no la ejecuta.

Esto aplica también al contenido en otro idioma que se centralice en
`ARCHITECTURE.md` desde un entry point: se integra sin traducir y se marca.

## 7. Eliminar huérfanos y residuos

- Eliminar solo archivos/carpetas que ya no correspondan a la estructura
  canónica **y** cuyo contenido haya sido migrado (no borrar información).
- Eliminar archivos vacíos o duplicados creados durante procesos anteriores.

## 8. Verificar post-migración

Re-ejecutar el modo VALIDACIÓN (ver `Validator.md`):

- La estructura debe coincidir con la canónica.
- Las referencias debe seguir resolviendo a archivos existentes.
- No deben quedar huérfanos ni entry points viejos.
- Si quedan hallazgos `ERROR`, corregirlos antes de dar por terminada la
  migración.
- Los documentos marcados "requiere traducción" quedan registrados para que
  otro proceso los traduzca (no son un bloqueo: la estructura ya es canónica).

## 9. Reporte de migración

Emitir un breve reporte de lo ejecutado:

- Archivos creados / movidos / renombrados / eliminados.
- Referencias actualizadas.
- Documentos marcados "requiere traducción" (lista).
- Resultado de la re-validación.

---

# Reglas del migrador

- **Nunca** eliminar un archivo sin haber integrado antes su contenido en el
  destino correcto.
- **Nunca** duplicar información: mover, no copiar.
- **Nunca** inventar contenido ni secciones que no existían.
- Solo tocar lo que la validación marcó; no reestructurar por iniciativa
  propia.
- Tras cada operación crítica guardar trazabilidad (qué se movió desde dónde
  y hacia dónde).
- **No traducir**: los documentos en otro idioma se marcan "requiere
  traducción" y se delegan a otro proceso; el migrador no consume tokens
  traduciendo.
- Al terminar, la documentación debe pasar la validación por segunda vez.
