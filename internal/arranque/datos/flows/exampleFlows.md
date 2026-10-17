# Flujos de LocalCli

Un flujo es una secuencia de etapas que el harness corre con el agente activo de
la sesión. Se invoca escribiendo:

```
/flow:<comando> <petición>
```

La **petición** es lo que el flujo debe resolver; viaja a todas las etapas como
«Petición original», junto al historial de la sesión. Sin petición el flujo no
arranca (la única excepción es la cola, `/flow:ejecutar`).

Los flujos oficiales viven en esta carpeta (`~/.config/localcli/flows/`) y los
propios del proyecto en `.localcli/flows/`. Los propios solo **añaden**: si
repiten el comando de un oficial, se conserva el oficial.

## Campos del flujo

| Campo     | Obligatorio | Qué es                                                                     |
| --------- | ----------- | -------------------------------------------------------------------------- |
| `comando` | sí          | El `/comando` que lo arranca. Debe empezar por `/`. Es su identidad.       |
| `reglas`  | no          | Lista de reglas que se inyectan en **todas** las etapas del flujo.         |
| `etapas`  | sí          | La secuencia ordenada de etapas. Al menos una.                             |

## Campos de una etapa

| Campo              | Obligatorio | Qué es                                                                                                     |
| ------------------ | ----------- | ---------------------------------------------------------------------------------------------------------- |
| `id`               | sí          | Identificador interno de la etapa.                                                                         |
| `nombre`           | sí          | Nombre humano que se muestra en la interfaz.                                                               |
| `orden`            | sí          | La orden de la etapa: qué debe cumplir. Es su instrucción.                                                  |
| `respuesta`        | en `linear` | La estructura de salida que la etapa debe producir y que recibe la etapa siguiente.                        |
| `type`             | no          | `linear` (una vez, por defecto) o `loop` (repite hasta que la etapa responda `true`).                       |
| `aprobacion`       | no          | `true` pausa el flujo **después** de la etapa para que el usuario apruebe y continúe.                       |
| `respuesta_en_chat`| no          | `true` pinta la respuesta de la etapa en el chat sin que sea la respuesta del turno.                        |
| `max_iteraciones`  | en `loop`   | Cota de vueltas de una etapa `loop`. Por defecto 5.                                                         |

La **última etapa** del flujo siempre es la entrega: su respuesta se transmite al
chat y se guarda como la respuesta del turno. No hay que marcarla con nada.

Las etapas corren con el **agente activo de la sesión** (el que elijas con Tab),
no con un agente fijo. Si una etapa necesita una herramienta que ese agente no
tiene concedida, el flujo se detiene y lo avisa: cambia de agente o concede el
permiso y vuelve a lanzarlo.

## Ejemplo lineal

```json
{
  "comando": "/revisar",
  "reglas": ["No inventes: cada afirmación cita el archivo del que sale."],
  "etapas": [
    {
      "id": "leer",
      "nombre": "Leer lo relevante",
      "orden": "Localiza y lee los archivos relacionados con la petición.",
      "respuesta": "Una lista en markdown: archivo; qué contiene; por qué es relevante."
    },
    {
      "id": "informe",
      "nombre": "Entregar el informe",
      "orden": "Redacta el informe con lo que devolvió la etapa anterior.",
      "respuesta": "Un informe en markdown: Hallazgos; Evidencia; Conclusión."
    }
  ]
}
```

## Ejemplo con bucle

Una etapa `loop` repite hasta que el agente cierra con una línea que sea `true`
(resuelto) o `false` (hay que volver a intentarlo). Lo que escriba antes de esa
línea es el resumen de su trabajo y alimenta el intento siguiente.

```json
{
  "comando": "/verde",
  "etapas": [
    {
      "id": "arreglar",
      "nombre": "Dejar los tests en verde",
      "orden": "Ejecuta los tests y corrige lo que falle. Repite hasta que pasen.",
      "type": "loop",
      "max_iteraciones": 8,
      "respuesta_en_chat": true
    },
    {
      "id": "entregar",
      "nombre": "Entregar",
      "orden": "Resume qué se cambió y cómo quedaron los tests.",
      "respuesta": "Un resumen en markdown: Cambios; Estado de los tests."
    }
  ]
}
```

## Errores frecuentes

- Un campo que no está en estas tablas: el flujo se ignora con un aviso (los
  archivos se leen con campos estrictos).
- Falta `orden` o (en una etapa `linear`) falta `respuesta`: el flujo no carga.
- `type` distinto de `linear` o `loop`: el flujo no carga.
- Un archivo con el esquema antiguo: bórralo y ejecuta `localcli init` para que
  el binario vuelva a sembrar el oficial con el formato nuevo.
