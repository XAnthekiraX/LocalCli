# Base de ERRORS.md

Plantilla de contenido para `ai/docs/frontend/07-errors/ERRORS.md`.
Define el contrato que debe cumplir el documento generado, no el frontend en sí.

# Reglas

- Idioma: español.
- Determinista: sin suposiciones; cada afirmación verificable.
- Terminología estable entre documentos.
- El backend define el contrato del error; el frontend define qué hace con él.
- Sigue las reglas de `documentation_agent.md`.
- Usa formato wiki link `[[carpeta/ARCHIVO]]` para referencias.

# Secciones (orden fijo)

## 1. Tipos de Errores

- Tipos de errores que puede recibir el frontend.
- Referencia a [[backend/ERRORS]] para el contrato de errores del backend.

## 2. Comportamiento por Error

- Comportamiento de la UI para cada error.

## 3. Mensajes al Usuario

- Mensajes mostrados al usuario.

## 4. Errores Recuperables

- Errores recuperables y cómo se recuperan.

## 5. Errores No Recuperables

- Errores no recuperables y su manejo.

## 6. Reintentos

- Reintentos y su lógica.

## 7. Redirecciones

- Redirecciones al ocurrir errores.

## 8. Estados de Error

- Estados de error de la interfaz.

# Guía de entrevista

Para documentar ERRORS, compara `PROJECT.md` y los user flows con estas
secciones. Lo que falte, preguntarlo al usuario (máx 5 preguntas a la vez).

# Checklist de calidad

- Tipos de errores y comportamiento por error definidos.
- Recuperables/no recuperables y reintentos documentados.
- Estados de error de la UI explicitados.
- Referencias a [[backend/ERRORS]] para el contrato de errores.
