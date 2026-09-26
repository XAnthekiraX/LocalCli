#!/usr/bin/env bash
# scripts/verify.sh — T-B015-07: la verificación completa del backend.
#
# Fuente de verdad: ai/tasks/backend/015-task-validaciones.md (T-B015-07) y
# ai/docs/backend/05-quality/TESTING.md (estrategia de pruebas).
#
# Uso: ./scripts/verify.sh
# Sale en verde si: formateo correcto, vet limpio, build completo, tests con
# -race en verde e informe de cobertura de todo el proyecto.

set -euo pipefail
cd "$(dirname "$0")/.."

echo "── gofmt (comprobación de formato)"
sin_formato=$(gofmt -l . | grep -v '^\.localcli' || true)
if [ -n "$sin_formato" ]; then
  echo "FALLO: estos archivos no están formateados:"
  echo "$sin_formato"
  exit 1
fi

echo "── go vet"
go vet ./...

echo "── go build ./..."
go build ./...

echo "── go test ./... -race (suite completa)"
go test ./... -race -count=1

echo "── cobertura"
if go test ./... -count=1 -coverprofile=coverage.out > /dev/null; then
  go tool cover -func=coverage.out | tail -1
  echo "(informe completo: go tool cover -html=coverage.out)"
else
  echo "FALLO: la cobertura no se pudo calcular"
  exit 1
fi

echo "✓ verificación completa en verde"
