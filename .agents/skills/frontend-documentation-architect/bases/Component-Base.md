# Base de COMPONENTE

Plantilla de contenido para los archivos `ai/docs/frontend/02-components/<COMPONENTE>.md`.
Define el contrato que debe cumplir cada componente documentado. **Este documento es opcional.**
Solo se crea si el usuario indica que hay componentes personalizados que la IA no puede generar.

# Reglas

- Idioma: español.
- Determinista: sin suposiciones; cada afirmación verificable.
- Terminología estable entre documentos.
- Sigue las reglas de `documentation_agent.md`.
- Usa formato wiki link `[[carpeta/ARCHIVO]]` para referencias.

# Secciones (orden fijo)

## 1. Nombre del Componente

- Nombre del componente.

## 2. Objetivo

- Para qué se usa el componente y qué resuelve.

## 3. Código del Componente

- Código del diseño personalizado del componente (no se debe generar
  automáticamente; la IA solo lo referencia).

## 4. Uso

- Cómo se usa el componente, props o variantes relevantes, si aplican.

# Guía de entrevista

Para documentar un componente, pregunta al usuario el nombre, objetivo y
código del diseño personalizado. Lo que falte se pregunta (máx 5 preguntas
a la vez).

# Checklist de calidad

- Nombre y objetivo definidos.
- Código del componente presente.
- La IA no lo genera; solo lo referencia.
