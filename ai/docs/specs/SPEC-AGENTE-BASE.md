---
title: SPEC — Agentes incluidos
tags: [specs, requisito]
depende_de:
  - "[[IDEA]]"
  - "[[specs/SPEC-TOOLS]]"
  - "[[specs/SPEC-NODO-CONTEXTO]]"
  - "[[specs/SPEC-ARCHIVOS]]"
  - "[[specs/SPEC-OLLAMA-PERFIL]]"
  - "[[specs/SPEC-SESIONES]]"
  - "[[specs/SPEC-COLA-TAREAS]]"
relacionado:
  - "[[backend/DECISIONS]]"
  - "[[specs/SPEC-CICLO-TRABAJO]]"
  - "[[specs/SPEC-RESOLVER]]"
  - "[[specs/SPEC-SKILLS]]"
---
# SPEC — Agentes incluidos

Prioridad: P0 (núcleo)

## Propósito

Los dos agentes incluidos, con permisos distintos: uno que solo mira y propone, otro que escribe.

**Vienen incluidos, pero no están hardcodeados.** Los dos se definen en archivos JSON con estructura fija, en `.localcli/agents/`. El usuario puede modificarlos o derivar otros agentes de ellos sin tocar el código.

**No hay skills por defecto.** Los agentes vienen sin ninguna; las crea el usuario en markdown. Ver [[specs/SPEC-SKILLS]].

## Alcance

Incluye `plan` y `build`: qué hace cada uno, a qué herramientas tiene acceso y cómo se pasa de uno a otro. Incluye además cargar agentes propios: cualquier `.localcli/agents/*.json` válido se añade a los disponibles y `Tab` recorre todos.
No incluye skills ni flujos.

## Actores

- **Usuario**: elige agente, responde preguntas, aprueba propuestas.
- **Agente `plan`**: investiga, propone y pide aprobación. No escribe.
- **Agente `build`**: aplica lo aprobado. Es el único que escribe.

## Dónde se definen

Cada agente es un `.localcli/agents/*.json` con campos fijos: `nombre`, `descripcion`, `prompt`, `permisos` y `skills`. Se eligió JSON y no un README porque un README deja margen a interpretación de qué significa cada parte.

**Todos los `*.json` de la carpeta se cargan**, no solo `plan.json` y `build.json`: el nombre del archivo no decide nada, manda el campo `nombre`. Dejar un `.localcli/agents/<nombre>.json` válido basta para tener un agente nuevo disponible —sin tocar el código—: aparece en la lista por la que cicla `Tab` y puede referenciarse desde una etapa de un flujo propio. `plan` y `build` siempre están disponibles aunque falte o no cargue su archivo (quedan con un prompt de respaldo), porque los flujos oficiales los referencian. Un archivo inválido se ignora sin impedir el arranque.

El campo `permisos` es el que sostiene la garantía: declara, por acción (`leer`, `editar`, `ejecutar`, `internet`, `tareas`), si el agente puede (`permitir`) o no (`denegar`). Si `plan` deniega `editar`, no tiene ninguna herramienta que escriba en el proyecto, aunque se la pidan. La acción `tareas` (la lista de pasos de la sesión) no escribe en el proyecto: es estado de la sesión, y la tienen los dos agentes. El catálogo efectivo de herramientas se **deriva** de los permisos contra el catálogo cerrado de `tools`: no hay una segunda lista que pueda contradecirlos. La garantía vive en los datos, no en el código.

Un agente es una configuración sobre un mismo ciclo conversacional: su prompt, sus permisos y su catálogo. No hay un camino «de chat» y otro «de herramientas»: las herramientas son una capacidad que el modelo pide dentro del mismo ciclo.

El usuario ve qué herramientas tiene el agente activo en la línea de estado bajo la entrada, no dentro de su mensaje de sistema. El catálogo ya no se inyecta en el prompt: viaja por el canal de herramientas, junto a la definición de cada una. El prompt del agente se dedica a lo que el agente es.

