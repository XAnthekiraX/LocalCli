# Base de CONFIGURATION.md

Plantilla de contenido para `ai/docs/backend/04-infrastructure/CONFIGURATION.md`.
Define el contrato que debe cumplir el documento generado, no el backend en sí.

# Reglas

- Idioma: español.
- Determinista: sin suposiciones; cada afirmación verificable.
- Terminología estable entre documentos.
- Sigue las reglas de `documentation_agent.md`.
- Usa formato wiki link `[[carpeta/ARCHIVO]]` para referencias.

# Secciones (orden fijo)

## 1. Variables de Entorno

- Variables de entorno del backend.
- Nombre, propósito y obligatoriedad.

## 2. Configuración por Ambiente

- Desarrollo.
- Pruebas.
- Producción.

## 3. Servicios Requeridos

- Servicios externos que el backend necesita para operar.

# Guía de entrevista

Para documentar CONFIGURATION, revisa `PROJECT.md` y cualquier documentación
existente relacionada con la configuración (incluida ARCHITECTURE) con
estas secciones. Lo que falte, preguntarlo al usuario (máx 5 preguntas
a la vez).

# Checklist de calidad

- Variables de entorno listadas con su propósito.
- Configuración por ambiente diferenciada.
- No contiene secretos en texto plano; solo referencia su uso.
