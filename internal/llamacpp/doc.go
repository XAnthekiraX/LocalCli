// Package llamacpp — adaptador compatible con OpenAI para `llama-server` de
// llama.cpp, implementando la frontera `llm.Motor`.
//
// `llama-server` en modo router habla el protocolo OpenAI: streaming SSE por
// `/v1/chat/completions`, lista de modelos en `/models` (con `/v1/models` de
// reserva) y propiedades del servidor en `/props` (de donde se lee la ventana
// `n_ctx` y las capacidades). El razonamiento llega en `delta.reasoning_content`
// y las herramientas en `delta.tool_calls` acumuladas por índice.
//
// No muta el servidor: no descarga ni carga modelos (`POST /models`,
// `POST /models/load`) ni cambia su ventana (`POST /props`). Ver
// ai/docs/backend/03-security/SECURITY.md §3.2.
//
// Fuente de verdad: ai/docs/backend/04-infrastructure/INTEGRATIONS.md §llama.cpp
// y ai/docs/backend/BACKEND.md §3.
package llamacpp
