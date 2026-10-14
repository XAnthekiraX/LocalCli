# 024-task-compactacion-contexto.md

> T-B024 — Compactación del historial: resumen persistido, disparo por umbral y
> por desbordamiento, formato y modelo configurables.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[specs/SPEC-HISTORIAL-CONVERSACION]] §Reglas de negocio, §Disparadores y configuración, §Formato y modelo del resumen — la fuente de verdad funcional
- [[backend/DECISIONS]] — la decisión «La compactación del historial se hace por resumen persistido en la sesión…»
- [[backend/04-infrastructure/CONFIGURATION]] §7 — el bloque `compactacion` de `config.json`
- [[database/01-schema/TABLES]] §`sessions` — `summary` y `summary_upto`
- [[database/03-operations/MIGRATIONS]] §`008-resumen-sesion` — las dos columnas aditivas
- [[specs/SPEC-PANEL-CONTEXTO]] — el porcentaje que motiva la compactación
- [[specs/SPEC-INTERFAZ-ATAJOS]] — el comando `/compactar` en la paleta

## Contexto

El historial se armaba con un presupuesto derivado de la ventana, pero sin
compactación real: cuando no cabía, se recortaba por recencia y se perdía lo
antiguo sin más. Con la ventana real ya resuelta (`023-task-ventana-contexto-real.md`),
un historial largo se entrega como **resumen + tramo reciente**, y ese resumen se
persiste para no rehacerlo. El disparo es doble: por **umbral** (una fracción de la
ventana) antes de enviar, y por **desbordamiento** —cuando el servidor rechaza la
petición real— con reintento único. El formato del resumen y el modelo que resume son
configurables. Incluye la **normalización de roles** en el cliente OpenAI, sin la
cual una plantilla estricta rechaza el resumen inyectado como `system` a mitad del
turno.

## Tareas pequeñas

| ID         | Acción     | Tarea                                                                                                                                                                                                                                                       | Estado     | Archivos                                                                 | Verificación                                                                                                       |
| ---------- | ---------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------- | ------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------- |
| T-B024-01  | actualizar | Config: tipo `Compactacion` y campo `compactacion` en `Preferencias`, con valores por defecto (`auto`, `umbral` 0.60, `conservar_tokens` 15000, `formato` `opencode`, `modelo`, `plantilla`) y métodos `*Efectivo`/`AutoActivo`                          | completada | `internal/tui/config.go`                                                  | Un `config.json` sin el bloque usa los defectos; `umbral` se acota a (0, 1]                                              |
| T-B024-02  | actualizar | Migración `008-resumen-sesion`: columnas `sessions.summary` y `sessions.summary_upto` (nullable); DDL de creación en v8, `user_version = 8`; jerarquía de `todos` renumerada a `009` (tarea T-B022)                                                       | completada | `internal/store/migraciones.go`, `internal/store/schema.go`, `internal/store/base.go` | Una base v7 abre y pasa a v8 sin tocar filas; el DDL nuevo y la migración dejan el mismo esquema                  |
| T-B024-03  | actualizar | Store: `ResumenHistorial{Texto, HastaID}` con `ObtenerResumenHistorial`, `GuardarResumenHistorial` y `LimpiarResumenHistorial`, acotados por `project_id`                                                                                                 | completada | `internal/store/sesiones.go`                                             | El resumen va y vuelve por la sesión; sin resumen `ok=false`; limpiar lo deja ausente                                |
| T-B024-04  | crear      | Paquete `compact`: formatos `opencode`/`legacy`/`custom` y prompts inicial e incremental (`Opciones.PromptInicial`, `Opciones.PromptIncremental`)                                                                                                         | completada | `internal/compact/compactacion.go`                                       | El formato por defecto pide las secciones de OpenCode; `custom` usa la plantilla; un formato desconocido cae en OpenCode |
| T-B024-05  | actualizar | `session`: `historialPara` pegajoso (el resumen persistido manda), `compactar` incremental (reutiliza y amplía), `Compactar(ctx, sesionID)` manual, `presupuestoHistorial(ventana, prefs)` con umbral y `recenciaDeCompactacion`; limpieza del resumen obsoleto al deshacer | completada | `internal/session/historial.go`                                          | Con un presupuesto pequeño se resume una sola vez; llegar mensajes nuevos amplía; `/compactar` persiste aunque quepa |
| T-B024-06  | actualizar | Orquestador: disparo automático por umbral y por desbordamiento (`esErrorDeVentana`), con `recompactar` y reintento único del turno                                                                                                                        | completada | `internal/session/orquestador.go`                                        | Un error 4xx de «exceeds the available context» recompacta y reintenta una vez; sin compactación automática no reintenta |
| T-B024-07  | actualizar | Cliente OpenAI: `normalizarMensajes` funde los `system` iniciales y degrada a `user` los `system` intermedios, para que el resumen inyectado a mitad de turno no rompa plantillas estrictas (Qwen3.5)                                                     | completada | `internal/llm/openai/cliente.go`                                         | Un bloque inicial de `system` se funde en uno; un `system` posterior viaja como `user`                                |
| T-B024-08  | actualizar | TUI: `Compactar` en el puerto, caso `/compactar` en `slash.go`, entrada en la paleta de `comandos.go`                                                                                                                                                     | completada | `internal/tui/wire.go`, `internal/tui/slash.go`, `internal/tui/comandos.go` | `/compactar` compacta la sesión activa y avisa; sin sesión solo avisa                                                |
| T-B024-09  | crear      | Tests: formatos de `compact`, ida y vuelta del resumen en `store`, compactación incremental y manual en `session`, `/compactar` en `tui`                                                                                                                    | completada | `internal/compact/compactacion_test.go`, `internal/store/resumen_test.go`, `internal/session/historial_test.go`, `internal/tui/slash_test.go` | `go test ./...` en verde salvo el fallo pre-existente y ajeno del logotipo (`logo_test.go`)                       |

**Fuente de verdad:** [[specs/SPEC-HISTORIAL-CONVERSACION]] (aprobada) y
[[backend/DECISIONS]]. No se implementa nada que no esté documentado ahí.

**Dependencias:** `01`→`05`; `02`→`03`; `03`,`04`,`07`→`05`; `05`→`06`,`08`; `09` cierra.
**Fuera de alcance:** medir tokens con un tokenizador real (se sigue estimando por
longitud); exponer el bloque `compactacion` en la interfaz (se edita a mano);
implementar la jerarquía `todos.parent_position` de la tarea T-B022, que solo se
renumeró de `008` a `009`.
