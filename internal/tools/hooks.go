package tools

// Hooks son los tres puntos de enganche de la capa universal. Opcionales: sin
// ellos la ejecución funciona igual.
type Hooks struct {
	AntesDeEjecutar   func(nombre string, args any, meta map[string]any)
	DespuesDeEjecutar func(nombre string, r Resultado, err error, meta map[string]any)
	// DefinirHerramienta ajusta nombre, descripción y esquema antes de que lleguen
	// al modelo.
	DefinirHerramienta func(nombre, descripcion string, e *Esquema) (string, *Esquema)
}
