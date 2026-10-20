---
title: SPEC — Operaciones de archivos
tags: [specs, requisito]
depende_de:
  - "[[IDEA]]"
  - "[[specs/SPEC-SESIONES]]"
relacionado:
  - "[[specs/SPEC-TOOLS]]"
  - "[[specs/SPEC-AGENTE-BASE]]"
  - "[[specs/SPEC-INTERFAZ-ATAJOS]]"
---
# SPEC — Operaciones de archivos

Prioridad: P0 (núcleo)

## Propósito

Todo lo que el agente puede hacer sobre archivos y carpetas, y cuándo tiene que pedirte permiso.

## Alcance

Incluye crear, leer, editar y borrar archivos y carpetas en cualquier formato de texto, y las reglas de permiso.
No incluye la forma de mostrar el panel de aprobaciones, que está en [[specs/SPEC-INTERFAZ-ATAJOS]], ni el catálogo de herramientas, que está en [[specs/SPEC-TOOLS]].

## Actores

- **Usuario**: aprueba o declina.
- **Agente**: pide la operación y explica qué pretende hacer.
- **Sistema**: comprueba, aplica y registra.

## Flujo principal

1. El agente pide una operación con la ruta y el contenido.
2. El sistema comprueba si la ruta está dentro de la carpeta abierta.
3. Dentro de la carpeta: se prepara el cambio y se aplica sin preguntar, salvo que sea un borrado.
4. Fuera de la carpeta: se pide aprobación y además el agente explica por qué busca eso.
5. El usuario aprueba o declina.
6. Aprobado: se aplica y se devuelve el resultado al agente.

## Flujos alternativos

- Declinado: el agente recibe el rechazo y propone otra cosa.
- La ruta no existe.
- El archivo ya existe con contenido y se intenta escribir entero: no se toca, y el agente recibe el motivo para editar o para borrar y crear.
- Borrado: pide confirmación explícita, no solo aprobación genérica.
- Operación fuera de la carpeta: además de la aprobación exige la explicación.

## Reglas de negocio

- Dentro de la carpeta abierta, todos los archivos y carpetas son accesibles sin preguntar.
- Fuera de la carpeta, hace falta permiso **y** la explicación del agente sobre por qué busca eso.
- **Solo piden confirmación dos cosas: lo que queda fuera de la carpeta y lo que es destructivo.** Crear, rellenar un archivo vacío o editar dentro de la carpeta no interrumpe al usuario, también dentro de un flujo en curso.
- **Escribir entero nunca pisa contenido existente.** `escribir_archivo` solo escribe archivos vacíos o inexistentes; si el archivo ya tiene contenido, no se toca nada y el agente recibe un error corregible que lo remite a `editar_archivo` (o a borrar y crear, que pide confirmación explícita).
- Borrar siempre requiere confirmación explícita, aunque la ruta esté dentro de la carpeta.
- Editar es por sustitución parcial: cada fragmento debe coincidir **exactamente una vez**; si no está o es ambiguo, no se escribe nada y el agente recibe el motivo. La comparación tolera finales de línea CRLF, BOM, Unicode tipográfico (comillas, guiones, elipsis, NBSP) y espacios finales o de sangría, y el resultado conserva el estilo del archivo.
- Una ruta relativa se resuelve desde el directorio de trabajo del harness; `..` y las rutas absolutas son válidas. No se expande `~` ni `$HOME` y no se busca en otras ubicaciones. La normalización no elude las restricciones: una ruta que resuelve fuera de la carpeta cuenta como «fuera».
- Cada cambio aplicado queda registrado con lo que había antes y lo que quedó.
- **Cada cambio aplicado deja ver el antes y el después.** El hilo muestra bajo la línea de la herramienta la diferencia del archivo —las líneas quitadas y las añadidas—, recortada con aviso si es larga. El bloque es de pantalla: no entra al contexto del modelo ni sustituye al registro completo de `change_history`. Un cambio que no se aplica no deja diferencia.
- Nada se borra de la base de datos: el registro de cambios se conserva para poder revertir y auditar.
- Solo `build` tiene herramientas de escritura. `plan` no puede aplicarlas ni siquiera con permiso.
- La herramienta de terminal no puede escribir en el proyecto, así que no sirve para saltarse estas reglas.

## Criterios de aceptación

- [ ] El agente puede crear, leer, editar y borrar archivos y carpetas dentro del proyecto.
- [ ] Funciona con cualquier archivo de texto, sin importar la extensión.
- [ ] Un cambio dentro de la carpeta que no es un borrado se aplica sin interrumpir al usuario.
- [ ] Un cambio fuera de la carpeta no se aplica sin aprobación y sin la explicación del agente.
- [ ] El usuario puede declinar y el agente recibe el rechazo.
- [ ] Todo cambio aplicado queda registrado con el antes y el después.
- [ ] Todo cambio aplicado deja ver su diferencia bajo la línea de la herramienta; un cambio no aplicado no deja diferencia.
- [ ] Escribir entero un archivo que ya tiene contenido no modifica nada y el agente recibe un error corregible que nombra `editar_archivo`.
- [ ] Borrar un archivo exige confirmación explícita, esté donde esté.
- [ ] Editar un archivo con un fragmento ambiguo o ausente no modifica nada y el agente recibe un error corregible.
- [ ] Editar funciona en archivos CRLF, con BOM y con texto Unicode tipográfico sin que el agente tenga que compensarlo.
- [ ] Si el agente busca fuera de la carpeta, la explicación queda visible para el usuario.
- [ ] `plan` no puede escribir ningún archivo.
- [ ] La terminal no puede usarse para escribir archivos y saltarse la aprobación.

## Requisitos no funcionales

- Ninguna escritura destructiva se aplica sin aprobación, sin excepción.
- Ninguna operación fuera de la carpeta se aplica sin aprobación.
- El registro de cambios no se modifica ni se borra una vez escrito.

## Dependencias funcionales

- [[specs/SPEC-SESIONES]]

## Referencias

- [[IDEA]]
