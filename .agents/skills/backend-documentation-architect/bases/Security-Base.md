# Base de SECURITY.md

Plantilla de contenido para `ai/docs/backend/03-security/SECURITY.md`.
Define el contrato que debe cumplir el documento generado, no el backend en sí.

# Reglas

- Idioma: español.
- Determinista: sin suposiciones; cada afirmación verificable.
- Terminología estable entre documentos.
- Sigue las reglas de `documentation_agent.md`.
- Usa formato wiki link `[[carpeta/ARCHIVO]]` para referencias.

# Secciones (orden fijo)

## 1. Autenticación

- Cómo se autentica el cliente (tokens, sesiones, OAuth, API keys).

## 2. Autorización

- Roles y permisos.
- Recurso/operación por rol.

## 3. Protección de Endpoints

- Endpoints protegidos y regla de acceso.

## 4. Sanitización y Datos Sensibles

- Sanitización de entradas.
- Manejo de datos sensibles (PII).

## 5. Riesgos Relevantes

- Rate limiting.
- CORS.
- Secrets.
- Amenazas relevantes.

# Guía de entrevista

Para documentar SECURITY, revisa `PROJECT.md` y cualquier documentación
existente relacionada con seguridad (incluidos los endpoints) con estas
secciones. Lo que falte, preguntarlo al usuario (máx 5 preguntas a la vez).

# Checklist de calidad

- Autenticación y autorización definidas.
- Endpoints protegidos listados.
- Datos sensibles identificados.
- No duplica reglas de `[[database/SECURITY]]`; las referencia.
