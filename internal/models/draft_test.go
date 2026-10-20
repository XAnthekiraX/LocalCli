package models

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// tensorPrueba es una entrada de la tabla de tensores de un GGUF de prueba.
type tensorPrueba struct {
	nombre string
	dims   []uint64
}

// escribirGGUFConTensores arma un GGUF mínimo con metadatos y la tabla de
// tensores, sin datos de tensor: basta para probar la inspección de cabecera.
func escribirGGUFConTensores(t *testing.T, tensores []tensorPrueba, pares ...parKV) string {
	t.Helper()
	var b bytes.Buffer
	b.WriteString("GGUF")
	escribirU32(&b, 3)
	escribirU64(&b, uint64(len(tensores)))
	escribirU64(&b, uint64(len(pares)))
	for _, p := range pares {
		escribirCadena(&b, p.clave)
		escribirU32(&b, p.tipo)
		b.Write(p.valor)
	}
	for _, tensor := range tensores {
		escribirCadena(&b, tensor.nombre)
		escribirU32(&b, uint32(len(tensor.dims)))
		for _, dim := range tensor.dims {
			escribirU64(&b, dim)
		}
		escribirU32(&b, 0) // tipo
		escribirU64(&b, 0) // offset
	}
	ruta := filepath.Join(t.TempDir(), "m.gguf")
	if err := os.WriteFile(ruta, b.Bytes(), 0o644); err != nil {
		t.Fatalf("no se pudo escribir el GGUF de prueba: %v", err)
	}
	return ruta
}

func TestEsCabezaDraftNombre(t *testing.T) {
	casos := []struct {
		ruta string
		si   bool
	}{
		{"mtp-Qwen3.8-Flash-Next-Q4_K_M.gguf", true},
		{"/home/u/.config/localcli/models/mtp-x/mtp-x.gguf", true},
		{"/home/u/repo/MTP/mtp-x.gguf", true},
		{"/home/u/repo/MTP/modelo.gguf", true},
		{"https://huggingface.co/org/repo/resolve/main/MTP/MTP-X.gguf", true},
		{"/models/qwen3-14b/qwen3-14b-q4_k_m.gguf", false},
		{"/models/mtp-helper/modelo.gguf", false},
		{"/home/MTP/models/qwen3-14b/qwen.gguf", false},
		{"", false},
	}
	for _, c := range casos {
		if got := EsCabezaDraftNombre(c.ruta); got != c.si {
			t.Errorf("EsCabezaDraftNombre(%q) = %v, quiero %v", c.ruta, got, c.si)
		}
	}
}

func TestMotivoCabezaDraftPorEstructura(t *testing.T) {
	ruta := escribirGGUFConTensores(t,
		[]tensorPrueba{
			{nombre: "token_embd.weight", dims: []uint64{2560, 248320}},
			{nombre: "blk.48.attn_q.weight", dims: []uint64{2560, 12288}},
			{nombre: "blk.48.nextn.eh_proj.weight", dims: []uint64{5120, 2560}},
		},
		parKV{"general.architecture", tipoString, valorCadena("qwen4exp")},
		parKV{"qwen4exp.block_count", tipoUint32, valorU32(49)},
		parKV{"qwen4exp.nextn_predict_layers", tipoUint32, valorU32(1)},
	)
	if motivo := MotivoCabezaDraft(ruta); motivo == "" {
		t.Fatal("un GGUF con solo la última capa y nextn debería detectarse como cabeza MTP")
	}
}

func TestMotivoCabezaDraftNoConfundeModeloCompleto(t *testing.T) {
	ruta := escribirGGUFConTensores(t,
		[]tensorPrueba{
			{nombre: "token_embd.weight", dims: []uint64{2560, 248320}},
			{nombre: "blk.0.attn_q.weight", dims: []uint64{2560, 12288}},
			{nombre: "blk.1.attn_q.weight", dims: []uint64{2560, 12288}},
			{nombre: "blk.2.attn_q.weight", dims: []uint64{2560, 12288}},
			{nombre: "output.weight", dims: []uint64{2560, 248320}},
		},
		parKV{"general.architecture", tipoString, valorCadena("qwen4exp")},
		parKV{"qwen4exp.block_count", tipoUint32, valorU32(3)},
		parKV{"qwen4exp.nextn_predict_layers", tipoUint32, valorU32(1)},
	)
	if motivo := MotivoCabezaDraft(ruta); motivo != "" {
		t.Fatalf("un modelo con varias capas no es una cabeza MTP: %s", motivo)
	}
}

func TestMotivoCabezaDraftPorNombre(t *testing.T) {
	if motivo := MotivoCabezaDraft("/no/existe/MTP/mtp-x.gguf"); motivo == "" {
		t.Fatal("la convención de nombre MTP debería bastar sin leer el archivo")
	}
	if motivo := MotivoCabezaDraft("/no/existe/qwen.gguf"); motivo != "" {
		t.Fatalf("un archivo ilegible sin nombre MTP no se puede acusar: %q", motivo)
	}
}

func TestParsearURLRechazaCabezaMTP(t *testing.T) {
	raw := "https://huggingface.co/unsloth/Qwen3.8-Flash-Next-GGUF/resolve/main/MTP/mtp-Qwen3.8-Flash-Next-Q4_K_M.gguf"
	if _, err := ParsearURL(raw); err == nil {
		t.Fatal("una URL de cabeza MTP debería rechazarse")
	}
}

func TestParsearURLAceptaModeloPrincipal(t *testing.T) {
	raw := "https://huggingface.co/Qwen/Qwen3-14B-GGUF/resolve/main/Qwen3-14B-Q4_K_M.gguf"
	if _, err := ParsearURL(raw); err != nil {
		t.Fatalf("un modelo principal no debería rechazarse: %v", err)
	}
}

func TestAltaRechazaCabezaMTP(t *testing.T) {
	g, dirModels := raizCatalogo(t)
	ruta := filepath.Join(dirModels, "mtp-x", "mtp-x.gguf")
	if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, []byte("gguf"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := g.AltaDescarga("mtp-x", Variante{Archivo: "mtp-x.gguf"}, ruta, 0, ""); err == nil {
		t.Fatal("una cabeza MTP no debería registrarse en el catálogo")
	}
}
