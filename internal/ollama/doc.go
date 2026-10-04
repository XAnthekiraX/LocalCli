// Package ollama — adaptador de Ollama para la frontera `llm.Motor`.
//
// Implementa el contrato neutro sobre la API HTTP local de Ollama: streaming
// NDJSON por `/api/chat`, extracción del razonamiento, canal nativo de
// herramientas, perfil de hardware y capacidades del modelo (`/api/show`). No
// decide nada y no conoce a los demás módulos: traduce el contrato neutro al
// formato de Ollama.
//
// Fuente de verdad: ai/docs/backend/BACKEND.md §3 y
// ai/docs/backend/04-infrastructure/INTEGRATIONS.md §Ollama.
package ollama
