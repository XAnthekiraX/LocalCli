---
title: LocalCli — payloads de las herramientas
tags: [backend, interfaces]
depende_de:
  - "[[backend/02-interfaces/TOOLS]]"
  - "[[specs/SPEC-TOOLS]]"
relacionado:
  - "[[backend/05-quality/VALIDATION]]"
  - "[[backend/01-domain/DOMAIN]]"
  - "[[backend/05-quality/ERRORS]]"
  - "[[database/01-schema/TABLES]]"
---
# TOOLS-DTO — Payloads de las herramientas

Los cuerpos de petición y respuesta de cada herramienta. `LocalCli` no tiene HTTP, así que esto no son cuerpos JSON: son los argumentos que el modelo pasa a la herramienta y lo que esta devuelve. Los tipos son los de Go en la frontera. Ver [[backend/02-interfaces/TOOLS]] para el comportamiento y [[backend/05-quality/VALIDATION]] para las reglas de validación.

## 1. Recurso

Pertenece al recurso **herramientas del agente**, que es la superficie que consume el modelo a través de `agent` y `tools`. Ver [[specs/SPEC-TOOLS]].

## 2. Request Schemas

Los argumentos que el modelo envía. Todos obligatorios salvo lo marcado.

### Archivos de lectura

| Herramienta | Campo | Tipo | Obligatorio | Notas |
|---|---|---|---|---|
| `leer_archivo` | `ruta` | string | Sí | Relativa a la carpeta del proyecto |
| `listar_carpeta` | `ruta` | string | Sí | Relativa; un nivel |
| `buscar_archivos` | `patron` | string | Sí | Coincide contra nombres de archivo |
| `buscar_en_archivos` | `patron` | string | Sí | Coincide contra el contenido |
| `buscar_en_archivos` | `ruta` | string | No | Limita la búsqueda a una subcarpeta |

### Archivos de escritura (solo `build`)

| Herramienta | Campo | Tipo | Obligatorio | Notas |
|---|---|---|---|---|
| `crear_archivo` | `ruta` | string | Sí | Falla si ya existe |
| `crear_archivo` | `contenido` | string | Sí | Texto plano, cualquier extensión |
| `escribir_archivo` | `ruta` | string | Sí | Sobrescribe |
| `escribir_archivo` | `contenido` | string | Sí | Reemplaza el contenido entero |
| `editar_archivo` | `ruta` | string | Sí | |
| `editar_archivo` | `cambio` | string | Sí | La edición a aplicar |
| `eliminar_archivo` | `ruta` | string | Sí | Pide confirmación explícita |
| `crear_carpeta` | `ruta` | string | Sí | Falla si ya existe |
| `eliminar_carpeta` | `ruta` | string | Sí | Pide confirmación explícita |

### Terminal

| Herramienta | Campo | Tipo | Obligatorio | Notas |
|---|---|---|---|---|
| `ejecutar_comando` | `comando` | string | Sí | Se valida contra la lista blanca, no contra su texto |
| `ejecutar_comando` | `carpeta` | string | No | Por defecto, la carpeta del proyecto |

### Internet

| Herramienta | Campo | Tipo | Obligatorio | Notas |
|---|---|---|---|---|
| `buscar_en_internet` | `consulta` | string | Sí | Lo único que sale de la máquina |
| `abrir_pagina` | `direccion` | string | Sí | URL de la página a abrir |

## 3. Response Schemas

Lo que cada herramienta devuelve al agente. El agente lo ve en su respuesta, así que el contenido importa: va al mismo presupuesto de contexto que el resto.

### Lectura de archivos

| Herramienta | Devuelve | Tipo |
|---|---|---|
| `leer_archivo` | El contenido del archivo | string |
| `listar_carpeta` | Las entradas de un nivel | lista de nombres |
| `buscar_archivos` | Las rutas que coinciden | lista de rutas |
| `buscar_en_archivos` | Las coincidencias | lista de (archivo, línea, fragmento) |

### Escritura de archivos

| Herramienta | Devuelve | Tipo |
|---|---|---|
| `crear_archivo` | Confirmación y la ruta | string |
| `escribir_archivo` | Confirmación y la ruta | string |
| `editar_archivo` | Confirmación y la ruta | string |
| `eliminar_archivo` | Confirmación | string |
| `crear_carpeta` | Confirmación y la ruta | string |
| `eliminar_carpeta` | Confirmación | string |

Toda escritura aprobada queda registrada en `change_history` con el antes y el después. La herramienta no devuelve ese historial al agente; vive en la base. Ver [[database/01-schema/TABLES]].

### Terminal

| Campo | Tipo | Notas |
|---|---|---|
| `salida` | string | Salida del comando, recortada al límite |
| `error` | string | Salida de error cuando la hay |
| `codigo` | int | Código de salida |
| `truncado` | bool | Si la salida se cortó por tamaño o tiempo |
| `termino` | bool | Si el comando terminó |

Un comando que sale con error no es un fallo de la herramienta: devuelve su salida de error y el agente sigue. Ver [[backend/05-quality/ERRORS]].

### Internet

| Herramienta | Devuelve | Tipo |
|---|---|---|
| `buscar_en_internet` | Resultados | lista de (titulo, direccion, fragmento) |
| `abrir_pagina` | El contenido de la página | string |

Lo que vuelve entra al presupuesto de contexto, se recorta y se audita como cualquier otro documento. Ver [[backend/01-domain/DOMAIN]].

## Referencias

- [[backend/02-interfaces/TOOLS]] — el catálogo y su comportamiento.
- [[backend/05-quality/VALIDATION]] — qué campos son obligatorios y cómo se validan.
- [[specs/SPEC-TOOLS]] — la especificación funcional.
- [[database/01-schema/TABLES]] — dónde queda lo que las herramientas de escritura registran.
