---
title: LocalCli — catálogo de herramientas
tags: [backend, interfaces]
depende_de:
  - "[[specs/SPEC-TOOLS]]"
  - "[[backend/DECISIONS]]"
relacionado:
  - "[[backend/02-interfaces/dto/TOOLS-DTO]]"
  - "[[backend/01-domain/DOMAIN]]"
  - "[[backend/02-interfaces/INTERFACES-GENERAL]]"
  - "[[backend/03-security/SECURITY]]"
  - "[[backend/05-quality/ERRORS]]"
  - "[[backend/05-quality/VALIDATION]]"
  - "[[specs/SPEC-ARCHIVOS]]"
---
# TOOLS — Catálogo de herramientas

Las trece herramientas del agente, su reparto y sus controles. La especificación funcional está en [[specs/SPEC-TOOLS]]; aquí está el contrato para quien implemente. Los payloads están en [[backend/02-interfaces/dto/TOOLS-DTO]].

## 1. El catálogo

Cerrado. El agente no puede inventar herramientas fuera de esta lista.

### Archivos

| Herramienta | Lee o escribe | Agente | Contrato |
|---|---|---|---|
| `leer_archivo` | Lee | Ambos | Ruta relativa al proyecto; devuelve el contenido como texto |
| `listar_carpeta` | Lee | Ambos | Ruta relativa; devuelve la lista de entradas de un nivel |
| `buscar_archivos` | Lee | Ambos | Patrón; devuelve las rutas que coinciden |
| `buscar_en_archivos` | Lee | Ambos | Patrón; devuelve las coincidencias con archivo y línea |
| `crear_archivo` | Escribe | Solo `build` | Ruta y contenido; falla si el archivo ya existe |
| `escribir_archivo` | Escribe | Solo `build` | Ruta y contenido; sobrescribe |
| `editar_archivo` | Escribe | Solo `build` | Ruta y el cambio; aplica una edición parcial |
| `eliminar_archivo` | Escribe | Solo `build` | Ruta; pide confirmación explícita |
| `crear_carpeta` | Escribe | Solo `build` | Ruta; falla si ya existe |
| `eliminar_carpeta` | Escribe | Solo `build` | Ruta; pide confirmación explícita |

### Terminal

| Herramienta | Lee o escribe | Agente | Contrato |
|---|---|---|---|
| `ejecutar_comando` | Lee | Ambos | Comando y carpeta de trabajo; devuelve salida, error y si terminó |

### Internet

| Herramienta | Lee o escribe | Agente | Contrato |
|---|---|---|---|
| `buscar_en_internet` | Lee | Ambos | Consulta; devuelve título, dirección y fragmento de cada resultado |
| `abrir_pagina` | Lee | Ambos | Dirección; devuelve el contenido de la página |

`crear_archivo` y `escribir_archivo` están separadas a propósito: el agente no destruye algo por accidente cuando pretendía crear.

## 2. El reparto: `plan` mira, `build` escribe

- `plan` recibe solo herramientas que **recopilan información**: las diez de lectura de archivos, la de terminal y las dos de internet. No tiene ninguna que escriba.
- `build` recibe el catálogo completo. Es el único que crea, modifica y borra.

Esto no es una restricción de estilo que se pueda desactivar: es la garantía estructural de que nada cambia sin que `plan` lo haya propuesto y tú lo hayas aprobado. Ver [[backend/03-security/SECURITY]].

## 3. El relevo entre agentes

```
plan lee  →  plan propone  →  tú apruebas  →  cambias a build  →  build aplica
```

1. `plan` investiga con herramientas de lectura.
2. `plan` propone el cambio concreto.
3. Tú lo apruebas.
4. Cambias a `build`.
5. `build` aplica exactamente lo aprobado.

**La aprobación vale para el cambio propuesto, no para lo que siga.** Si tras el relevo `build` necesita hacer algo que `plan` no propuso, vuelve a preguntar. Una aprobación no es un permiso general.

## 4. Herramientas de archivo

