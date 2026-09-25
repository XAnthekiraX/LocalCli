# 007-task-tools.md

> T-B007 — tools: registro de las 13 herramientas, comprobación de permiso y enrutado (+ DTOs).
> Acción de esta descomposición: `crear`.

## Referencias

- `ai/docs/backend/02-interfaces/TOOLS.md` [5-113] (catálogo, reparto plan/build, controles, enrutado)
- `ai/docs/backend/02-interfaces/dto/TOOLS-DTO.md` [5-96] (request/response schemas por herramienta)
- `ai/docs/specs/SPEC-TOOLS.md` (especificación funcional de las 13)
- `ai/docs/backend/03-security/SECURITY.md` [11-24] (autorización por agente)
- `ai/docs/backend/DECISIONS.md` [18] (permiso se comprueba en tools, se aplica en fileops/exec)

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-B007-01 | crear | Definir el catálogo cerrado de las 13 herramientas con su nombre y categoría | pendiente | `internal/tools/catalog.go` | Test: el registro contiene exactamente 13 entradas |
| T-B007-02 | crear | Crear los structs DTO de request por categoría (lectura, escritura, terminal, internet) | pendiente | `internal/tools/dto_request.go` | Test: fixture JSON de cada herramienta del DTO decodea |
| T-B007-03 | crear | Crear los structs DTO de response por categoría (incluye campo `truncado`) | pendiente | `internal/tools/dto_response.go` | Test: encode coincide con Response Schemas |
| T-B007-04 | crear | Implementar el registro que asocia cada herramienta a su handler (fileops/exec/internet) | pendiente | `internal/tools/registry.go` | Test: lookup devuelve handler por nombre; desconocida error |
| T-B007-05 | crear | Comprobar permiso: rechazar herramienta fuera del catálogo del agente que la pide | pendiente | `internal/tools/permission.go` | Test: plan pidiendo write → denegación documentada |
| T-B007-06 | crear | Enrutar herramientas de archivo hacia fileops y de terminal hacia exec | pendiente | `internal/tools/route.go` | Test con stubs: cada categoría llega al destino correcto |
| T-B007-07 | crear | Enrutar las de internet respetando LOCALCLI_ALLOW_INTERNET (denegar si apagado) | pendiente | `internal/tools/route.go` | Test: sin variable, web_search devuelve denegación |
| T-B007-08 | crear | Validar el payload de cada request antes de enrutar (campos obligatorios) | pendiente | `internal/tools/validate.go` | Test: payload incompleto → error de validación, no llega al handler |
| T-B007-09 | crear | Escribir tests del módulo con handlers stub | pendiente | `internal/tools/*_test.go`, `internal/tools/testdata/` | `go test ./internal/tools/...` pasa |

**Dependencias:** 01→{04,05}; {02,03}→08; 04→{06,07}; 08→09; 05→09. Requiere T-B001 completada (T-B006-03 consume T-B007-01).
