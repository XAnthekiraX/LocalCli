# Base de RELATIONSHIPS.md

Plantilla de contenido para `ai/docs/database/01-schema/RELATIONSHIPS.md`.
Define el contrato que debe cumplir el documento generado, no la base en sí.

# Reglas

- Idioma: español.
- Determinista: sin suposiciones; cada afirmación verificable.
- Terminología estable entre documentos.
- Sigue las reglas de `documentation_agent.md`.
- Usa formato wiki link `[[carpeta/ARCHIVO]]` para referencias.

# Secciones (orden fijo)

## 1. Relaciones

- Relaciones entre entidades: 1:1, 1:N, N:N.

## 2. Foreign Keys

- Foreign keys entre tablas.

## 3. Cascadas

- Comportamiento de cascada en las relaciones.

## 4. Dependencias

- Qué entidades dependen de otras.

# Guía de entrevista

Para documentar RELATIONSHIPS, parte de [[database/SCHEMA]] y [[database/TABLES]] y define
cada relación. Lo que falte, preguntarlo al usuario (máx 5 preguntas
a la vez).

# Checklist de calidad

- Cada relación indica cardinalidad.
- FKs y cascadas documentadas.
- Dependencias entre entidades claras.
