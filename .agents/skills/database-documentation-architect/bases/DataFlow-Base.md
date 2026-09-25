# Base de DATA_FLOW.md

Plantilla de contenido para `ai/docs/database/02-rules/DATA_FLOW.md`.
Define el contrato que debe cumplir el documento generado, no la base en sí.

# Reglas

- Idioma: español.
- Determinista: sin suposiciones; cada afirmación verificable.
- Terminología estable entre documentos.
- Sigue las reglas de `documentation_agent.md`.
- Usa formato wiki link `[[carpeta/ARCHIVO]]` para referencias.

# Secciones (orden fijo)

## 1. Circulación de Datos

- Cómo circulan los datos: Request → Backend → Validation → DB → Response.

## 2. Creación

- Flujo de creación de datos.

## 3. Actualización

- Flujo de actualización de datos.

## 4. Eliminación

- Flujo de eliminación de datos.

## 5. Transacciones

- Transacciones y su uso.

## 6. Relaciones entre Operaciones

- Relaciones entre operaciones y su orden.

# Guía de entrevista

Para documentar DATA_FLOW, parte de PROJECT.md y [[database/BUSINESS_RULES]].
Lo que falte, preguntarlo al usuario (máx 5 preguntas a la vez).

# Checklist de calidad

- Ciclo completo del dato documentado.
- Creación, actualización y eliminación cubiertas.
- Transacciones y relaciones entre operaciones explicitadas.
