package modals

import (
	"fmt"
	"strings"

	"localcli/internal/tui/render"
)

// EstadoDescarga es el punto en que va el modal de descarga.
type EstadoDescarga int

const (
	// DescargaPidiendo: la URL se está escribiendo; todavía no se lanzó nada.
	DescargaPidiendo EstadoDescarga = iota
	// DescargaBajando: la descarga está en curso.
	DescargaBajando
	// DescargaHecha: la descarga terminó y el modelo quedó registrado.
	DescargaHecha
	// DescargaFallo: la descarga no pudo completarse.
	DescargaFallo
)

// Pies del modal de descarga: lo que se puede hacer en cada estado.
const (
	PieDescarga        = "(enter descargar · Tab cambiar campo · esc cerrar)"
	PieDescargaEnCurso = "(esc cancelar)"
	PieDescargaFin     = "(esc cerrar)"
)

// Descarga es el modal de descarga de modelos (`Ctrl+X d`): dos campos de URL
// (GGUF obligatorio, mmproj opcional), el nombre derivado y una barra de progreso.
// Es solo estado y pintado; el `app` traduce sus teclas y entrega las URLs al
// puerto, y los eventos de descarga van rellenando su progreso.
type Descarga struct {
	// Abierto dice si el modal se pinta.
	Abierto bool
	// URL es lo escrito: la dirección del GGUF a descargar.
	URL string
	// MMProjURL es lo escrito: la dirección del proyector multimodal (opcional).
	MMProjURL string
	// Pos es la posición del cursor dentro del campo activo, en runas.
	Pos int
	// CampoActivo indica qué campo está editando el usuario: 0 = URL GGUF, 1 = mmproj.
	CampoActivo int
	// Estado es el punto en que va la descarga.
	Estado EstadoDescarga
	// Nombre es el nombre derivado de la URL del GGUF (el archivo sin `.gguf`).
	Nombre string
	// Hecho y Total son los bytes descargados y el total conocido (0 si no se
	// conoce).
	Hecho int64
	Total int64
	// Aviso es el error de la última descarga fallida o el recordatorio de que
	// falta la URL.
	Aviso string
}

// Abrir deja el modal en blanco, listo para teclear una URL. Cada apertura
// empieza de cero: no se arrastra lo de la anterior.
func (d *Descarga) Abrir() {
	d.URL = ""
	d.MMProjURL = ""
	d.Pos = 0
	d.CampoActivo = 0
	d.Estado = DescargaPidiendo
	d.Nombre = ""
	d.Hecho = 0
	d.Total = 0
	d.Aviso = ""
	d.Abierto = true
}

// Cerrar lo esconde.
func (d *Descarga) Cerrar() { d.Abierto = false }

// Bajando dice si hay una descarga en curso.
func (d *Descarga) Bajando() bool { return d.Estado == DescargaBajando }

// FijarNombre deja el nombre derivado de la URL mientras se escribe.
func (d *Descarga) FijarNombre(nombre string) { d.Nombre = nombre }

// Escribir inserta texto en la posición del cursor del campo activo.
// Solo se escribe mientras se pide la URL: con una descarga en marcha el campo queda fijo.
func (d *Descarga) Escribir(s string) {
	if d.Estado != DescargaPidiendo || s == "" {
		return
	}
	if d.CampoActivo == 0 {
		runas := []rune(d.URL)
		d.Pos = acotar(d.Pos, 0, len(runas))
		d.URL = string(runas[:d.Pos]) + s + string(runas[d.Pos:])
		d.Pos += len([]rune(s))
	} else {
		runas := []rune(d.MMProjURL)
		d.Pos = acotar(d.Pos, 0, len(runas))
		d.MMProjURL = string(runas[:d.Pos]) + s + string(runas[d.Pos:])
		d.Pos += len([]rune(s))
	}
}

// Borrar elimina el carácter anterior al cursor en el campo activo.
func (d *Descarga) Borrar() {
	if d.Estado != DescargaPidiendo || d.Pos <= 0 {
		return
	}
	if d.CampoActivo == 0 {
		runas := []rune(d.URL)
		d.URL = string(runas[:d.Pos-1]) + string(runas[d.Pos:])
		d.Pos--
	} else {
		runas := []rune(d.MMProjURL)
		d.MMProjURL = string(runas[:d.Pos-1]) + string(runas[d.Pos:])
		d.Pos--
	}
}

