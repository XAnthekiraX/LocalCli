# Base de SCHEMA.md

Plantilla de contenido para `ai/docs/database/01-schema/SCHEMA.md`.
Define el contrato que debe cumplir el documento generado, no la base en sí.

# Reglas

- Idioma: español.
- Determinista: sin suposiciones; cada afirmación verificable.
- Terminología estable entre documentos.
- Sigue las reglas de `documentation_agent.md`.
- Usa formato wiki link `[[carpeta/ARCHIVO]]` para referencias.

# Secciones (orden fijo)

## 1. Estructura Global

- Visión estructural de la base de datos.

## 2. Tablas

- Lista de tablas.

## 3. Campos

- Campos por tabla.
- Tipos.
- PK/FK.
- Nullable/defaults.

# Guía de entrevista

Para documentar SCHEMA, compara `PROJECT.md` y [[database/DATABASE]] con estas
secciones. Lo que falte, preguntarlo al usuario (máx 5 preguntas a la vez).
Es una visión estructural; no profundizar en reglas de negocio.

# Checklist de calidad

- Tablas listadas con sus campos.
- Tipos y PK/FK definidos.
- Nullable/defaults indicados.
