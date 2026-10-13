package chat

import (
	"strings"
	"time"
)

// AnotarInvocacion abre la línea de una herramienta: su verbo de pantalla y su
// objetivo —«LEER [ruta]»—. No cierra el turno: la línea se completa con
// CerrarHerramienta cuando llega el resultado, y mientras corre el indicador en
// vivo dice «Usando herramienta: X». El nombre del agente que la pidió no entra
// en la línea (INTERFACES §1.1).
func (m *Model) AnotarInvocacion(nombre, verbo, tema string) {
	if verbo == "" {
		return
	}
	m.herramienta = nombre
	m.AñadirLinea(Linea{Tipo: LineaHerramienta, Texto: LineaHerramientaAbierta(verbo, tema)})
	m.pendientes = append(m.pendientes, len(m.piezas)-1)
}

// CerrarHerramienta completa la línea abierta con la marca, la medida del
// resultado, el aviso de recorte, el motivo del fallo y el tiempo que tardó la
// ejecución: una sola línea cerrada, sin volcar la salida cruda. Sin invocación
// pendiente —un resultado suelto— deja una línea nueva con el nombre de la
// herramienta.
func (m *Model) CerrarHerramienta(nombre string, ok, truncado bool, medida, motivo string, duracion time.Duration) {
	m.herramienta = ""
	idx := m.siguientePendiente()
	base := nombre
	if l := m.lineaEn(idx); l != nil {
		base = l.Texto
	}
	texto := LineaHerramientaCerrada(base, ok, truncado, medida, motivo, duracion)
	if l := m.lineaEn(idx); l != nil {
		l.Texto = texto
		return
	}
	m.AñadirLinea(Linea{Tipo: LineaHerramienta, Texto: texto})
}

// AnotarEtapa deja la línea `[Sub Proceso] <nombre>` de la etapa que empieza. El
// texto de un paso intermedio no se pinta: la etapa solo se anuncia por su
// nombre.
func (m *Model) AnotarEtapa(nombre string) {
	if strings.TrimSpace(nombre) == "" {
		return
	}
	m.AñadirLinea(Linea{Tipo: LineaSubProceso, Texto: LineaProceso(nombre)})
}

// AnotarAviso deja un aviso suelto en el hilo: una notificación, el desenlace de
// una etapa que falló o un cambio aplicado. Va como línea del sistema, sin bloque
// ni color de hablante.
func (m *Model) AnotarAviso(texto string) {
	if strings.TrimSpace(texto) == "" {
		return
	}
	m.AñadirLinea(Linea{Tipo: LineaAviso, Texto: texto})
}

// lineaEn devuelve la línea del sistema que hay en ese índice (nil si no hay).
func (m *Model) lineaEn(idx int) *Linea {
	if idx < 0 || idx >= len(m.piezas) {
		return nil
	}
	linea, _ := m.piezas[idx].(*Linea)
	return linea
}

// siguientePendiente saca la línea de herramienta abierta más antigua y la deja
// fuera de la cola. Devuelve -1 si no hay ninguna.
func (m *Model) siguientePendiente() int {
	if len(m.pendientes) == 0 {
		return -1
	}
	idx := m.pendientes[0]
	m.pendientes = m.pendientes[1:]
	return idx
}
