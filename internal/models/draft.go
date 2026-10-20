package models

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Una cabeza MTP (draft head) es un GGUF que solo contiene la capa de
// predicción multi-token, no un modelo completo: se carga junto al modelo
// principal con `-md` y `--spec-type draft-mtp`. LocalCli todavía no arranca
// cabezas MTP, así que registrarlas o pasarlas a `llama-server` como si fueran
// un modelo termina en un fallo del motor.

// MotivoCabezaDraft indica por qué un GGUF no es un modelo completo, o cadena
// vacía si no hay indicios de que sea una cabeza MTP. No bloquea por no poder
// leer el archivo: la ausencia de datos no es una prueba.
func MotivoCabezaDraft(ruta string) string {
	if EsCabezaDraftNombre(ruta) {
		return "es una cabeza MTP (draft head), no un modelo completo; se usa junto al modelo principal con -md y --spec-type draft-mtp"
	}
	motivo, err := motivoCabezaDraftGGUF(ruta)
	if err != nil {
		return ""
	}
	return motivo
}

// EsCabezaDraftNombre reconoce la convención de nombres de las cabezas MTP: el
// archivo empieza por `mtp-`/`mtp_` o vive en una carpeta `MTP/`. Acepta rutas
// locales y rutas de repositorio (`MTP/mtp-....gguf`).
func EsCabezaDraftNombre(ruta string) bool {
	limpia := strings.ReplaceAll(strings.TrimSpace(ruta), "\\", "/")
	if limpia == "" {
		return false
	}
	base := strings.ToLower(filepath.Base(limpia))
	if strings.HasPrefix(base, "mtp-") || strings.HasPrefix(base, "mtp_") {
		return true
	}
	return strings.EqualFold(filepath.Base(filepath.Dir(limpia)), "MTP")
}

// motivoCabezaDraftGGUF detecta una cabeza MTP por su estructura: declara capas
// `nextn`, pero solo trae tensores de la última capa. Un modelo completo, aun
// repartido en shards, no cumple estas condiciones a la vez.
func motivoCabezaDraftGGUF(ruta string) (string, error) {
	f, err := os.Open(ExpandirRuta(ruta))
	if err != nil {
		return "", err
	}
	defer f.Close()
	ficha, err := leerFichaGGUF(bufio.NewReader(f))
	if err != nil {
		return "", err
	}
	if ficha.nextn < 1 || ficha.bloque < 2 {
		return "", nil
	}
	capas := make(map[int]struct{})
	for _, nombre := range ficha.tensores {
		if capa, ok := capaDeTensor(nombre); ok {
			capas[capa] = struct{}{}
		}
	}
	if len(capas) != 1 {
		return "", nil
	}
	for capa := range capas {
		if capa != ficha.bloque-1 {
			return "", nil
		}
	}
	return fmt.Sprintf("es una cabeza MTP (draft head): declara %d capas pero solo trae tensores de la capa %d; se usa junto al modelo principal con -md y --spec-type draft-mtp", ficha.bloque, ficha.bloque-1), nil
}

// capaDeTensor extrae el índice de bloque de un nombre como `blk.48.attn_q.weight`.
func capaDeTensor(nombre string) (int, bool) {
	resto, ok := strings.CutPrefix(nombre, "blk.")
	if !ok {
		return 0, false
	}
	corte := strings.IndexByte(resto, '.')
	if corte <= 0 {
		return 0, false
	}
	capa, err := strconv.Atoi(resto[:corte])
	if err != nil || capa < 0 {
		return 0, false
	}
	return capa, true
}
