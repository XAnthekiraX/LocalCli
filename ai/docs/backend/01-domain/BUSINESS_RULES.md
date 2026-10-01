---
title: LocalCli — reglas de negocio del motor
tags: [backend, reglas]
depende_de:
  - "[[backend/01-domain/DOMAIN]]"
  - "[[database/02-rules/BUSINESS_RULES]]"
relacionado:
  - "[[backend/02-interfaces/TOOLS]]"
  - "[[backend/04-infrastructure/EVENTS]]"
  - "[[database/01-schema/ENUMS]]"
---
# BUSINESS_RULES — Motor

Las reglas que rigen el motor. Las que dependen de cómo se guardan los datos están en [[database/02-rules/BUSINESS_RULES]] y no se repiten aquí; estas son las del comportamiento.

## 1. Reglas de Negocio

### Sesiones

- Una sesión pertenece a un solo proyecto, y el proyecto es la carpeta abierta. El chat de una carpeta nunca aparece en otra.
- El contenido de una sesión no se ve desde otra sesión del mismo proyecto. El contexto acumulado no se comparte.
- Una sesión en segundo plano sigue trabajando aunque el usuario cambie de sesión o salga de la vista.
- Cambiar de sesión no detiene nada.
- Cada sesión expone siempre su estado: inactiva, trabajando, esperando permiso, terminada o con error. Los estados y sus transiciones están en [[database/01-schema/ENUMS]].

### Notificaciones

- Cuando una sesión pasa a **esperando permiso**, se manda una notificación. Es el caso que más duele si no te enteras: el trabajo está parado y solo tú puedes desbloquearlo.
- Cuando una sesión pasa a **terminada**, se manda una notificación, para que te entere sin tener que estar mirando.
- La notificación se manda aunque no estés viendo esa sesión, y aunque hayas cambiado a otra sesión o a otra carpeta.
- Dice qué pasó y qué hacer a continuación: qué aprobación espera, o que el trabajo terminó.
- El cierre de un turno de **chat** no se anuncia como trabajo: dice «la sesión terminó de responder» y «revisa su respuesta en el historial», porque no corrió ninguna etapa y el usuario no debe pensar que arrancó un flujo que no pidió.
- Es un aviso, no una interrupción: no detiene la sesión ni cambia lo que se está haciendo.
- No duplica información. Los detalles están donde siempre: la fila en `approvals` y el historial de la sesión.

### Motor de etapas

- Un flujo no arranca solo: lo solicita el usuario con un comando explícito (`/planificar`, `/crear`, `/actualizar`, `/eliminar`, `/resolver` o `/ejecutar`). Una petición que no sea comando se responde en el chat.
- Las etapas se ejecutan en orden y cada una arranca cuando la anterior terminó.
- Cada etapa recibe solo el contexto que necesita, no todo lo que produjo la anterior.
- Si una etapa falla, el flujo se detiene y el usuario elige reintentar, saltar esa etapa o cancelar.
- Un flujo pausado por un permiso se retoma desde la misma etapa, sin repetir lo ya hecho.
- El usuario puede cancelar en cualquier momento, y un flujo cancelado no deja etapas corriendo.
- Todo lo que hace cada etapa queda registrado: qué recibió, qué hizo y qué produjo.
- Cada etapa pide su contexto identificándose: es lo que permite que `context_audit` tenga una fila por documento **y etapa** en lugar de mezclarlas.
- Un flujo no recibe el contexto de otro flujo.

### Detección de trabajo ordenado

- Una petición tuya que implique una lista ordenada de trabajo **propone un TODO**. El sistema lo detecta, pero no lo ejecuta sin tu confirmación o el comando `/ejecutar`.
- "Documentar capa por capa", "ejecutar tarea 1, tarea 2, tarea 3", "primero esto, luego esto" son todas peticiones que crean un TODO.
- Quien detecta es `flow`, no `queue`. `queue` consume lo que hay.
- El TODO se crea en el flujo que corresponde:
  - En el **flujo de planificar**, se crea el TODO con todas las fases de la planificación.
  - En el **flujo de ejecutar tareas**, se crea el TODO con los pasos para ejecutar cada tarea.
