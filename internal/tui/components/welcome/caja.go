package welcome

import (
	"strings"

	"localcli/internal/tui/render"
)

const (
	placeholder       = "Escribe para iniciar la conversacion"
	cursor            = "▌"
	anchoMaximoCaja   = 72
	anchoEntradaLlana = 60
	// anchoMinimoLogotipoCompleto es el ancho de terminal a partir del cual cabe
	// el logotipo completo; por debajo se pinta el compacto.
	anchoMinimoLogotipoCompleto = 70
)

// caja compone la caja de entrada: fondo propio, aire a los lados y bandas arriba
// y abajo. Recibe sus líneas interiores ya compuestas.
func caja(m Model, ancho int, lineas []string) string {
	if ancho < 3 {
		return strings.Join(lineas, "\n")
	}
	banda := render.PintarFondo("", ancho, render.FondoCaja)
	var b strings.Builder
	b.WriteString(banda)
	for _, linea := range lineas {
		b.WriteString("\n" + render.PintarFondo(" "+render.RecortarColumnas(linea, ancho-2), ancho, render.FondoCaja))
	}
	b.WriteString("\n" + banda)
	return b.String()
}

// lineasCaja es el texto con su cursor, una fila de aire y la línea de estado, ya
// envuelta al interior de la caja. Devuelve también dónde quedó el interruptor de
// razonamiento —fila dentro de las líneas y columna—, o -1 si no se compuso.
func lineasCaja(m Model, interior int, datos Datos) (lineas []string, filaPensar, columnaPensar int) {
	lineas = append(textoConCursor(m, interior), "")
	filasEstado, posiciones := estado(datos, interior)
	filaPensar = -1
	if punto, ok := posiciones["pensar"]; ok {
		filaPensar = len(lineas) + punto.Fila
		columnaPensar = punto.Columna
	}
	return append(lineas, filasEstado...), filaPensar, columnaPensar
}

// estado reparte el agente, el modelo y el interruptor de razonamiento en filas
// que quepan en `ancho`: en una caja estrecha el interruptor no se pierde, cae a
// la fila siguiente.
func estado(datos Datos, ancho int) ([]string, map[string]render.Punto) {
	sep := render.EstiloSutil.Render(" · ")
	return render.EnvolverChips([]render.Chip{
		{Texto: prefijoEstado(datos)},
		{Texto: chipPensar(datos.Pensar), Clave: "pensar"},
	}, sep, ancho)
}

// textoConCursor envuelve lo escrito y pinta el cursor en su sitio.
func textoConCursor(m Model, ancho int) []string {
	if m.Peticion() == "" {
		return []string{render.EstiloSutil.Render(`"` + placeholder + `"`)}
	}
	envueltas, fila, columna := render.EnvolverConCursor(m.Texto, m.Pos, ancho)
	lineas := make([]string, 0, len(envueltas))
	for i, linea := range envueltas {
		if i != fila {
			lineas = append(lineas, m.adjuntos.Estilizar(linea))
			continue
		}
		runas := []rune(linea)
		lineas = append(lineas, m.adjuntos.Estilizar(string(runas[:columna]))+
			cursor+m.adjuntos.Estilizar(string(runas[columna:])))
	}
	return lineas
}

// prefijoEstado pinta el agente activo y el modelo en uso con el motor que lo sirve.
func prefijoEstado(datos Datos) string {
	return render.EstiloAgente.Render("["+datos.Agente+"]") +
		render.EstiloBlanco.Render(" • "+rotuloModelo(datos))
}

// chipPensar es el interruptor de razonamiento: `pensar [x]` encendido.
func chipPensar(activo bool) string {
	marca := "[ ]"
	if activo {
		marca = "[x]"
	}
	return render.EstiloBlanco.Render("pensar " + marca)
}

func rotuloModelo(datos Datos) string {
	switch {
	case datos.Modelo == "":
		return "—"
	case datos.Motor == "":
		return datos.Modelo
	default:
		return datos.Modelo + " (" + datos.Motor + ")"
	}
}