Un cambio de contrato: un JSON viejo con el campo `herramientas` ya no se carga; la lista de herramientas no se declara, se deriva. El formato exacto de cada campo está en [[backend/DECISIONS]].

## El ciclo de un turno

Un turno no es una pregunta y una respuesta. Es una conversación corta entre el modelo y las herramientas, que puede necesitar más de una ronda.

1. El modelo recibe el contexto y la lista de sus herramientas.
2. Responde con texto, con una petición de herramienta, o con las dos cosas.
3. Si pidió una herramienta, se ejecuta y su resultado vuelve al modelo.
4. El modelo ve el resultado y decide: seguir, pedir otra, o responder.
5. Se repite desde el 2 hasta que el modelo responde en texto o se alcanza el máximo de rondas.

Ese máximo existe para que un modelo que se equivoca en bucle no se quede generando indefinidamente.

**Al agotarse, el turno redacta.** Cerrarlo devolviendo el preámbulo que acompañaba a la última petición de herramienta sería devolver un texto que no contesta nada, así que el turno cierra con una **pasada de redacción sin herramientas**: se le pide al modelo que escriba su resultado con lo que consiguió. Esa pasada es la que hace que la entrega final no dependa de que el modelo deje de pedir herramientas por sí solo.

Una redacción no es un cierre, y puede fallar de dos maneras. Si el modelo vuelve a pedir herramientas —aun sin que se le ofrezcan— se le responde con una instrucción explícita y se **reintenta una sola vez**. Si aun así no entrega texto, el turno **falla con el motivo a la vista**: se avisa de que se agotaron las rondas y de que el modelo no entregó nada. Nunca se guarda un turno mudo como si hubiera respondido, y nunca se termina en silencio.

**Un fallo no termina el turno.** Si una herramienta falla por un motivo que el modelo puede corregir —le faltan argumentos, la ruta no existe— ve el motivo y puede reintentarlo en la misma ronda. Si el fallo no es suyo —se le ha denegado el permiso, el sistema no responde— se lo dice y sigue con otra cosa.

Las herramientas se ejecutan **una detrás de otra**, en el orden pedido. No en paralelo: el orden importa y las aprobaciones tienen que llegar al usuario en orden.

## Un agente sin herramientas

Si el modelo en uso no sabe pedir herramientas —porque no lo declares, o porque el modelo no lo soporta— el agente activo funciona en **modo conversación**: contesta con el contexto que recibe y no ejecuta nada.

Se le avisa al usuario, pero no se le bloquea el modelo. Un modelo que no usa herramientas puede ser el correcto para conversar. Lo que no hace LocalCli es fingir que el agente está trabajando cuando no puede.

## Los dos agentes

### `plan` — solo lee

Tiene **solo herramientas que recopilan información**. No tiene ninguna forma de escribir, y no la necesita: existe para entender el proyecto y decidir qué hay que hacer.

- Lee archivos, lista carpetas y busca por contenido o por nombre.
- Ejecuta comandos de consulta: compilar, correr pruebas, revisar estilo, ver el estado del repositorio.
- Busca en internet y abre páginas.
- Mantiene la lista de pasos de la sesión (`actualizar_todo`), que es estado de la sesión, no un cambio en el proyecto.
- Pregunta al usuario lo que no sabe y propone ideas en vez de suponer.
- Nunca escribe archivos. Su salida es una propuesta, no un cambio.

Su trabajo termina en una propuesta aprobada.

### `build` — escribe

Tiene **el catálogo completo**. Es el único que crea, modifica y borra.

- Aplica exactamente lo que `plan` propuso y tú aprobaste.
- Dentro de un flujo arrancado con comando, toma el siguiente elemento del TODO.
- Trata **un elemento del TODO por iteración**, sin importar si es de código o de documentación.
- La documentación es su única fuente de verdad.
- Si falta información, marca la tarea como bloqueada y dice exactamente qué falta. No rellena huecos inventando.
- Verifica el resultado antes de dar la tarea por terminada.
- Al terminar la tarea, toma la siguiente de la cola, siempre dentro del flujo que arrancó el usuario.

