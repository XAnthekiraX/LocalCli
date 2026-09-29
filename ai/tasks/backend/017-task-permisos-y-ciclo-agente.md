> T-B017 — permisos explícitos por acción y un único ciclo conversacional: el agente declara `permisos` (`permitir`/`denegar` por acción) y su catálogo de herramientas se deriva de ellos; el bucle modelo → herramienta → resultado → modelo vive en `agent` y el catálogo del agente se inyecta en su mensaje de sistema.
> Acción de esta descomposición: `actualizar` (sobre T-B006 agent y T-B007 tools).

## Referencias

- [[specs/SPEC-AGENTE-BASE]] — §Dónde se definen: campos `nombre`, `descripcion`, `prompt`, `permisos`, `skills`.
- [[specs/SPEC-TOOLS]] — §El reparto por acciones y presentación del catálogo al modelo.
- [[backend/DECISIONS]] — JSON de agente con `permisos`; ciclo en `agent.Ejecutor`; catálogo inyectado.
- [[backend/02-interfaces/TOOLS]] — §2 el reparto por acciones, §7 el enrutado.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-B017-01 | actualizar | Sustituir el campo `herramientas` por `permisos` (acción × efecto), default deny y sin contradicciones | completada | `internal/agent/model.go`, `internal/agent/catalog.go`, `.localcli/agents/*.json` | Test: un JSON con `herramientas` se rechaza; permisos contradictorios se rechazan; el catálogo efectivo se deriva |
| T-B017-02 | actualizar | Añadir acciones (`leer`/`editar`/`ejecutar`/`internet`) al catálogo, derivadas de categoría y modo | completada | `internal/tools/catalog.go`, `internal/tools/permission.go` | Test: las cuatro acciones cubren las trece herramientas sin solaparse |
| T-B017-03 | actualizar | Extraer el ciclo conversacional a `agent.Ejecutor` con `Sink` y adelgazar el arranque | completada | `internal/agent/loop.go`, `arranque.go` | Test: texto directo en una pasada; herramienta → resultado → respuesta; tope de pasadas |
| T-B017-04 | actualizar | Inyectar el catálogo del agente y el formato de llamada en el mensaje de sistema | completada | `internal/agent/run.go`, `internal/tools/describe.go` | Test: el sistema lista las herramientas del agente; `plan` no ve escritura |
| T-B017-05 | actualizar | Leer el modelo elegido por turno al llamar al modelo (regresión del selector de modelos) | completada | `arranque.go` | El cambio de modelo en el selector se refleja en la siguiente respuesta |

Dependencias: T-B017-01 y T-B017-02 son la base del reparto; T-B017-03 y T-B017-04 van después; T-B017-05 es independiente.
