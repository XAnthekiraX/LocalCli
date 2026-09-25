package task

// parse.go — T-B004-02: parsear tablas Markdown hacia Elementos.
//
// El TODO vive en tablas Markdown (MAIN-TASKS.md: columna "Dep" con IDs o
// "—"; NNN-task-*.md: columnas ID/Acción/Tarea/Estado/Archivos/Verificación).
// Este archivo convierte cada fila de datos en un Elemento con los 7 campos
// del frontmatter aprobado ([[backend/DECISIONS]] [28]):
//
//   - id           ← columna "ID"
//   - capa         ← la capa del archivo (se pasa por parámetro; el nombre
//                    del directorio ai/tasks/<capa>/ la determina)
//   - accion       ← columna "Acción"
//   - estado       ← columna "Estado"
//   - depende_de   ← columna "Dep"/"Dependencias": lista separada por ",",
//                    ";" o "→"; "—", "-", "ninguna" y vacío ⇒ sin dependencias
//   - bloqueada_por← columna homónima si existe (MAIN no la tiene: queda vacía)
//   - documentos   ← celdas wiki-link [[...]] de las columnas de contexto
//                    ("Referencia"/"Documentos") + enlaces a archivos del TODO
//                    ("Detalle"). En MAIN-TASKS.md, si la fila no declara
//                    documentos, se recoge el destino del enlace Detalle.
//
// Las filas de cabecera y separadoras (|---|) se ignoran. Un guion bajo "—"
// es el vacío documentado en MAIN-TASKS.md para "sin dependencias".

import (
	"fmt"
	"regexp"
	"strings"
)

// wikiLinkRE captura los destinos [[carpeta/ARCHIVO]] dentro de una celda.
var wikiLinkRE = regexp.MustCompile(`\[\[([^\]\[]+)\]\]`)

// enlaceMDRE captura el destino de un enlace Markdown [texto](ruta).
var enlaceMDRE = regexp.MustCompile(`\[[^\]\[]*\]\(([^)\s]+)\)`)

// ParseTabla parsea el contenido Markdown de un archivo del TODO devolviendo
// sus elementos en el orden de aparición. `capa` es la capa deducida de la
// ruta (ver locate.go). Devuelve error solo si no encuentra ninguna fila de
// datos con columna ID reconocible.
//
// Postura estricta: una vez fijada la cabecera, toda fila de datos de esa
// tabla debe producir un elemento; una fila malformada (id vacío, acción o
// estado fuera del catálogo) es un error del documento, nunca se descarta en
// silencio — así el parseo "no pierde filas" (verificación T-B004-02).
func ParseTabla(contenido []byte, capa Capa) ([]Elemento, error) {
	lineas := strings.Split(strings.ReplaceAll(string(contenido), "\r\n", "\n"), "\n")

	var cabecera []string // nombres de columna normalizados (minúsculas)
	var elems []Elemento
	for _, ln := range lineas {
		t := strings.TrimSpace(ln)
		if !strings.HasPrefix(t, "|") {
			cabecera = nil // salimos de la tabla al encontrar texto normal
			continue
		}
		celdas := dividirFila(t)
		if len(celdas) == 0 {
			continue
		}
		if esSeparadora(celdas) {
			continue
		}
		if cabecera == nil {
			if indiceColumna(normalizarCabecera(celdas), "id") < 0 {
				continue // no es una tabla de elementos
			}
			cabecera = normalizarCabecera(celdas)
			continue
		}
		// Una segunda tabla sin columna ID (p. ej. el listado de specs de un
		// SPEC) interrumpe la actual: evitamos leer sus filas con la cabecera
		// de la tabla de elementos anterior.
		if indiceColumna(normalizarCabecera(celdas), "id") < 0 && len(celdas) != len(cabecera) {
			cabecera = nil
			continue
		}
		e, err := filaAElemento(cabecera, celdas, capa)
		if err != nil {
			return nil, fmt.Errorf("task: fila %q: %w", celdas[0], err)
		}
		elems = append(elems, e)
	}
	if len(elems) == 0 {
		return nil, fmt.Errorf("task: la tabla no contiene filas de elementos")
	}
	return elems, nil
}

// dividirFila parte una fila "| a | b |" en sus celdas recortadas.
func dividirFila(fila string) []string {
	fila = strings.TrimSuffix(strings.TrimPrefix(fila, "|"), "|")
	parts := strings.Split(fila, "|")
	out := make([]string, len(parts))
	for i, p := range parts {
		out[i] = strings.TrimSpace(p)
	}
	return out
}

// esSeparadora detecta la línea |----|: todas las celdas hechas de -, : y espacios.
func esSeparadora(celdas []string) bool {
	if len(celdas) == 0 {
		return false
	}
	for _, c := range celdas {
		if c == "" {
			return false
		}
		for _, r := range c {
			if r != '-' && r != ':' && r != ' ' {
				return false
			}
		}
	}
	return true
}

// normalizarCabecera baja de nombre las cabeceras para comparar sin tildes.
func normalizarCabecera(celdas []string) []string {
	out := make([]string, len(celdas))
	for i, c := range celdas {
		out[i] = normalizarNombre(c)
	}
	return out
}

