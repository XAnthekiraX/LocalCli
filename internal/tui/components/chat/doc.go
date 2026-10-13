// Package chat guarda el hilo de la sesión activa: los segmentos de texto y las
// líneas del sistema, en el orden en que ocurrieron. El hilo de un turno no se
// funde: el texto anterior a una herramienta y el posterior son segmentos
// distintos y cada uno conserva su razonamiento.
//
// Es solo guardar y pintar: no gestiona sesiones, no habla con el motor y no
// decide nada. Lo que se ve aquí llegó por eventos o lo envió el usuario.
package chat
