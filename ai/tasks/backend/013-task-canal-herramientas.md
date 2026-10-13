# 013-task-canal-herramientas.md

> T-B013 — Canal de herramientas en `llm/openai` (tipos neutros + soporte para tool_calls).
> Acción de esta descomposición: `crear`.

## Referencias

- [[specs/SPEC-TOOLS]] §Cómo se le pide una herramienta al modelo, §El canal estructurado
- [[backend/02-interfaces/TOOLS]] §4 El cable: cómo llegan al modelo
- [[backend/01-domain/DOMAIN]] §1 Módulos (`agent`, `llm`, `openai`)
- [[backend/02-interfaces/INTERFACES-GENERAL]] §5 Contratos
- T-B006 (cliente de inferencia solo texto)
- T-B008 (agente consume esquemas de `tools`, despacha sin implementarlos)
- T-B012 (catálogo y capa universal de ejecución)

## Contexto

T-B006 implementó el cliente `llm/openai` solo para respuesta de texto (streaming). El agente (T-B008) debe consumir esquemas de herramientas y despacharlas, pero el transporte necesita soportar el canal estructurado de herramientas (peticiones `tool_calls`, resultados con `role:"tool"` y `tool_call_id`, y construir la petición con `tools[]` y `tool_choice` cuando proceda). También hace falta definir/integrar los tipos neutros para herramientas en la frontera `llm` (según el diseño target: `llm` como frontera neutra).

Esta tarea añade el soporte del canal de herramientas en el núcleo común (`llm/openai`) sin duplicar el catálogo. El catálogo (T-B012) inyecta/propone esquemas; el agente (T-B008) los incluye en la petición; esta tarea garantiza que el transporte los envía, recibe y parsea correctamente (incluyendo acumulación por índice de `delta.tool_calls`), y que los resultados con `tool` se formen correctamente.

## Tareas pequeñas

| ID           | Acción | Tarea                                                                                      | Estado    | Archivos                           | Verificación                                                                 |
| ------------ | ------ | ------------------------------------------------------------------------------------------ | --------- | ---------------------------------- | ---------------------------------------------------------------------------- |
| T-B013-01    | crear  | Tipos neutros para herramientas en frontera `llm` (si faltan): `Herramienta`, petición con `tools` | completada | `internal/llm/*.go`                | Tipos compartidos y coherentes con `llm/openai`                              |
| T-B013-02    | crear  | Soportar `tools[]` y `tool_choice` en petición del cliente OpenAI-compatible              | completada | `internal/llm/openai/cliente.go`   | La petición incluye esquemas cuando el agente los pasa                       |
| T-B013-03    | crear  | Parsear `delta.tool_calls` acumulando por índice (formato OpenAI streaming)               | completada | `internal/llm/openai/cliente.go`   | Se acumulan fragmentos por `index` antes de ejecutar                          |
| T-B013-04    | crear  | Añadir soporte para mensajes con `role:"tool"` y `tool_call_id` en construcción de petición | completada | `internal/llm/openai/cliente.go`   | Resultados de herramientas se envían con `tool_call_id` correcto             |
| T-B013-05    | crear  | Test de integración del canal: streaming con tool_call completo                           | completada | `internal/llm/openai/cliente_test.go` | Test verifica acumulación y entrega de tool_calls                           |

**Dependencias:** T-B006.

**Fuera de alcance:** ejecutar herramientas (T-B012), ciclo conversacional completo (T-B008), persistencia de mensajes de herramienta (no se persisten según DECISIONS).

