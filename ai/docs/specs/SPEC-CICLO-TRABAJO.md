# SPEC — Ciclo de trabajo

Prioridad: P0 (núcleo)

## Propósito

El flujo oficial que convierte un cambio pedido en una tarea con su contexto adjunto y la deja lista para ejecutarse. Trabaja siempre junto a la documentación: la tarea nunca va por delante de ella.

## Alcance

Incluye el flujo único de cambio con sus tres entradas: crear, actualizar y eliminar.
No incluye la cola que las ejecuta, que está en [[specs/SPEC-COLA-TAREAS]], ni la documentación del proyecto desde cero, que está en [[specs/SPEC-CICLO-PLANIFICACION]], ni el arreglo de problemas, que está en [[specs/SPEC-RESOLVER]].

## Actores

- **Usuario**: pide el cambio, aprueba el plan, aprueba cada propuesta.
- **Agente `plan`**: analiza el impacto, actualiza la documentación y produce la tarea con su contexto adjunto. No escribe.
- **Agente `build`**: escribe la documentación y la tarea aprobadas, y luego implementa.
- **Nodo de contexto**: decide qué entra en la tarea.

## Un solo flujo, tres entradas

El flujo es siempre el mismo. Lo único que cambia es por dónde se entra y qué acción declara la tarea.

```
impacto  →  plan  →  documentación  →  tarea  →  cola
```

| Entrada | Cuándo | Acción |
|---------|--------|--------|
| Crear | La funcionalidad no existía en el proyecto | `crear` |
| Actualizar | La funcionalidad ya existe y se modifica | `actualizar` |
| Eliminar | La funcionalidad se retira del proyecto | `eliminar` |

La entrada la determina lo que pidió el usuario, no el flujo. La acción la describe lo que hará la implementación, no lo que se borra del archivo de planificación.

## El esqueleto, paso a paso

1. El usuario pide el cambio, indicando la tarea afectada si la conoce.
2. El flujo analiza el impacto: qué funcionalidades, entidades, endpoints, componentes, flujos y datos se ven afectados.
3. `plan` presenta el plan y **espera confirmación**.
4. `plan` propone la documentación de cada archivo afectado, uno a la vez. Cada uno se aprueba antes de pasar al siguiente, y `build` lo escribe.
5. `build` crea o actualiza el elemento del TODO con su acción y su contexto.
6. El elemento entra en la cola.
7. `build` lo ejecuta, un elemento por iteración.

Si la documentación no refleja el cambio, la tarea no se crea. Es la regla que evita que `build` implemente algo que nadie escribió.

## Entrada 1 — crear

Ejemplo: en una aplicación del clima ya construida, se pide generar una imagen para compartir el clima en redes sociales. No estaba solicitado al inicio, así que es funcionalidad nueva.

1. Se reconoce qué archivos y documentos son relevantes.
2. Se busca el contexto de esos archivos.
3. Se optimiza ese contexto reduciéndolo a lo necesario.
4. `plan` propone la especificación de la funcionalidad.
5. `plan` propone los documentos afectados, uno a la vez, y `build` los escribe tras tu aprobación.
6. Se crea la tarea con acción `crear`, su contexto, su ubicación y su alcance.
7. Entra en la cola.

## Entrada 2 — actualizar

Ejemplo: la generación de imágenes existe, pero ahora hay que añadirle marca de agua.

1. Se identifica la tarea existente, por identificador o por nombre.
2. Se lee su especificación y su documentación para entender el estado actual.
3. Se determina exactamente qué cambia: entidades, endpoints, servicios, DTOs, componentes, estados, flujos.
4. Se presenta el plan y se espera confirmación.
5. `plan` propone la especificación y los documentos afectados, uno a la vez, y `build` los escribe.
6. Se actualiza la tarea con acción `actualizar`.
7. Entra en la cola.

### Reglas de actualizar

- Si el usuario indicó una tarea existente, se actualiza esa. No se crea una tarea nueva que duplique su alcance.
- Los elementos no afectados conservan su acción y su estado.
- Una tarea ya completada no se modifica salvo que el usuario lo pida explícitamente.
- Si el cambio no cabe en ninguna tarea existente, se crea una nueva, y su acción sigue siendo `actualizar`.
- Nunca se usa la acción `crear` para un cambio de actualización.

## Entrada 3 — eliminar

Ejemplo: se decide que la generación de imágenes sobra.

