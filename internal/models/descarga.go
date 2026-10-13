package models

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Descargador baja variantes GGUF al directorio global de modelos.
type Descargador struct {
	hf       *HF
	raiz     string
	progreso func(hecho, total int64)
}

// NuevoDescargador devuelve un descargador que guarda los modelos bajo `raiz`.
func NuevoDescargador(hf *HF, raiz string) *Descargador {
	return &Descargador{hf: hf, raiz: raiz}
}

// ConProgreso fija la función que recibe el avance (bytes hechos y total).
func (d *Descargador) ConProgreso(f func(hecho, total int64)) *Descargador {
	d.progreso = f
	return d
}

// Descargar baja la variante al subdirectorio `modelo` y devuelve la ruta final.
// Si ya está completa no descarga; si hay un `.part`, reanuda desde su tamaño.
// Escribe en un `.part` y solo lo renombra cuando cuadra, así que un fallo de
// red no toca los modelos existentes.
func (d *Descargador) Descargar(ctx context.Context, v Variante, modelo string) (string, error) {
	if strings.TrimSpace(modelo) == "" {
		return "", errorf("falta el nombre del modelo")
	}
	destDir := filepath.Join(d.raiz, modelo)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", errorf("no se pudo crear %s: %w", destDir, err)
	}
	destino := filepath.Join(destDir, v.Archivo)
	parcial := destino + ".part"

	if completo(destino, v.Tamano) {
		return destino, nil
	}

	offset := int64(0)
	if info, err := os.Stat(parcial); err == nil {
		offset = info.Size()
	}

	resp, err := d.peticion(ctx, v, offset)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		offset = 0 // el servidor ignoró el rango: se reempieza
	case http.StatusPartialContent:
	default:
		return "", errorf("descarga rechazada: %s: %s", resp.Status, trozoDeRespuesta(resp.Body))
	}

	if err := d.escribir(ctx, parcial, resp.Body, offset, v.Tamano); err != nil {
		return "", err // el `.part` queda para reanudar
	}
	if err := verificarDescarga(parcial, v); err != nil {
		return "", err
	}
	if err := os.Rename(parcial, destino); err != nil {
		return "", errorf("no se pudo colocar %s: %w", destino, err)
	}
	return destino, nil
}

// DescargarURL baja una variante identificada por una URL directa: completa su
// tamaño (y su hash, si es de Hugging Face) antes de empezar y delega en
// `Descargar`.
func (d *Descargador) DescargarURL(ctx context.Context, v Variante, modelo string) (string, error) {
	return d.Descargar(ctx, d.completar(ctx, v), modelo)
}

// completar rellena el tamaño y el hash de una variante cuando se pueden
// resolver: por la API de Hugging Face si es un repositorio, o por una petición
// HEAD al servidor. Sin poder resolverlos la descarga sigue: el tamaño solo
// aporta la barra de progreso y la comprobación de integridad.
func (d *Descargador) completar(ctx context.Context, v Variante) Variante {
	if d.hf == nil {
		return v
	}
	if v.Repo != "" {
		if completa, err := d.hf.Seleccionar(ctx, v.Repo, v.Archivo); err == nil {
			v.Tamano, v.SHA256 = completa.Tamano, completa.SHA256
			return v
		}
	}
	if v.Tamano <= 0 {
		v.Tamano = d.hf.TamanoURL(ctx, v.URL)
	}
	return v
}

// peticion pide el resto del archivo a partir de `offset` cuando hay algo que
// reanudar.
func (d *Descargador) peticion(ctx context.Context, v Variante, offset int64) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.hf.URLDescarga(v), nil)
	if err != nil {
		return nil, errorf("no se pudo preparar la descarga: %w", err)
	}
	if offset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}
	resp, err := d.hf.http.Do(req)
	if err != nil {
		return nil, errorf("no se pudo descargar %s: %w", v.Archivo, err)
	}
	return resp, nil
}

// escribir vuelca el cuerpo al `.part` desde `offset`, informando del avance.
func (d *Descargador) escribir(ctx context.Context, parcial string, cuerpo io.Reader, offset, total int64) error {
	banderas := os.O_CREATE | os.O_WRONLY
	if offset == 0 {
		banderas |= os.O_TRUNC
	}
	f, err := os.OpenFile(parcial, banderas, 0o644)
	if err != nil {
		return errorf("no se pudo abrir %s: %w", parcial, err)
	}
	defer f.Close()
	if offset > 0 {
		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			return errorf("no se pudo reanudar %s: %w", parcial, err)
		}
	}

	buf := make([]byte, 256*1024)
	hecho := offset
	for {
		if err := ctx.Err(); err != nil {
			return errorf("descarga cancelada: %w", err)
		}
		n, err := cuerpo.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				return errorf("no se pudo escribir %s: %w", parcial, werr)
			}
			hecho += int64(n)
			if d.progreso != nil {
				d.progreso(hecho, total)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return errorf("la descarga se interrumpió: %w", err)
		}
	}
	if err := f.Sync(); err != nil {
		return errorf("no se pudo volcar %s a disco: %w", parcial, err)
	}
	return nil
}

// completo indica si el archivo final existe con el tamaño esperado. Sin tamaño
// conocido, un archivo presente se da por bueno.
func completo(destino string, tamano int64) bool {
	info, err := os.Stat(destino)
	if err != nil {
		return false
	}
	if tamano <= 0 {
		return true
	}
	return info.Size() == tamano
}

// verificarDescarga valida tamaño y hash. Un parcial incompleto se conserva para
// reanudar; uno corrupto se descarta.
func verificarDescarga(parcial string, v Variante) error {
	info, err := os.Stat(parcial)
	if err != nil {
		return errorf("no está el archivo descargado: %w", err)
	}
	if v.Tamano > 0 && info.Size() != v.Tamano {
		if info.Size() < v.Tamano {
			return errorf("la descarga quedó incompleta: %d de %d bytes", info.Size(), v.Tamano)
		}
		_ = os.Remove(parcial)
		return errorf("la descarga quedó más grande de lo esperado: %d de %d bytes", info.Size(), v.Tamano)
	}
	if v.SHA256 != "" {
		suma, err := sha256De(parcial)
		if err != nil {
			return err
		}
		if !strings.EqualFold(suma, v.SHA256) {
			_ = os.Remove(parcial)
			return errorf("el hash no coincide: esperado %s, obtenido %s", v.SHA256, suma)
		}
	}
	return nil
}

// sha256De calcula el sha256 de un archivo por bloques.
func sha256De(ruta string) (string, error) {
	f, err := os.Open(ruta)
	if err != nil {
		return "", errorf("no se pudo leer %s: %w", ruta, err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", errorf("no se pudo leer %s: %w", ruta, err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
