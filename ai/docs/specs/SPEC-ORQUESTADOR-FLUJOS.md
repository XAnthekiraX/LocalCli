---
title: SPEC — Orquestador de flujos
tags: [specs, requisito]
depende_de:
    - "[[IDEA]]"
    - "[[specs/SPEC-NODO-CONTEXTO]]"
    - "[[specs/SPEC-SESIONES]]"
    - "[[specs/SPEC-AGENTE-BASE]]"
relacionado:
    - "[[specs/SPEC-FLUJO-PERSONALIZADO]]"
    - "[[specs/SPEC-COLA-TAREAS]]"
    - "[[specs/SPEC-TOOLS]]"
---

# SPEC — Orquestador de flujos

Prioridad: P0 (núcleo)

## Propósito

Ejecutar trabajos de varias etapas encadenadas en las que **cada etapa responde una pregunta declarada** y su respuesta es lo que viaja a la siguiente.

## Alcance

Incluye el orquestador de etapas, los flujos oficiales, la ventana de contexto del flujo, el encadenamiento, el control cuando una etapa falla y el registro de lo que hizo cada etapa.
No incluye la cola que el orquestador ejecuta, que está en [[specs/SPEC-COLA-TAREAS]], ni que el usuario defina sus propios flujos, que está en [[specs/SPEC-FLUJO-PERSONALIZADO]].

## Actores

- **Usuario**: lanza el flujo con su petición, decide cuando algo falla y puede parar una etapa antes de que corra.
- **Orquestador**: encadena las etapas, mantiene la ventana y conserva el estado.
- **Etapa**: pide su contexto, responde su pregunta y produce una respuesta.
- **Modelo**: decide qué documentación es relevante y produce cada respuesta.

## La pregunta de cada etapa

Un flujo declara **qué entrega** (`pregunta` del flujo) y **qué responde cada etapa** (`pregunta` de la etapa). Son campos obligatorios: una etapa sin pregunta no sabe qué se le pide, y un flujo sin pregunta no sabe qué entrega.

La pregunta es lo que convierte una etapa en algo verificable. En lugar de «explora el proyecto», la etapa responde «¿qué archivos tiene el proyecto?». La diferencia se ve en la etapa siguiente: recibe la **respuesta** a esa pregunta, no el volcado de una exploración.

## Invocación del flujo

Un flujo tiene **dos entradas**, y las dos corren las mismas etapas, en el mismo orden, con el mismo bloque, las mismas pausas y la misma frontera de salida:

1. **El comando del usuario** (`/flow:planificar`, `/flow:crear`, `/flow:actualizar`, `/flow:eliminar`, `/flow:resolver`, `/flow:ejecutar`), seguido de su petición. El prefijo `flow:` es el que marca que un texto arranca un flujo: el `/comando` crudo queda para los comandos de la interfaz.
2. **La herramienta `ejecutar_flujo`**, que el agente pide por el mismo canal nativo que cualquier otra herramienta cuando decide que el trabajo pide etapas encadenadas. Es una **sola herramienta genérica** —`flujo` elige el comando y `peticion` es el objetivo—, no una herramienta por flujo. El catálogo de flujos viaja en el **esquema** de la herramienta, que se arma por turno con el agente activo.

### Qué puede lanzar cada agente

El flujo corre con los agentes que declaran sus etapas, no con el agente que lo invoca. Pero para **poder lanzarlo**, el agente que invoca necesita el permiso **derivado del flujo**: si todas las etapas corren con `plan`, el flujo es de lectura y lo lanza cualquier agente con `read`; si alguna etapa corre con `build` —es decir, escribe—, el flujo exige `write`/`edit` y `plan` no lo puede lanzar. Así la regla «`plan` mira, `build` escribe» se mantiene aunque el flujo lo dispare el agente y no el usuario. El esquema solo lista los flujos que el agente activo puede lanzar.

### Frontera de salida

Lo que vuelve a la sesión que invocó el flujo es **la respuesta de la etapa `entrega`** más el **bloque de contexto resumido** —una aportación por etapa, condensada—. Nada más cruza: ni las etapas, ni las herramientas que corrieron dentro, ni la ventana del flujo. La sesión principal sabe qué flujo lanzó y qué entregó, no lo que pasó dentro.

### Profundidad

Un flujo **no invoca flujos**: `ejecutar_flujo` no está disponible dentro de una etapa de flujo. La profundidad es uno, la misma regla que impide que una herramienta llame a otra herramienta.

## Flujo principal — flujo con objetivo