1. Se identifica qué se elimina y qué tareas la consumen.
2. Se buscan referencias en todo el proyecto: llamadas, rutas, componentes, relaciones, datos.
3. Se presenta el alcance de la eliminación: qué entidades, endpoints, servicios, componentes, estados y rutas se van, qué datos se pierden, qué migraciones hacen falta, qué dependencias se pueden retirar.
4. Se **avisa de las funcionalidades afectadas y de los datos que se perderían**.
5. Se **espera confirmación explícita**. Sin ella, el flujo no continúa.
6. `plan` propone la especificación y los documentos afectados, uno a la vez, y `build` los escribe. No quedan referencias rotas.
7. Se crea o actualiza la tarea con acción `eliminar`.
8. Entra en la cola.

### Reglas de eliminar

- Nunca se elimina nada sin confirmación explícita.
- Se avisa de las dependencias afectadas antes de eliminar.
- Los datos que puedan seguir siendo necesarios se proponen migrar, no borrar.
- Una tarea nueva creada por una eliminación tiene acción `eliminar`, nunca `crear` ni `actualizar`.
- No se modifican tareas ya completadas sin autorización explícita.

## Ejecutar

La ejecución no se pide aquí. El elemento entra en la cola de [[specs/SPEC-COLA-TAREAS]] y `build` lo toma cuando le toca, un elemento por iteración, pidiendo aprobación en cada cambio.

## Flujos alternativos

- La tarea ya existe y se vuelve a pedir: se ofrece actualizarla en vez de duplicarla.
- La tarea no tiene contexto suficiente: se vuelve a buscar contexto.
- Una etapa falla: el flujo se detiene y el usuario decide.
- Falta documentación: avisa y se detiene en vez de suponer.
- El cambio no cae en ninguna capa: se registra que no aplica.
- El usuario cancela: se detiene y se conserva lo que ya estaba hecho.
- El cambio resulta ser un arreglo de algo roto, no una funcionalidad: se deriva a [[specs/SPEC-RESOLVER]].

## Reglas de negocio

- Una tarea no se ejecuta sin su contexto adjunto.
- El contexto de la tarea es el que entrega el flujo de optimización, no el proyecto completo.
- La tarea declara su ubicación dentro del proyecto.
- La acción de cada elemento del TODO coincide con la de su tarea principal.
- `plan` propone y `build` escribe. `plan` no escribe ningún archivo.
- Cada fase pide aprobación antes de pasar a la siguiente.
- Ningún cambio de código se aplica sin aprobación, también dentro de la cola.
- El ciclo no es una secuencia fija e inmutable: es un flujo oficial compuesto por etapas, y admite etapas intermedias.
- El resultado se guarda y se puede volver a ejecutar más adelante.

## Criterios de aceptación

- [ ] Crear, actualizar y eliminar son el mismo flujo con entradas distintas.
- [ ] La documentación se actualiza antes que la tarea, con aprobación archivo por archivo.
- [ ] `plan` propone cada documento y `build` lo escribe.
- [ ] El impacto se presenta y se espera confirmación antes de tocar nada.
- [ ] "Crear tarea" devuelve una tarea con contexto, ubicación y alcance, con acción `crear`.
- [ ] "Actualizar tarea" modifica la tarea existente cuando el usuario indicó una.
- [ ] "Eliminar tarea" busca referencias en todo el proyecto antes de actuar.
- [ ] "Eliminar tarea" advierte qué datos se perderían y espera confirmación explícita.
- [ ] Toda tarea y elemento del TODO declara su acción, y coincide con la de su tarea principal.
- [ ] Un cambio de actualización nunca se marca con acción `crear`.
- [ ] Un cambio de eliminación nunca se marca con acción `crear` ni `actualizar`.
- [ ] Una tarea completada no se modifica sin pedirlo explícitamente.
- [ ] Tras eliminar no quedan referencias rotas en la documentación.
- [ ] `plan` produce la tarea y `build` la ejecuta.
- [ ] La tarea queda registrada y se puede volver a ejecutar.
- [ ] Si falta información, el flujo avisa y se detiene en vez de inventar.
- [ ] Cancelar el flujo conserva lo que ya estaba hecho y lo informa.

## Requisitos no funcionales

- El contexto que recibe la ejecución es una fracción del tamaño del proyecto.
- La tarea guardada es legible y editable por el usuario.

## Dependencias funcionales

- [[specs/SPEC-MOTOR-FLUJOS]]
- [[specs/SPEC-COLA-TAREAS]]
- [[specs/SPEC-NODO-CONTEXTO]]
- [[specs/SPEC-ARCHIVOS]]
- [[specs/SPEC-AGENTE-BASE]]
- [[specs/SPEC-CICLO-PLANIFICACION]]
- [[specs/SPEC-RESOLVER]]

## Supuestos

- Las etapas internas de los flujos quedan por definir. Esta spec fija el esqueleto, no la lista.
- El archivo de tarea se apoya en la convención del proyecto, del tipo `NNN-task-<nombre>.md`, con un bloque de contexto.

## Referencias

- [[IDEA]]
