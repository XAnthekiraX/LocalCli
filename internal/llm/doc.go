// Package llm es la frontera neutra con el modelo: los tipos que comparten los
// motores y las reglas que no dependen de ningún runtime concreto (el estado de
// una capacidad y la ventana de contexto). No habla HTTP ni conoce ningún
// endpoint: el transporte vive en `llm/openai` y las extensiones nativas en su
// adaptador.
package llm
