// Package input es la caja de entrada de la vista principal: una línea de texto
// que crece en filas al desbordar el ancho, un pie con el agente activo, el
// modelo en uso y sus capacidades, y el conteo de tokens del turno justo debajo.
// Los archivos, carpetas y textos pegados o arrastrados se muestran como tokens
// ([nombre.ext], [CARPETA N elementos], [PEGADO N líneas]) y se expanden a su
// valor real al enviar.
//
// Es solo composición y edición: no valida reglas de negocio. Quien conoce la
// sesión activa confirma el envío (el `EnviadoMsg` que emite al pulsar `Enter`).
package input