Siguen [[specs/SPEC-ARCHIVOS]]: dentro de la carpeta del proyecto son accesibles, fuera se pide permiso y el agente explica por qué, y **toda escritura pasa por aprobación**, incluso dentro de un flujo en curso. El borrado exige confirmación explícita, no solo aprobación genérica. Cada ruta es relativa a la carpeta abierta. Ver [[backend/02-interfaces/dto/TOOLS-DTO]] para las reglas de validación de rutas.

## 5. Herramientas de terminal

Dos controles independientes.

### Control 1 — lista blanca

Se ejecutan sin preguntar, porque son los que se repiten en cada iteración:

- Compilar el proyecto.
- Correr las pruebas.
- Revisar estilo y tipos.
- Ver el estado y las diferencias del repositorio: estado, diferencias, historial.

Cualquier otro comando pide aprobación antes de ejecutarse.

### Control 2 — no toca tus archivos

El bloqueo **no depende de revisar el texto del comando**. La terminal no puede crear, editar ni borrar ningún archivo del usuario, por más indirecto que sea: `find -delete`, `tee`, una tubería hacia un archivo, `python3 -c` o un `sh -c` con redirección se colarían por un filtro de texto, y por eso el bloqueo es estructural, con Landlock.

Lo que sí puede hacer: usar su propio espacio (caché de compilación, temporales, carpeta de trabajo por sesión) y correr los comandos de la lista blanca.

Lo que no puede hacer, aunque se le pida: crear, editar o borrar archivos del proyecto (para eso están las herramientas de escritura, que solo tiene `build` y que pasan por aprobación), descartar cambios del repositorio, hacer commits o subir cambios.

### Otras reglas de la terminal

- El comando corre en la carpeta del proyecto.
- La salida está limitada: un comando que no termina o genera salida sin fin se corta y se avisa, sin colgar el harness ni llenar el contexto.
- El resultado indica si el comando terminó bien o mal, con la salida de error cuando la hay.
- Un comando que falla no detiene el trabajo: el agente ve el error y sigue.

## 6. Herramientas de internet

Son las únicas que hacen salir información de la máquina.

- **Qué sale:** solo la consulta que redacta el modelo. Nunca el contenido de tus archivos, de la documentación, del historial de la conversación ni de tus tareas. Si una búsqueda necesita el proyecto para ser útil, el agente te lo pide a ti.
- **Qué vuelve:** `buscar_en_internet` devuelve resultados; `abrir_pagina` devuelve el contenido de una página. Ese contenido entra al contexto, ocupa el mismo espacio que el resto, y está sujeto al mismo recorte y al mismo registro de auditoría que en [[backend/01-domain/DOMAIN]].
- **Quién las usa:** los dos agentes. `plan` las necesita para consultar documentación de librerías mientras planifica; `build` para verificar versiones y APIs.

## 7. Cómo enruta `tools`

`tools` es el registro y el enrutado, no el permiso aplicado ni la herramienta ejecutada:

1. Recibe la petición del agente (nombre de herramienta y argumentos).
2. Comprueba que la herramienta existe en el catálogo y que el agente activo la tiene (`plan` no tiene ninguna de escritura).
3. Enruta: archivo → `fileops`; terminal → `exec`; internet → el cliente de internet.
4. `fileops` o `exec` comprueban el permiso concreto, lo aplican y devuelven el resultado.
5. `tools` devuelve el resultado al agente.

`tools` no inventa herramientas y no aplica permisos: los aplica `fileops` y `exec`. Ver [[backend/DECISIONS]].

## Referencias

- [[specs/SPEC-TOOLS]] — la especificación funcional y los criterios de aceptación.
- [[specs/SPEC-ARCHIVOS]] — reglas de permiso sobre archivos.
- [[backend/02-interfaces/INTERFACES-GENERAL]] — las tres superficies.
- [[backend/02-interfaces/dto/TOOLS-DTO]] — los payloads de cada herramienta.
- [[backend/03-security/SECURITY]] — la garantía de escritura y el bloqueo de la terminal.
- [[backend/05-quality/VALIDATION]] — validación de rutas y argumentos.
- [[backend/05-quality/ERRORS]] — errores que puede producir cada herramienta.
