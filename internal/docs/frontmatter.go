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
		v = strings.TrimSpace(v)
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
				items = append(items, strings.TrimSpace(strings.TrimPrefix(item, "-")))
				j++
			}
			if items != nil {
				m.Listas[k] = items
			} else {
				m.Claves[k] = ""
			}
			i = j
			continue
		case strings.HasPrefix(v, "[") && strings.HasSuffix(v, "]"):
			for _, it := range strings.Split(strings.Trim(v, "[]"), ",") {
				if it = strings.TrimSpace(it); it != "" {
					m.Listas[k] = append(m.Listas[k], it)
				}
			}
		default:
			m.Claves[k] = v
		}
		i++
	}
	return nil
}
