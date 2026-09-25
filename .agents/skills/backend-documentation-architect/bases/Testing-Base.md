# Base de TESTING.md

Plantilla de contenido para `ai/docs/backend/05-quality/TESTING.md`.
Define el contrato que debe cumplir el documento generado, no el backend en sí.

# Reglas

- Idioma: español.
- Determinista: sin suposiciones; cada afirmación verificable.
- Terminología estable entre documentos.
- Sigue las reglas de `documentation_agent.md`.
- Usa formato wiki link `[[carpeta/ARCHIVO]]` para referencias.

# Secciones (orden fijo)

## 1. Estrategia de Testing

- Unit.
- Integration.
- E2E.

## 2. Qué debe probarse

- Cobertura esperada por módulo o funcionalidad.

## 3. Fixtures y Mocks

- Fixtures disponibles.
- Mocks a usar.

## 4. Reglas para Nuevos Tests

- Convenciones al añadir tests.

# Guía de entrevista

Para documentar TESTING, revisa `PROJECT.md` y cualquier documentación
existente relacionada con testing (incluidos los módulos) con estas secciones.
Lo que falte, preguntarlo al usuario (máx 5 preguntas a la vez).

# Checklist de calidad

- Estrategia por nivel definida.
- Qué se prueba está explicitado.
- Convenciones para nuevos tests documentadas.