1. El flujo arranca por una de sus dos entradas: el usuario escribe el comando con su petición, con sus palabras, o el agente pide `ejecutar_flujo` con el flujo elegido y la petición.
2. Cada etapa pide su contexto al nodo, **excluyendo lo que las etapas anteriores ya recibieron** (§Nodo de contexto en [[specs/SPEC-NODO-CONTEXTO]]). Es lo que hace auditable qué recibió cada una: `context_audit` guarda una fila por documento y etapa.
3. El orquestador corre las etapas en orden. En cada una:
   1. Si la etapa tiene `continuacion`, el flujo **se detiene y pregunta** `continuamos con la etapa <nombre>?`. El usuario contesta sí o no, y puede cambiar de modelo mientras la decisión está en pantalla: al continuar, la etapa corre con el modelo que esté elegido, y las siguientes también, porque el modelo es de la sesión y se relee en cada etapa.
   2. La etapa recibe la ventana (§Ventana de contexto del flujo) más su propio contexto.
   3. Corre con herramientas, salvo que tenga `entrega`: esa corre **sin herramientas**, porque solo redacta.
   4. Su respuesta se optimiza —una generación corta que la condensa sin inventar—, se guarda en el bloque de contexto y se suma a la ventana.
   5. Si la etapa tiene `aprobacion`, el flujo se detiene y pregunta `aprobar la etapa <nombre>`, igual que hasta ahora.
   6. Si la etapa tiene `respuesta_en_chat`, su respuesta se pinta en la pantalla y se guarda (§Visibilidad de una respuesta).
4. El flujo termina con la respuesta de la etapa `entrega`, que es lo único que el chat muestra como conversación.

## Ventana de contexto del flujo

La ventana es lo que una etapa hereda de las anteriores. No es el historial del chat: es un texto que el orquestador compone y que contiene dos cosas.

1. **La petición original**, completa y con su comando. Cuando el flujo lo arranca el usuario, es su invocación (`/flow:<nombre>` con sus palabras); cuando lo arranca el agente, es la `peticion` que trajo `ejecutar_flujo`. Va siempre, en todas las etapas: es el encargo que ninguna etapa puede perder de vista.
2. **La pregunta y la respuesta de cada etapa anterior**, en orden. `## <nombre>` · `Pregunta: <pregunta>` · `Respuesta: <respuesta>`.

La ventana viaja porque el objetivo es que **ninguna etapa repita lo que otra ya resolvió**. Una etapa que ya sabe qué archivos tiene el proyecto no vuelve a listarlos: busca entre lo que ya tiene en la ventana.

### Presupuesto

La ventana tiene tope, porque la ventana de contexto del modelo es limitada y una etapa con herramientas la agota enseguida. El tope **no es un número fijo**: es lo que queda de la ventana real del modelo —el menor entre la que declara y el máximo que el harness permite pedir— después del prompt del agente, los esquemas de las herramientas y el historial.

Se recalcula **en cada etapa, con el modelo que vaya a correrla**. Con un modelo pequeño sobra sitio; con uno grande aprieta, y el cálculo se rehace porque el modelo se relee al empezar cada etapa.

### Cuando no cabe

Si la ventana completa no cabe en el presupuesto, se **caen las entradas más antiguas** y el texto dice cuáles se han caído. Se descartan las viejas porque llegan ya digeridas por su etapa, mientras que la petición original y lo más reciente son lo que la etapa necesita para trabajar.

Caer una entrada no cancela la etapa: la sigue, sabiendo por el aviso qué parte de lo anterior no tiene a la vista.

## Bloque de contexto

Toda etapa entrega su respuesta, el modelo la **optimiza** y queda en el bloque de contexto, una aportación por etapa. No hay bandera que lo active: **todos los flujos** llevan bloque.

El bloque vive en `flow_context` (sesión, flujo, etapa, nombre visible, posición y contenido). La pregunta no se persiste: se deriva del flujo al recomponer la ventana, porque el flujo es dato y los ids de etapa son estables. Una ejecución nueva del mismo flujo lo vacía: no hereda la anterior. Se guarda porque es lo que permite **retomar un flujo pausado sin repetir las etapas ya hechas**, y porque la etapa `entrega` lo lee entero para redactar.

## Visibilidad de una respuesta

Una respuesta de etapa tiene tres niveles, y el nivel no lo decide el orquestador: lo declara la etapa.

| Nivel | Cuándo | Dónde queda | ¿Llega al modelo? | Herramientas |
|---|---|---|---|---|
| Silenciosa | la etapa no pide nada de esto | en la ventana y el bloque | no | sí |
| `respuesta_en_chat: true` | el usuario quiere ver lo que encontró | en la pantalla, como bloque del hilo, y en `chat_evento` con tipo `respuesta` | **no** | sí |
| `entrega: true` | el flujo ya tiene todo y hay que redactar | en la pantalla y en `messages` | sí | **no** |