var sustituciones = strings.NewReplacer(
	"ó", "o", "Ó", "O", "á", "a", "é", "e", "í", "i", "ú", "u", "ñ", "n",
	"“", "\"", "”", "\"", "`", "", "'", "",
)

func normalizarNombre(s string) string {
	s = strings.Trim(s, "*_ ")
	s = strings.ReplaceAll(s, "`", "")
	return strings.ToLower(strings.TrimSpace(sustituciones.Replace(s)))
}

// indiceColumna devuelve el índice de la primera celda de cabecera cuyo nombre
// normalizado coincide con alguno de los alias dados, o -1.
func indiceColumna(cabecera []string, alias ...string) int {
	for i, h := range cabecera {
		for _, a := range alias {
			if h == a {
				return i
			}
		}
	}
	return -1
}

// filaAElemento construye el Elemento de una fila de datos. Devuelve error si
// la celda ID está vacía o su vocabulario (acción/estado) es inválido: una
// fila malformada del TODO es un error duro, no se descarta en silencio.
func filaAElemento(cabecera, celdas []string, capa Capa) (Elemento, error) {
	celda := func(alias ...string) string {
		i := indiceColumna(cabecera, alias...)
		if i < 0 || i >= len(celdas) {
			return ""
		}
		return celdas[i]
	}

	id := limpiarCelda(celda("id"))
	if id == "" {
		return Elemento{}, fmt.Errorf("celda ID vacía")
	}
	accion := Accion(limpiarCelda(celda("accion")))
	estado := Estado(limpiarCelda(celda("estado")))
	if !EsAccion(string(accion)) {
		return Elemento{}, fmt.Errorf("%s: acción inválida %q", id, accion)
	}
	if !EsEstado(string(estado)) {
		return Elemento{}, fmt.Errorf("%s: estado inválido %q", id, estado)
	}

	dependencias := dividirLista(limpiarCelda(celda("dep", "dependencias", "depre")))
	bloqueadaPor := dividirLista(limpiarCelda(celda("bloqueada_por", "bloqueada por")))

	documentos := wikiLinks(celda("documentos", "referencias", "referencia"))
	if detalle := enlaces(celda("detalle")); len(detalle) > 0 {
		documentos = append(documentos, detalle...)
	}

	e := NuevoElemento(id, capa, accion, estado, dependencias, bloqueadaPor, documentos)
	if err := e.validarVocabulario(); err != nil {
		return Elemento{}, err
	}
	return e, nil
}

// limpiarCelda quita marcas de código en línea (**x**, `x`) y espacios.
func limpiarCelda(s string) string {
	s = strings.ReplaceAll(s, "`", "")
	s = strings.ReplaceAll(s, "**", "")
	return strings.TrimSpace(s)
}

var vacios = map[string]bool{"": true, "—": true, "-": true, "--": true, "ninguna": true, "ningunas": true, "n/a": true}

// dividirLista parte una celda de dependencias por ",", ";" o "→" y descarta
// los marcadores de vacío documentados.
func dividirLista(s string) []string {
	if vacios[strings.ToLower(s)] {
		return nil
	}
	reemplazos := strings.NewReplacer(";", ",", "→", ",", " y ", ",", "&", ",")
	parts := strings.Split(reemplazos.Replace(s), ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = limpiarCelda(p)
		// "T-B002..T-B014" es un rango: se guarda tal cual como referencia.
		if p != "" && !vacios[strings.ToLower(p)] {
			out = append(out, p)
		}
	}
	return out
}

// wikiLinks extrae los destinos [[...]] de una celda, sin duplicados.
func wikiLinks(celda string) []string {
	matches := wikiLinkRE.FindAllStringSubmatch(celda, -1)
	var out []string
	visto := map[string]bool{}
	for _, m := range matches {
		d := strings.TrimSpace(m[1])
		if d != "" && !visto[d] {
			visto[d] = true
			out = append(out, d)
		}
	}
	return out
}

// archivoTareaRE captura referencias a archivos del TODO tipo `004-task-task.md`
// o `002..015-task-*.md` dentro de una celda Detalle.
var archivoTareaRE = regexp.MustCompile(`[0-9]{3}(?:\.\.[0-9]{3})?-task-[a-z-]+\.(?:md|\*)`)

// enlaces extrae las rutas documentales de una celda: primero los enlaces
// Markdown [texto](ruta); si no hay, los nombres de archivo del TODO citados
// entre marcas de código (`NNN-task-*.md`). Sin duplicados, en orden.
func enlaces(celda string) []string {
	var out []string
	visto := map[string]bool{}
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s != "" && !visto[s] {
			visto[s] = true
			out = append(out, s)
		}
	}
	for _, m := range enlaceMDRE.FindAllStringSubmatch(celda, -1) {
		add(m[1])
	}
	if len(out) == 0 {
		for _, m := range archivoTareaRE.FindAllString(celda, -1) {
			add(m)
		}
	}
	return out
}
