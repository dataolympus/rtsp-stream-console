#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TERRAFORM_DIR="${ROOT_DIR}/deploy/terraform/azure"

REPO_URL="https://github.com/dataolympus/rtsp-stream-console.git"
REMOTE_REPO_NAME="rtsp-stream-console"

GIT_REF=""
SITE_ADDRESS=""
USERNAME="reviewer"

ENABLE_DEMO=false
REUSE_ENV=false
NO_BUILD=false

usage() {
  cat <<'EOF'
Usage:
  ./scripts/deploy-azure.sh --ref <git-ref> [options]

Options:
  --ref <git-ref>       Exact Git ref to deploy, for example v0.1.0 or a commit SHA
  --site <hostname>     Public hostname, for example streams.example.com
  --username <name>     Basic Auth username (default: reviewer)
  --demo                Enable MediaMTX and the synthetic RTSP camera
  --reuse-env           Reuse the existing production environment on the VM
  --no-build            Reuse existing images instead of rebuilding them
  -h, --help            Show this help

Examples:
  ./scripts/deploy-azure.sh \
    --ref v0.1.0 \
    --site streams.example.com

  ./scripts/deploy-azure.sh \
    --ref f4b3afb \
    --reuse-env \
    --demo \
    --no-build
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --ref)
      GIT_REF="${2:?missing value for --ref}"
      shift 2
      ;;
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
      NO_BUILD=true
      shift
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

if [[ -z "${GIT_REF}" ]]; then
  echo "--ref is required" >&2
  exit 1
fi

if [[ "${REUSE_ENV}" == false && -z "${SITE_ADDRESS}" ]]; then
  echo "--site is required unless --reuse-env is used" >&2
  exit 1
fi

command -v terraform >/dev/null 2>&1 || {
  echo "terraform is required" >&2
  exit 1
}

command -v ssh >/dev/null 2>&1 || {
  echo "ssh is required" >&2
  exit 1
}

if [[ ! -d "${TERRAFORM_DIR}" ]]; then
  echo "Terraform directory not found: ${TERRAFORM_DIR}" >&2
  exit 1
fi

SSH_COMMAND="$(
  terraform \
    -chdir="${TERRAFORM_DIR}" \
    output \
    -raw ssh_command
)"

if [[ "${SSH_COMMAND}" != ssh\ * ]]; then
  echo "Unexpected Terraform ssh_command output" >&2
  exit 1
fi

SSH_TARGET="${SSH_COMMAND#ssh }"

if [[ -z "${SSH_TARGET}" || "${SSH_TARGET}" == *" "* ]]; then
  echo "Unable to determine SSH target from Terraform output" >&2
  exit 1
fi

echo "Deployment target: ${SSH_TARGET}"
echo "Git ref:           ${GIT_REF}"

if [[ "${REUSE_ENV}" == true ]]; then
  echo "Environment:       reuse existing production environment"
else
  echo "Site:              ${SITE_ADDRESS}"
  echo "Username:          ${USERNAME}"
fi

if [[ "${ENABLE_DEMO}" == true ]]; then
  echo "Demo profile:      enabled"
else
  echo "Demo profile:      disabled"
fi

echo
echo "Preparing repository on Azure VM..."

ssh "${SSH_TARGET}" \
  bash -s -- \
  "${REPO_URL}" \
  "${REMOTE_REPO_NAME}" \
  "${GIT_REF}" <<'REMOTE'
set -euo pipefail

REPO_URL="$1"
REMOTE_REPO_NAME="$2"
GIT_REF="$3"

APP_DIR="${HOME}/${REMOTE_REPO_NAME}"

if [[ ! -d "${APP_DIR}/.git" ]]; then
  echo "Cloning repository..."
  git clone "${REPO_URL}" "${APP_DIR}"
fi

cd "${APP_DIR}"

if [[ -n "$(git status --porcelain --untracked-files=normal)" ]]; then
  echo "Remote repository has uncommitted changes." >&2
  echo "Refusing to change Git refs." >&2
  git status --short >&2
  exit 1
fi

echo "Fetching repository..."
git fetch --prune origin
git fetch --tags --force origin

if ! COMMIT="$(git rev-parse --verify "${GIT_REF}^{commit}" 2>/dev/null)"; then
  echo "Git ref does not resolve to a commit: ${GIT_REF}" >&2
  exit 1
fi

echo "Checking out ${GIT_REF} (${COMMIT})..."
git checkout --detach "${COMMIT}"

echo
echo "Remote revision:"
git log -1 --oneline
REMOTE

DEPLOY_COMMAND="cd \"\$HOME/${REMOTE_REPO_NAME}\" && ./scripts/deploy.sh"

if [[ "${REUSE_ENV}" == true ]]; then
  DEPLOY_COMMAND+=" --reuse-env"
else
  printf -v SITE_Q '%q' "${SITE_ADDRESS}"
  printf -v USERNAME_Q '%q' "${USERNAME}"

  DEPLOY_COMMAND+=" --site ${SITE_Q}"
  DEPLOY_COMMAND+=" --username ${USERNAME_Q}"
fi

if [[ "${ENABLE_DEMO}" == true ]]; then
  DEPLOY_COMMAND+=" --demo"
fi

if [[ "${NO_BUILD}" == true ]]; then
  DEPLOY_COMMAND+=" --no-build"
fi

echo
echo "Running application deployment..."

ssh -t "${SSH_TARGET}" "${DEPLOY_COMMAND}"

echo
echo "Azure deployment completed."