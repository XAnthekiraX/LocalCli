# IDEA — LocalCli

## Resumen

Harness de terminal especializado en modelos locales (Ollama), pensado para máquinas pequeñas (4 GB de VRAM, 16 GB de RAM como mínimo) y para uso personal. Su tesis es que con modelos pequeños el cuello de botella no es el modelo sino el contexto: se entrega solo la información necesaria, mediante flujos de trabajo que encadenan etapas y van reduciendo el contexto en cada paso. Está enfocado a desarrollo de software: planear, crear tareas y ejecutarlas.

## Problema

Las herramientas open source del estilo opencode están diseñadas asumiendo modelos grandes en la nube o en GPU con memoria. Al conectarlas a Ollama en un equipo modesto:

- El contexto que se le manda al modelo no encaja en 4 GB de VRAM junto con el modelo, y las respuestas se degradan o la máquina se queda sin memoria.
- Se manda toda la documentación y todo el historial, sin filtrar, así que el modelo se distrae con información irrelevante y responde más lento.
- No existen flujos que automaticen el ciclo de trabajo (planear → crear tareas → ejecutar tareas) entregando a cada etapa solo lo que esa etapa necesita.

El resultado es una herramienta que, con modelos locales, se siente lenta, imprecisa y frágil.

## Solución

Un harness de terminal en Go, construido sobre la experiencia de herramientas como opencode (misma familiaridad: sesiones por proyecto, herramientas, agentes, skills) pero no una copia: se especializa en lo que las otras no hacen bien con modelos locales.

Sus tres pilares:

1. **Perfil de modelo para hardware pequeño.** Habla con Ollama y ajusta el modelo, el tamaño de contexto y las opciones de inferencia para que todo quepa en 4 GB de VRAM.
2. **Flujos de trabajo listos para usar.** Flujos predefinidos que ejecutan una tarea completa de desarrollo (por ejemplo: planear el proyecto, generar tareas, ejecutar una tarea). Son procesos encadenados en secuencia donde cada etapa recibe únicamente el contexto que necesita, no el historial completo. Trae flujos oficiales y el usuario puede definir los suyos en configuración.
3. **Nodo de contexto.** Dentro de cada flujo, una etapa se encarga de recopilar la información de partida (por ejemplo, la documentación ya escrita del proyecto), elegir el subconjunto suficiente para la tarea en curso, y pasarlo a la etapa siguiente. El modelo recibe poco y relevante, lo que reduce consumo de VRAM, acelera la ejecución y mejora la calidad de la respuesta.

   La propia documentación declara qué otros documentos necesita: un documento de APIs indica qué DTOs usa, un documento de DTOs indica qué entities necesita, y así sucesivamente. El frontmatter con enlaces funciona como **mapa de navegación** por el que el harness recorre el proyecto, pero la decisión de qué es relevante la sigue tomando el modelo. Se guarda y se muestra qué documentación entró en cada etapa y cuál quedó fuera.

4. **Varias sesiones por proyecto.** Cada carpeta abierta es un proyecto y puede tener varias sesiones. Cada sesión tiene su contexto completamente independiente: se puede dejar una sesión ejecutando el frontend en segundo plano y abrir otra para el backend. El trabajo sigue corriendo aunque no estés mirando esa sesión.

Además, las tareas, la documentación, las skills y los agentes se guardan en base de datos local.

## Público

Uso personal del autor. Público secundario: cualquier desarrollador con una máquina modesta que quiera trabajar con modelos locales en un ciclo de desarrollo completo sin depender de servicios en la nube.

## Alcance inicial

Incluye:

- Chat en terminal con historial separado por ruta de trabajo.
- Integración con Ollama y perfil de modelo ajustado a 4 GB de VRAM / 16 GB de RAM.
- Panel de información: razonamiento del modelo, tokens usados, contexto cargado.
- Persistencia de conversaciones, flujos, tareas y estado del agente en base de datos local.
- Operaciones de archivo y carpeta: crear, leer, editar, borrar, en cualquier formato de texto.
- Varias sesiones por proyecto, con contexto independiente y ejecución simultánea en segundo plano con estado visible.
- Flujos de trabajo oficiales y predefinidos: planear proyecto, crear tareas, ejecutar tarea.
- Nodo de contexto que filtra y entrega solo la información necesaria por etapa, con registro de lo seleccionado y lo descartado.
- Agente base incluido, listo para usar.
- Toda escritura pide aprobación antes de aplicarse, también dentro de un flujo en curso.

