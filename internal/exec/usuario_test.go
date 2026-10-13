package exec

import (
	"context"
	"strings"
	"testing"

	"localcli/internal/tools"
)

func TestHerramientaDeUsuarioSinShellYConAprobacion(t *testing.T) {
	u := tools.HerramientaUsuario{Nombre: "saluda", Descripcion: "saluda", Equipo: []string{"echo", "a;b"}}
	e := &Ejecutor{Proyecto: t.TempDir()}
	handler := e.HandlerUsuario(u)

	salida, err := handler(context.Background(), map[string]any{"x": 1}, aprobador())
	if err != nil {
		t.Fatalf("no debería fallar: %v", err)
	}
	if !strings.Contains(salida.Salida, "a;b") {
		t.Fatalf("un argumento con `;` es un argumento literal: %q", salida.Salida)
	}

	declinador := tools.Contexto{Ask: func(context.Context, tools.Solicitud) (tools.Decision, error) {
		return tools.Declinar, nil
	}}
	res, err := handler(context.Background(), map[string]any{}, declinador)
	if err != nil {
		t.Fatalf("declinar no es error de Go: %v", err)
	}
	if !strings.Contains(res.Error, tools.CodApprovalDeclined) {
		t.Fatalf("una herramienta del usuario siempre pide aprobación: %q", res.Error)
	}
}
