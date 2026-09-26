---
title: LocalCli — decisiones técnicas
tags: [backend, decisiones]
depende_de:
  - "[[backend/BACKEND]]"
  - "[[PROJECT]]"
relacionado:
  - "[[specs/SPEC-AGENTE-BASE]]"
  - "[[specs/SPEC-CICLO-TRABAJO]]"
  - "[[specs/SPEC-INTERFAZ-ATAJOS]]"
---
# DECISIONS — LocalCli (backend)

Decisiones técnicas de la capa de motor. Lo aprobado en [[PROJECT]] y en las specs no se repite como decisión pendiente: aquí solo está lo que afecta al backend y aún necesita contexto o sigue abierto.

## 1. Decisiones Confirmadas

Ya están tomadas. Se registran aquí para que el implementador sepa qué es fijo.

| Decisión | Por qué | Alternativa descartada |
|---|---|---|
| Un solo proceso, canales de Go | Sin red entre módulos, arranque inmediato y concurrencia nativa | Varios procesos con IPC, que añadiría latencia y complejidad sin ganancia |
| Sin ORM, SQL directo en `store` | El esquema son seis tablas y las consultas son directas; un ORM estorbaría | Un ORM, que escondería las consultas y añadiría una dependencia |
| `store` es el único que escribe en SQLite | Un solo punto de escritura hace que las transacciones y el WAL sean predecibles | Que cada módulo escriba por su cuenta, que multiplicaría los problemas de bloqueo |
| `PRAGMA foreign_keys = ON` en cada conexión | SQLite ignora las cascadas sin esto; sin él, la política de borrado de sesiones no se sostiene | Dejarlo off por defecto, que es el error clásico de SQLite y rompería el borrado en cascada |
| Cola derivada de los archivos de tarea, no almacenada | La fuente de verdad son los archivos; la cola se reconstruye al arrancar y refleja el estado real | Guardar el estado de la cola en SQLite, que puede divergir del archivo y obligar a reconciliar dos verdades |
| Grafo de documentación en memoria, reconstruido al arrancar | Es pequeño y solo hace falta cuando corre una etapa; persistirlo sería otra cosa que puede quedar vieja | Persistir el grafo, que añade invalidación y no aporta consultas |
| `plan` y `build` como dos agentes separados con herramientas distintas | La garantía de que nada se escribe sin propuesta previa viene de que `plan` sencillamente no tiene cómo escribir | Un solo agente con un interruptor, que se podría activar por error y debilitar la garantía |
| Permiso comprobado en `tools`, aplicado en `fileops` y `exec` | Separar "quién puede pedir" de "quién aplica" evita que un módulo se auto-conceda permiso | Comprobar y aplicar en el mismo sitio, que mezcla decisión con efecto |
| Landlock para bloquear escritura en la terminal | Garantía del kernel, sin privilegios ni binarios externos, y no se esquiva con `find -delete` ni redirecciones | Filtrar el texto del comando, que se cuela por `tee`, tuberías o `sh -c`; o bubblewrap, que exige instalar un paquete |
| Razonamiento en tabla propia, no dentro del mensaje | El streaming escribe a menudo y el mensaje es inmutable una vez completo; separarlos simplifica ambos | Guardarlo en la misma fila, que obliga a reescribir el mensaje en cada token |
| Definir los agentes en JSON con estructura fija, no en código ni en README | Un README deja margen a interpretación de qué significa cada parte; con campos fijos el motor sabe qué leer. Permite modificar el agente base y derivar otros sin recompilar | Hardcodear el agente en Go, que obliga a recompilar para cualquier ajuste; o un README, ambiguo |
| `change_history` sobrevive al borrado de sesión con `session_id` a NULL | Es el registro de qué se tocó en los archivos; no puede depender de que la sesión siga existiendo | Borrar el historial con la sesión, que perdería lo único que no se puede reconstruir |
| Archivo SQLite en `.localcli/state.db` dentro del proyecto, y `.localcli/` fuera de git | Punto de entrada oculto y predecible: la ruta se deriva de la carpeta abierta sin variables; el estado de ejecución es local por máquina y no hay nada que versionar | Un archivo en la raíz (`localcli.db`), que mezcla estado con contenido del proyecto; versionarlo, que crearía conflictos entre máquinas |
| Serialización de la inferencia con un canal de capacidad 1 en `ollama` (FIFO) | Es idiomático en Go, no añade dependencias, respeta el orden de llegada y da un punto único donde marcar que una sesión está esperando al modelo | Dejar la cola a Ollama, que no distingue sesiones ni da visibilidad del estado de espera; una cola de prioridad, sin caso de uso porque las sesiones son iguales |
| Razonamiento en streaming: se persiste con límite de frecuencia, cada 200 ms y al terminar la respuesta | La pantalla recibe los tokens por eventos, no desde la base; la fila de `reasoning` solo existe para auditoría y recuperación. 200 ms son ~5 escrituras/s en lugar de decenas | Por token, que multiplica escrituras y churn del WAL; solo al terminar, que pierde el razonamiento parcial si el proceso muere a mitad |
| Límites de comando fijos en la primera versión: 120 s de tiempo y 10 KB de salida | Los comandos de la lista blanca (compilar, probar, lint, git de lectura) terminan con holgura dentro de esos límites en proyectos personales; 10 KB basta para errores de compilación y resúmenes de pruebas, y lo cortado se marca con `truncado` | Hacerlos configurables, que añade superficie de configuración sin un caso real; se abren más adelante si aparece |
| Solo el motor lanza colas; no hay modo de tarea suelta en la primera versión | Pausar, cancelar y re-derivar ya cubren el "quiero solo esto" (pausar detiene tras el elemento en curso); un segundo punto de entrada duplicaría el camino de ejecución y debilitaría la regla de un elemento por iteración | Modo de tarea individual lanzada por el usuario, que duplica el punto de entrada y complica la interfaz de `queue` |
| Frontmatter del elemento del TODO: `id`, `capa`, `accion`, `estado`, `depende_de`, `bloqueada_por`, `documentos`; el contexto se referencia por rutas, no se incrusta | Los campos cubren lo que la cola necesita: orden topológico por `depende_de` (con el prefijo `NNN` del nombre como desempate), estado en el archivo y acción de [[specs/SPEC-CICLO-TRABAJO]]. Referenciar documentos evita una segunda copia que diverge del original, que es justo lo que prohíbe el principio de archivos como fuente de verdad. `bloqueada_por` solo aparece con `estado = bloqueada` | Incrustar el contenido del contexto en la tarea, que duplicaría documentación, ensuciaría el historial de git y envejecería mal |
| JSON de agente con cinco campos: `nombre`, `descripcion`, `prompt`, `herramientas`, `skills`; sin herencia entre agentes | Confirma lo esbozado en [[specs/SPEC-AGENTE-BASE]] y refuerza la garantía: qué puede hacer un agente se lee en un solo archivo, sin seguir cadenas de herencia. La carga valida contra el catálogo cerrado y rechaza una herramienta inexistente; `herramientas` vacío produce un agente de solo conversación | Un campo `hereda_de`, que haría que un cambio en un padre alterara en silencio los permisos de los hijos, justo lo que la garantía auditable quiere evitar |

Las siete últimas cierran las decisiones que estaban pendientes al terminar la documentación. Quedan a la espera del visto bueno del usuario: si se rechaza una, se revierte a pendiente y se cambia por la alternativa.

## 2. Decisiones Pendientes

No queda ninguna pendiente en la capa backend. Las siete abiertas se han cerrado en la sección 1: el archivo SQLite, la serialización de la inferencia, el ritmo del razonamiento en streaming, los límites de los comandos, el lanzamiento de colas, el frontmatter del elemento del TODO y los campos del JSON de agente.

Queda abierta a nivel de spec, no de backend: la tercera opción del panel de aprobaciones de [[specs/SPEC-INTERFAZ-ATAJOS]].

## Referencias

- [[PROJECT]] — decisiones técnicas globales ya aprobadas.
- [[backend/BACKEND]] — mapa de la capa.
