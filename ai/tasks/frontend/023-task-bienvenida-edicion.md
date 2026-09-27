> T-F023 — bienvenida: la línea de entrada se edita en cualquier punto.
> Acción de esta descomposición: `actualizar`.

## Referencias

- [[frontend/01-domain/DOMAIN]] — §3: la línea de entrada de la bienvenida se edita como la principal.
- [[specs/SPEC-INTERFAZ]] — §Pantalla de bienvenida: línea de entrada editable.

## Tareas pequeñas

| ID | Acción | Tarea | Estado | Archivos | Verificación |
|----|--------|-------|--------|----------|--------------|
| T-F023-01 | actualizar | `Bienvenida` gana posición de cursor y operaciones de edición (insertar, retroceso, suprimir, inicio/fin) | completada | `internal/tui/welcome.go` | Test `TestLaBienvenidaEditaEnCualquierPunto` en verde |
| T-F023-02 | actualizar | El cursor se pinta en su posición y el teclado de la bienvenida enruta flechas, `home`/`end` y `supr` | completada | `internal/tui/welcome.go` | Mismo test |