El nivel intermedio no es contexto por construcción: lo que se pinta pero no es conversación se guarda en `chat_evento`, y el contexto que se le entrega al modelo se arma solo con `messages`. No hay un filtro que se pueda olvidar. La respuesta se pinta en su sitio cronológico —después de lo que la etapa ya dejó en el hilo y antes del aviso de etapa terminada— y se recupera igual al volver a la sesión. La única excepción es la etapa cuyo texto ya viaja token a token al turno: esa no se pinta dos veces.

El nivel de entrega es el único que corre sin herramientas. Lo declara la etapa, y no se deduce de que sea la última: un flujo puede terminar escribiendo código, y en ese caso su última etapa necesita herramientas.

## Flujos alternativos

- Una etapa falla: el flujo se detiene y el usuario decide.
- El usuario contesta que no a una `continuacion`: el flujo se detiene.
- Una etapa con `aprobacion` espera: el flujo queda pausado.
- El usuario cancela: se detiene en la etapa actual.
- Una etapa no puede continuar por falta de contexto: avisa y se detiene.
- El usuario vuelve más tarde: el flujo pausado se retoma donde estaba, leyendo el bloque.

## Reglas de negocio

- Las etapas se ejecutan en orden y cada una arranca cuando la anterior terminó.
- La `pregunta` del flujo y la `pregunta` de cada etapa son obligatorias. Sin ellas el flujo no carga, y se informa de cuál falta.
- Un flujo declara exactamente una etapa `entrega`, y es la última.
- `continuacion` se pregunta **antes** de correr la etapa; `aprobacion`, **después**. Una etapa puede tener las dos: primero se decide si se sigue, y después se aprueba el resultado.
- El modelo es el de la sesión, y se relee en cada etapa. Un flujo no declara modelo: quien lo elige es el usuario, también a mitad de un flujo.
- Un documento se entrega **una sola vez** por ejecución: las etapas siguientes ya no lo ven, porque la respuesta de quien lo recibió lo resume.
- Cada etapa corre sin el historial del chat y recibe de las anteriores la ventana, no su salida entera.
- La ventana lleva siempre la petición original —la del usuario con su comando, o la que trae `ejecutar_flujo`—, completa.
- La ventana tiene tope, se recalcula por etapa con el modelo que la corre y, al exceder, se caen sus entradas más antiguas avisando de cuáles.
- Una etapa `entrega` corre sin herramientas. Las demás, con las del agente que declara.
- Una etapa con `respuesta_en_chat` se ve en la pantalla y se guarda, y su texto nunca entra al contexto del modelo.
- La pregunta de cada etapa viaja en el brief que recibe el agente: es lo que la etapa debe responder, y es la que la ventana empareja con la respuesta.
- Una etapa que no responde a su pregunta se reintenta **una vez**; si sigue muda, falla (`E_STAGE_FAILED`) y el flujo se detiene. Una etapa sin respuesta no alimenta la ventana y no pasa por buena.
- Una etapa que no pide nada de esto corre en silencio: la vista solo anuncia su nombre (`[Sub Proceso] <nombre>`).
- Si una etapa falla, el flujo se detiene. El usuario elige reintentar, saltar esa etapa o cancelar.
- Un flujo pausado por un permiso se retoma desde la misma etapa, sin repetir lo ya hecho, leyendo el bloque.
- Todo lo que hace cada etapa queda registrado: qué recibió, qué hizo y qué produjo.
- Los flujos se cargan al arrancar de `~/.config/localcli/flows/*.json` (los oficiales, sembrados desde el binario) y de `<proyecto>/.localcli/flows/*.json` (los propios del proyecto, aditivos): un `comando` sin archivo no existe y no se lista, y un archivo con un `comando` nuevo añade ese comando ([[specs/SPEC-FLUJO-PERSONALIZADO]]).
- Un flujo que se cancela no deja etapas ejecutándose.
- Un flujo arranca por dos entradas: el comando explícito del usuario (`/flow:planificar`, `/flow:crear`, `/flow:actualizar`, `/flow:eliminar`, `/flow:resolver` o `/flow:ejecutar`), o la herramienta `ejecutar_flujo` que el agente pide cuando decide que el trabajo lo necesita.
- Una petición que no es un flujo (`/flow:…`) y que el agente no resuelve con `ejecutar_flujo` se responde en el chat, no arranca etapas.
- El agente solo lanza los flujos que admite su permiso derivado: con `read`, los flujos cuyas etapas corren todas con `plan`; un flujo con alguna etapa `build` exige `write`/`edit`. `plan` no lanza un flujo que escribe.
- `ejecutar_flujo` no está disponible dentro de una etapa de flujo: un flujo nunca invoca a otro.
- Lo que un flujo devuelve a quien lo invocó es la respuesta de su etapa `entrega` más el bloque de contexto resumido; el interior del flujo no cruza.
- El orquestador puede detectar trabajo ordenado y proponer un TODO, pero no lo ejecuta hasta que el usuario confirme o escriba el comando.
- Una tarea de la cola no arranca si sus dependencias no están cumplidas.

