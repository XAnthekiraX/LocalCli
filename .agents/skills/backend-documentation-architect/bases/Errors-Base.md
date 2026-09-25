# Base de ERRORS.md

Plantilla de contenido para `ai/docs/backend/05-quality/ERRORS.md`.
Define el contrato que debe cumplir el documento generado, no el backend en sí.

# Reglas

- Idioma: español.
- Determinista: sin suposiciones; cada afirmación verificable.
- Terminología estable entre documentos.
- Sigue las reglas de `documentation_agent.md`.
- Usa formato wiki link `[[carpeta/ARCHIVO]]` para referencias.

# Secciones (orden fijo)

## 1. Sistema de Errores

- Formato global de respuesta de error.

## 2. Errores Conocidos

- Errores conocidos del sistema.

## 3. Códigos Internos

- Códigos de error internos y su significado.

## 4. Manejo de Excepciones

- Cómo se manejan las excepciones.

## 5. Errores por Operación

- Qué errores debe producir cada operación.

# Guía de entrevista

Para documentar ERRORS, revisa los endpoints y las reglas de negocio y
cualquier documentación existente relacionada con estas secciones. Lo que
falte, preguntarlo al usuario (máx 5 preguntas a la vez).

# Checklist de calidad

- Formato de error global definido.
- Códigos internos documentados.
- Qué errores produce cada operación esta explicitado.
