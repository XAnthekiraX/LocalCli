# MAIN-TASKS — fixture de cola detenida

T-B002 está declarado bloqueado en el TODO (con su motivo) y T-B001 espera a
T-B002: nada puede arrancar, así que la cola se detiene y avisa.

| ID | Acción | Tarea | Dep | Estado | Bloqueada por | Detalle |
|----|--------|-------|-----|--------|---------------|---------|
| T-B001 | crear | Primera | T-B002 | pendiente | | |
| T-B002 | crear | Segunda | — | bloqueada | T-B001 | |
