# Base de DATABASE.md

Plantilla de contenido para `ai/docs/database/DATABASE.md` (raíz de la
carpeta). Es el entry point de la base de datos: centraliza la información
global y el mapa de navegación.
Define el contrato que debe cumplir el documento generado, no la base en sí.

# Reglas

- Idioma: español.
- Determinista: sin suposiciones; cada afirmación verificable.
- Terminología estable entre documentos.
- Sigue las reglas de `documentation_agent.md`.
- Usa formato wiki link `[[carpeta/ARCHIVO]]` para referencias.

# Secciones (orden fijo)

## 1. Visión General

Descripción breve de la base de datos y su propósito.

## 2. Motor

- Motor de base de datos usado (PostgreSQL, SQLite, MySQL, ...).

## 3. ORM

- ORM utilizado, si aplica.

## 4. Ubicación del Schema

- Ruta donde vive el schema y los modelos.

## 5. Convenciones

- Convenciones de nombres (tablas, columnas, constraints).

## 6. Reglas Generales

- Reglas generales que la base de datos impone.

## 7. Mapa de Navegación

Guía de lectura para la IA según la tarea:

- Estructura completa → [[database/SCHEMA]], [[database/TABLES]].
- Relaciones → [[database/RELATIONSHIPS]].
- Reglas de negocio → [[database/BUSINESS_RULES]].
- Flujo de datos → [[database/DATA_FLOW]].
- Consultas críticas → [[database/QUERIES]].

Al implementar un endpoint, leer la subcadena necesaria:

```
DATABASE.md → [[database/TABLES]] → [[database/RELATIONSHIPS]] → [[database/BUSINESS_RULES]] → [[database/QUERIES]]
```

# Guía de entrevista

Para documentar DATABASE, compara `PROJECT.md` con estas secciones.
Lo que falte, preguntarlo al usuario (máx 5 preguntas a la vez).

# Checklist de calidad

- Motor, ORM y ubicación del schema definidos.
- Convenciones declaradas.
- Mapa de navegación presente.
