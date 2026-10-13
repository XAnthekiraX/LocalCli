# 008-task-agente-base.md

> T-B008 — Agente base y construcción de la llamada (prompt de sistema + esquemas).
> Acción de esta descomposición: `crear`.

## Referencias

- [[specs/SPEC-AGENTE-BASE]] §El ciclo de un turno, §Un agente sin herramientas, §Los dos agentes, §Reglas de negocio
- [[backend/01-domain/DOMAIN]] §1 Módulos (`agent`), §2 Entidades del dominio
- [[backend/01-domain/BUSINESS_RULES]] §Agentes, §Invariantes
- [[backend/02-interfaces/INTERFACES-GENERAL]] §5 Contratos (`agent`)
- [[backend/02-interfaces/TOOLS]] §2 El reparto, §4 El cable: cómo llegan al modelo
- [[backend/DECISIONS]] — el ciclo conversacional vive en `agent.Ejecutor`; permisos derivan el catálogo
- [[database/01-schema/ENUMS]] §Rol de mensaje — los mensajes de herramienta no se persisten

## Contexto

`agent` carga las definiciones desde carpeta (`agent.yaml` + `prompt.md`, ya cargadas de forma aditiva en T-B002) y corre el ciclo conversacional del agente activo. Del `agent.yaml` salen el nombre y las herramientas —derivadas de `permissions`—, y del `prompt.md` el mensaje de sistema, que se inyecta **íntegro**. El catálogo de herramientas **no** viaja en el prompt: va por el canal nativo del motor, junto a la definición de cada herramienta.

`agent` no ejecuta herramientas: pide sus esquemas a `tools` y despacha cada ejecución. Si el modelo en uso no admite herramientas (capacidad `no soportada` o `desconocida`), cae a **modo conversación** y avisa, en vez de fingir que trabaja. El ciclo se acota con un máximo de rondas y cierra con una redacción sin herramientas.

El catálogo de herramientas y su capa universal de ejecución son de T-B012: aquí el agente consume los esquemas y despacha contra esa frontera, sin implementarla.

## Tareas pequeñas

| ID        | Acción | Tarea                                                                                                 | Estado    | Archivos                     | Verificación                                                                   |
| --------- | ------ | ----------------------------------------------------------------------------------------------------- | --------- | ---------------------------- | ------------------------------------------------------------------------------ |
| T-B008-01 | crear  | Catálogo efectivo del agente: derivar las herramientas de sus `permissions` contra el catálogo cerrado | completada | `internal/agent/agente.go`   | `plan` (solo `read`) no incluye ninguna herramienta de escritura               |
| T-B008-02 | crear  | Construir la petición: `prompt.md` íntegro como mensaje de sistema; sin el catálogo en el prompt       | completada | `internal/agent/peticion.go` | El prompt va entero y el catálogo no aparece dentro del mensaje de sistema     |
| T-B008-03 | crear  | Añadir al canal de herramientas los esquemas del catálogo efectivo del agente activo                   | completada | `internal/agent/peticion.go` | La petición lleva el esquema solo de las herramientas concedidas               |
| T-B008-04 | crear  | Caer a modo conversación si la capacidad de herramientas es `no soportada` o `desconocida`, avisando   | completada | `internal/agent/ejecutor.go` | Sin capacidad `soportada` no se ofrecen herramientas y la interfaz lo avisa    |
| T-B008-05 | crear  | Ciclo conversacional: modelo → herramienta → resultado → modelo, con tope de rondas                    | completada | `internal/agent/ejecutor.go` | Un turno de leer y escribir hace al menos dos rondas y se detiene en el tope   |
| T-B008-06 | crear  | Herramientas del turno en el orden pedido, una tras otra; un fallo corregible vuelve al modelo         | completada | `internal/agent/ejecutor.go` | Un fallo corregible no corta el turno y las ejecuciones respetan el orden       |
| T-B008-07 | crear  | Cierre con redacción sin herramientas; un reintento; si no entrega, falla con el motivo a la vista     | completada | `internal/agent/ejecutor.go` | Agotado el tope, el turno redacta; sin texto, falla con aviso visible          |

**Dependencias:** `01`→`02`→`03`; `03`→`04`; `03`→`05`→`06`→`07`.
Fuera de alcance: el contenido de los agentes base (sembrado en T-B000 y cargado en T-B002); la selección de documentos del contexto (T-B009); la persistencia de sesiones y mensajes (T-B010); la TUI de chat (T-B011).
