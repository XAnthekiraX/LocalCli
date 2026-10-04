// Package llm — la frontera neutra con el modelo de LocalCli.
//
// Es el contrato único con el modelo: los tipos que viajan (Mensaje,
// Herramienta, Evento, Modelo, Peticion), la interfaz Motor que cada
// adaptador implementa, el registro de motores declarados (`motores.json`,
// con un adaptador vivo por `id`), la cola de inferencia FIFO —una por
// motor— y la regla de la ventana de contexto. No conoce el formato de ningún
// runtime: no sabe qué es NDJSON, ni SSE, ni `num_ctx`, ni `n_ctx`.
//
// Fuente de verdad: ai/docs/backend/BACKEND.md §3, ai/docs/backend/DECISIONS.md
// («El harness habla con un solo contrato de modelo (`llm.Motor`) y cada
// runtime lo traduce en su adaptador») y ai/docs/specs/SPEC-MODELO-MOTOR.
//
// Quien llama al modelo —`agent`, `context`— importa este paquete y no los
// adaptadores: `ollama` y `llamacpp` son detalles de transporte que se eligen al
// arrancar y no se filtran hacia arriba. Es el único sitio que instancia un
// adaptador por tipo cerrado.
package llm
