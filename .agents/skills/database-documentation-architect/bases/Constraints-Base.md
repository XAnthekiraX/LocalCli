# Base de CONSTRAINTS.md

Plantilla de contenido para `ai/docs/database/01-schema/CONSTRAINTS.md`.
Define el contrato que debe cumplir el documento generado, no la base en sí.

# Reglas

- Idioma: español.
- Determinista: sin suposiciones; cada afirmación verificable.
- Terminología estable entre documentos.
- Sigue las reglas de `documentation_agent.md`.
- Usa formato wiki link `[[carpeta/ARCHIVO]]` para referencias.

# Secciones (orden fijo)

## 1. Constraints

- NOT NULL.
- UNIQUE.
- CHECK.
- Foreign keys.

## 2. Restricciones Especiales

- Restricciones especiales, si aplican.

## 3. Reglas que la DB Impone

- Reglas que la base de datos impone a los datos.

# Guía de entrevista

Para documentar CONSTRAINTS, parte de [[database/SCHEMA]] y [[database/TABLES]]. Lo que falte,
preguntarlo al usuario (máx 5 preguntas a la vez).

# Checklist de calidad

- NOT NULL, UNIQUE, CHECK y FKs explicitados.
- Restricciones especiales documentadas.
- Reglas impuestas por la DB claras.
