# MAIN-TASKS — fixture de queue

Todo derivado de estos archivos, sin estado propio. T-B002 y T-B003 pueden
arrancar; T-B004 espera a T-B002 y T-B005 a T-B004.

| ID | Acción | Tarea | Dep | Estado | Detalle |
|----|--------|-------|-----|--------|---------|
| T-B001 | crear | Primera | — | completada | |
| T-B002 | crear | Segunda | T-B001 | pendiente | |
| T-B003 | crear | Tercera | — | pendiente | |
| T-B004 | crear | Cuarta | T-B002 | pendiente | |
| T-B005 | crear | Quinta | T-B004 | pendiente | |
