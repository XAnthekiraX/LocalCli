---
name: context-obtainer
description: >
  Usar cuando se necesite obtener el contexto necesario para ejecutar una tarea
  (dada por el usuario o descrita en ai/tasks/<capa>/NNN-task-<nombre>.md),
  siguiendo la cadena de referencias de documentación del proyecto (Agents.md →
  FRONTEND.md/BACKEND.md/DATABASE.md → otros archivos) bajo ai/docs/. Busca
  los archivos y lee los contenidos referenciados, recursivamente, hasta agotar
  el contexto de la tarea. NO documenta: solo recolecta contexto.
---

# Context Obtainer

## Objetivo

Recolectar el contexto completo y necesario para ejecutar una tarea, siguiendo recursivamente las referencias de documentación bajo `ai/docs/`.

## Entrada

La tarea, dada por el usuario o descrita en `ai/tasks/<capa>/NNN-task-<nombre>.md`.

## Proceso

1. Ubica `Agents.md` con `find` (nunca asumas su ubicación).
2. Léelo; identifica las referencias en formato wiki link (`[[carpeta/ARCHIVO]]`).
3. Para cada wiki link, resuelve la ruta y lee el archivo completo.
4. Sigue recursivamente cada referencia que este o cualquier archivo contenga, **filtrando por relevancia a la tarea**.
5. Cada archivo es autónomo; léelo completo (no hay rangos de líneas que extraer).
6. Repite hasta agotar las referencias relevantes a la tarea.

## Reglas

- Usa `find` para localizar archivos (nada hardcodeado).
- Lee archivos completos; no extraes rangos parciales.
- Respeta el formato wiki link `[[carpeta/ARCHIVO]]` definido por `documentation_agent`.
- Sigue recursivamente las referencias relevantes a la tarea.
- Solo lectura: nunca modifiques la documentación.

## Restricciones

- No hardcodees nombres de archivos o rutas (sigue lo que exista realmente).
- No inventes referencias que no estén en los archivos.
- Esta skill NO documenta; solo recolecta contexto.
