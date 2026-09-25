# Base de STATE.md

Plantilla de contenido para `ai/docs/frontend/05-state/STATE.md`.
Define el contrato que debe cumplir el documento generado, no el frontend en sí.

# Reglas

- Idioma: español.
- Determinista: sin suposiciones; cada afirmación verificable.
- Terminología estable entre documentos.
- Distingue el estado del frontend de los datos que persiste el backend.
- Sigue las reglas de `documentation_agent.md`.
- Usa formato wiki link `[[carpeta/ARCHIVO]]` para referencias.

# Secciones (orden fijo)

## 1. Server State

- Datos del servidor que el frontend gestiona.
- Referencia a [[backend/API-<RECURSO>]] para endpoints de datos.

## 2. UI State

- Estado de la interfaz.

## 3. Local State

- Estado local de componentes.

## 4. Derived State

- Estado derivado.

## 5. Estado Global

- Estado global del frontend.

## 6. Estado por Funcionalidad

- Estado específico por funcionalidad.

## 7. Persistencia Local

- Persistencia local, si existe.

## 8. Ciclo de Actualización/Invalidación

- Ciclo de actualización e invalidación del estado.

# Guía de entrevista

Para documentar STATE, compara `PROJECT.md` con estas secciones.
Lo que falte, preguntarlo al usuario (máx 5 preguntas a la vez).

# Checklist de calidad

- Server, UI, local y derived state diferenciados.
- Persistencia local y ciclo de actualización documentados.
- No duplica datos que persiste el backend.
- Referencias a [[backend/...]] cuando se obtenga server state.
