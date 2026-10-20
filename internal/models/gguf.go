package models

import (
	"bufio"
	"encoding/binary"
	"io"
	"os"
	"strings"
)

// clavesContexto son las claves a las que buscar el contexto del modelo, en orden
// de preferencia. Los GGUF modernos usan la clave prefijada por arquitectura
// (p. ej. `qwen35.context_length`, `qwen3.context_length`, `qwen2.context_length`).
var clavesContexto = []string{
	"general.context_length",
}

// tamanoMaximoCadena acota una cadena de metadatos para no reservar de más ante
// un archivo corrupto.
const tamanoMaximoCadena = 1 << 20

// maxTensores acota cuántos nombres de tensor se leen de la cabecera. Un GGUF
// normal declara unos pocos miles; un archivo corrupto no debe poder reservar
// memoria sin límite.
const maxTensores = 1 << 20

// Tipos de valor de un metadato GGUF.
const (
	tipoUint8   = 0
	tipoInt8    = 1
	tipoUint16  = 2
	tipoInt16   = 3
	tipoUint32  = 4
	tipoInt32   = 5
	tipoFloat32 = 6
	tipoBool    = 7
	tipoString  = 8
	tipoArray   = 9
	tipoUint64  = 10
	tipoInt64   = 11
	tipoFloat64 = 12
)

// ContextoDeGGUF lee el contexto declarado por el modelo en su GGUF
// (`general.context_length`) sin tocar sus tensores: solo la cabecera y su tabla
// de metadatos. Devuelve 0 cuando el archivo no declara esa clave.
func ContextoDeGGUF(ruta string) (int, error) {
	f, err := os.Open(ruta)
	if err != nil {
		return 0, errorf("no se pudo abrir el GGUF %s: %w", ruta, err)
	}
	defer f.Close()
	return leerContextoGGUF(bufio.NewReader(f))
}

// fichaGGUF es lo que el harness necesita saber de un GGUF sin cargarlo: su
// arquitectura, su contexto, la estructura de capas y los nombres de sus
// tensores.
type fichaGGUF struct {
	arch      string
	contexto  int
	bloque    int
	nextn     int
	nTensores uint64
	tensores  []string
}

// leerContextoGGUF recorre la cabecera GGUF y su tabla de metadatos.
func leerContextoGGUF(r io.Reader) (int, error) {
	f, err := leerMetadatosGGUF(r)
	if err != nil {
		return 0, err
	}
	return f.contexto, nil
}

// leerFichaGGUF lee metadatos y la tabla de tensores, sin tocar los datos de los
// tensores. Es lo que permite distinguir un modelo completo de una cabeza MTP.
func leerFichaGGUF(r io.Reader) (fichaGGUF, error) {
	f, err := leerMetadatosGGUF(r)
	if err != nil {
		return f, err
	}
	if f.nTensores > maxTensores {
		return f, errorf("el GGUF declara demasiados tensores: %d", f.nTensores)
	}
	f.tensores = make([]string, 0, f.nTensores)
	for i := uint64(0); i < f.nTensores; i++ {
		nombre, err := leerCadena(r)
		if err != nil {
			return f, err
		}
		dimensiones, err := leerU32(r)
		if err != nil {
			return f, err
		}
		for j := uint32(0); j < dimensiones; j++ {
			if _, err := leerU64(r); err != nil {
				return f, err
			}
		}
		if _, err := leerU32(r); err != nil { // tipo
			return f, err
		}
		if _, err := leerU64(r); err != nil { // offset
			return f, err
		}
		f.tensores = append(f.tensores, nombre)
	}
	return f, nil
}

// leerMetadatosGGUF recorre la cabecera GGUF y su tabla de metadatos.
func leerMetadatosGGUF(r io.Reader) (fichaGGUF, error) {
	var f fichaGGUF
	var magic [4]byte
	if _, err := io.ReadFull(r, magic[:]); err != nil {
		return f, errorf("no se pudo leer la cabecera GGUF: %w", err)
	}
	if magic != [4]byte{'G', 'G', 'U', 'F'} {
		return f, errorf("el archivo no es un GGUF (falta la firma)")
	}
	if _, err := leerU32(r); err != nil { // version
		return f, err
	}
	nTensores, err := leerU64(r)
	if err != nil {
		return f, err
	}
	f.nTensores = nTensores
	kv, err := leerU64(r)
	if err != nil {
		return f, err
	}
	for i := uint64(0); i < kv; i++ {
		clave, err := leerCadena(r)
		if err != nil {
			return f, err
		}
		tipo, err := leerU32(r)
		if err != nil {
			return f, err
		}
		leido := false
		if clave == "general.architecture" && tipo == tipoString {
			val, err := leerCadenaValor(r)
			if err != nil {
				return f, err
			}
			f.arch = val
			leido = true
		}
		if !leido && f.contexto == 0 && esClaveContexto(clave, f.arch) {
			c, err := leerContexto(r, tipo)
			if err != nil {
				return f, err
			}
			f.contexto = c
			leido = true
		}
		if !leido && f.bloque == 0 && esClaveDe(clave, ".block_count") {
			b, err := leerContexto(r, tipo)
			if err != nil {
				return f, err
			}
			f.bloque = b
			leido = true
		}
		if !leido && f.nextn == 0 && esClaveDe(clave, ".nextn_predict_layers") {
			n, err := leerContexto(r, tipo)
			if err != nil {
				return f, err
			}
			f.nextn = n
			leido = true
		}
		if !leido {
			if err := saltarValor(r, tipo); err != nil {
				return f, err
			}
		}
	}
	return f, nil
}

