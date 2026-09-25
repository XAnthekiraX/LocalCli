# Base de USER-FLOW

Plantilla de contenido para los archivos `ai/docs/frontend/01-user-flow/<seccion>.md`.
Define el contrato que debe cumplir cada documento generado, no el frontend en sí.

# Reglas

- Idioma: español.
- Determinista: sin suposiciones; cada afirmación verificable.
- Terminología estable entre documentos.
- Describe el flujo del usuario en el frontend, no el backend.
- Sigue las reglas de `documentation_agent.md`.
- Usa formato wiki link `[[carpeta/ARCHIVO]]` para referencias.

# Secciones (orden fijo)

## 1. Sección

- Qué sección o funcionalidad cubre este flujo.

## 2. Flujo del Usuario

- Cómo se mueve el usuario paso a paso en el frontend.
- Cada paso describe la acción y la respuesta de la interfaz.
  Ejemplo: crear una tarea → se abre un modal → se ingresan datos →
  se envían a la API.

## 3. Funciones

- Cómo funcionan las funciones concretas de esta sección.

## 4. Referencias

- Specs del project-planner o documentos asociados que se referencian.
- Referencia a [[backend/API-<RECURSO>]] cuando se consuman endpoints.

# Guía de entrevista

Para documentar un user flow, compara `PROJECT.md` y las specs con estas
secciones. Lo que falte, preguntarlo al usuario (máx 5 preguntas a la vez).

# Checklist de calidad

- Flujo paso a paso del usuario definido.
- Respuesta de la interfaz por paso explicitada.
- Referencias a specs indicadas.
- Referencias a [[backend/...]] cuando se consuman endpoints.
