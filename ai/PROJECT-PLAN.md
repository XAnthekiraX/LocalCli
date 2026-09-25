# PROJECT-PLAN.md

## Fases del Proyecto

- [x] FASE 0 - IDEA/VISIÓN
- [x] FASE 1 - ESPECIFICACIÓN FUNCIONAL
- [x] FASE 2 - PROPUESTA TÉCNICA
- [x] FASE 3 - DISEÑO POR SECCIÓN/MÓDULO
- [x] FASE 4 - DOCUMENTACIÓN POR CAPAS
  - [x] Backend: 12 documentos en ai/docs/backend/
  - [x] Database: 12 documentos en ai/docs/database/
  - [x] Frontend: 4 documentos en ai/docs/frontend/ (ampliado con la pantalla de bienvenida estilo opencode)
  - [x] Logotipo canónico fijado: internal/tui/testdata/logo.txt (6 filas x 53 columnas), salida dorada de la prueba de bienvenida
  - [x] Aprobación de la capa frontend y de las 7 decisiones cerradas
- [x] FASE 5 - ROADMAP (adaptado: el detalle vive en ai/tasks/ y se genera al iniciar la ejecución)
- [x] FASE 6 - DOCUMENTACIÓN FINAL
  - [x] Validación ejecutada: veredicto CONSISTENT, 528 enlaces sin rotos (reporte en ai/DOCS-VALIDATION-REPORT.md)
  - [x] Hallazgos corregidos: W01, W02, W03, I01 (I02 aceptado, O01 documentado)
## Capas

En una aplicación web estas capas se separan por red. Aquí es un proceso único: se comunican por canales de Go. **Database** es persistencia, **Backend** es el motor, **Frontend** es la interfaz de terminal.

### Frontend

**M13 `tui`** — Toda la pantalla Bubble Tea: chat, panel de datos, selector de sesiones, panel de aprobaciones y razonamiento en vivo.

- Límite: presentación pura. No ejecuta nada por su cuenta ni escribe en disco.
- Entradas: teclado, y eventos de `session`, `flow`, `store` y el streaming de `ollama`.
- Salidas: peticiones a `session` (escribir, cambiar de sesión, aprobar o declinar).
- Dependencias: `session`, `flow`, `store`, `ollama`.
- Reglas: el razonamiento se muestra en vivo antes de la respuesta y se puede ocultar; el panel de datos es de solo lectura; con el panel cerrado se ve cuántas aprobaciones hay pendientes; cambiar de sesión no detiene nada.
- Verificación: los nueve datos del panel visibles; streaming en vivo; cambiar de sesión no detiene otra; aprobaciones de cualquier sesión visibles.

### Backend

**M4 `session`** — Ciclo de vida de las sesiones, sus estados y la ejecución en segundo plano.

- Límite: no ejecuta tareas; eso es `flow` y `queue`.
- Entradas: petición del usuario, cambio de sesión, resultado de etapa. Salidas: estado de sesión y eventos a la TUI.
- Dependencias: `flow`, `store`, `ollama`, `tools`.
- Reglas: dos sesiones nunca comparten contexto; una sesión que espera permiso muestra ese estado.
- Verificación: dos sesiones simultáneas quedan aisladas; una bloqueada muestra su estado.

**M5 `flow`** — Motor de etapas y encadenamiento. Ejecuta los ciclos oficiales de planificación, trabajo y resolver.

- Límite: orquesta; no habla con el modelo ni con herramientas directamente.
- Entradas: objetivo, tipo de flujo, contexto de etapa. Salidas: resultado y decisión de seguir, detener o esperar.
- Dependencias: `session`, `context`, `agent`, `queue`, `store`.
- Reglas: ninguna etapa arranca sin contexto; un fallo detiene el flujo; un permiso lo pausa sin matarlo.
- Verificación: N etapas encadenadas; un flujo pausado se reanuda desde la misma etapa sin repetir.

**M6 `queue`** — Cola por capa, orden por dependencias, consumo autónomo y manejo de bloqueadas.

- Límite: no ejecuta; entrega la siguiente tarea elegible. No es una cola global.
- Entradas: archivos de tarea, tareas completadas o bloqueadas. Salidas: siguiente tarea y estado de la cola.
- Dependencias: `task`, `store`, `flow`.
- Reglas: una subtarea por iteración; una bloqueada no frena la cola si nada depende de ella; una capa no bloquea a otra.
- Verificación: orden por dependencias correcto; bloqueada saltada y registrada; la cola arranca sola.

**M7 `context`** — Nodo de contexto: arma lo que recibe cada etapa.

- Límite: la relevancia la decide siempre el modelo; el harness solo recorta y registra.
- Entradas: objetivo de la etapa, grafo de documentos, límite de contexto. Salidas: contexto dentro del límite más el registro de auditoría.
- Dependencias: `docs`, `ollama`, `store`, `fileops`.
- Reglas: lo entregado nunca supera el límite; todo descarte queda registrado con su motivo.
- Verificación: el recorte reduce de forma medible frente a leer el proyecto entero; lo entregado cabe.

**M8 `agent`** — Define `plan` y `build` con sus herramientas, y el relevo entre ellos. Alberga agentes propios y skills.