// esClaveDe indica si la clave termina en el sufijo dado, para no depender de
// que la arquitectura se haya leído ya.
func esClaveDe(clave, sufijo string) bool {
	return strings.HasSuffix(clave, sufijo)
}

func esClaveContexto(clave, arch string) bool {
	for _, c := range clavesContexto {
		if clave == c {
			return true
		}
	}
	if arch != "" && clave == arch+".context_length" {
		return true
	}
	if len(clave) > 15 && clave[len(clave)-15:] == ".context_length" {
		return true
	}
	return false
}

func leerCadenaValor(r io.Reader) (string, error) {
	// tipo ya leído; leer el valor string (con su longitud + contenido)
	n, err := leerU64(r)
	if err != nil {
		return "", err
	}
	if n > tamanoMaximoCadena {
		return "", errorf("cadena de metadatos GGUF demasiado larga: %d", n)
	}
	b := make([]byte, n)
	if _, err := io.ReadFull(r, b); err != nil {
		return "", truncado(err)
	}
	return string(b), nil
}

// leerContexto lee el valor de la clave de contexto; se aceptan las variantes
// entera y flotante que usan los GGUF reales.
func leerContexto(r io.Reader, tipo uint32) (int, error) {
	switch tipo {
	case tipoUint8, tipoInt8:
		var b [1]byte
		if _, err := io.ReadFull(r, b[:]); err != nil {
			return 0, truncado(err)
		}
		return int(b[0]), nil
	case tipoUint16, tipoInt16:
		var b [2]byte
		if _, err := io.ReadFull(r, b[:]); err != nil {
			return 0, truncado(err)
		}
		return int(binary.LittleEndian.Uint16(b[:])), nil
	case tipoUint32, tipoInt32, tipoFloat32:
		v, err := leerU32(r)
		if err != nil {
			return 0, err
		}
		return int(v), nil
	case tipoUint64, tipoInt64, tipoFloat64:
		v, err := leerU64(r)
		if err != nil {
			return 0, err
		}
		return int(v), nil
	default:
		return 0, errorf("el contexto del GGUF tiene un tipo de valor inesperado: %d", tipo)
	}
}

// saltarValor avanza sobre el valor de un metadato que no interesa.
func saltarValor(r io.Reader, tipo uint32) error {
	if n, ok := tamanoFijo(tipo); ok {
		_, err := io.CopyN(io.Discard, r, int64(n))
		return truncado(err)
	}
	switch tipo {
	case tipoString:
		_, err := leerCadena(r)
		return err
	case tipoArray:
		elemento, err := leerU32(r)
		if err != nil {
			return err
		}
		n, err := leerU64(r)
		if err != nil {
			return err
		}
		if sz, ok := tamanoFijo(elemento); ok {
			_, err := io.CopyN(io.Discard, r, int64(sz)*int64(n))
			return truncado(err)
		}
		for i := uint64(0); i < n; i++ {
			if err := saltarValor(r, elemento); err != nil {
				return err
			}
		}
		return nil
	default:
		return errorf("tipo de metadato GGUF desconocido: %d", tipo)
	}
}

// tamanoFijo devuelve el ancho en bytes de un tipo de valor de tamaño fijo.
func tamanoFijo(tipo uint32) (int, bool) {
	switch tipo {
	case tipoUint8, tipoInt8, tipoBool:
		return 1, true
	case tipoUint16, tipoInt16:
		return 2, true
	case tipoUint32, tipoInt32, tipoFloat32:
		return 4, true
	case tipoUint64, tipoInt64, tipoFloat64:
		return 8, true
	}
	return 0, false
}

func leerU32(r io.Reader) (uint32, error) {
	var b [4]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return 0, truncado(err)
	}
	return binary.LittleEndian.Uint32(b[:]), nil
}

func leerU64(r io.Reader) (uint64, error) {
	var b [8]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return 0, truncado(err)
	}
	return binary.LittleEndian.Uint64(b[:]), nil
}

func leerCadena(r io.Reader) (string, error) {
	n, err := leerU64(r)
	if err != nil {
		return "", err
	}
	if n > tamanoMaximoCadena {
		return "", errorf("cadena de metadatos GGUF demasiado larga: %d", n)
	}
	b := make([]byte, n)
	if _, err := io.ReadFull(r, b); err != nil {
		return "", truncado(err)
	}
	return string(b), nil
}

func truncado(err error) error {
	if err == nil {
		return nil
	}
	return errorf("GGUF truncado: %w", err)
}
