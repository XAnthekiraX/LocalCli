package tui

import tea "github.com/charmbracelet/bubbletea"

// copiarBotonEn resuelve un clic sobre el botón de copiar de la cabecera de un
// bloque de código del chat: si el punto cae sobre uno, copia su cuerpo al
// portapapeles y muestra el aviso [Copiado], sin iniciar una selección. Devuelve
// false si el clic no cae sobre ningún botón, y entonces el clic sigue su curso
// normal. El patrón es el de los botones de aprobación (SPEC-VISTA-CODIGO §Reglas
// de negocio).
func (a *App) copiarBotonEn(p punto) (tea.Cmd, bool) {
	zona := a.zonaChat
	if a.vista != VistaPrincipal || zona.Alto <= 0 || zona.Ancho <= 0 || !zona.contiene(p) {
		return nil, false
	}
	boton, ok := a.chat.BotonEnVentana(p.Fila-zona.Fila, p.Columna-zona.Columna)
	if !ok {
		return nil, false
	}
	return tea.Batch(copiarCmd(boton.Codigo), a.avisarCopiado()), true
}
