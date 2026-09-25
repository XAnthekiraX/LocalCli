# Base de DTO (por recurso)

Plantilla de contenido para los DTOs por recurso en `ai/docs/backend/02-api/dto/`
(ej. `AUTH-DTO.md`, `USERS-DTO.md`). Define el contrato que debe cumplir cada
documento generado, no el backend en sí.

# Reglas

- Idioma: español.
- Determinista: sin suposiciones; cada afirmación verificable.
- Terminología estable entre documentos.
- Referencia [[database/ARCHIVO]] para los tipos de datos; no duplicar.
- Sigue las reglas de `documentation_agent.md`.
- Usa formato wiki link `[[carpeta/ARCHIVO]]` para referencias.

# Convenciones de Naming

- **DTOs por recurso:** `AUTH-DTO.md`, `USERS-DTO.md`, `TASKS-DTO.md`.
- **Ubicación:** Siempre en `02-api/dto/`.
- **Referencia:** Desde `API-<RECURSO>.md` usando `[[backend/dto/RECURSO-DTO]]`.

# Secciones (orden fijo)

## 1. Recurso

- Recurso al que pertenece este DTO.

## 2. Request Schemas

- Cuerpos de petición (body, query, path).
- Campos, tipos y obligatoriedad.

## 3. Response Schemas

- Cuerpos de respuesta.
- Campos y tipos.

# Guía de entrevista

Para documentar un DTO, revisa el endpoint correspondiente del recurso y
cualquier documentación existente relacionada con estas secciones. Lo que
falte, preguntarlo al usuario (máx 5 preguntas a la vez).

# Checklist de calidad

- Los campos tienen tipo y obligatoriedad.
- Los tipos son coherentes con `database/`.
- Coincide con el endpoint del recurso.