- Límite: no ejecuta herramientas; las despacha a `tools` según el grant de cada agente.
- Entradas: petición, estado del relevo, herramientas solicitadas. Salidas: llamada al modelo y peticiones de herramienta.
- Dependencias: `ollama`, `tools`, `context`.
- Reglas: `plan` sin herramientas de escritura, `build` con todas; toda escritura pasó por propuesta y aprobación; la aprobación vale solo para lo propuesto.
- Verificación: `plan` no logra una herramienta de escritura; `build` sí; el relevo se exige antes de aplicar.

**M9 `ollama`** — Cliente HTTP a Ollama con streaming, extracción de razonamiento y perfil de hardware.

- Límite: solo habla con Ollama; no decide qué documentos son relevantes.
- Entradas: prompt, modelo, parámetros. Salidas: respuesta en streaming separada en razonamiento y respuesta, conteo de tokens y errores.
- Dependencias: ninguna; es el borde exterior.
- Reglas: un modelo que no cabe se rechaza con aviso; si Ollama no está, avisa con instrucción clara; la inferencia se serializa entre sesiones.
- Verificación: el streaming se ve en vivo; razonamiento y respuesta llegan separados; un modelo que no cabe se rechaza.

**M10 `tools`** — Registro de las trece herramientas, comprobación de permiso y enrutado.

- Límite: no inventa herramientas; los permisos los aplica `fileops`.
- Entradas: petición de herramienta con su grant. Salidas: resultado o error.
- Dependencias: `fileops`, `exec`, `session`, `store`.
- Reglas: catálogo cerrado; una herramienta fuera del grant falla; toda escritura pasa por aprobación.
- Verificación: `plan` no obtiene herramienta de escritura; cada herramienta del grant responde; fuera de grant se rechaza.

**M11 `fileops`** — Aplica operaciones de archivo y carpeta, valida la frontera de rutas y guarda el historial.

- Límite: no ejecuta comandos de terminal.
- Entradas: operación, ruta, contenido y aprobación. Salidas: archivo en disco y registro de cambio con lo anterior y lo nuevo.
- Dependencias: `store`, `session`.
- Reglas: fuera de la carpeta exige permiso y explicación; borrar exige confirmación explícita; sobrescribir es distinto de crear.
- Verificación: escribir pide aprobación; fuera de la carpeta pide explicación; el historial permite revertir.

**M12 `exec`** — Terminal con lista blanca y bloqueo estructural de escritura mediante Landlock.

- Límite: no puede escribir en los archivos del proyecto.
- Entradas: comando y carpeta de trabajo de la sesión. Salidas: salida acotada, código de salida y aviso si se cortó.
- Dependencias: `session` y Landlock, que es del sistema.
- Reglas: la lista blanca corre sin preguntar y el resto pide aprobación; sin checkout, reset, clean, commit ni push; salida y tiempo limitados.
- Verificación: un comando que intente escribir en el proyecto falla; un comando fuera de la lista pide aprobación; una salida excesiva se corta con aviso.

### Database

**M1 `store`** — Único acceso a SQLite: esquema, WAL y transacciones.

- Límite: no sabe de agentes, flujos ni interfaz; no decide permisos.
- Entradas: operaciones tipadas de sesión, mensaje, cola y auditoría. Salidas: estado persistido.
- Dependencias: ninguna.
- Reglas: WAL activo; una transacción por operación lógica; si una escritura falla, la etapa se marca con error en vez de perderse en silencio.
- Verificación: escritura y lectura de cada tabla; una transacción fallida no deja estado parcial.

**M2 `docs`** — Carga la documentación del proyecto, lee el frontmatter y construye el grafo de dependencias.

- Límite: solo lee; no decide relevancia; no sale de la carpeta del proyecto.
- Entradas: raíz del proyecto. Salidas: lista de documentos con sus dependencias.
- Dependencias: `fileops`.
- Reglas: un documento sin dependencias declaradas se avisa; una dependencia rota se detecta y se informa.
- Verificación: grafo válido; dependencia rota detectada; documento sin frontmatter avisado.

**M3 `task`** — Lee y escribe los archivos `NNN-task-<nombre>.md`.

- Límite: el archivo es la fuente de verdad; la posición en la cola no vive aquí.
- Entradas: ruta, acción y contexto. Salidas: archivo en disco y metadatos leídos.
- Dependencias: `store`, `queue`.
- Reglas: la acción es `crear`, `actualizar` o `eliminar`; toda escritura pasa por aprobación.
- Verificación: escribir y leer conserva todos los campos; una acción inválida se rechaza.

## Roadmap

El roadmap detallado no se mantiene aquí. Al iniciar la ejecución de cada capa se genera en archivos, que es donde vive mientras se trabaja:

- [ ] `ai/tasks/backend/MAIN-TASKS.md` — se genera al iniciar la ejecución del backend.
- [ ] `ai/tasks/frontend/MAIN-TASKS.md` — se genera al iniciar la ejecución del frontend.
- [ ] Cada archivo `NNN-task-<nombre>.md` identifica la acción de sus subtareas: `crear`, `actualizar` o `eliminar`.
