# Base de DOMAIN.md

Plantilla de contenido para `ai/docs/backend/01-domain/DOMAIN.md`.
Define el contrato que debe cumplir el documento generado, no el backend en sí.

# Reglas

- Idioma: español.
- Determinista: sin suposiciones; cada afirmación verificable.
- Terminología estable entre documentos.
- Referencia [[database/ARCHIVO]] para entidades y atributos; no duplicar.
- Sigue las reglas de `documentation_agent.md`.
- Usa formato wiki link `[[carpeta/ARCHIVO]]` para referencias.

# Secciones (orden fijo)

## 1. Módulos

- Lista de módulos del backend.
- Responsabilidad de cada uno.

## 2. Entidades del Dominio

- Entidades principales por módulo.
- Referencia a [[database/...]] para su definición detallada.

## 3. Relaciones de Dominio

- Cómo se relacionan las entidades a nivel de negocio.
- Referencia a [[database/...]] para el modelo de datos.

# Guía de entrevista

Para documentar DOMAIN, revisa `PROJECT.md` y cualquier documentación
existente relacionada con el dominio del sistema. Lo que falte, preguntarlo
al usuario (máx 5 preguntas a la vez).

# Checklist de calidad

- Los módulos están listados con su responsabilidad.
- Las entidades referencian `[[database/...]]` sin duplicar sus atributos.
- No contradice BACKEND.md.
