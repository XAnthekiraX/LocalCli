package context

// assemble.go — T-B011-06: ensamblar el bloque de contexto final.
//
// Fuente de verdad: [[specs/SPEC-NODO-CONTEXTO]] ("Se entrega el contexto a la
// etapa siguiente") y VALIDATION.md §1 ("Un documento entra al contexto solo
// después de leerse"). El bloque lleva solo lo que aprobó el recorte, cada
// documento con su ruta para que quede trazable.

import "strings"

// Ensamblar construye el texto que recibe la etapa: una cabecera con el
// objetivo y un apartado por documento incluido.
func Ensamblar(objetivo, etapa string, incluidos []DocumentoSeleccionado) string {
	var b strings.Builder
	b.WriteString("# Contexto para: " + objetivo + "\n")
	if etapa != "" {
		b.WriteString("Etapa: " + etapa + "\n")
	}
	for _, d := range incluidos {
		b.WriteString("\n## " + d.Ruta + "\n\n")
		b.WriteString(strings.TrimRight(d.Contenido, "\n"))
		b.WriteString("\n")
	}
	return b.String()
}