Personalización (se puede diferir sin bloquear el resto):
- Agentes propios definidos por el usuario.
- Flujos propios definidos por el usuario.
- Carga de skills.

No incluye en esta primera versión:

- Interfaz gráfica o web.
- Multiusuario, cuentas o sincronización en la nube.
- Ejecución remota del harness.
- Proveedores de modelos distintos de Ollama.
- Ejecución del harness en una máquina distinta de la que corre Ollama.

## Resultados esperados

- Abrir la herramienta en una carpeta con un equipo modesto y conversar con un modelo local que ya funciona, sin configurar nada a mano.
- Ver en un vistazo cuántos tokens se usaron y qué parte del contexto está ocupada, para entender por qué una respuesta es lenta o mala.
- Retomar la conversación exactamente donde se dejó, en la misma carpeta, y no ver conversaciones de otras carpetas.
- Dejar una sesión trabajando en segundo plano, abrir otra para otra cosa y volver después para ver el resultado.
- Lanzar "crear tarea": el flujo reconoce los archivos relevantes, busca el contexto, lo optimiza y devuelve una tarea ya con su contexto adjunto.
- Lanzar "ejecutar tarea" y el harness implementa los cambios en los archivos por su cuenta. Antes de tocar nada te muestra el cambio propuesto y espera tu aprobación.
- Ver qué documentación recibió el modelo en cada etapa y cuál quedó fuera, con el motivo del descarte.
- Pedirle al agente que cree, lea, edite o borre archivos y carpetas del proyecto.
- Navegar la interfaz con atajos de teclado, iguales a los de una herramienta que ya conoces.
- Ver la interfaz como un chat a pantalla completa, con un panel de datos que se pliega a la derecha: sesión, contexto ocupado, tarea en curso, ruta, git, cola, aprobaciones pendientes, agente activo y versión.
- Ver el razonamiento del modelo en vivo, mientras genera, antes de la respuesta.

## Restricciones y supuestos

- Hardware base: 4 GB de VRAM y 16 GB de RAM como mínimo. Es una restricción dura de diseño, no una optimización opcional. En la práctica la RAM es el recurso que se agota primero: el modelo pesa poco en la GPU y el consumo fuerte está en el proceso y el contexto.
- Uso personal: no hace falta multisuario, permisos ni aislamiento de datos más allá del de los sistemas de archivos.
- Ollama corre de forma local y es la única fuente de modelos en esta versión.
- El harness planifica y ejecuta: puede escribir y modificar los archivos del proyecto por sí mismo, pero nunca sin que apruebes el cambio antes.
- Dentro de la carpeta abierta el agente tiene acceso a todos los archivos y carpetas sin preguntar. Si necesita salir de ella, debe pedir permiso y explicar por qué busca eso.
- Todas las escrituras, incluso dentro de un flujo, pasan por aprobación. Una sesión en segundo plano que necesita permiso queda esperando y lo indica claramente.
- Los flujos de trabajo se ejecutan en secuencia y en un solo proceso; la recuperación ante fallos a mitad de un flujo hay que diseñarla.
- Varias sesiones del mismo proyecto pueden trabajar a la vez, cada una con su contexto independiente. Más de una sesión ejecutando flujo queda para una versión posterior.

## Decisiones pendientes

- Arquitectura de referencia: estudiar cómo resuelve opencode el ciclo de sesión, herramientas y agentes, para acercarnos en experiencia sin copiar su estructura.
- Composición interna del ciclo de desarrollo: qué etapas hay entre planear, crear tareas y ejecutar. Está definido el mecanismo de etapas, no la lista definitiva.
- Tercera opción del panel de aprobaciones, además de aprobar y declinar.
- Formato exacto del bloque de contexto que se adjunta a una tarea.
- Formato y ubicación de la definición de flujos y agentes propios por parte del usuario.
- Comportamiento del flujo cuando una etapa falla a mitad: la regla general es detenerse y preguntar, falta concretizar las opciones de reintento.
- Alcance del trabajo de varias sesiones concurrentes sobre la misma base de datos.
