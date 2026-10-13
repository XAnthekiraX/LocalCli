package models

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// parKV es un par clave/valor de la tabla de metadatos de un GGUF de prueba.
type parKV struct {
	clave string
	tipo  uint32
	valor []byte
}

// escribirGGUF arma un GGUF mínimo —cabecera y tabla de metadatos, sin
// tensores— y devuelve su ruta.
func escribirGGUF(t *testing.T, pares ...parKV) string {
	t.Helper()
	var b bytes.Buffer
	b.WriteString("GGUF")
	escribirU32(&b, 3)
	escribirU64(&b, 0) // tensor_count
	escribirU64(&b, uint64(len(pares)))
	for _, p := range pares {
		escribirCadena(&b, p.clave)
		escribirU32(&b, p.tipo)
		b.Write(p.valor)
	}
	ruta := filepath.Join(t.TempDir(), "m.gguf")
	if err := os.WriteFile(ruta, b.Bytes(), 0o644); err != nil {
		t.Fatalf("no se pudo escribir el GGUF de prueba: %v", err)
	}
	return ruta
}

func escribirU32(b *bytes.Buffer, v uint32) {
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], v)
	b.Write(buf[:])
}

func escribirU64(b *bytes.Buffer, v uint64) {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], v)
	b.Write(buf[:])
}

func escribirCadena(b *bytes.Buffer, s string) {
	escribirU64(b, uint64(len(s)))
	b.WriteString(s)
}

func valorU32(v uint32) []byte {
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], v)
	return buf[:]
}

func valorU64(v uint64) []byte {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], v)
	return buf[:]
}

// valorArray arma un valor de tipo array con elementos de tamaño fijo.
func valorArray(elemento uint32, elementos ...uint32) []byte {
	var b bytes.Buffer
	escribirU32(&b, elemento)
	escribirU64(&b, uint64(len(elementos)))
	for _, e := range elementos {
		escribirU32(&b, e)
	}
	return b.Bytes()
}

// valorCadena arma un valor de tipo string.
func valorCadena(s string) []byte {
	var b bytes.Buffer
	escribirCadena(&b, s)
	return b.Bytes()
}

func TestContextoDeGGUFLeeUint32(t *testing.T) {
	ruta := escribirGGUF(t, parKV{claveContexto, tipoUint32, valorU32(32768)})
	ctx, err := ContextoDeGGUF(ruta)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if ctx != 32768 {
		t.Fatalf("contexto = %d, quiero 32768", ctx)
	}
}

func TestContextoDeGGUFLeeUint64(t *testing.T) {
	ruta := escribirGGUF(t, parKV{claveContexto, tipoUint64, valorU64(131072)})
	ctx, err := ContextoDeGGUF(ruta)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if ctx != 131072 {
		t.Fatalf("contexto = %d, quiero 131072", ctx)
	}
}

func TestContextoDeGGUFSaltaClavesPrevias(t *testing.T) {
	ruta := escribirGGUF(t,
		parKV{"general.architecture", tipoString, valorCadena("llama")},
		parKV{"tokenizer.ggml.merges", tipoArray, valorArray(tipoUint32, 1, 2, 3)},
		parKV{"llama.block_count", tipoUint32, valorU32(32)},
		parKV{claveContexto, tipoUint32, valorU32(8192)},
	)
	ctx, err := ContextoDeGGUF(ruta)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if ctx != 8192 {
		t.Fatalf("contexto = %d, quiero 8192", ctx)
	}
}

func TestContextoDeGGUFSinLaClave(t *testing.T) {
	ruta := escribirGGUF(t,
		parKV{"general.architecture", tipoString, valorCadena("llama")},
		parKV{"llama.block_count", tipoUint32, valorU32(32)},
	)
	ctx, err := ContextoDeGGUF(ruta)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if ctx != 0 {
		t.Fatalf("contexto = %d, quiero 0 (sin declarar)", ctx)
	}
}

func TestContextoDeGGUFMagicInvalido(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "m.gguf")
	if err := os.WriteFile(ruta, []byte("NO-GGUF-data"), 0o644); err != nil {
		t.Fatalf("no se pudo escribir: %v", err)
	}
	if _, err := ContextoDeGGUF(ruta); err == nil {
		t.Fatal("un archivo sin firma GGUF debería fallar")
	}
}

func TestContextoDeGGUFTruncado(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "m.gguf")
	if err := os.WriteFile(ruta, []byte("GGUF"), 0o644); err != nil {
		t.Fatalf("no se pudo escribir: %v", err)
	}
	if _, err := ContextoDeGGUF(ruta); err == nil {
		t.Fatal("un GGUF truncado debería fallar")
	}
}
