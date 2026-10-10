#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_DIR="${ROOT_DIR}/deploy/compose"
PROD_ENV="${COMPOSE_DIR}/prod.env"

SITE_ADDRESS=""
USERNAME="reviewer"
ENABLE_DEMO=false
REUSE_ENV=false
BUILD_IMAGES=true
RELEASE_TAG=""

usage() {
  cat <<'EOF'
Usage:
  ./scripts/deploy.sh --site <hostname> [options]

Options:
  --site <hostname>     Public hostname, for example streams.example.com
  --username <name>     Basic Auth username (default: reviewer)
  --demo                Enable MediaMTX and the synthetic RTSP camera
  --reuse-env           Reuse an existing deploy/compose/prod.env
  --no-build            Do not build local images before starting
  --release <tag>       Pull published GHCR images instead of building locally
  -h, --help            Show this help
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --site)
      SITE_ADDRESS="${2:?missing value for --site}"
      shift 2
      ;;
    --username)
      USERNAME="${2:?missing value for --username}"
      shift 2
      ;;
    --demo)
      ENABLE_DEMO=true
      shift
      ;;
    --reuse-env)
      REUSE_ENV=true
      shift
      ;;
    --no-build)
      BUILD_IMAGES=false
      shift
      ;;
    --release)
      RELEASE_TAG="${2:?missing value for --release}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

command -v docker >/dev/null 2>&1 || {
  echo "docker is required" >&2
  exit 1
}

docker compose version >/dev/null 2>&1 || {
  echo "Docker Compose is required" >&2
  exit 1
}

if [[ -n "${RELEASE_TAG}" ]]; then
  if [[ ! "${RELEASE_TAG}" =~ ^[A-Za-z0-9][A-Za-z0-9._-]*$ ]]; then
    echo "Invalid release tag: ${RELEASE_TAG}" >&2
    exit 1
  fi

  export BACKEND_IMAGE="ghcr.io/dataolympus/rtsp-stream-console-backend:${RELEASE_TAG}"
  export FRONTEND_IMAGE="ghcr.io/dataolympus/rtsp-stream-console-frontend:${RELEASE_TAG}"

  BUILD_IMAGES=false

  echo "Release:            ${RELEASE_TAG}"
  echo "Backend image:      ${BACKEND_IMAGE}"
  echo "Frontend image:     ${FRONTEND_IMAGE}"
fi

if [[ "${REUSE_ENV}" == false ]]; then
  if [[ -z "${SITE_ADDRESS}" ]]; then
    echo "--site is required when creating a production environment" >&2
    exit 1
  fi

  read -rsp "Reviewer password: " PASSWORD
  echo

  read -rsp "Confirm password: " PASSWORD_CONFIRM
  echo

  if [[ "${PASSWORD}" != "${PASSWORD_CONFIRM}" ]]; then
    echo "Passwords do not match" >&2
    unset PASSWORD PASSWORD_CONFIRM
    exit 1
  fi

  if [[ -z "${PASSWORD}" ]]; then
    echo "Password cannot be empty" >&2
    unset PASSWORD PASSWORD_CONFIRM
    exit 1
  fi

  PASSWORD_HASH="$(
    printf '%s\n' "${PASSWORD}" |
      docker run --rm -i caddy:2-alpine \
        caddy hash-password
  )"

  unset PASSWORD PASSWORD_CONFIRM

  umask 077

  {
    printf 'SITE_ADDRESS=%s\n\n' "${SITE_ADDRESS}"
    printf 'DEMO_USERNAME=%s\n' "${USERNAME}"
    printf "DEMO_PASSWORD_HASH='%s'\n\n" "${PASSWORD_HASH}"
    printf 'RTSP_ALLOW_PRIVATE_NETWORKS=false\n'
    printf 'RTSP_ALLOWED_HOSTS=mediamtx\n\n'
    printf 'MAX_ACTIVE_STREAMS=2\n'
    printf 'MAX_VIEWERS_PER_STREAM=4\n'
    printf 'EXPENSIVE_REQUESTS_PER_MINUTE=10\n'
    printf 'TRUST_PROXY_HEADERS=true\n\n'
    printf 'BACKEND_CPUS=2.0\n'
    printf 'BACKEND_MEMORY_LIMIT=1g\n'
    printf 'BACKEND_PIDS_LIMIT=128\n'
  } > "${PROD_ENV}"

  unset PASSWORD_HASH

  chmod 600 "${PROD_ENV}"
else
  if [[ ! -f "${PROD_ENV}" ]]; then
    echo "${PROD_ENV} does not exist" >&2
    exit 1
  fi
fi

COMPOSE_ARGS=(
  --env-file "${PROD_ENV}"
  -f "${COMPOSE_DIR}/compose.yaml"
  -f "${COMPOSE_DIR}/compose.prod.yaml"
)

if [[ "${ENABLE_DEMO}" == true ]]; then
  COMPOSE_ARGS+=(--profile demo)
fi

docker compose "${COMPOSE_ARGS[@]}" config >/dev/null

if [[ -n "${RELEASE_TAG}" ]]; then
  echo
  echo "Pulling release images..."

  docker compose "${COMPOSE_ARGS[@]}" pull backend frontend
elif [[ "${BUILD_IMAGES}" == true ]]; then
  echo
  echo "Building application images..."

  COMPOSE_PARALLEL_LIMIT=1 \
    docker compose "${COMPOSE_ARGS[@]}" build
fi

docker compose "${COMPOSE_ARGS[@]}" up -d

docker compose "${COMPOSE_ARGS[@]}" ps