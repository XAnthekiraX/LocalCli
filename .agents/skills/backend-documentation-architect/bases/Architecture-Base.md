# Base de BACKEND.md

Plantilla de contenido para `ai/docs/backend/BACKEND.md` (raíz de la
carpeta). Es el entry point del backend: centraliza y da contexto global.
Define el contrato que debe cumplir el documento generado, no el backend en sí.

# Reglas

- Idioma: español.
- Determinista: sin suposiciones; cada afirmación verificable.
- Terminología estable entre documentos.
- Referencia [[database/ARCHIVO]] cuando toque datos; no duplicar.
- Sigue las reglas de `documentation_agent.md`.
- Usa formato wiki link `[[carpeta/ARCHIVO]]` para referencias.

# Secciones (orden fijo)

## 1. Visión General

Descripción breve de cómo está dividido el sistema y cuáles son sus
partes principales.

## 2. Estructura del Proyecto

Árbol simplificado de las carpetas y módulos principales.

## 3. Responsabilidades de los Componentes

Qué responsabilidad tiene cada módulo o capa principal.

## 4. Relaciones entre Componentes

Cómo se comunican las partes principales del sistema.

# Stack

- Lenguaje y versión.
- Framework.
- ORM.
- Base de datos.
- Otras dependencias principales.

# Guía de entrevista

Para documentar ARCHITECTURE, revisa `PROJECT.md` y cualquier documentación
existente relacionada con la arquitectura del sistema. Lo que falte,
preguntarlo al usuario (máx 5 preguntas a la vez).

# Checklist de calidad

- Las 4 secciones tienen contenido (no en blanco).
- El stack está definido.
- Los nombres de módulos son estables.
- No duplica datos que ya están en `[[database/...]]`.

# Mapa de Navegación

Guía de lectura para la IA según la tarea:

- Estructura completa → [[database/SCHEMA]], [[database/TABLES]].
- Relaciones → [[database/RELATIONSHIPS]].
- Reglas de negocio → [[database/BUSINESS_RULES]].
- Flujo de datos → [[database/DATA_FLOW]].
- Consultas críticas → [[database/QUERIES]].

Al implementar un endpoint, leer la subcadena necesaria:

```
BACKEND.md → [[database/TABLES]] → [[database/RELATIONSHIPS]] → [[database/BUSINESS_RULES]] → [[database/QUERIES]]
```
