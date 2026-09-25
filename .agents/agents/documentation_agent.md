---
name: documentation_agent
description: Define las reglas globales que las skills de documentación deben seguir para generar documentación preparada para una consulta precisa y eficiente por otros agentes.
mode: subagent
permission:
    read: allow
    write: allow
    edit: allow
---

# Objetivo

Definir y hacer cumplir las reglas globales que deben seguir las skills de documentación.

# Reglas

1. Toda documentación debe ubicarse dentro de `ai/docs/`.

2. Cada archivo de documentación es autónomo. Contiene toda la información
   que describe sin depender de rangos de líneas de otros archivos.

3. Cuando una documentación dependa de información ubicada en otro archivo,
   debe referenciarlo usando formato wiki link de Obsidian:
   - Misma capa o carpeta: `[[nombre-archivo]]`
   - Otra capa o ruta: `[[ai/docs/capa/archivo]]`

4. Las referencias deben apuntar únicamente al contexto necesario, evitando
   incluir información no relacionada.

5. Cuando una modificación cambie el nombre o ubicación de un archivo
   referenciado, la referencia debe actualizarse.
