# MAIN-TASKS — fixture con ciclo de dependencias

Dos elementos que se esperan entre sí: no hay orden de ejecución posible.

| ID | Acción | Tarea | Dep | Estado | Detalle |
|----|--------|-------|-----|--------|---------|
| T-B001 | crear | Primera | T-B002 | pendiente | |
| T-B002 | crear | Segunda | T-B001 | pendiente | |
