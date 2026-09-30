// sesion.go — el id de sesión viaja en el contexto del turno.
//
// La cadena `session → flow → agent → tools` no lleva la sesión como parámetro
// en cada interfaz: va en el contexto, que `session` estampa UNA vez por turno
// (`ConSesion`). Así cada evento que produce el motor se atribuye a la sesión
// que lo produjo, sin una variable global mutable compartida por todas las
// sesiones. `tools` no importa nada interno: es la hoja natural donde vive,
// junto a los `SesionID` que ya llevan `Peticion` y `Contexto`.
package tools

import "context"

// claveSesion es el tipo de la clave del contexto. Un tipo propio evita
// colisiones con otras claves del mismo contexto.
type claveSesion struct{}

// ConSesion devuelve un contexto que lleva el id de sesión del turno. Lo llama
// `session` antes de entregar el turno al motor.
func ConSesion(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, claveSesion{}, id)
}

// SesionDe devuelve el id de sesión del turno, o "" si no viaja en el contexto.
func SesionDe(ctx context.Context) string {
	id, _ := ctx.Value(claveSesion{}).(string)
	return id
}
