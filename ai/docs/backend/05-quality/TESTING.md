---
title: LocalCli — pruebas del backend
tags: [backend, calidad]
depende_de:
  - "[[PROJECT]]"
  - "[[specs/SPEC-TOOLS]]"
relacionado:
  - "[[backend/01-domain/BUSINESS_RULES]]"
  - "[[backend/03-security/SECURITY]]"
  - "[[backend/05-quality/VALIDATION]]"
  - "[[database/03-operations/SEEDING]]"
---
# TESTING — Pruebas del backend

Qué se prueba, cómo, y qué debe sostenerse con una prueba. Ver [[PROJECT]] para la estrategia global y [[specs/SPEC-TOOLS]] para los criterios de aceptación que estas pruebas cubren.

## 1. Estrategia de Testing

| Nivel | Qué cubre | Cómo |
|---|---|---|
| Unitario | Lógica pura: grafo de dependencias, selección de contexto, orden de la cola, comprobación de permisos, reparto de herramientas, validación de rutas | Funciones aisladas, sin base de datos ni red |
| De integración | Conexión y streaming con Ollama, ciclo completo de una etapa con lectura, aplicación de un cambio con su aprobación, ciclo de la cola | Ollama real, base temporal, sistema de archivos temporal |
| De aislamiento | Un intento de escribir en el proyecto a través de la terminal, que debe fallar | Landlock real, no simulado |
| De contexto | Que el recorte reduzca de forma medible frente a leer el proyecto entero, y que lo entregado quepa en el límite del modelo | Ollama real, con el modelo que elija la prueba |

La prueba de aislamiento es la que sostiene la garantía de Landlock. Si pasa con Landlock simulado, no prueba nada: tiene que usar el mecanismo real.

## 2. Qué debe probarse

Cobertura esperada por módulo:

| Módulo | Qué hay que probar |
|---|---|
| `docs` | El frontmatter se lee bien; el grafo toma las aristas del frontmatter y no solo de los enlaces del cuerpo; una etiqueta no cuenta como dependencia; un frontmatter roto se reporta sin tumbar la carga; un fallo de carga se reconoce como `E_STAGE_FAILED` sin leer el mensaje |
| `context` | El modelo recibe solo lo que declaró necesario; el recorte cabe en el límite; lo descartado queda registrado con motivo; un documento inexistente detiene la etapa |
| `queue` | El orden respeta las dependencias; un rango se cumple entero y en el orden; un rango no se da por cumplido con un ID de otra capa; una tarea bloqueada no detiene la cola si nada depende de ella; una tarea nueva entra en su posición; reanudar no repite subtareas completadas |
| `agent` | `plan` no tiene ninguna herramienta de escritura; `build` tiene el catálogo completo; el relevo es explícito; una aprobación no habilita lo no propuesto |
| `tools` | Una herramienta fuera del catálogo se rechaza; una herramienta que el agente no tiene se rechaza; el enrutado va al módulo correcto |
| `fileops` | Dentro de la carpeta se escribe con aprobación; fuera sin explicación no; borrar pide confirmación; cada cambio queda registrado con el antes y el después |
| `exec` | La lista blanca corre sin preguntar; un comando fuera de la lista pide permiso; la salida sin fin se corta; la terminal no puede escribir en el proyecto |
| `ollama` | El streaming entrega token a token; el razonamiento se distingue de la respuesta; el perfil de hardware avisa si el modelo no cabe |
| `store` | Las escrituras son transaccionales; el borrado de sesión deja `change_history` intacto; WAL permite leer mientras se escribe; un flujo compuesto que falla en un paso no deja escrituras a medias; las escrituras concurrentes no se pierden; `id` no admite nulo en ninguna tabla; un esquema con `user_version` correcto pero DDL viejo se rechaza |
| `session` | El estado de una sesión cambia correctamente; una sesión en segundo plano sigue al cambiar de vista; el contenido no se filtra entre sesiones |
| `flow` | Las etapas van en orden; una etapa que falla detiene el flujo; un flujo pausado se retoma donde estaba; cancelar no deja etapas corriendo |

## 3. Fixtures y mocks

- **Base de datos:** base en memoria o archivo temporal por prueba, creada desde el esquema y destruida al terminar. Nunca se usa la base de un proyecto real. Ver [[database/03-operations/SEEDING]].
- **Ollama:** para las pruebas de integración, un Ollama real. Si el test debe ser rápido, se puede sustituir el cliente con un doble que emita tokens fijos, pero eso no prueba el streaming de verdad.
- **Sistema de archivos:** un directorio temporal por prueba. El proyecto de prueba tiene su propia carpeta, y las rutas se resuelven siempre relativas a ella.
- **Landlock:** no se simula en la prueba de aislamiento. Si el sistema no lo tiene, esa prueba se salta, pero no se da por buena.
- **Reloj y esperas:** para la cola y los flujos pausados, el tiempo se controla o se inyecta un reloj falso, para que las pruebas no dependan de dormir.

No hay datos de ejemplo sembrados en proyectos reales: `change_history` debe reflejar solo cambios reales. Ver [[database/03-operations/SEEDING]].

## 4. Reglas para nuevos tests

- **Una prueba, una regla.** El nombre dice la regla que comprueba: "plan no puede escribir", "una tarea bloqueada no detiene la cola". Si el nombre necesita una conjunción, probablemente son dos pruebas.
- **Toda regla de negocio de [[backend/01-domain/BUSINESS_RULES]] tiene al menos una prueba.** Si añades una regla, añades su prueba.
- **Nada se escribe sin pasar por la aprobación.** Una prueba que escriba en un directorio de prueba sin pasar por el flujo real no está probando la garantía; está probando otra cosa.
- **Las invariantes se prueban como invariantes**, no como casos sueltos. Un caso especial que rompe una invarianta es un fallo del código, no un test que se ajusta.
- **La prueba de aislamiento no se puede falsear.** Si Landlock no está disponible, se marca como no ejecutada, no como pasada. Ver [[backend/03-security/SECURITY]].
- **Nada depende del orden entre pruebas.** Cada una monta y desmonta lo suyo.

## Referencias

- [[PROJECT]] — estrategia de pruebas global.
- [[backend/01-domain/BUSINESS_RULES]] — las reglas que hay que probar.
- [[backend/05-quality/VALIDATION]] — validaciones que las pruebas cubren.
- [[backend/03-security/SECURITY]] — garantías que las pruebas de aislamiento sostienen.
- [[database/03-operations/SEEDING]] — por qué no hay datos de ejemplo.
