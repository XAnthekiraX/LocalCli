// Package panel es el sidebar de datos de la sesión activa: el título de la
// conversación, CONTEXTO (tokens y porcentaje), TODO, LISTA DE TAREAS, ESTADO
// (capa y cola, aprobaciones y agente) y un pie de tres filas con el estado de
// git, la ruta del proyecto y la firma del harness. Arranca visible y se pliega
// con `Ctrl+D`.
//
// Es de lectura y no decide nada: recibe los datos ya resueltos y los pinta. No
// consulta el motor, no abre la base y no calcula estados; una estimación se
// marca como estimación y lo que no se sabe se dice, nunca se disfraza.
package panel
