# 009-task-nodo-contexto.md

> T-B009 — Nodo de contexto: selección y recorte por etapa.
> Acción de esta descomposición: `crear`.

## Referencias

- [[specs/SPEC-NODO-CONTEXTO]] §Flujo principal, §Flujos alternativos, §Reglas de negocio
- [[specs/SPEC-ORQUESTADOR-FLUJOS]] §Nodo de contexto (etapas), §Ventana de contexto del flujo
- [[backend/01-domain/DOMAIN]] §1 Módulos (`context`), §3 Relaciones de Dominio
- [[backend/01-domain/BUSINESS_RULES]] §Nodo de contexto
- [[backend/02-interfaces/INTERFACES-GENERAL]] §5 Contratos (`context`)
- [[backend/04-infrastructure/EVENTS]] — `contexto_auditado`
- [[database/01-schema/TABLES]] §`context_audit`

## Contexto

`context` decide qué pequeña parte de la documentación del proyecto se le entrega a una etapa. Lee el grafo de dependencias del frontmatter, **deja que el modelo decida** qué es relevante para el objetivo de la etapa, carga solo esos documentos, recorta lo que sobre y registra qué entró y qué salió con su motivo. El grafo no decide por sí solo: orienta.

El nodo no llama al modelo por su cuenta —solo para que elija—, y no lee fuera de la carpeta del proyecto sin permiso. La auditoría es por terna (sesión, etapa, documento): una etapa repetida conserva la fila ya registrada. La única excepción a la consulta al modelo es el **chat sin semilla**, cuyo bloque va solo con el objetivo y sin documentos.

La carga de documentación y la construcción del grafo de frontmatter son de `docs` (T-B019): aquí se consumen. El registro de auditoría en `context_audit` es de `store` (T-B010). La ventana de contexto de un flujo y su presupuesto son de T-B017.

## Tareas pequeñas

| ID        | Acción | Tarea                                                                                                          | Estado    | Archivos                         | Verificación                                                                |
| --------- | ------ | -------------------------------------------------------------------------------------------------------------- | --------- | -------------------------------- | --------------------------------------------------------------------------- |
| T-B009-01 | crear  | Pedir al modelo qué documentos necesita para el objetivo de la etapa (si no decide, usar el objetivo)          | completada | `internal/context/seleccion.go`  | Dada una etapa con objetivo, devuelve la lista de documentos elegidos       |
| T-B009-02 | crear  | Excluir de la selección lo ya entregado a etapas anteriores de la misma ejecución                               | completada | `internal/context/seleccion.go`  | Un documento ya entregado no se vuelve a ofrecer a la etapa siguiente      |
| T-B009-03 | crear  | Leer cada documento seleccionado antes de entregarlo; no leer fuera del proyecto sin permiso                    | completada | `internal/context/carga.go`      | Un documento entra al bloque solo tras leerse                               |
| T-B009-04 | crear  | Recortar por sección los documentos grandes contra el presupuesto de contexto                                   | completada | `internal/context/recorte.go`    | Un documento que desborda se recorta por sección y cabe en el tope          |
| T-B009-05 | crear  | Registrar la auditoría por (sesión, etapa, documento); todo descarte lleva motivo; repetir conserva la fila     | completada | `internal/context/auditoria.go`  | Hay una fila por documento y etapa; repetir la etapa no duplica ni falla    |
| T-B009-06 | crear  | Chat sin semilla: el bloque va con el objetivo y sin documentos, sin consultar al modelo                        | completada | `internal/context/seleccion.go`  | En chat sin semilla no se consulta al modelo y no se cargan documentos      |
| T-B009-07 | crear  | Detener la etapa si el modelo pide un documento inexistente; avisar de documentos sin dependencias declaradas   | completada | `internal/context/seleccion.go`  | Un documento pedido que no existe detiene la etapa con aviso                |
| T-B009-08 | crear  | Emitir `contexto_auditado` al terminar de armar el contexto de una etapa                                        | completada | `internal/context/auditoria.go`  | Al terminar la etapa se emite el evento con su sesión                       |

**Dependencias:** `01`→`02`→`03`→`04`→`05`→`08`; `01`→`06`; `01`→`07`.
**Fuera de alcance:** la carga de documentación y el grafo de frontmatter (T-B019); la ventana de contexto y su presupuesto por etapa (T-B017); la persistencia de `context_audit` (T-B010); el recorte del historial del chat (T-B007); la TUI (T-B011).
