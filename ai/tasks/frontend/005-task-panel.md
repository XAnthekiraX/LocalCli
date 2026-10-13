# 005-task-panel.md

> T-F005 — Panel lateral de datos plegable.
> Acción de esta descomposición: `crear`.

## Referencias

- [[frontend/FRONTEND]] §2 Estructura del Proyecto (`panel.go`), §3 Contratos
- [[frontend/01-domain/DOMAIN]] §1 Componentes (`panel`), §2 Reglas de presentación
- [[frontend/02-interfaces/INTERFACES]] §3 Lecturas a `store`, §3.1 Lecturas del arranque
- [[specs/SPEC-INTERFAZ]] §3 Sidebar de datos
- [[specs/SPEC-PANEL-CONTEXTO]] (significado de los números de contexto)
- [[frontend/05-quality/TESTING]] §2 Qué debe probarse (panel refleja la sesión activa)

## Tareas pequeñas

| ID         | Acción | Tarea                                                                               | Estado    | Archivos               | Verificación                                                                     |
| ---------- | ------ | ----------------------------------------------------------------------------------- | --------- | ---------------------- | -------------------------------------------------------------------------------- |
| T-F005-01  | crear  | Modelo del panel: datos de la sesión activa y estado plegado/desplegado              | completada | `internal/tui/panel.go` | El panel guarda sus datos y si está plegado o desplegado                        |
| T-F005-02  | crear  | Título de la conversación: nombre de la sesión activa con su estado entre paréntesis | completada | `internal/tui/panel.go` | El título muestra el nombre y el estado de la sesión activa                     |
| T-F005-03  | crear  | Bloque CONTEXTO: tokens usados y porcentaje ocupado                                 | completada | `internal/tui/panel.go` | El bloque pinta tokens y porcentaje; la estimación se marca como estimación     |
| T-F005-04  | crear  | Bloque TODO: elemento en curso y cuántos quedan                                     | completada | `internal/tui/panel.go` | El bloque muestra el elemento del TODO en curso y el resto                      |
| T-F005-05  | crear  | Bloque LISTA DE TAREAS con `[•]`/`[✓]`/`[ ]`, oculto si no queda nada accionable     | completada | `internal/tui/panel.go` | Cada paso se pinta con su marca y la lista se oculta sin pasos accionables      |
| T-F005-06  | crear  | Bloque ESTADO: capa y cola, aprobaciones y agente                                   | completada | `internal/tui/panel.go` | Se muestran capa, cola, aprobaciones pendientes y agente activo                 |
| T-F005-07  | crear  | Pie de tres filas: git (o `sin iniciar`), ruta `[~…]` y firma del harness            | completada | `internal/tui/panel.go` | Las tres filas se pintan; sin git la fila dice `sin iniciar`                    |
| T-F005-08  | crear  | Plegado y despliegue con `Ctrl+D`; el panel arranca visible                         | completada | `internal/tui/panel.go` | `Ctrl+D` pliega y despliega; al arrancar está visible                           |
| T-F005-09  | crear  | Separación de dos columnas y ancho mínimo del chat (20 columnas)                    | completada | `internal/tui/panel.go` | El sidebar cede ancho antes que el chat; el chat nunca baja de 20 columnas      |
| T-F005-10  | crear  | Integrar el panel en `app` como columna derecha y reflejar la sesión activa          | completada | `internal/tui/app.go`  | El panel se pinta a la derecha y solo con los datos de la sesión activa         |

**Dependencias:** `01`→`02`–`08`; `08`→`10`; `09`→`10`.
