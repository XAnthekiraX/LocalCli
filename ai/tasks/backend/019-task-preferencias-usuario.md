> T-B019 — preferencias del usuario: el arranque reutiliza el último modelo y expone el último agente.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[backend/04-infrastructure/CONFIGURATION]] — §7 Preferencias del usuario (`~/.config/localcli/config.json`).
- [[specs/SPEC-OLLAMA-PERFIL]] — el modelo lo elige el usuario y su elección se recuerda.
- [[backend/BACKEND]] — `tui` es presentación pura; el motor conoce el archivo de preferencias.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-B019-01 | actualizar | `elegirModelo` acepta el modelo preferido y lo usa si sigue instalado | completada | `arranque.go`, `arranque_test.go` | Test `TestElegirModeloPrefiereElUltimoUsado` en verde |
| T-B019-02 | actualizar | El arranque lee las preferencias y expone el último agente por el puerto (`AgenteRecordado`) | completada | `arranque.go`, `internal/tui/wire.go` | `go build ./...` sin errores |
| T-B019-03 | actualizar | Test de que un modelo preferido desinstalado no rompe la autodetección | completada | `arranque_test.go` | Test `TestElegirModeloIgnoraUnPreferidoDesinstalado` en verde |

Dependencias: T-B019-02 depende de T-F021-01 (el helper de preferencias); T-B019-03 depende de T-B019-01.
