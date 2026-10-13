package models

import "path/filepath"

// NombreDe deriva el nombre de un modelo a partir de la carpeta que contiene su
// GGUF: `.../models/qwen3-14b/qwen3-14b-q4_k_m.gguf` → `qwen3-14b`.
func NombreDe(ruta string) string {
	return filepath.Base(filepath.Dir(ruta))
}
