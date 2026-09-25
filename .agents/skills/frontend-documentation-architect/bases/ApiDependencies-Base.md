# Base de API-DEPENDENCIES.md

Plantilla de contenido para `ai/docs/frontend/09-api-dependencies/API-DEPENDENCIES.md`.
Define el contrato que debe cumplir el documento generado, no el frontend en sí.

# Reglas

- Idioma: español.
- Determinista: sin suposiciones; cada afirmación verificable.
- Terminología estable entre documentos.
- La API del propio backend no se documenta aquí si ya existe su contrato
  en la documentación del backend.
- Sigue las reglas de `documentation_agent.md`.
- Usa formato wiki link `[[carpeta/ARCHIVO]]` para referencias.

# Secciones (orden fijo)

## 1. APIs Externas y Servicios de Terceros

- Servicios externos utilizados directamente por la aplicación.

## 2. Propósito de Cada Integración

- Para qué se usa cada integración.

## 3. Features que las Utilizan

- Funcionalidades que usan cada integración.

## 4. Datos Enviados

- Datos enviados a cada servicio.

## 5. Datos Recibidos

- Datos recibidos de cada servicio.

## 6. Configuración Requerida

- Configuración necesaria.

## 7. Restricciones

- Restricciones de cada integración.

## 8. Dependencias

- Dependencias relacionadas.

## 9. Endpoints del Backend Utilizados

- Referencia a los endpoints del backend que consume el frontend.
- Formato: [[backend/API-<RECURSO>]] para cada endpoint.

# Guía de entrevista

Para documentar API-DEPENDENCIES, compara `PROJECT.md` (integraciones) con
estas secciones. Lo que falte, preguntarlo al usuario (máx 5 preguntas
a la vez).

# Checklist de calidad

- Integraciones externas listadas con su propósito.
- Datos enviados/recibidos y configuración documentados.
- No duplica el contrato del propio backend.
- Referencias a [[backend/API-...]] para endpoints utilizados.
