> T-F021 — preferencias de usuario en la TUI: agente recordado, persistencia y lectura en el arranque.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[frontend/FRONTEND]] — §2 `config.go`, §3/§4: la preferencia vive en `~/.config/localcli/config.json`.
- [[frontend/01-domain/DOMAIN]] — §2: el último modelo y el último agente se recuerdan.
- [[specs/SPEC-OLLAMA-PERFIL]] — el perfil (modelo y agente) se recuerda y se aplica.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-F021-01 | actualizar | Preferencias (`ultimo_modelo`, `ultimo_agente`): carga y guardado tolerantes en `~/.config/localcli/config.json` | completada | `internal/tui/config.go` | Test `TestLasPreferenciasSeRecuerdanYSeToleran` en verde |
| T-F021-02 | actualizar | La vista arranca en el agente recordado (vía `Puerto.AgenteRecordado`), con `plan` por defecto | completada | `internal/tui/app.go`, `internal/tui/wire.go` | Test: `AgenteRecordado` vacío cae en `plan` |
| T-F021-03 | actualizar | Persistir al elegir modelo y al cambiar de agente | completada | `internal/tui/app.go` | Test: aplicar modelo / `Tab` dejan el `config.json` escrito (HOME temporal) |

Dependencias: T-F021-02..03 dependen de T-F021-01; el modelo recordado del arranque lo consume T-B019.
