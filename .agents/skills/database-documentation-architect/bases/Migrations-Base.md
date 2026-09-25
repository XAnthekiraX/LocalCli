# Base de MIGRATIONS.md

Plantilla de contenido para `ai/docs/database/03-operations/MIGRATIONS.md`.
Define el contrato que debe cumplir el documento generado, no la base en sí.

# Reglas

- Idioma: español.
- Determinista: sin suposiciones; cada afirmación verificable.
- Terminología estable entre documentos.
- Sigue las reglas de `documentation_agent.md`.
- Usa formato wiki link `[[carpeta/ARCHIVO]]` para referencias.

# Secciones (orden fijo)

## 1. Estrategia de Migraciones

- Cómo se gestionan las migraciones del schema.

## 2. Modificación del Schema

- Cómo se modifica el schema.

## 3. Convenciones de Nombres

- Convenciones de nombres para migraciones.

## 4. Datos Existentes

- Qué hacer con los datos existentes al migrar.

## 5. Migraciones Destructivas

- Reglas para migraciones destructivas.

# Guía de entrevista

Para documentar MIGRATIONS, parte de [[database/SCHEMA]]. Lo que falte,
preguntarlo al usuario (máx 5 preguntas a la vez).

# Checklist de calidad

- Estrategia y convenciones definidas.
- Manejo de datos existentes explicitado.
- Reglas para migraciones destructivas presentes.
