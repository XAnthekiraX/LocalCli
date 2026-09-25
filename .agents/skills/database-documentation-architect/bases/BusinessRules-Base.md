# Base de BUSINESS_RULES.md

Plantilla de contenido para `ai/docs/database/02-rules/BUSINESS_RULES.md`.
Define el contrato que debe cumplir el documento generado, no la base en sí.

# Reglas

- Idioma: español.
- Determinista: sin suposiciones; cada afirmación verificable.
- Terminología estable entre documentos.
- Sigue las reglas de `documentation_agent.md`.
- Usa formato wiki link `[[carpeta/ARCHIVO]]` para referencias.

# Secciones (orden fijo)

## 1. Reglas de Negocio

- Reglas que no están necesariamente expresadas por la base de datos.
- Ejemplo: una factura no puede modificarse después de ser emitida.

## 2. Invariantes

- Condiciones que siempre deben cumplirse.
- Ejemplo: una tarea padre solo puede completarse cuando sus
  subtareas están completas.

## 3. Casos Especiales

- Casos límite o excepciones.

# Guía de entrevista

Para documentar BUSINESS_RULES, parte de PROJECT.md, [[database/SCHEMA]] y
[[database/TABLES]]. Lo que falte, preguntarlo al usuario (máx 5 preguntas a la vez).

Este documento es clave para que la IA no genere un backend técnicamente
válido pero funcionalmente incorrecto.

# Checklist de calidad

- Cada regla es verificable.
- No hay contradicciones con [[database/SCHEMA]].
- Casos especiales documentados.
