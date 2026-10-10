# Contributing

Contributions to RTSP Stream Console are welcome.

## Development setup

Clone the repository:

```bash
git clone https://github.com/dataolympus/rtsp-stream-console.git
cd rtsp-stream-console
```

The easiest way to run the complete local environment is Docker Compose:

```bash
docker compose \
  --env-file deploy/compose/demo.env \
  -f deploy/compose/compose.yaml \
  -f deploy/compose/compose.local.yaml \
  --profile demo \
  up --build -d
```

Open:

```text
http://localhost:8082
```

The included synthetic RTSP source is:

```text
rtsp://mediamtx:8554/camera-1
```

## Verify your changes

Before submitting a change, run:

```bash
./scripts/check.sh
```

This verifies:

- Go tests
- Go vet
- frontend tests
- frontend production build
- Docker Compose configuration

Pull requests run the corresponding checks automatically in GitHub Actions.

## Backend

The backend lives under:

```text
backend/
```

Run its checks directly with:

```bash
cd backend

go test ./...
go vet ./...
```

## Frontend

The frontend lives under:

```text
frontend/
```

Install dependencies and run:

```bash
cd frontend

npm ci
npm test
npm run build
```

For frontend changes, include screenshots or a short recording when the visual or interaction behavior changes.

## Infrastructure

Azure Terraform lives under:

```text
deploy/terraform/azure/
```

Before submitting infrastructure changes:

```bash
cd deploy/terraform/azure

terraform fmt -check -recursive
terraform init -backend=false
terraform validate
```

Do not commit:

- Terraform state
- `terraform.tfvars`
- production environment files
- passwords or authentication headers
- private RTSP credentials

## Commits

Keep commits focused and describe the change rather than the implementation process.

Sign off commits with:

```bash
git commit -s -m "..."
```

Example:

```bash
git commit -s \
  -m "fix: release viewer slot after disconnect"
```

## Pull requests

A pull request should:

- explain the problem or change
- keep unrelated changes separate
- include tests when behavior changes
- keep CI green
- include screenshots or recordings for relevant UI changes
- update documentation when behavior or configuration changes

## Security issues

Do not publish sensitive vulnerability details in a normal public issue.

See the repository security policy for reporting guidance once available.
