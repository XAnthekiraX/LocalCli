package agent

import (
	"strings"
	"testing"
)

func recoger(fragmentos *[]string) func(Delta) error {
	return func(d Delta) error {
		if !d.Razonamiento {
			*fragmentos = append(*fragmentos, d.Texto)
		}
		return nil
	}
}

func TestUnTextoQueNoEsLlamadaSeTransmiteEnVivo(t *testing.T) {
	var vistos []string
	r := nuevoRetenedor(recoger(&vistos))

	if err := r.Fragmento(Delta{Texto: "Voy a mirar"}); err != nil {
		t.Fatal(err)
	}
	if len(vistos) != 1 {
		t.Fatalf("la prosa debería salir en el acto: %v", vistos)
	}
	if err := r.Fragmento(Delta{Texto: " el archivo"}); err != nil {
		t.Fatal(err)
	}
	if strings.Join(vistos, "") != "Voy a mirar el archivo" {
		t.Fatalf("texto = %q", strings.Join(vistos, ""))
	}
}

func TestUnaLlamadaSeRetieneYSeDescarta(t *testing.T) {
	var vistos []string
	r := nuevoRetenedor(recoger(&vistos))

	if err := r.Fragmento(Delta{Texto: "```json\n{\"name\":"}); err != nil {
		t.Fatal(err)
	}
	if len(vistos) != 0 {
		t.Fatalf("una posible llamada no debería pintarse: %v", vistos)
	}
	r.Descartar()
	if err := r.Fragmento(Delta{Texto: `"x"}`}); err != nil {
		t.Fatal(err)
	}
	if len(vistos) != 0 {
		t.Fatalf("tras descartar, el resto de la llamada tampoco sale: %v", vistos)
	}
}

func TestUnJsonQueNoEsLlamadaSeVuelcaAlCerrarLaRonda(t *testing.T) {
	var vistos []string
	r := nuevoRetenedor(recoger(&vistos))

	if err := r.Fragmento(Delta{Texto: `{"resultado":42}`}); err != nil {
		t.Fatal(err)
	}
	if len(vistos) != 0 {
		t.Fatalf("mientras podría ser una llamada no se pinta: %v", vistos)
	}
	if err := r.Volcar(); err != nil {
		t.Fatal(err)
	}
	if strings.Join(vistos, "") != `{"resultado":42}` {
		t.Fatalf("texto = %q", strings.Join(vistos, ""))
	}
}

func TestElRazonamientoNoSeRetiene(t *testing.T) {
	var vistos []string
	r := nuevoRetenedor(func(d Delta) error {
		if d.Razonamiento {
			vistos = append(vistos, "R:"+d.Texto)
		} else {
			vistos = append(vistos, "T:"+d.Texto)
		}
		return nil
	})

	if err := r.Fragmento(Delta{Texto: "pienso", Razonamiento: true}); err != nil {
		t.Fatal(err)
	}
	if len(vistos) != 1 || vistos[0] != "R:pienso" {
		t.Fatalf("el razonamiento debería pasar en el acto: %v", vistos)
	}
}

func TestRetenedorSinCallbackNoHaceNada(t *testing.T) {
	r := nuevoRetenedor(nil)
	if err := r.Fragmento(Delta{Texto: "{..."}); err != nil {
		t.Fatal(err)
	}
	if err := r.Volcar(); err != nil {
		t.Fatal(err)
	}
}

func TestUnTextoQueCreceDemasiadoDejaDeRetenerse(t *testing.T) {
	var vistos []string
	r := nuevoRetenedor(recoger(&vistos))

	if err := r.Fragmento(Delta{Texto: "{" + strings.Repeat("a", maxRetenido)}); err != nil {
		t.Fatal(err)
	}
	if len(vistos) == 0 {
		t.Fatal("al superar el tope, el texto debería volcarse")
	}
	if err := r.Fragmento(Delta{Texto: "resto"}); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(strings.Join(vistos, ""), "resto") {
		t.Fatalf("tras volcar debería seguir en vivo: %q", strings.Join(vistos, ""))
	}
}

func TestUnJsonLargoSinNombreDeLlamadaSeVuelca(t *testing.T) {
	var vistos []string
	r := nuevoRetenedor(recoger(&vistos))

	texto := `{"datos":"` + strings.Repeat("a", umbralNombre) + `"}`
	if err := r.Fragmento(Delta{Texto: texto}); err != nil {
		t.Fatal(err)
	}
	if len(vistos) == 0 {
		t.Fatal("un JSON que no nombra una herramienta no debería quedar retenido")
	}
	if strings.Join(vistos, "") != texto {
		t.Fatalf("debería salir el JSON tal cual: %q", strings.Join(vistos, ""))
	}
}
