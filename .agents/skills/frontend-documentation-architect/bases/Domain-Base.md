# Base de DOMAIN.md

Plantilla de contenido para `ai/docs/frontend/DOMAIN.md`.
Define el contrato que debe cumplir el documento generado, no el frontend en sí.

# Reglas

- Idioma: español.
- Determinista: sin suposiciones; cada afirmación verificable.
- Terminología estable entre documentos.
- No duplica modelos técnicos del backend.
- Sigue las reglas de `documentation_agent.md`.
- Usa formato wiki link `[[carpeta/ARCHIVO]]` para referencias.

# Secciones (orden fijo)

## 1. Conceptos Principales

- Conceptos del sistema desde la perspectiva del producto.

## 2. Entidades Conceptuales

- Entidades conceptuales (sin DTOs ni schemas).

## 3. Relaciones

- Relaciones entre entidades conceptuales.

## 4. Jerarquías

- Jerarquías del dominio, si aplican.

## 5. Terminología

- Terminología utilizada por el producto.

## 6. Conceptos Relevantes para la UI

- Conceptos relevantes para la interfaz.

# Guía de entrevista

Para documentar DOMAIN, compara `PROJECT.md` con estas secciones.
Lo que falte, preguntarlo al usuario (máx 5 preguntas a la vez).

# Checklist de calidad

- Conceptos y entidades conceptuales definidos.
- Relaciones y jerarquías claras.
- Terminología del producto documentada.
- No incluye DTOs, schemas ni tablas del backend.
- Cuando se mencionen entidades del backend, referenciar [[backend/...]].