- Una petición sin orden ni lista no crea TODO: se responde en el chat, como cualquier otra.
- El TODO se crea vacío de estado: todos sus elementos empiezan pendientes. El orden lo pone quien lo creó.

### Cola

- La cola no es por capa ni es una cola global. Es la cola de **esta** ejecución, y por eso no es global: cada petición ordenada crea su propio TODO, y ese TODO es su cola.
- La cola tiene forma de TODO. Contiene las fases, tareas o pasos que van a seguir, todo en orden.
- El TODO es la cola. No hay una representación paralela: lo pendiente es literalmente lo que está en el TODO.
- Dos ejecuciones simultáneas pueden tener cada una su TODO, y una no ve el de la otra.
- El orden del TODO es el orden de ejecución, y se respeta.
- Se ejecuta un elemento del TODO por iteración, nunca varios a la vez.
- Un elemento de documentación y uno de código se tratan igual: mismo criterio, mismos permisos, misma verificación.
- Un elemento solo se marca completado cuando se terminó, con su verificación hecha.
- Un elemento bloqueado se marca y se informa qué falta; si nada depende de él, la cola sigue con el siguiente.
- Un elemento nuevo hace que la cola se re-derive y entre en su posición.
- El estado de la cola es visible desde cualquier sesión.
- La cola solo avanza cuando el usuario la arranca con `/ejecutar` o confirma la propuesta del TODO.
- Al vaciarse, la cola se detiene sola y avisa.
- La cola refleja siempre el estado real del TODO. Como se deriva de él, si el TODO cambia, la vista cambia; no hay una segunda verdad que reconciliar.

### Nodo de contexto

- La documentación declara sus dependencias en el frontmatter. Ese mapa orienta, pero no decide por sí solo.
- La decisión de qué es relevante la toma siempre el modelo.
- El contexto entregado nunca supera el límite de contexto del modelo.
- Todo lo que se descartó queda registrado con su motivo, en `context_audit`.
- La auditoría es por terna (sesión, etapa, documento): si una etapa se repite —reinteto, resumen o segundo turno del chat—, la fila ya registrada se conserva y el turno sigue; la base no admite dos filas para la misma terna.
- En el chat, si el modelo no selecciona nada y no hay semilla, el bloque va **sin documentos**: el proyecto entero no entra en una conversación casual. Las etapas de flujo conservan el respaldo a los candidatos.
- El nodo de contexto no lee fuera de la carpeta del proyecto sin permiso.
- Un documento que entra en el contexto se lee primero; no se entrega nada sin leer.
- El contexto de una etapa no arrastra el de etapas anteriores si no lo necesita.
- Si el modelo pide un documento que no existe, la etapa se detiene y avisa, en lugar de continuar suponiendo.
- Si hay documentos sin dependencias declaradas, se avisa para que se completen.
- Un documento demasiado grande se recorta por sección.
- Si el modelo no puede decidir, se usa por defecto el objetivo declarado por la etapa.

### Agentes

- `plan` solo tiene herramientas que leen. No tiene ninguna forma de escribir.
- `build` tiene el catálogo completo y es el único que crea, modifica y borra.
- El relevo es explícito: `plan` investiga, propone, tú apruebas, cambias a `build`, `build` aplica.
- Una aprobación vale para el cambio propuesto, no para lo que siga. Si `build` necesita algo que `plan` no propuso, vuelve a preguntar.
- El relevo no se puede saltar cambiando de agente con un trick: no hay interruptor, son dos catálogos distintos.
- Los agentes se definen en carpetas con dos archivos —`agent.yaml` y `prompt.md`—, no en el código. El agente base existe en disco, y se puede modificar o derivar de él otros sin recompilar.
- El nombre del agente lo decide `name`, no el nombre de la carpeta.
- Un `agent.yaml` con un campo desconocido se rechaza, incluidos `prompt`, `herramientas` y `skills`. Un `*.json` de agente, del formato antiguo, se avisa nombrándolo y no se carga.
- `permissions` es un mapa de permiso a efecto con tres permisos (`read`, `write`, `edit`); el catálogo efectivo se deriva de él y no se declara aparte.
- `default` es opcional y solo admite `deny`; ausente equivale a `deny`. Un `default: allow` se rechaza con el motivo, porque concedería todo en silencio.
- Un permiso o un efecto fuera de vocabulario se rechaza, no se ignora.
- Sin `permissions`, el agente es de solo conversación.
- Un agente sin `write` ni `edit` no puede escribir en el proyecto, aunque se le pida. Es lo que hace que la garantía se sostenga en datos y no en una promesa del código.
- **`crear_todo` añade un paso al final y no toca los que ya había; `actualizar_todo` reemplaza la lista entera.** Las dos devuelven la lista resultante.
- **No hay skills.** No es que no haya ninguna por defecto: la capacidad no está implementada y ningún sitio las declara. Ver [[specs/SPEC-SKILLS]].
- En el chat, cada agente responde con las herramientas derivadas de sus permisos; el chat no añade ni quita herramientas.

