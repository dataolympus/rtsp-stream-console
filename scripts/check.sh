#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo
echo "==> Backend tests"
(
  cd "${ROOT_DIR}/backend"
  go test ./...
)

echo
echo "==> Backend vet"
(
  cd "${ROOT_DIR}/backend"
  go vet ./...
)

echo
echo "==> Frontend dependencies"
(
  cd "${ROOT_DIR}/frontend"
  npm ci
)

echo
echo "==> Frontend tests"
(
  cd "${ROOT_DIR}/frontend"
  npm test
)

echo
echo "==> Frontend production build"
(
  cd "${ROOT_DIR}/frontend"
  npm run build
)

echo
echo "==> Local Compose configuration"
docker compose \
  --env-file "${ROOT_DIR}/deploy/compose/demo.env" \
  -f "${ROOT_DIR}/deploy/compose/compose.yaml" \
  -f "${ROOT_DIR}/deploy/compose/compose.local.yaml" \
  --profile demo \
  config >/dev/null

echo
echo "All source checks passed."
