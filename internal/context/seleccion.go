package context

import "context"

// elegir consulta al modelo qué documentos necesita. Si no decide —error, lista
// vacía o sin modelo—, se cae al objetivo declarado: el bloque va sin documentos.
func (n *Nodo) elegir(ctx context.Context, objetivo string, catalogo []string) []string {
	if n.Modelo == nil {
		return nil
	}
	elegidos, err := n.Modelo.Elegir(ctx, objetivo, catalogo)
	if err != nil || len(elegidos) == 0 {
		return nil
	}
	return elegidos
}

func (n *Nodo) avisarDocumentosSinDependencias(documentos []Documento) {
	if n.Aviso == nil {
		return
	}
	for _, d := range documentos {
		if len(d.Dependencias) == 0 {
			n.Aviso("documento sin dependencias declaradas: " + d.Ruta)
		}
	}
}
