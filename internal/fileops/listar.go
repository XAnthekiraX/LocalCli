package fileops

import (
	"context"
	"encoding/json"
	"os"
	"sort"

	"localcli/internal/tools"
)

// ListarHandler es el `Ejecutar` de listar_carpeta.
func (o *Operaciones) ListarHandler() tools.Ejecutar {
	return func(ctx context.Context, args any, c tools.Contexto) (tools.Resultado, error) {
		peticion, ok := args.(tools.PeticionListarCarpeta)
		if !ok {
			return corregible(tools.CodBadArgs + ": argumentos inesperados")
		}
		abs, codigo := Resolver(o.Raiz, peticion.Ruta)
		if codigo != "" {
			return corregible(codigo)
		}
		if err := ValidarAcceso(ctx, o.Raiz, abs, peticion.Motivo, c); err != nil {
			return resultado(err)
		}
		entradas, err := os.ReadDir(abs)
		if err != nil {
			return corregible("no se pudo listar la carpeta: " + Canonica(o.Raiz, abs))
		}
		nombres := make([]string, 0, len(entradas))
		for _, e := range entradas {
			nombre := e.Name()
			if e.IsDir() {
				nombre += "/"
			}
			nombres = append(nombres, nombre)
		}
		sort.Strings(nombres)
		salida, _ := json.Marshal(nombres)
		return tools.Resultado{
			Salida: string(salida),
			Meta:   map[string]any{"medida": pluralEntradas(len(nombres))},
		}, nil
	}
}

func pluralEntradas(n int) string {
	if n == 1 {
		return "1 entrada"
	}
	return itoa(n) + " entradas"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var d []byte
	for n > 0 {
		d = append([]byte{byte('0' + n%10)}, d...)
		n /= 10
	}
	return string(d)
}
