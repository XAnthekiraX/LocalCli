package fileops

import (
	"fmt"
	"strings"

	"localcli/internal/tools"
)

const (
	// lineasContexto son las líneas iguales que se muestran alrededor de cada
	// cambio para situarlo dentro del archivo.
	lineasContexto = 3
	// topeBloque es el máximo de líneas que se pintan del diff; lo que sobra se
	// resume con un aviso. El antes y el después completos siguen en
	// `change_history` (SPEC-TOOLS §La diferencia a la vista).
	topeBloque = 40
	// topeCeldasLCS acota la tabla del algoritmo: por encima de este tamaño el
	// tramo cambiado se muestra entero quitado y añadido, sin buscar el detalle.
	topeCeldasLCS = 1_000_000

	tipoBloqueDiferencia = "diferencia"
)

// claseOp es el papel de una línea en el diff: se queda igual, se quita o se
// añade.
type claseOp byte

const (
	opIgual  claseOp = ' '
	opQuitar claseOp = '-'
	opPoner  claseOp = '+'
)

type operacion struct {
	clase claseOp
	texto string
}

// Diferencia devuelve el antes y el después de un cambio como un diff unificado
// por líneas: las quitadas con `-`, las añadidas con `+`, tres líneas de contexto
// y una cabecera `@@ -inicio,n +inicio,n @@` por tramo. Si el cambio no cabe en
// el tope, cierra con `… N líneas más`; el registro completo vive en
// `change_history`. Un contenido idéntico no produce diferencia.
func Diferencia(antes, despues string) string {
	if antes == despues {
		return ""
	}
	ops := comparar(partirEnLineas(antes), partirEnLineas(despues))
	return recortar(componer(ops, bloques(ops)), topeBloque)
}

// conDiferencia arma el resultado de una escritura aplicada con su bloque de
// diferencia. Si el texto no cambió —o solo cambió el fin de línea—, el bloque
// no se adjunta: no hay nada que enseñar.
func conDiferencia(mensaje, antes, despues string) tools.Resultado {
	res := tools.Resultado{Salida: mensaje}
	if texto := Diferencia(antes, despues); texto != "" {
		res.Meta = map[string]any{
			"bloque":      texto,
			"bloque_tipo": tipoBloqueDiferencia,
		}
	}
	return res
}

// partirEnLineas separa el contenido en líneas sin conservar el salto final: un
// archivo que termina en `\n` no gana una línea vacía. El `\r` de un fin de
// línea CRLF se conserva para comparar y se quita al pintar.
func partirEnLineas(contenido string) []string {
	// El BOM no se pinta: es una marca del archivo, no contenido del cambio.
	contenido = strings.TrimPrefix(contenido, "\ufeff")
	contenido = strings.TrimSuffix(contenido, "\n")
	if contenido == "" {
		return nil
	}
	return strings.Split(contenido, "\n")
}

// comparar empareja dos listas de líneas: recorta el prefijo y el sufijo iguales
// y busca la mejor secuencia en el tramo del medio.
func comparar(a, b []string) []operacion {
	prefijo := 0
	for prefijo < len(a) && prefijo < len(b) && a[prefijo] == b[prefijo] {
		prefijo++
	}
	sufijo := 0
	for sufijo < len(a)-prefijo && sufijo < len(b)-prefijo &&
		a[len(a)-1-sufijo] == b[len(b)-1-sufijo] {
		sufijo++
	}

	ops := make([]operacion, 0, len(a)+len(b))
	for _, linea := range a[:prefijo] {
		ops = append(ops, operacion{opIgual, linea})
	}
	ops = append(ops, compararTramo(a[prefijo:len(a)-sufijo], b[prefijo:len(b)-sufijo])...)
	for _, linea := range a[len(a)-sufijo:] {
		ops = append(ops, operacion{opIgual, linea})
	}
	return ops
}

// compararTramo resuelve la parte distinta. Con tramos pequeños usa la
// subsecuencia común más larga, que da el diff mínimo; con tramos grandes —más
// de un millón de celdas— evita la tabla y muestra el tramo entero quitado y
// añadido, que sigue siendo correcto.
func compararTramo(a, b []string) []operacion {
	if len(a) == 0 {
		return todas(opPoner, b)
	}
	if len(b) == 0 {
		return todas(opQuitar, a)
	}
	if len(a)*len(b) > topeCeldasLCS {
		return append(todas(opQuitar, a), todas(opPoner, b)...)
	}

	n, m := len(a), len(b)
	// tabla[i][j] es la longitud de la subsecuencia común desde i, j. Se guarda
	// plana para no crear un slice por fila.
	ancho := m + 1
	tabla := make([]int32, (n+1)*ancho)
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				tabla[i*ancho+j] = tabla[(i+1)*ancho+j+1] + 1
			} else if tabla[(i+1)*ancho+j] >= tabla[i*ancho+j+1] {
				tabla[i*ancho+j] = tabla[(i+1)*ancho+j]
			} else {
				tabla[i*ancho+j] = tabla[i*ancho+j+1]
			}
		}
	}

	ops := make([]operacion, 0, n+m)
	i, j := 0, 0
	for i < n && j < m {
		switch {
		case a[i] == b[j]:
			ops = append(ops, operacion{opIgual, a[i]})
			i++
			j++
		case tabla[(i+1)*ancho+j] >= tabla[i*ancho+j+1]:
			ops = append(ops, operacion{opQuitar, a[i]})
			i++
		default:
			ops = append(ops, operacion{opPoner, b[j]})
			j++
		}
	}
	for ; i < n; i++ {
		ops = append(ops, operacion{opQuitar, a[i]})
	}
	for ; j < m; j++ {
		ops = append(ops, operacion{opPoner, b[j]})
	}
	return ops
}

