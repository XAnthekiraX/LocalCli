# Base de BUSINESS_RULES.md

Plantilla de contenido para `ai/docs/backend/01-domain/BUSINESS_RULES.md`.
Define el contrato que debe cumplir el documento generado, no el backend en sí.

# Reglas

- Idioma: español.
- Determinista: sin suposiciones; cada afirmación verificable.
- Terminología estable entre documentos.
- Sigue las reglas de `documentation_agent.md`.
- Usa formato wiki link `[[carpeta/ARCHIVO]]` para referencias.

# Secciones (orden fijo)

## 1. Reglas de Negocio

- Reglas que rigen cada operación.
- Qué puede y qué no puede hacer cada operación.

## 2. Invariantes

- Condiciones que siempre deben cumplirse.

## 3. Casos Especiales

- Casos límite o excepciones al comportamiento normal.

# Guía de entrevista

Para documentar BUSINESS_RULES, revisa `PROJECT.md` y cualquier documentación
existente relacionada con las reglas del negocio (incluida BACKEND.md)
con estas secciones. Lo que falte, preguntarlo al usuario (máx 5 preguntas
a la vez). Este documento es clave para que la IA no deduzca el
comportamiento empresarial desde el código.

# Checklist de calidad

- Cada regla es verificable.
- No hay contradicciones con DOMAIN.md.
- Casos especiales documentados.
