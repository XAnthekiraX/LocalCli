# Base de ENUMS.md

Plantilla de contenido para `ai/docs/database/01-schema/ENUMS.md`.
Define el contrato que debe cumplir el documento generado, no la base en sí.

# Reglas

- Idioma: español.
- Determinista: sin suposiciones; cada afirmación verificable.
- Terminología estable entre documentos.
- Sigue las reglas de `documentation_agent.md`.
- Usa formato wiki link `[[carpeta/ARCHIVO]]` para referencias.

# Secciones (orden fijo)

## 1. Enums

- Todos los enums de la base de datos.

## 2. Valores Válidos

- Valores válidos de cada enum.

## 3. Significado

- Significado de cada valor.

## 4. Estados y Transiciones

- Estados y transiciones permitidas, cuando corresponda.

# Guía de entrevista

Para documentar ENUMS, parte de [[database/SCHEMA]] y [[database/TABLES]]. Lo que falte,
preguntarlo al usuario (máx 5 preguntas a la vez).

# Checklist de calidad

- Todos los enums listados con sus valores.
- Cada valor tiene significado.
- Transiciones de estado documentadas si aplican.