func todas(clase claseOp, lineas []string) []operacion {
	ops := make([]operacion, len(lineas))
	for i, linea := range lineas {
		ops[i] = operacion{clase, linea}
	}
	return ops
}

// bloques agrupa los cambios que comparten contexto y devuelve, por cada grupo,
// el tramo de operaciones que se pinta: desde tres líneas antes del primer
// cambio hasta tres después del último.
func bloques(ops []operacion) [][2]int {
	var cambios []int
	for i, op := range ops {
		if op.clase != opIgual {
			cambios = append(cambios, i)
		}
	}
	if len(cambios) == 0 {
		return nil
	}

	var grupos [][2]int
	ini, fin := cambios[0], cambios[0]+1
	for _, idx := range cambios[1:] {
		if idx-fin <= 2*lineasContexto {
			fin = idx + 1
			continue
		}
		grupos = append(grupos, [2]int{ini, fin})
		ini, fin = idx, idx+1
	}
	grupos = append(grupos, [2]int{ini, fin})

	for i := range grupos {
		desde := grupos[i][0] - lineasContexto
		if desde < 0 {
			desde = 0
		}
		hasta := grupos[i][1] + lineasContexto
		if hasta > len(ops) {
			hasta = len(ops)
		}
		grupos[i] = [2]int{desde, hasta}
	}

	// Dos grupos con el mismo contexto son un solo tramo.
	unidos := [][2]int{grupos[0]}
	for _, grupo := range grupos[1:] {
		ultimo := &unidos[len(unidos)-1]
		if grupo[0] <= ultimo[1] {
			if grupo[1] > ultimo[1] {
				ultimo[1] = grupo[1]
			}
			continue
		}
		unidos = append(unidos, grupo)
	}
	return unidos
}

// componer convierte las operaciones en las líneas del diff, con la cabecera de
// cada tramo y los números de línea reales del antes y del después.
func componer(ops []operacion, grupos [][2]int) []string {
	// aAntes[i] y bAntes[i] son las líneas consumidas antes de la operación i,
	// contadas desde cero; la cabecera las publica desde uno.
	aAntes := make([]int, len(ops)+1)
	bAntes := make([]int, len(ops)+1)
	for i, op := range ops {
		aAntes[i+1], bAntes[i+1] = aAntes[i], bAntes[i]
		if op.clase != opPoner {
			aAntes[i+1]++
		}
		if op.clase != opQuitar {
			bAntes[i+1]++
		}
	}

	var lineas []string
	for _, grupo := range grupos {
		inicio, fin := grupo[0], grupo[1]
		aIni, bIni := aAntes[inicio]+1, bAntes[inicio]+1
		aLen := aAntes[fin] - aAntes[inicio]
		bLen := bAntes[fin] - bAntes[inicio]
		if aLen == 0 {
			aIni = aAntes[inicio]
		}
		if bLen == 0 {
			bIni = bAntes[inicio]
		}
		lineas = append(lineas, fmt.Sprintf("@@ -%d,%d +%d,%d @@", aIni, aLen, bIni, bLen))
		for _, op := range ops[inicio:fin] {
			lineas = append(lineas, string(op.clase)+mostrarLinea(op.texto))
		}
	}
	return lineas
}

// mostrarLinea quita el `\r` de un fin de línea CRLF para que no ensucie el
// bloque; el byte sigue intacto en el archivo y en `change_history`.
func mostrarLinea(linea string) string {
	return strings.TrimSuffix(linea, "\r")
}

// recortar une las líneas y, si pasan del tope, corta y avisa cuántas quedaron
// fuera. El aviso cuenta como cierre, no como línea del diff.
func recortar(lineas []string, tope int) string {
	if len(lineas) == 0 {
		return ""
	}
	if len(lineas) <= tope {
		return strings.Join(lineas, "\n")
	}
	salida := make([]string, 0, tope+1)
	salida = append(salida, lineas[:tope]...)
	salida = append(salida, fmt.Sprintf("… %d líneas más", len(lineas)-tope))
	return strings.Join(salida, "\n")
}
