# Taste

- Prefiere comunicarse en español (tanto peticiones como contexto técnico). Confidence: 0.85
- Trabaja con un flujo spec-driven: invoca comandos personalizados (p. ej. `/resolver`) que exigen leer las specs afectadas en `ai/docs/specs` y pedir aprobación antes de ejecutar cambios. Confidence: 0.6
- Prefiere instrumentar/medir el comportamiento (p. ej. contadores de latencia) antes de optimizar, para decidir con datos. Confidence: 0.55
- Prefiere un único camino de código parametrizado por configuración frente a duplicar funciones por variante: p. ej. que `plan` y `build` sean el mismo componente `Agent` (con distinto system prompt, permisos y tools) sobre un mismo bucle, en lugar de dos programas con sus propios flujos. Confidence: 0.65
- Prefiere una sola fuente de verdad para evitar estados que puedan contradecirse (que la lista de herramientas se derive de los permisos, y no declarar dos listas que puedan discrepar). Confidence: 0.55
