package arranque

import (
	"embed"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"

	"localcli/internal/store"
)

//go:embed datos/agents/*/* datos/flows/*
var datosFS embed.FS

// Init realiza la inicialización global: crea la raíz y siembra lo que falte.
func Init() error {
	raiz, err := RaizMaquina()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(raiz, 0o755); err != nil {
		return err
	}

	// Crear subdirectorios
	if err := os.MkdirAll(filepath.Join(raiz, "agents"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(raiz, "flows"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(raiz, "models"), 0o755); err != nil {
		return err
	}

	// Sembrar agentes base solo si faltan
	if err := sembrarAgentes(raiz); err != nil {
		return err
	}

	// Sembrar flujos base solo si faltan
	if err := sembrarFlujos(raiz); err != nil {
		return err
	}

	// Sembrar archivos de configuración solo si faltan
	if err := sembrarConfig(raiz); err != nil {
		return err
	}

	// Crear state.db global si no existe
	rutaDB, err := store.RutaGlobal()
	if err != nil {
		return err
	}
	if _, err := os.Stat(rutaDB); os.IsNotExist(err) {
		db, err := store.CrearDB(rutaDB)
		if err != nil {
			return err
		}
		db.Close()
	} else if err == nil {
		// Si ya existe, abrirlo para asegurar que esté bien (pero no sobrescribir)
		db, err := store.AbrirDBDesde(rutaDB)
		if err != nil {
			return err
		}
		db.Close()
	} else {
		return err
	}

	return nil
}

func sembrarAgentes(raiz string) error {
	entries, err := fs.ReadDir(datosFS, "datos/agents")
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		agente := entry.Name()
		agenteDir := filepath.Join(raiz, "agents", agente)
		if err := os.MkdirAll(agenteDir, 0o755); err != nil {
			return err
		}
		agentFiles, err := fs.ReadDir(datosFS, filepath.Join("datos/agents", agente))
		if err != nil {
			return err
		}
		for _, af := range agentFiles {
			if af.IsDir() {
				continue
			}
			dest := filepath.Join(agenteDir, af.Name())
			if _, err := os.Stat(dest); err == nil {
				// Ya existe, no sobrescribir
				continue
			}
			data, err := datosFS.ReadFile(filepath.Join("datos/agents", agente, af.Name()))
			if err != nil {
				return err
			}
			if err := os.WriteFile(dest, data, 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}

func sembrarFlujos(raiz string) error {
	flowsDir := filepath.Join(raiz, "flows")
	if err := os.MkdirAll(flowsDir, 0o755); err != nil {
		return err
	}
	entries, err := fs.ReadDir(datosFS, "datos/flows")
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		dest := filepath.Join(flowsDir, entry.Name())
		if _, err := os.Stat(dest); err == nil {
			// Ya existe, no sobrescribir
			continue
		}
		data, err := datosFS.ReadFile(filepath.Join("datos/flows", entry.Name()))
		if err != nil {
			return err
		}
		if err := os.WriteFile(dest, data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func sembrarConfig(raiz string) error {
	// config.json
	configPath := filepath.Join(raiz, "config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		def := map[string]interface{}{
			"ultimo_agente":    "build",
			"historial_tokens": 4096,
		}
		data, err := json.MarshalIndent(def, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(configPath, data, 0o644); err != nil {
			return err
		}
	}

	// keys.json
	keysPath := filepath.Join(raiz, "keys.json")
	if _, err := os.Stat(keysPath); os.IsNotExist(err) {
		def := map[string]interface{}{}
		data, err := json.MarshalIndent(def, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(keysPath, data, 0o644); err != nil {
			return err
		}
	}

	// localcli.json
	localcliPath := filepath.Join(raiz, "localcli.json")
	if _, err := os.Stat(localcliPath); os.IsNotExist(err) {
		def := map[string]string{
			"proveedor": "llama.cpp",
			"api":       "http://localhost:8080",
		}
		data, err := json.MarshalIndent(def, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(localcliPath, data, 0o644); err != nil {
			return err
		}
	}

	// models.json
	modelsPath := filepath.Join(raiz, "models.json")
	if _, err := os.Stat(modelsPath); os.IsNotExist(err) {
		def := map[string]interface{}{
			"modelos":      []interface{}{},
			"seleccionado": nil,
		}
		data, err := json.MarshalIndent(def, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(modelsPath, data, 0o644); err != nil {
			return err
		}
	}

	return nil
}
