# Base de EVENTS.md

Plantilla de contenido para `ai/docs/backend/04-infrastructure/EVENTS.md`.
Define el contrato que debe cumplir el documento generado, no el backend en sí.

# Reglas

- Idioma: español.
- Determinista: sin suposiciones; cada afirmación verificable.
- Terminología estable entre documentos.
- Sigue las reglas de `documentation_agent.md`.
- Usa formato wiki link `[[carpeta/ARCHIVO]]` para referencias.

# Secciones (orden fijo)

## 1. Eventos Disponibles

- Lista de eventos del backend.

## 2. Productores y Consumidores

- Quién produce cada evento.
- Quién consume cada evento.

## 3. Payloads

- Estructura del payload de cada evento.

## 4. Sincronía y Consistencia

- Eventos síncronos o asíncronos.
- Orden y consistencia esperada.

# Guía de entrevista

Para documentar EVENTS, revisa `PROJECT.md` y cualquier documentación
existente relacionada con eventos (incluidos los flujos) con estas secciones.
Lo que falte, preguntarlo al usuario (máx 5 preguntas a la vez).
Este documento solo se crea si el backend utiliza eventos.

# Checklist de calidad

- Cada evento indica productor, consumidor y payload.
- Sincronía/asincronía y consistencia documentadas.
- Solo se crea si el backend usa eventos.
