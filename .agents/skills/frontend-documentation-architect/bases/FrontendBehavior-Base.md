# Base de FRONTEND-BEHAVIOR.md

Plantilla de contenido para `ai/docs/frontend/04-behavior/FRONTEND-BEHAVIOR.md`.
Define el contrato que debe cumplir el documento generado, no el frontend en sí.

# Reglas

- Idioma: español.
- Determinista: sin suposiciones; cada afirmación verificable.
- Terminología estable entre documentos.
- La regla de negocio pertenece al backend; aquí solo se documenta cómo la
  UI la refleja.
- Sigue las reglas de `documentation_agent.md`.
- Usa formato wiki link `[[carpeta/ARCHIVO]]` para referencias.

# Secciones (orden fijo)

## 1. Comportamiento de la UI

- Cómo se comporta la interfaz.

## 2. Estados Visuales

- Estados visuales de la interfaz.

## 3. Acciones Según Estado

- Acciones disponibles según el estado.

## 4. Elementos Habilitados/Deshabilitados

- Elementos según estado habilitado/deshabilitado.

## 5. Estados de Carga

- Estados de carga.

## 6. Estados Vacíos

- Estados vacíos.

## 7. Respuestas tras Operaciones

- Respuestas de la UI después de operaciones.

## 8. Actualización de Información

- Cómo se actualiza la información.

## 9. Reacción a Cambios y Reglas

- Comportamiento ante cambios y reglas del dominio reflejadas en la UI.
- Referencia a [[database/BUSINESS_RULES]] para reglas de negocio.

# Guía de entrevista

Para documentar FRONTEND-BEHAVIOR, compara `PROJECT.md` y DOMAIN con estas
secciones. Lo que falte, preguntarlo al usuario (máx 5 preguntas a la vez).

# Checklist de calidad

- Comportamiento y estados visuales definidos.
- Estados de carga/vacíos y respuestas documentados.
- Reacciones a reglas del dominio reflejadas, sin definir la regla en sí.
- Referencias a [[database/BUSINESS_RULES]] cuando se mencionen reglas.
