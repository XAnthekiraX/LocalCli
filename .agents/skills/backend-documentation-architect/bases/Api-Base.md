# Base de API (por recurso)

Plantilla de contenido para archivos de API por recurso en `ai/docs/backend/02-api/`
(ej. `API-AUTH.md`, `API-USERS.md`). Define el contrato que debe cumplir cada
documento generado, no el backend en sí.

# Reglas

- Idioma: español.
- Determinista: sin suposiciones; cada afirmación verificable.
- Terminología estable entre documentos.
- Referencia [[database/ARCHIVO]] cuando toque datos; no duplicar.
- Sigue las reglas de `documentation_agent.md`.
- Usa formato wiki link `[[carpeta/ARCHIVO]]` para referencias.

# Convenciones de Naming

- **Un recurso = Un archivo:** `API-AUTH.md` agrupa todos los endpoints de auth.
- **DTOs por recurso:** `dto/AUTH-DTO.md` en lugar de `dto/auth.md`.
- **Sufijos:** Usar `-BACK` solo si hay conflicto con otras capas.

# Secciones (orden fijo)

## 1. Recursos

- Lista de recursos de la API (users, tasks, auth, ...).

## 2. Base y Versión

- Base path de la API.
- Versión de la API.

## 3. Convenciones

- Formato de respuestas.
- Paginación y filtros (si aplican).

## 4. Estilo

- REST, GraphQL u otro.

# Guía de entrevista

Para documentar API, revisa `PROJECT.md` y cualquier documentación existente
relacionada con la interfaz (incluidos los módulos) con estas secciones.
Lo que falte, preguntarlo al usuario (máx 5 preguntas a la vez).

# Checklist de calidad

- Los recursos están listados.
- Base path y versión definidos.
- Convenciones de respuesta/paginación explicitadas.
- No duplica datos que ya están en `database/`.
