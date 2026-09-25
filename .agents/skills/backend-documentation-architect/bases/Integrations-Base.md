# Base de INTEGRATIONS.md

Plantilla de contenido para `ai/docs/backend/04-infrastructure/INTEGRATIONS.md`.
Define el contrato que debe cumplir el documento generado, no el backend en sí.

# Reglas

- Idioma: español.
- Determinista: sin suposiciones; cada afirmación verificable.
- Terminología estable entre documentos.
- Sigue las reglas de `documentation_agent.md`.
- Usa formato wiki link `[[carpeta/ARCHIVO]]` para referencias.

# Secciones (orden fijo)

## 1. APIs y Servicios Externos

- Cada integración externa del backend.
- Propósito de cada una.

## 2. Webhooks

- Webhooks entrantes o salientes, si aplican.

## 3. SDKs

- SDKs o librerías usadas para integrarse.

## 4. Credenciales y Configuración Requerida

- Qué credenciales o configuración necesita cada integración.

## 5. Contratos Externos

- Contratos o comportamiento esperado de cada servicio externo.

# Guía de entrevista

Para documentar INTEGRATIONS, revisa `PROJECT.md` (sección Integraciones)
y cualquier documentación existente relacionada con integraciones con estas
secciones. Lo que falte, preguntarlo al usuario (máx 5 preguntas a la vez).

# Checklist de calidad

- Cada integración tiene propósito y contrato.
- Credenciales documentadas sin exponer secretos.
- Webhooks y SDKs cubiertos si aplican.