// BorrarAdelante elimina el carácter bajo el cursor en el campo activo.
func (d *Descarga) BorrarAdelante() {
	if d.Estado != DescargaPidiendo {
		return
	}
	if d.CampoActivo == 0 {
		runas := []rune(d.URL)
		if d.Pos >= len(runas) {
			return
		}
		d.URL = string(runas[:d.Pos]) + string(runas[d.Pos+1:])
	} else {
		runas := []rune(d.MMProjURL)
		if d.Pos >= len(runas) {
			return
		}
		d.MMProjURL = string(runas[:d.Pos]) + string(runas[d.Pos+1:])
	}
}

// Izquierda y Derecha mueven el cursor por el campo activo.
func (d *Descarga) Izquierda() {
	if d.Pos > 0 {
		d.Pos--
	}
}

func (d *Descarga) Derecha() {
	if d.CampoActivo == 0 {
		if d.Pos < len([]rune(d.URL)) {
			d.Pos++
		}
	} else {
		if d.Pos < len([]rune(d.MMProjURL)) {
			d.Pos++
		}
	}
}

// Inicio y Fin llevan el cursor a los extremos del campo activo.
func (d *Descarga) Inicio() { d.Pos = 0 }
func (d *Descarga) Fin() {
	if d.CampoActivo == 0 {
		d.Pos = len([]rune(d.URL))
	} else {
		d.Pos = len([]rune(d.MMProjURL))
	}
}

// CambiarCampo alterna entre el campo GGUF (0) y mmproj (1).
func (d *Descarga) CambiarCampo() {
	if d.Estado != DescargaPidiendo {
		return
	}
	d.CampoActivo = 1 - d.CampoActivo
	d.Pos = 0
}

// esGGUFURL verifica si la URL apunta a un archivo .gguf (ignora query params).
func esGGUFURL(raw string) bool {
	u := strings.TrimSpace(raw)
	if u == "" {
		return false
	}
	// Quitar query string y fragmento
	if i := strings.IndexByte(u, '?'); i >= 0 {
		u = u[:i]
	}
	if i := strings.IndexByte(u, '#'); i >= 0 {
		u = u[:i]
	}
	return strings.HasSuffix(strings.ToLower(u), ".gguf")
}

// Empezar pasa el modal a «bajando» si hay una URL de GGUF que bajar.
// El mmproj es opcional. Sin URL de GGUF deja el recordatorio y devuelve false.
func (d *Descarga) Empezar() bool {
	if !esGGUFURL(d.URL) {
		d.Aviso = "escribe la URL de un archivo .gguf"
		return false
	}
	if strings.TrimSpace(d.MMProjURL) != "" && !esGGUFURL(d.MMProjURL) {
		d.Aviso = "el mmproj debe ser un archivo .gguf"
		return false
	}
	d.Estado = DescargaBajando
	d.Aviso = ""
	d.Hecho, d.Total = 0, 0
	return true
}

// Progreso actualiza la barra mientras la descarga avanza. Un evento fuera de la
// descarga en curso se ignora.
func (d *Descarga) Progreso(hecho, total int64) {
	if d.Estado != DescargaBajando {
		return
	}
	d.Hecho, d.Total = hecho, total
}

// Terminada marca la descarga como completada. El nombre que trae el evento
// manda sobre el derivado: es el que quedó registrado.
func (d *Descarga) Terminada(nombre string) {
	if d.Estado != DescargaBajando {
		return
	}
	if nombre != "" {
		d.Nombre = nombre
	}
	d.Estado = DescargaHecha
}

// Fallida marca la descarga como fallida con su motivo.
func (d *Descarga) Fallida(motivo string) {
	if d.Estado != DescargaBajando {
		return
	}
	d.Estado = DescargaFallo
	d.Aviso = motivo
}

// pie devuelve la línea de teclas del estado actual.
func (d *Descarga) pie() string {
	switch d.Estado {
	case DescargaBajando:
		return PieDescargaEnCurso
	case DescargaHecha, DescargaFallo:
		return PieDescargaFin
	default:
		return PieDescarga
	}
}

