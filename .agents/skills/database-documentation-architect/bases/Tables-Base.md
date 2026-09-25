# Base de TABLES.md

Plantilla de contenido para `ai/docs/database/01-schema/TABLES.md`.
Define el contrato que debe cumplir el documento generado, no la base en sí.

# Reglas

- Idioma: español.
- Determinista: sin suposiciones; cada afirmación verificable.
- Terminología estable entre documentos.
- Sigue las reglas de `documentation_agent.md`.
- Usa formato wiki link `[[carpeta/ARCHIVO]]` para referencias.

# Secciones (orden fijo)

## 1. Tablas

Documentación detallada de cada tabla.

## 2. Propósito de la Tabla

- Para qué existe cada tabla.

## 3. Columnas

- Cada columna con:
  - Qué representa.
  - Valores permitidos.
  - Nullable/defaults.

## 4. Relaciones

- Relaciones de cada tabla.
- Referencia a [[database/RELATIONSHIPS]] para detalles.

## 5. Ejemplo Conceptual

- Ejemplo conceptual de registros.

# Guía de entrevista

Para documentar TABLES, parte de [[database/SCHEMA]] y documenta cada tabla en
detalle. Lo que falte, preguntarlo al usuario (máx 5 preguntas a la vez).

# Checklist de calidad

- Cada tabla tiene propósito documentado.
- Cada columna tiene descripción y valores permitidos.
- Relaciones y ejemplo conceptual presentes.
- Referencias a [[database/RELATIONSHIPS]] cuando aplique.
