# Base de FRONTEND-DATA.md

Plantilla de contenido para `ai/docs/frontend/03-data/FRONTEND-DATA.md`.
Define el contrato que debe cumplir el documento generado, no el frontend en sí.

# Reglas

- Idioma: español.
- Determinista: sin suposiciones; cada afirmación verificable.
- Terminología estable entre documentos.
- No duplica DTOs, schemas ni modelos técnicos del backend.
- Referencia la documentación del backend cuando corresponde.
- Sigue las reglas de `documentation_agent.md`.
- Usa formato wiki link `[[carpeta/ARCHIVO]]` para referencias.

# Secciones (orden fijo)

## 1. Datos por Funcionalidad

- Datos que utiliza cada funcionalidad.

## 2. Datos por Vista

- Datos necesarios para cada vista.

## 3. Relaciones Relevantes para la UI

- Relaciones de datos relevantes para la interfaz.

## 4. Datos Derivados y Calculados

- Datos derivados o calculados por el frontend.

## 5. Transformaciones API → Frontend

- Transformaciones de los datos del API al frontend.
- Referencia a [[backend/dto/<RECURSO>-DTO]] para esquemas.

## 6. Datos para Renderizar

- Información necesaria para renderizar componentes.

## 7. Datos de Responsabilidad del Frontend

- Datos exclusivamente del frontend.

# Guía de entrevista

Para documentar FRONTEND-DATA, compara `PROJECT.md` con estas secciones.
Lo que falte, preguntarlo al usuario (máx 5 preguntas a la vez).

# Checklist de calidad

- Datos por vista y funcionalidad definidos.
- Transformaciones API → frontend explicitadas.
- No duplica DTOs/schemas del backend; los referencia con [[backend/...]].
