# Base de VALIDATION.md

Plantilla de contenido para `ai/docs/backend/05-quality/VALIDATION.md`.
Define el contrato que debe cumplir el documento generado, no el backend en sí.

# Reglas

- Idioma: español.
- Determinista: sin suposiciones; cada afirmación verificable.
- Terminología estable entre documentos.
- Referencia [[database/ARCHIVO]] para tipos; no duplicar.
- Sigue las reglas de `documentation_agent.md`.
- Usa formato wiki link `[[carpeta/ARCHIVO]]` para referencias.

# Secciones (orden fijo)

## 1. Validaciones de Entrada

- Reglas de validación por campo/recurso.

## 2. Schemas y Tipos

- Schemas de validación.
- Tipos esperados.

## 3. Campos Obligatorios y Opcionales

- Obligatoriedad por campo.

## 4. Transformaciones

- Transformaciones aplicadas a los datos de entrada.

## 5. Reglas de Validación

- Reglas cruzadas o dependientes.

# Guía de entrevista

Para documentar VALIDATION, revisa los DTOs y endpoints y cualquier
documentación existente relacionada con estas secciones. Lo que falte,
preguntarlo al usuario (máx 5 preguntas a la vez).

# Checklist de calidad

- Cada campo tiene su regla de validación.
- Obligatoriedad definida.
- Tipos coherentes con `[[database/...]]` y DTOs.