// Render pinta el modal centrado: cabecera, dos campos de URL (GGUF + mmproj opcional)
// con su cursor, el nombre derivado y, según el estado, la barra de progreso o el desenlace.
func (d *Descarga) Render(ancho, alto int) string {
	if !d.Abierto {
		return ""
	}
	contenido := min(max(ancho-8, 24), 72)
	var b strings.Builder
	b.WriteString(render.EstiloTitulo.Render("DESCARGAR MODELOS") + "\n\n")

	switch d.Estado {
	case DescargaPidiendo:
		// Campo GGUF (obligatorio)
		estiloGGUF := render.EstiloSutil
		prefijoGGUF := "  "
		if d.CampoActivo == 0 {
			estiloGGUF = render.EstiloBlanco
			prefijoGGUF = "› "
		}
		b.WriteString(estiloGGUF.Render("URL del modelo (.gguf):") + "\n")
		for _, linea := range urlConCursor(d.URL, d.Pos, contenido) {
			b.WriteString(render.EstiloUsuario.Render(prefijoGGUF) + linea + "\n")
		}
		b.WriteString("\n")

		// Campo mmproj (opcional)
		estiloMM := render.EstiloSutil
		prefijoMM := "  "
		if d.CampoActivo == 1 {
			estiloMM = render.EstiloBlanco
			prefijoMM = "› "
		}
		b.WriteString(estiloMM.Render("URL del mmproj (.gguf, opcional):") + "\n")
		for _, linea := range urlConCursor(d.MMProjURL, d.Pos, contenido) {
			b.WriteString(render.EstiloUsuario.Render(prefijoMM) + linea + "\n")
		}
		b.WriteString("\n")

		nombre := d.Nombre
		if nombre == "" {
			nombre = "—"
		}
		b.WriteString(render.EstiloSutil.Render("nombre del modelo: ") + render.EstiloBlanco.Render(nombre) + "\n")
		b.WriteString(render.EstiloSutil.Render("(Tab para cambiar campo)") + "\n")
		if d.Aviso != "" {
			b.WriteString("\n" + render.EstiloAviso.Render(d.Aviso) + "\n")
		}
	case DescargaBajando:
		b.WriteString(render.EstiloBlanco.Render(d.Nombre) + "\n\n")
		b.WriteString(barraDeProgreso(d.Hecho, d.Total) + "\n")
	case DescargaHecha:
		b.WriteString(render.EstiloAgente.Render("✓ ") + render.EstiloBlanco.Render(d.Nombre) + "\n")
		b.WriteString(render.EstiloSutil.Render("modelo registrado; ya aparece en el modal de modelos") + "\n")
	case DescargaFallo:
		b.WriteString(render.EstiloAviso.Render("no se pudo descargar") + "\n")
		if d.Aviso != "" {
			b.WriteString(render.EstiloBlanco.Render(d.Aviso) + "\n")
		}
	}
	b.WriteString(render.EstiloSutil.Render(d.pie()))
	return centrar(ajustar(b.String(), alto), ancho, alto)
}

// urlConCursor envuelve la URL al ancho dado y marca con el cursor la posición
// donde se escribe.
func urlConCursor(url string, pos, ancho int) []string {
	if ancho <= 0 {
		ancho = 1
	}
	lineas, fila, col := render.EnvolverConCursor(url, pos, ancho)
	if len(lineas) == 0 {
		lineas = []string{""}
	}
	if fila >= 0 && fila < len(lineas) {
		lineas[fila] = insertarCursor(lineas[fila], col)
		return lineas
	}
	lineas[len(lineas)-1] += "▏"
	return lineas
}

// insertarCursor mete el glifo del cursor en la columna dada.
func insertarCursor(linea string, col int) string {
	runas := []rune(linea)
	col = acotar(col, 0, len(runas))
	return string(runas[:col]) + "▏" + string(runas[col:])
}

// barraDeProgreso compone `[████····] 47% · 3.2/6.0 GB`. Sin total conocido deja
// la barra vacía y solo muestra lo descargado.
func barraDeProgreso(hecho, total int64) string {
	const ancho = 28
	llenos, etiqueta := 0, ""
	if total > 0 {
		pct := int(hecho * 100 / total)
		if pct > 100 {
			pct = 100
		}
		llenos = pct * ancho / 100
		etiqueta = fmt.Sprintf(" %3d%% · %s/%s", pct, humanoBytes(hecho), humanoBytes(total))
	} else {
		etiqueta = " " + humanoBytes(hecho)
	}
	barra := "[" + strings.Repeat("█", llenos) + strings.Repeat("·", ancho-llenos) + "]"
	return render.EstiloUsuario.Render(barra) + render.EstiloSutil.Render(etiqueta)
}

// humanoBytes pasa un tamaño a una etiqueta legible (B, KB, MB, GB, …).
func humanoBytes(n int64) string {
	const unidad = 1024
	if n < unidad {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unidad), 0
	for x := n / unidad; x >= unidad; x /= unidad {
		div *= unidad
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}

// acotar mete un valor en un rango cerrado.
func acotar(valor, minimo, maximo int) int {
	if valor < minimo {
		return minimo
	}
	if valor > maximo {
		return maximo
	}
	return valor
}
