package models

import (
	"bufio"
	"encoding/binary"
	"io"
	"os"
)

// claveContexto es la clave del metadato GGUF que declara el contexto de
// entrenamiento del modelo (`general.context_length`).
const claveContexto = "general.context_length"

// tamanoMaximoCadena acota una cadena de metadatos para no reservar de más ante
// un archivo corrupto.
const tamanoMaximoCadena = 1 << 20

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

// leerContextoGGUF recorre la cabecera GGUF y su tabla de metadatos.
func leerContextoGGUF(r io.Reader) (int, error) {
	var magic [4]byte
	if _, err := io.ReadFull(r, magic[:]); err != nil {
		return 0, errorf("no se pudo leer la cabecera GGUF: %w", err)
	}
	if magic != [4]byte{'G', 'G', 'U', 'F'} {
		return 0, errorf("el archivo no es un GGUF (falta la firma)")
	}
	if _, err := leerU32(r); err != nil { // version
		return 0, err
	}
	if _, err := leerU64(r); err != nil { // tensor_count
		return 0, err
	}
	kv, err := leerU64(r)
	if err != nil {
		return 0, err
	}
	for i := uint64(0); i < kv; i++ {
		clave, err := leerCadena(r)
		if err != nil {
			return 0, err
		}
		tipo, err := leerU32(r)
		if err != nil {
			return 0, err
		}
		if clave == claveContexto {
			return leerContexto(r, tipo)
		}
		if err := saltarValor(r, tipo); err != nil {
			return 0, err
		}
	}
	return 0, nil
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
