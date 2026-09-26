package context

// trim.go — T-B011-04: recortar la selección hasta el límite.
//
// Fuente de verdad: [[specs/SPEC-NODO-CONTEXTO]] §Reglas ("El contexto
// entregado nunca supera el límite de contexto del modelo") y §Flujos
// alternativos ("Un documento es demasiado grande: se recorta por sección").
//
// La prioridad es el orden en que el modelo devolvió los documentos: ese orden
// es su juicio de relevancia. Se van incluyendo en ese orden hasta agotar el
// límite; el primero que no cabe se recorta por secciones, y lo que no cabe se
// descarta con su motivo. El resultado nunca queda vacío: si el primer
// documento no cabe entero, se entrega su primera parte.

import "strings"

// Recortar ajusta la selección al límite de tokens.
//
// `limite <= 0` significa sin tope: se incluye todo. Devuelve lo incluido, lo
// descartado con su motivo y los tokens incluidos.
func Recortar(seleccion []DocumentoSeleccionado, limite int) (incluidos []DocumentoSeleccionado, descartados []Descarte, tokens int) {
	if limite <= 0 {
		for _, d := range seleccion {
			tokens += d.Tokens
			incluidos = append(incluidos, d)
		}
		return incluidos, nil, tokens
	}

	for _, d := range seleccion {
		cupo := limite - tokens
		if cupo <= 0 {
			descartados = append(descartados, Descarte{Ruta: d.Ruta, Motivo: "no cabe en el límite de contexto"})
			continue
		}
		if d.Tokens <= cupo {
			incluidos = append(incluidos, d)
			tokens += d.Tokens
			continue
		}
		// El documento no cabe entero: se recorta por secciones. Si ya hay algo
		// incluido y ni la primera sección cabe, se descarta.
		recortado := recortarPorSeccion(d.Contenido, cupo)
		if recortado == "" {
			descartados = append(descartados, Descarte{Ruta: d.Ruta, Motivo: "no cabe en el límite de contexto"})
			continue
		}
		tks := EstimarTokens(recortado)
		if len(incluidos) > 0 && tks == 0 {
			descartados = append(descartados, Descarte{Ruta: d.Ruta, Motivo: "no cabe en el límite de contexto"})
			continue
		}
		incluidos = append(incluidos, DocumentoSeleccionado{Ruta: d.Ruta, Contenido: recortado, Tokens: tks})
		tokens += tks
	}
	return incluidos, descartados, tokens
}

// recortarPorSeccion conserva secciones enteras (delimitadas por encabezados)
// mientras quepan, y corta por líneas la última si hace falta. Devuelve "" solo
// si no cabe nada.
func recortarPorSeccion(contenido string, cupo int) string {
	if cupo <= 0 || strings.TrimSpace(contenido) == "" {
		return ""
	}
	secciones := partirSecciones(contenido)
	var b strings.Builder
	for _, s := range secciones {
		cand := s
		if b.Len() > 0 {
			cand = "\n" + s
		}
		if EstimarTokens(b.String()+cand) <= cupo {
			b.WriteString(cand)
			continue
		}
		if b.Len() == 0 {
			// La primera sección tampoco cabe: se corta por líneas para no
			// entregar vacío.
			b.WriteString(recortarLineas(s, cupo))
		}
		break
	}
	return strings.TrimRight(b.String(), "\n")
}

// partirSecciones divide el markdown en bloques que empiezan por un encabezado.
func partirSecciones(contenido string) []string {
	lineas := strings.Split(contenido, "\n")
	var (
		secciones []string
		actual    []string
	)
	for _, ln := range lineas {
		if strings.HasPrefix(strings.TrimSpace(ln), "#") && len(actual) > 0 {
			secciones = append(secciones, strings.Join(actual, "\n"))
			actual = nil
		}
		actual = append(actual, ln)
	}
	if len(actual) > 0 {
		secciones = append(secciones, strings.Join(actual, "\n"))
	}
	return secciones
}

// recortarLineas toma líneas enteras hasta agotar el cupo.
func recortarLineas(bloque string, cupo int) string {
	var b strings.Builder
	for _, ln := range strings.Split(bloque, "\n") {
		cand := ln
		if b.Len() > 0 {
			cand = "\n" + ln
		}
		if EstimarTokens(b.String()+cand) > cupo {
			break
		}
		b.WriteString(cand)
	}
	return b.String()
}
