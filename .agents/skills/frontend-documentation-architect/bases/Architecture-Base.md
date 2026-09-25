# Base de FRONTEND.md

Plantilla de contenido para `ai/docs/frontend/FRONTEND.md`.
Define el contrato que debe cumplir el documento generado, no el frontend en sí.

# Reglas

- Idioma: español.
- Determinista: sin suposiciones; cada afirmación verificable.
- Terminología estable entre documentos.
- No incluye la arquitectura interna del backend.
- Sigue las reglas de `documentation_agent.md`.
- Usa formato wiki link `[[carpeta/ARCHIVO]]` para referencias.

# Secciones (orden fijo)

## 1. Arquitectura Utilizada

- Patrón o arquitectura del frontend.

## 2. Capas y Responsabilidades

- Capas del frontend y su responsabilidad.

## 3. Módulos Principales

- Módulos principales y su propósito.

## 4. Estructura de Carpetas

- Árbol de carpetas del frontend.

## 5. Flujo de Datos

- Cómo circulan los datos en el frontend.

## 6. Comunicación entre Módulos

- Cómo se comunican los módulos.

## 7. Gestión del Estado

- Cómo se gestiona el estado.

## 8. Comunicación con el Backend

- Cómo se comunica el frontend con el backend.
- Referencia a [[backend/API-GENERAL]] o [[backend/API-<RECURSO>]] para endpoints.

## 9. Patrones y Convenciones

- Patrones y convenciones arquitectónicas.

## 10. Dependencias Relevantes

- Dependencias principales del frontend.

# Guía de entrevista

Para documentar FRONTEND, compara `PROJECT.md` con estas secciones.
Lo que falte, preguntarlo al usuario (máx 5 preguntas a la vez).

# Checklist de calidad

- Arquitectura, capas y módulos definidos.
- Estructura de carpetas y flujo de datos claros.
- Gestión de estado y comunicación con el backend explicitadas.
- No incluye arquitectura interna del backend.
- Referencias a [[backend/...]] cuando se mencionen endpoints.
