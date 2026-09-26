package docs

// frontmatter.go — T-B003-02: parsear el frontmatter YAML de cada documento.
//
// El bloque es el estándar: líneas entre dos separadores "---" al inicio del
// archivo (DOMAIN.md: "Frontmatter con etiquetas y enlaces que declaran
// dependencias"). Para no arrastrar una dependencia YAML externa, se soporta
// el subconjunto real que usan estos documentos:
//
//	clave: valor            → Meta.Claves[clave] = valor
//	clave: [a, b]           → lista inline
//	clave:                  → lista en bloque con items "- x"
//	  - a
//	  - b
//
// Cualquier línea que no encaja (tabulaciones, dos puntos duplicados, bloques
// anidados) produce ErrFrenteInvalido, que el cargador reporta como error
// localizado por archivo sin abortar la carga (T-B003-06).

import (
	"errors"
	"fmt"
	"strings"
)

// ErrFrenteInvalido indica que el frontmatter no pudo parsearse. El cargador
// lo envuelve con el código E_DOC_PARSE y la ruta del documento.
var ErrFrenteInvalido = errors.New("frontmatter inválido")

// Meta es el frontmatter parseado. Tags recoge las claves convencionales de
// etiquetado (tags/etiquetas/keywords); el resto queda en Claves.
type Meta struct {
	Tags   []string
	Claves map[string]string
	Listas map[string][]string
}

// ParseFrontmatter separa el bloque inicial del cuerpo. Devuelve el meta, el
// cuerpo (sin el bloque), si había frontmatter y el error de parseo. Un
// archivo sin "---" inicial no es un error: devuelve Meta vacío y todo el
// texto como cuerpo.
func ParseFrontmatter(raw []byte) (Meta, string, bool, error) {
	meta := Meta{Claves: map[string]string{}, Listas: map[string][]string{}}
	texto := strings.ReplaceAll(string(raw), "\r\n", "\n")
	if !strings.HasPrefix(texto, "---\n") {
		return meta, texto, false, nil
	}
	cierre := strings.Index(texto[4:], "\n---")
	if cierre < 0 {
		return meta, texto, true, fmt.Errorf("%w: bloque sin cerrar", ErrFrenteInvalido)
	}
	bloque := texto[4 : 4+cierre]
	cuerpo := texto[4+cierre+len("\n---"):]
	if i := strings.Index(cuerpo, "\n"); i >= 0 {
		cuerpo = cuerpo[i+1:] // descarta la línea de cierre (--- o ---fin)
	} else {
		cuerpo = ""
	}
	if err := parseBloque(bloque, &meta); err != nil {
		return meta, cuerpo, true, err
	}
	for _, k := range []string{"tags", "etiquetas", "keywords"} {
		if v, ok := meta.Listas[k]; ok {
			meta.Tags = append(meta.Tags, v...)
		} else if v, ok := meta.Claves[k]; ok && v != "" {
			for _, t := range strings.Split(v, ",") {
				if t = strings.TrimSpace(t); t != "" {
					meta.Tags = append(meta.Tags, t)
				}
			}
		}
	}
	return meta, cuerpo, true, nil
}