### Herramientas y terminal

- El catálogo es cerrado: el agente no puede inventar herramientas.
- Los comandos de la lista blanca (compilar, probar, revisar estilo y tipos, ver estado y diferencias de git) se ejecutan sin preguntar.
- Cualquier otro comando pide aprobación antes de ejecutarse.
- La terminal no puede crear, editar ni borrar archivos del proyecto, y esa garantía es estructural, no depende de revisar el texto del comando.
- La terminal no puede descartar cambios del repositorio, hacer commits ni subir cambios.
- Un comando que no termina o genera salida sin fin se corta y se avisa; no puede colgar el harness ni llenar el contexto.
- Un comando que falla no detiene el trabajo: el agente ve el error y sigue.
- El comando corre en la carpeta del proyecto.

## 2. Invariantes

Condiciones que siempre se cumplen, sin excepción:

- **Ninguna escritura sin aprobación.** Ni dentro de la carpeta, ni fuera, ni dentro de un flujo en curso.
- **`plan` nunca escribe.** No es una política que se pueda desactivar: sencillamente no tiene en su catálogo ninguna herramienta de escritura ni de edición.
- **El modelo nunca ve el proyecto entero.** Ve lo que el nodo de contexto decidió y recortó.
- **Lo que entra al contexto se leyó antes** y **lo que salió queda registrado**.
- **La cola refleja los archivos de tarea**, no una copia que pueda quedar vieja.
- **El contenido de una sesión no se filtra a otra.**
- **El historial de cambios no se borra.** Sobrevive a la sesión que lo produjo.
- **Una etapa que falla detiene el flujo.** No se encadena la siguiente como si nada.
- **La inferencia se serializa** entre sesiones que comparten modelo.
- **Esperar permiso o terminar siempre notifica**, estés donde estés.

## 3. Casos Especiales

- **Una etapa necesita permiso:** la etapa espera, el flujo queda pausado, aparece la aprobación y se manda la notificación. Al aprobar, se retoma desde esa misma etapa.
- **El usuario cancela con un flujo en marcha:** se pregunta qué hacer con el flujo antes de cerrar la sesión. Cancelado, no queda ninguna etapa corriendo.
- **El usuario vuelve al día siguiente:** las sesiones se listan y se retoma cualquiera con su historial. Un flujo pausado por permiso se retoma donde estaba.
- **Un elemento del TODO ya empezado:** se retoma sin repetir lo ya completado.
- **Termina una sesión que no estabas mirando:** te llega la notificación, y al volver tienes el resultado en su historial.
- **El modelo elige mal el contexto:** el registro de auditoría muestra qué entró y qué salió; el usuario puede consultarlo y corregir las dependencias declaradas.
- **El modelo pide internet y la búsqueda necesita el proyecto:** el agente te lo pide a ti en lugar de mandar el contenido del proyecto. Por internet solo sale la consulta.
- **Una capa termina antes que la otra:** su cola se detiene y avisa, la otra sigue. Ninguna bloquea a la otra.
- **La tarea activa depende de otra bloqueada:** no arranca hasta que se cumpla, y se marca con lo que falta.

## Referencias

- [[backend/01-domain/DOMAIN]] — los módulos y sus límites.
- [[database/02-rules/BUSINESS_RULES]] — reglas que dependen de los datos.
- [[backend/02-interfaces/TOOLS]] — el catálogo y los controles de la terminal.
- [[backend/04-infrastructure/EVENTS]] — cómo se comunican estos cambios.
