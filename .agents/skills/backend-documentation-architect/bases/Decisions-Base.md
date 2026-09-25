# Base de DECISIONS.md

Plantilla de contenido para `ai/docs/backend/DECISIONS.md` (raíz de la carpeta).
Define el contrato que debe cumplir el documento generado, no el backend en sí.

# Reglas

- Idioma: español.
- Determinista: sin suposiciones; cada afirmación verificable.
- Terminología estable entre documentos.
- Sigue las reglas de `documentation_agent.md`.
- Usa formato wiki link `[[carpeta/ARCHIVO]]` para referencias.

# Secciones (orden fijo)

## 1. Decisiones Confirmadas

- Decisiones técnicas y de diseño ya tomadas por el usuario.
- Una por ítem, con su justificación breve.
- Incluye decisiones que afectan al backend sin ser estrictamente
  arquitectónicas, por ejemplo: usar JWT, usar soft delete, versionar la
  API, devolver errores con un formato determinado, usar transacciones en
  determinadas operaciones.
- Cuando una decisión confirmada esté respaldada por código o documentación
  existente, debe indicarse su fuente con un wiki link `[[archivo]]` siguiendo
  las reglas de `documentation_agent.md` para que sea rastreable.

## 2. Decisiones Pendientes

- Decisiones aún no resueltas.
- Una por ítem, redactada como pregunta si aplica.
- Indicar qué decisión debe tomarse y qué información falta para resolverla.
- Ejemplo: "¿La API utilizará JWT o sesiones? Falta confirmar el mecanismo
  de autenticación."

# Guía de entrevista

Para documentar DECISIONS, revisa `PROJECT.md` y cualquier documentación
existente relacionada con arquitectura y decisiones. Identifica las decisiones
explícitamente confirmadas y separa aquellas que no puedan verificarse.
Lo que falte para completar las decisiones debe preguntarse al usuario
(máx. 5 preguntas a la vez).

# Checklist de calidad

- No presenta supuestos como decisiones confirmadas.
- Hay una lista de confirmadas y otra de pendientes.
- Cada decisión tiene justificación o está marcada como pendiente.
- Cada decisión confirmada respaldada por código o documentación indica su
  fuente con wiki link `[[archivo]]`; en caso contrario se marca como pendiente de verificar.
- Cada decisión pendiente indica qué información falta para poder resolverla.
