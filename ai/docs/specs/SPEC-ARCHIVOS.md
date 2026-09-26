---
title: SPEC — Operaciones de archivos
tags: [specs, requisito]
depende_de:
  - "[[IDEA]]"
  - "[[specs/SPEC-AGENTE-BASE]]"
  - "[[specs/SPEC-INTERFAZ-ATAJOS]]"
  - "[[specs/SPEC-SESIONES]]"
relacionado:
  - "[[specs/SPEC-TOOLS]]"
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
3. Dentro de la carpeta: se prepara el cambio y se pide aprobación.
4. Fuera de la carpeta: se pide aprobación y además el agente explica por qué busca eso.
5. El usuario aprueba o declina.
6. Aprobado: se aplica y se devuelve el resultado al agente.

## Flujos alternativos

- Declinado: el agente recibe el rechazo y propone otra cosa.
- La ruta no existe.
- El archivo ya existe y se va a sobrescribir.
- Borrado: pide confirmación explícita, no solo aprobación genérica.
- Operación fuera de la carpeta: además de la aprobación exige la explicación.

## Reglas de negocio

- Dentro de la carpeta abierta, todos los archivos y carpetas son accesibles sin preguntar.
- Fuera de la carpeta, hace falta permiso **y** la explicación del agente sobre por qué busca eso.
- Toda escritura requiere aprobación, también dentro de la carpeta y también dentro de un flujo en curso.
- Borrar siempre requiere confirmación explícita.
- Toda ruta es relativa a la carpeta abierta del proyecto.
- Cada cambio aplicado queda registrado con lo que había antes y lo que quedó.
- Nada se borra de la base de datos: el registro de cambios se conserva para poder revertir y auditar.
- Solo `build` tiene herramientas de escritura. `plan` no puede aplicarlas ni siquiera con permiso.
- La herramienta de terminal no puede escribir en el proyecto, así que no sirve para saltarse estas reglas.

## Criterios de aceptación

- [ ] El agente puede crear, leer, editar y borrar archivos y carpetas dentro del proyecto.
- [ ] Funciona con cualquier archivo de texto, sin importar la extensión.
- [ ] Un cambio dentro de la carpeta no se aplica hasta que el usuario lo aprueba.
- [ ] Un cambio fuera de la carpeta no se aplica sin aprobación y sin la explicación del agente.
- [ ] El usuario puede declinar y el agente recibe el rechazo.
- [ ] Todo cambio aplicado queda registrado con el antes y el después.
- [ ] Borrar un archivo exige confirmación explícita.
- [ ] Si el agente busca fuera de la carpeta, la explicación queda visible para el usuario.
- [ ] `plan` no puede escribir ningún archivo.
- [ ] La terminal no puede usarse para escribir archivos y saltarse la aprobación.

## Requisitos no funcionales

- Ninguna escritura se aplica sin aprobación, sin excepción.
- El registro de cambios no se modifica ni se borra una vez escrito.

## Dependencias funcionales

- [[specs/SPEC-AGENTE-BASE]]
- [[specs/SPEC-INTERFAZ-ATAJOS]]
- [[specs/SPEC-SESIONES]]

## Referencias

- [[IDEA]]