## Criterios de aceptación

- [ ] Lanzar un flujo oficial ejecuta sus etapas en orden.
- [ ] Un flujo arranca con `/flow:<comando>` —el comando crudo no arranca flujos— o con la herramienta `ejecutar_flujo` del agente.
- [ ] `ejecutar_flujo` es una sola herramienta: el esquema se arma por turno y solo lista los flujos que el agente activo puede lanzar.
- [ ] `plan` no puede lanzar un flujo con alguna etapa `build`; `build` sí, con `write`/`edit`.
- [ ] Lo que vuelve al agente que invocó el flujo es la entrega más el bloque resumido, y nada del interior.
- [ ] `ejecutar_flujo` no está disponible dentro de una etapa: un flujo no invoca a otro.
- [ ] Un flujo lanzado por el agente conserva sus pausas de `continuacion` y `aprobacion`.
- [ ] La cola avanza cuando el usuario escribe `/ejecutar`.
- [ ] Una petición sin comando se responde en el chat y no arranca etapas.
- [ ] Una tarea con dependencias sin cumplir no arranca.
- [ ] Un flujo sin `pregunta`, o con una etapa sin `pregunta`, no carga y se informa de cuál falta.
- [ ] Un flujo declara exactamente una etapa `entrega`, y es la última.
- [ ] Cada etapa recibe la ventana: la petición original y la pregunta y respuesta de cada etapa anterior.
- [ ] Un documento recibido por una etapa no se vuelve a entregar a las siguientes.
- [ ] La ventana se acota al presupuesto que deja la ventana real del modelo con la etapa que va a correr.
- [ ] Cuando la ventana no cabe, se caen sus entradas más antiguas y la etapa lo sabe por el aviso.
- [ ] Una etapa con `continuacion` detiene el flujo antes de correr, y el usuario puede cambiar de modelo antes de continuar.
- [ ] Cambiar el modelo durante una `continuacion` hace que la etapa y las siguientes corran con ese modelo.
- [ ] Una etapa con `aprobacion` detiene el flujo después de correr, y el usuario aprueba su resultado.
- [ ] La etapa `entrega` corre sin herramientas; una etapa que escribe código, con ellas.
- [ ] Una etapa con `respuesta_en_chat` se ve en la pantalla y se guarda, y su texto no aparece en el contexto que recibe el modelo.
- [ ] La pregunta de una etapa viaja en el brief del agente y la ventana empareja pregunta y respuesta de cada etapa anterior.
- [ ] Una etapa que no responde a su pregunta se reintenta una vez y, si sigue muda, falla con `E_STAGE_FAILED` y el flujo se detiene.
- [ ] Una etapa sin nada de esto no se muestra ni se persiste; la vista solo anuncia su nombre.
- [ ] Al terminar el flujo, el chat muestra la entrega como conversación.
- [ ] Todas las etapas guardan su respuesta optimizada en el bloque, sin importar el flujo.
- [ ] Si una etapa falla, el flujo se detiene y el usuario elige qué hacer.
- [ ] Un flujo pausado por un permiso se retoma exactamente donde estaba.
- [ ] El usuario puede cancelar un flujo en cualquier momento.
- [ ] Queda registro de lo que hizo cada etapa.
- [ ] Un flujo oficial se puede lanzar sin configurar nada.

## Requisitos no funcionales

- Aislamiento: un flujo no recibe el contexto de otro flujo.
- Recuperación: retomar un flujo pausado no repite etapas ya completadas.
- Eficiencia: el presupuesto de la ventana se deriva de la ventana real del modelo en vez de fijarse, para no desperdiciar la de un modelo pequeño ni desbordar la de uno grande.

## Dependencias funcionales

- [[specs/SPEC-NODO-CONTEXTO]]
- [[specs/SPEC-SESIONES]]
- [[specs/SPEC-AGENTE-BASE]]

## Referencias

- [[IDEA]]