## El relevo

El paso de `plan` a `build` es explícito y es el mecanismo central de la herramienta.

```
plan lee  →  plan propone  →  tú apruebas  →  cambias a build  →  build aplica
```

1. `plan` investiga con herramientas de lectura.
2. `plan` propone el cambio concreto y pide aprobación.
3. Tú apruebas.
4. Cambias a `build`.
5. `build` aplica exactamente lo aprobado.

**Una aprobación vale para el cambio propuesto, no para lo que siga.** Si `build` necesita hacer algo que `plan` no propuso, vuelve a preguntar. Una aprobación no es un permiso general.

Por eso `plan` no escribe: nada cambia en el proyecto sin que antes alguien lo propusiera y tú lo aceptaras.

## Flujo principal

1. El usuario abre la herramienta en una carpeta.
2. Elige agente: `plan` para decidir, `build` para aplicar.
3. Escribe su petición.
4. El agente recibe el contexto que entrega el nodo de contexto.
5. `plan` investiga, propone y pide aprobación.
6. El usuario cambia a `build`.
7. `build` aplica el cambio.
8. `build` verifica y continúa con la cola del flujo que arrancó el usuario.

## Flujos alternativos

- Elige `build` sin documentación suficiente: se lo dice y ofrece cambiar a `plan`.
- Elige `plan` en un proyecto ya documentado: pregunta qué hay que cambiar.
- `plan` termina sin proponer cambios: la tarea se cierra sin cambios.
- `build` se queda sin contexto: se lo dice y vuelve a pedir contexto.
- Necesita salir de la carpeta del proyecto → se aplica [[specs/SPEC-ARCHIVOS]].
- Necesita cambiar, actualizar o eliminar una funcionalidad → se aplica [[specs/SPEC-CICLO-TRABAJO]].
- Tiene que resolver algo que está roto → se aplica [[specs/SPEC-RESOLVER]].

## Reglas de negocio

- Ambos agentes vienen incluidos y están siempre disponibles.
- `plan` no tiene herramientas de escritura. `build` no cambia decisiones técnicas.
- El relevo entre agentes es explícito: propuesta, aprobación, cambio, aplicación.
- Una aprobación habilita solo el cambio propuesto.
- Ambos responden en el idioma en que el usuario escribió.
- Ninguno busca archivos por su cuenta: esperan el contexto que entrega el nodo de contexto.
- Si el contexto no alcanza, lo dicen y piden más. No rellenan los huecos con suposiciones.
- Ninguna escritura pasa sin que `plan` la haya propuesto y tú la hayas aprobado.
- `build` no inventa endpoints, entidades, reglas de negocio ni relaciones.
- Un intercambio de chat empieza con el contexto de esa sesión, no con el de otra.
- Fuera de un flujo, ambos agentes atienden el chat con el catálogo derivado de sus permisos: `plan` lee y propone, `build` escribe con aprobación.
- `plan` propone y `build` aplica dentro de un flujo; un flujo solo existe cuando el usuario lo arranca con un comando explícito.
- Los agentes se cargan de `.localcli/agents/*.json`: `plan` y `build` siempre están disponibles y cualquier agente propio válido se añade a los disponibles. Un archivo inválido se ignora sin impedir el arranque.
- La interfaz recorre los agentes disponibles con `Tab`; el agente activo es el que recibe la petición. Un agente propio vale igual que los base: responde con el catálogo derivado de sus permisos.
- El modelo pide herramientas por su nombre con un esquema declarado. No hay formato en prosa que deba imitar.
- El prompt del agente no incluye el catálogo ni el formato de llamada: el catálogo va por el canal de herramientas.
- Un turno puede necesitar varias rondas de pedir y recibir resultados de herramientas, con un máximo acotado.
- Las herramientas de un turno se ejecutan en el orden pedido, nunca en paralelo.
- Un fallo corregible —argumentos que no cuadran, una ruta que no existe— vuelve al modelo como resultado y el turno puede continuar corrigiendo.
- Un fallo no corregible se informa al modelo y no detiene el trabajo.
- Si el modelo en uso no puede usar herramientas, el agente responde en modo conversación y la interfaz lo avisa.
- Un turno que agota las rondas pide una redacción sin herramientas antes de cerrarse, y solo entonces.
- Si esa redacción vuelve a pedir herramientas, se reintenta una sola vez con una instrucción explícita.
- Si el turno no entrega texto ni así, falla con el motivo a la vista: no se guarda como si hubiera respondido ni se termina en silencio.