func parseBloque(bloque string, m *Meta) error {
	lineas := strings.Split(bloque, "\n")
	i := 0
	for i < len(lineas) {
		linea := lineas[i]
		if strings.TrimSpace(linea) == "" || strings.HasPrefix(strings.TrimSpace(linea), "#") {
			i++
			continue
		}
		if strings.HasPrefix(linea, " ") || strings.HasPrefix(linea, "\t") {
			return fmt.Errorf("%w: sangrado fuera de lista: %q", ErrFrenteInvalido, linea)
		}
		k, v, ok := strings.Cut(linea, ":")
		if !ok {
			return fmt.Errorf("%w: línea sin clave: valor: %q", ErrFrenteInvalido, linea)
		}
		k = strings.TrimSpace(k)
		if k == "" {
			return fmt.Errorf("%w: clave vacía en %q", ErrFrenteInvalido, linea)
		}
		// Aquí solo se quita el comentario y se CONSERVAN las comillas. Quitar
		// las comillas aquí rompía las listas inline: ["[[b]]"] perdía las
		// comillas de sus elementos y se leía como [[[b]]], que tras el Trim
		// se quedaba en "b", sin ningún enlace.
		v = sinComentario(strings.TrimSpace(v))
		switch {
		case v == "":
			// ¿lista en bloque en las siguientes líneas?
			var items []string
			j := i + 1
			for j < len(lineas) && strings.HasPrefix(lineas[j], "  ") {
				item := strings.TrimSpace(lineas[j])
				if !strings.HasPrefix(item, "- ") && item != "-" {
					return fmt.Errorf("%w: se esperaba '- ' en %q", ErrFrenteInvalido, lineas[j])
				}
				limpio, err := desencomentar(strings.TrimSpace(strings.TrimPrefix(item, "-")))
				if err != nil {
					return fmt.Errorf("%w: %v en %q", ErrFrenteInvalido, err, lineas[j])
				}
				items = append(items, limpio)
				j++
			}
			if items != nil {
				m.Listas[k] = items
			} else {
				m.Claves[k] = ""
			}
			i = j
			continue
		case strings.HasPrefix(v, "["):
			// Una lista inline que no cierra no es un valor: es basura que no
			// debe guardarse como una sola clave.
			if !strings.HasSuffix(v, "]") {
				return fmt.Errorf("%w: lista sin cerrar en %q", ErrFrenteInvalido, linea)
			}
			for _, it := range strings.Split(strings.Trim(v, "[]"), ",") {
				limpio, err := desencomentar(strings.TrimSpace(it))
				if err != nil {
					return fmt.Errorf("%w: %v en %q", ErrFrenteInvalido, err, linea)
				}
				if limpio != "" {
					m.Listas[k] = append(m.Listas[k], limpio)
				}
			}
		default:
			escalar, err := desencomentar(v)
			if err != nil {
				return fmt.Errorf("%w: %v en %q", ErrFrenteInvalido, err, linea)
			}
			m.Claves[k] = escalar
		}
		i++
	}
	return nil
}

// sinComentario quita el comentario YAML del final de un valor y conserva las
// comillas. Es el paso previo a decidir si el valor es escalar o lista: quitar
// las comillas aquí rompería los elementos de una lista inline.
func sinComentario(v string) string {
	var (
		cita  byte
		com   bool
		saida strings.Builder
	)
	for i := 0; i < len(v); i++ {
		c := v[i]
		switch {
		case com:
			continue
		case cita == 0 && c == '#' && (i == 0 || v[i-1] == ' ' || v[i-1] == '\t'):
			com = true
			continue
		case cita == 0 && (c == '"' || c == '\''):
			cita = c
		case cita != 0 && c == cita:
			cita = 0
		}
		saida.WriteByte(c)
	}
	return strings.TrimSpace(saida.String())
}

// desencomentar quita el comentario YAML de una línea de valor y quita las
// comillas que envuelven el valor.
//
// Sin esto, `title: Mi título # nota` se guardaba con el "# nota" dentro y
// `title: "Mi título"` se guardaba con las comillas. Las comillas también se
// validan: una comilla sin cerrar es un frontmatter roto y no un valor
// aceptable a medias.
func desencomentar(v string) (string, error) {
	// El comentario solo cuenta fuera de comillas.
	var (
		cita  byte
		com   bool // dentro de un comentario
		saida strings.Builder
	)
	for i := 0; i < len(v); i++ {
		c := v[i]
		switch {
		case com:
			continue // el resto de la línea es comentario
		case cita == 0 && c == '#' && (i == 0 || v[i-1] == ' ' || v[i-1] == '\t'):
			com = true
			continue
		case cita == 0 && (c == '"' || c == '\''):
			// Abre comilla:se marca el estado pero NO se copia el carácter.
			// Copiarlo guardaba el valor con las comillas dentro.
			cita = c
			continue
		case cita != 0 && c == cita:
			cita = 0 // cierra: tampoco se copia
			continue
		}
		saida.WriteByte(c)
	}
	if cita != 0 {
		// Una comilla abierta y nunca cerrada no es un valor: es un
		// frontmatter roto. Aceptarla a medias guardaba el resto de la línea
		// como contenido del documento, y el archivo pasaba por válido.
		return "", fmt.Errorf("comilla %q sin cerrar", string(cita))
	}
	return strings.TrimSpace(saida.String()), nil
}
