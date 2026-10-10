#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo
echo "==> Shell script syntax"

while IFS= read -r -d '' script; do
  bash -n "${script}"
done < <(
  find "${ROOT_DIR}/scripts" \
    -maxdepth 1 \
    -type f \
    -name '*.sh' \
    -print0
)

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
echo "==> Production Compose configuration"

PROD_CHECK_ENV="$(mktemp)"
trap 'rm -f "${PROD_CHECK_ENV}"' EXIT

cat > "${PROD_CHECK_ENV}" <<'EOF'
SITE_ADDRESS=streams.example.com
DEMO_USERNAME=reviewer
DEMO_PASSWORD_HASH=placeholder
BACKEND_IMAGE=ghcr.io/dataolympus/rtsp-stream-console-backend:v0.1.0
FRONTEND_IMAGE=ghcr.io/dataolympus/rtsp-stream-console-frontend:v0.1.0
EOF

docker compose \
  --env-file "${PROD_CHECK_ENV}" \
  -f "${ROOT_DIR}/deploy/compose/compose.yaml" \
  -f "${ROOT_DIR}/deploy/compose/compose.prod.yaml" \
  --profile demo \
  config >/dev/null

rm -f "${PROD_CHECK_ENV}"
trap - EXIT

echo
echo "All source checks passed."
