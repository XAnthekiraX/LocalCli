# Base de VALIDATION.md

Plantilla de contenido para `ai/docs/frontend/06-validation/VALIDATION.md`.
Define el contrato que debe cumplir el documento generado, no el frontend en sí.

# Reglas

- Idioma: español.
- Determinista: sin suposiciones; cada afirmación verificable.
- Terminología estable entre documentos.
- La validación definitiva y autoritativa pertenece al backend.
- Sigue las reglas de `documentation_agent.md`.
- Usa formato wiki link `[[carpeta/ARCHIVO]]` para referencias.

# Secciones (orden fijo)

## 1. Validaciones Inmediatas de UI

- Validaciones inmediatas de la interfaz.

## 2. Campos Obligatorios para UX

- Campos obligatorios desde la experiencia de usuario.

## 3. Límites que Conoce la UI

- Límites que la interfaz debe conocer.

## 4. Formatos

- Formatos esperados.

## 5. Mensajes de Validación

- Mensajes de validación mostrados.

## 6. Validación Antes de Enviar

- Validaciones antes de enviar datos.
- Referencia a [[backend/dto/<RECURSO>-DTO]] para validaciones del servidor.

## 7. Comportamiento ante Datos Inválidos

- Comportamiento de la UI ante datos inválidos.

# Guía de entrevista

Para documentar VALIDATION, compara `PROJECT.md` y los user flows con estas
secciones. Lo que falte, preguntarlo al usuario (máx 5 preguntas a la vez).

# Checklist de calidad

- Validaciones de UI y formatos definidos.
- Mensajes de validación documentados.
- Se distingue que la validación autoritativa es del backend.
- Referencias a [[backend/dto/...]] cuando se mencionen validaciones del servidor.
