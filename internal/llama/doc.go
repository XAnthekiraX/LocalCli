// Package llama gestiona el ciclo de vida de `llama-server`: localizarlo,
// arrancarlo con el modelo seleccionado, comprobar que responde, reiniciarlo al
// cambiar de modelo y detenerlo al salir. No habla con el modelo: eso es del
// cliente de inferencia.
package llama