## Criterios de aceptación

- [ ] Al abrir la herramienta sin configurar nada, los dos agentes están disponibles.
- [ ] `plan` no tiene ninguna herramienta que cree, modifique o borre.
- [ ] `plan` produce una propuesta, nunca un cambio.
- [ ] `build` tiene el catálogo completo y aplica lo aprobado.
- [ ] Nada se escribe sin que `plan` lo haya propuesto y tú lo hayas aprobado.
- [ ] Una aprobación no habilita cambios distintos de los propuestos.
- [ ] `build` implementa y nunca inventa requisitos.
- [ ] Ambos responden en el idioma en que el usuario escribió.
- [ ] Si le falta información del contexto, la pide en vez de inventarla.
- [ ] `build` marca la tarea como bloqueada y dice qué falta cuando no puede continuar.
- [ ] `build` verifica el resultado antes de cerrar la tarea.
- [ ] Dentro de un flujo arrancado por comando, `build` toma la siguiente tarea de la cola.
- [ ] Dejar un `.localcli/agents/<nombre>.json` válido añade un agente disponible sin tocar el código.
- [ ] `Tab` recorre todos los agentes disponibles, no solo `plan` y `build`.
- [ ] Un `.localcli/agents/*.json` inválido se ignora y el arranque sigue.
- [ ] El contenido de una sesión no aparece en otra sesión del mismo proyecto.
- [ ] El modelo pide herramientas por su nombre, sin imitar ningún formato escrito.
- [ ] El prompt del agente no lleva el catálogo de herramientas.
- [ ] Un turno que necesita leer y después escribir hace al menos dos rondas de herramientas.
- [ ] Un turno se detiene al llegar al máximo de rondas y avisa de que se agotaron.
- [ ] Un turno que gastó sus rondas en herramientas cierra con una redacción de su resultado, no con el preámbulo de su última petición de herramienta.
- [ ] Si el modelo sigue pidiendo herramientas al redactar, se reintenta una vez y, si tampoco responde, el turno falla con un aviso visible en vez de guardar una respuesta vacía.
- [ ] Un fallo de argumentos vuelve al modelo y puede corregirlo sin perder el turno.
- [ ] Las herramientas de un turno se ejecutan en el orden en que las pidió el modelo.
- [ ] Un modelo sin capacidad de herramientas hace que el agente responda en modo conversación, y la interfaz lo avisa.

## Requisitos no funcionales

- Ambos operan solo con el contexto que reciben, no con el proyecto completo.
- No añaden esperas propias al tiempo de respuesta del modelo.
- El reintento de la redacción es una sola pasada más y solo ocurre cuando el turno ya no iba a entregar nada: es el precio de no dejar un turno mudo, y no se repite.
- Una espera de aprobación de una sesión no retiene el turno de inferencia de las demás.

## Dependencias funcionales

- [[specs/SPEC-TOOLS]]
- [[specs/SPEC-NODO-CONTEXTO]]
- [[specs/SPEC-ARCHIVOS]]
- [[specs/SPEC-OLLAMA-PERFIL]]
- [[specs/SPEC-SESIONES]]
- [[specs/SPEC-COLA-TAREAS]]

## Supuestos

- Las instrucciones concretas de cada agente (tono, formato de respuesta, longitud) se ajustan después de verlos en uso.
- El número máximo de rondas de herramientas por turno se fija en FASE 2.

## Referencias

- [[IDEA]]
