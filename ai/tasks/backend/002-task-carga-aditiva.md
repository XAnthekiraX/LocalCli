# 002-task-carga-aditiva.md

> T-B002 — Carga aditiva de agentes y flujos (globales + propios del proyecto).
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[backend/02-interfaces/INTERFACES-GENERAL]] §5 Contratos (`agent`, `flow`)
- [[backend/04-infrastructure/CONFIGURATION]] §4 Agentes del proyecto
- [[backend/DECISIONS]] — decisión 2 (agentes) y 5 (flujos): carga aditiva, colisión conservando el global
- [[specs/SPEC-AGENTE-BASE]] §Dónde se definen y §Reglas de negocio
- [[specs/SPEC-FLUJO-PERSONALIZADO]] §Supuestos

## Contexto

Los agentes base (`plan`, `build`, `chat`) viven en `~/.config/localcli/agents/` y los flujos base en `~/.config/localcli/flows/`, sembrados desde el binario por `localcli init` ([`000-task-arranque-siembra.md`](000-task-arranque-siembra.md)). Un proyecto solo **añade** agentes y flujos propios en `<proyecto>/.localcli/agents/` y `<proyecto>/.localcli/flows/`. La carga es aditiva y nunca sobrescribe: en colisión (mismo `name` de agente o mismo `comando` de flujo) se avisa y se conserva el global. Lo que no está en archivos no aparece: si un base falta, la TUI no lo lista.

## Tareas pequeñas

| ID         | Acción | Tarea                                                                                          | Estado    | Archivos                  | Verificación                                                                                |
| ---------- | ------ | ---------------------------------------------------------------------------------------------- | --------- | ------------------------- | ------------------------------------------------------------------------------------------- |
| T-B002-01  | actualizar | Cargar agentes de las dos raíces: `~/.config/localcli/agents/` y `<proyecto>/.localcli/agents/` | completada | `internal/agent/loader.go` | Ambas carpetas se cargan al arrancar; sin proyecto propio solo cargan los base                |
| T-B002-02  | actualizar | Validar cada carpeta: `agent.yaml` + `prompt.md` obligatorios; campos desconocidos rechazados   | completada | `internal/agent/loader.go` | Una carpeta sin `prompt.md` no carga; `skills:`/`herramientas:`/`prompt:` rechaza el agente   |
| T-B002-03  | crear  | Colisión por `name`: un agente propio que repite un `name` base se avisa y se conserva el base  | completada | `internal/agent/loader.go` | Un `name: build` propio produce aviso y `build` sigue siendo el base                           |
| T-B002-04  | crear  | Aviso del formato antiguo: un `*.json` suelto en un directorio `agents/` se nombra sin cargar   | completada | `internal/agent/loader.go` | Un `agents/foo.json` produce un aviso con el nombre del archivo                              |
| T-B002-05  | actualizar | Cargar flujos de las dos raíces: `~/.config/localcli/flows/` y `<proyecto>/.localcli/flows/`    | completada | `internal/flow/flujos.go`  | Ambos directorios se cargan; un JSON roto no tumba el arranque (cae a los oficiales y avisa) |
| T-B002-06  | crear  | Colisión por `comando`: un flujo propio que repite un `comando` oficial se avisa y se conserva el oficial | completada | `internal/flow/flujos.go`  | Un `comando` duplicado produce aviso y el oficial sigue ganando                              |

**Dependencias:** `01`→`02`→`03`,`04`; `05`→`06`.