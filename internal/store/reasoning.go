package store

// reasoning.go — repositorio de la tabla reasoning (T-B002-06).
//
// El razonamiento vive en tabla propia porque llega en streaming y el mensaje
// es inmutable una vez completo (DECISIONS.md). Las operaciones concretas —
// UpsertRazonamiento, RazonamientoDe y el límite de persistencia de 200 ms—
// están en messages.go junto al resto del ciclo del turno, porque comparten el
// ejecutor de transacciones; este archivo documenta la frontera del repositorio:
//
//   - Un mensaje tiene como máximo un razonamiento: lo garantiza el índice
//     UNIQUE idx_reasoning_message (CONSTRAINTS.md §1, INDEXES.md §4).
//   - Solo los mensajes con role = agent tienen razonamiento
//     (ENUMS.md §2, BUSINESS_RULES.md); si el modelo no devolvió razonamiento,
//     simplemente no hay fila.
