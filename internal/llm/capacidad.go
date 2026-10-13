package llm

import "strings"

// Capacidad es el estado de una capacidad del modelo. Tiene tres valores porque
// «no saber» y «no tener» son estados distintos: la ausencia de dato nunca es
// permiso ni prohibición.
type Capacidad int

const (
	// Desconocida: el motor no expone la extensión que declara la capacidad. Es
	// el valor cero a propósito, para que lo no declarado no se lea como permiso.
	Desconocida Capacidad = iota
	// Soportada: el motor declara que el modelo puede.
	Soportada
	// NoSoportada: el motor declara que el modelo no puede.
	NoSoportada
)

// Capacidades reúne las tres capacidades que el harness normaliza.
type Capacidades struct {
	Herramientas Capacidad
	Vision       Capacidad
	Razonamiento Capacidad
}

// TDesconocidas es el juego de capacidades cuando el motor no declara nada.
func TDesconocidas() Capacidades {
	return Capacidades{Herramientas: Desconocida, Vision: Desconocida, Razonamiento: Desconocida}
}

// DeBooleano traduce un booleano presente en la extensión a su capacidad; solo
// se usa cuando el motor sí declara el dato.
func DeBooleano(v bool) Capacidad {
	if v {
		return Soportada
	}
	return NoSoportada
}

// DeLista traduce una lista declarada de capacidades a los tres estados. Una
// lista nula deja todo Desconocida (el motor no la expone); una lista vacía
// declara que no hay ninguna soportada.
func DeLista(declaradas []string) Capacidades {
	if declaradas == nil {
		return TDesconocidas()
	}
	return Capacidades{
		Herramientas: deLista(declaradas, "tools"),
		Vision:       deLista(declaradas, "vision"),
		Razonamiento: deLista(declaradas, "reasoning"),
	}
}

func deLista(declaradas []string, nombre string) Capacidad {
	for _, d := range declaradas {
		if strings.EqualFold(d, nombre) {
			return Soportada
		}
	}
	return NoSoportada
}
