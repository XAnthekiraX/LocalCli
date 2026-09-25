> T-F001 — keys: mapa de teclas reasignable con valores por defecto, carga/guardado en ~/.config/localcli/keys.json y rechazo de duplicados.
> Acción de esta descomposición: `crear`.

## Referencias

- [[frontend/FRONTEND]] — `keys.go` en la estructura (sección 2) y configuración en `~/.config/localcli/keys.json` (sección 3).
- [[frontend/02-interfaces/INTERFACES]] — sección 4: atajos por defecto y reglas (duplicado rechazado al guardar, cambio sin reiniciar).
- [[specs/SPEC-INTERFAZ-ATAJOS]] — reglas de negocio de los atajos.
- [[frontend/05-quality/TESTING]] — unitario: detección de atajos duplicados.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-F001-01 | crear | Tipos del mapa: acción identificable, binding (tecla/modificador) y KeyMap con valores por defecto de los 7 atajos | completada | `internal/tui/keys.go` | `go build ./internal/tui/` sin errores |
| T-F001-02 | crear | Coincidencia de pulsaciones contra el mapa (`Matches`) para enrutar teclado según teclas reasignadas | completada | `internal/tui/keys.go` | Test `TestMatchesDetectaAtajo` en verde |
| T-F001-03 | crear | Detección de atajos duplicados (`FindDuplicate`) que rechaza una asignación doble al guardar | completada | `internal/tui/keys.go` | Test `TestFindDuplicateRechazaRepetido` en verde |
| T-F001-04 | crear | Carga y guardado JSON en `~/.config/localcli/keys.json` (XDG), creando el directorio y tolerando archivo ausente | completada | `internal/tui/keys.go` | Test `TestSaveLoadRoundTrip` con HOME temporal en verde |
| T-F001-05 | crear | Pruebas unitarias puras del paquete keys (defectos, duplicados, ida y vuelta) | completada | `internal/tui/keys_test.go` | `go test ./internal/tui/ -run Keys` en verde |

Dependencias: T-F001-02..04 dependen de T-F001-01; T-F001-05 depende de T-F001-01..04.
