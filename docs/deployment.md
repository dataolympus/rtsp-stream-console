# Deployment

RTSP Stream Console supports two primary deployment paths:

1. local Docker Compose for development and evaluation
2. Azure VM deployment using Terraform, Docker Compose, Caddy, and GHCR images

The application and infrastructure layers are intentionally separated.

```text
Infrastructure
    Terraform
        |
        v
VM + network + public IP + Docker

Application
    deploy-azure.sh
        |
        v
deploy.sh
        |
        v
Docker Compose
        |
        v
GHCR images
```

## Local quick start

### Requirements

Install:

- Docker
- Docker Compose
- Git

Clone the repository:

```bash
git clone https://github.com/dataolympus/rtsp-stream-console.git
cd rtsp-stream-console
```

Start the local demo stack:

```bash
docker compose \
  --env-file deploy/compose/demo.env \
  -f deploy/compose/compose.yaml \
  -f deploy/compose/compose.local.yaml \
  --profile demo \
  up --build -d
```

The demo profile starts:

```text
backend
frontend
mediamtx
test-camera
```

Open:

```text
http://localhost:8082
```

Add the synthetic stream:

```text
Name: Camera 1
URL:  rtsp://mediamtx:8554/camera-1
```

Then click **Start**.

### Check local services

```bash
docker compose \
  --env-file deploy/compose/demo.env \
  -f deploy/compose/compose.yaml \
  -f deploy/compose/compose.local.yaml \
  --profile demo \
  ps
```

Expected services:

```text
backend
frontend
mediamtx
test-camera
```

### Stop local environment

```bash
docker compose \
  --env-file deploy/compose/demo.env \
  -f deploy/compose/compose.yaml \
  -f deploy/compose/compose.local.yaml \
  --profile demo \
  down
```

## Compose layout

The Compose configuration is split by environment.

```text
deploy/compose/
├── compose.yaml
├── compose.local.yaml
├── compose.prod.yaml
├── demo.env
└── prod.env
```

`compose.yaml` defines the shared application services.

`compose.local.yaml` adds local host exposure.

`compose.prod.yaml` adds the production Caddy edge, TLS ports, persistent Caddy volumes, and production runtime limits.

`prod.env` contains deployment-specific configuration and is intentionally ignored by Git.

## Demo profile

The demo profile is optional.

It adds:

```text
MediaMTX
synthetic FFmpeg test camera
```

The synthetic camera publishes:

```text
rtsp://mediamtx:8554/camera-1
```

Use the demo profile for:

```text
development
evaluation
hosted product demonstrations
integration testing
```

A normal installation using real cameras does not require it.

## Azure deployment

The repository includes Terraform configuration under:

```text
deploy/terraform/azure/
```

The current Azure deployment model provisions one Linux VM with Docker and runs the application through Docker Compose.

## Azure prerequisites

Install locally:

- Terraform
- Azure CLI
- Git
- OpenSSH

Authenticate to Azure:

```bash
az login
```

Confirm the active subscription:

```bash
az account show
```

If necessary:

```bash
az account set \
  --subscription "<subscription-id-or-name>"
```

## Configure Terraform

Move to:

```bash
cd deploy/terraform/azure
```

Review:

```text
terraform.tfvars.example
```

Create a local configuration:

```bash
cp terraform.tfvars.example terraform.tfvars
```

Edit values such as:

```text
Azure region
VM size
administrator source CIDR
SSH public key path
resource tags
```

`terraform.tfvars` is intentionally excluded from Git.

The SSH source should normally be a narrow administrator CIDR such as:

```text
203.0.113.10/32
```

rather than allowing SSH from the entire Internet.

## Provision Azure infrastructure

Initialize Terraform:

```bash
terraform init
```

Format and validate:

```bash
terraform fmt
terraform validate
```

Review the plan:

```bash
terraform plan
```

Apply only after reviewing the resources:

```bash
terraform apply
```

The current configuration provisions resources including:

```text
Resource Group
Virtual Network
Subnet
Network Security Group
Static Standard Public IP
Network Interface
Linux Virtual Machine
Docker bootstrap
```

## Terraform outputs

Useful outputs include:

```bash
terraform output
```

Individual outputs can be retrieved with:

```bash
terraform output -raw public_ip_address
terraform output -raw ssh_command
```

Example:

```text
public_ip_address = "203.0.113.20"
ssh_command       = "ssh azureuser@203.0.113.20"
```

## Cloud-init verification

After provisioning, connect to the VM:

```bash
ssh azureuser@<public-ip>
```

Wait for bootstrap completion:

```bash
cloud-init status --wait
```

Verify Docker:

```bash
docker --version
docker compose version
systemctl is-active docker
```

Expected:

```text
Docker installed
Docker Compose installed
docker service active
```

## Production application deployment

Use the Azure deployment wrapper from your local machine.

The wrapper:

```text
reads the VM target from Terraform outputs
connects over SSH
clones the repository if necessary
fetches the requested Git revision
checks out the exact revision
invokes the target-side deployment helper
```

## Development image deployment

For current development images:

```bash
./scripts/deploy-azure.sh \
  --ref <git-commit> \
  --release dev \
  --site streams.example.com
```

Enable the synthetic camera:

```bash
./scripts/deploy-azure.sh \
  --ref <git-commit> \
  --release dev \
  --site streams.example.com \
  --demo
```

The deployment helper will prompt for the production Basic Auth password.

There is no default production password.

## Existing environment reuse

To keep the existing production environment and credentials:

```bash
./scripts/deploy-azure.sh \
  --ref <git-commit> \
  --release dev \
  --reuse-env \
  --demo
```

This is useful for application updates where:

```text
domain stays the same
credentials stay the same
runtime configuration stays the same
```

## Exact Git revisions

Deployment hosts use detached Git checkouts.

For example:

```text
HEAD detached at e683546
```

This is intentional.

The VM runs the exact revision requested by the deployment command rather than automatically tracking `main`.

Production releases should use explicit version tags.

## GHCR images

Application containers are published to GitHub Container Registry.

```text
ghcr.io/dataolympus/rtsp-stream-console-backend
ghcr.io/dataolympus/rtsp-stream-console-frontend
```

The packages are public.

Deployment hosts do not need a GitHub token to pull public releases.

Test anonymous access with:

```bash
docker logout ghcr.io 2>/dev/null || true

docker pull \
  ghcr.io/dataolympus/rtsp-stream-console-backend:dev

docker pull \
  ghcr.io/dataolympus/rtsp-stream-console-frontend:dev
```

## Release mode

Release mode avoids compiling application source on the deployment VM.

```text
GitHub
   |
   | tag / workflow
   v
GitHub Actions
   |
   v
GHCR
   |
   v
Azure VM
   |
   | docker compose pull
   v
application
```

The deployment command is:

```bash
./scripts/deploy-azure.sh \
  --ref <release-ref> \
  --release <release-tag> \
  --site streams.example.com
```

For example, after the first tagged release:

```text
--ref v0.1.0
--release v0.1.0
```

The Git revision and container version should normally match.

Do not use a moving development tag for long-lived production deployments.

## Source-build mode

The target-side helper can also build the application from source.

This is primarily useful for development.

Without `--release`, `deploy.sh` uses the local Docker build definitions.

Release mode is preferred for normal hosted deployments because it:

```text
reduces VM CPU and memory usage during deployment
shortens deployment time
produces reproducible application artifacts
keeps builds in CI instead of production
```

## Production environment

Production settings live in:

```text
deploy/compose/prod.env
```

This file is not committed.

Typical content resembles:

```dotenv
SITE_ADDRESS=streams.example.com

DEMO_USERNAME=reviewer
DEMO_PASSWORD_HASH='$2a$...'

RTSP_ALLOW_PRIVATE_NETWORKS=false
RTSP_ALLOWED_HOSTS=

MAX_ACTIVE_STREAMS=4
MAX_VIEWERS_PER_STREAM=8
EXPENSIVE_REQUESTS_PER_MINUTE=10
TRUST_PROXY_HEADERS=true

BACKEND_CPUS=2.0
BACKEND_MEMORY_LIMIT=1g
BACKEND_PIDS_LIMIT=128
```

The deployment helper generates the password hash interactively.

## Manual password generation

If manual configuration is required:

```bash
read -rsp "Reviewer password: " DEMO_PASSWORD
echo

DEMO_PASSWORD_HASH="$(
  printf '%s\n' "$DEMO_PASSWORD" |
    docker run --rm -i caddy:2-alpine \
      caddy hash-password
)"

unset DEMO_PASSWORD
```

Store only:

```text
DEMO_PASSWORD_HASH
```

not the plaintext password.

Restrict the file:

```bash
chmod 600 deploy/compose/prod.env
```

## DNS

Production HTTPS requires a hostname that resolves to the Azure static public IP.

After Terraform apply:

```bash
terraform output -raw public_ip_address
```

Create an A record with your DNS provider.

Example:

```text
Type:   A
Name:   streams
Value:  <Azure static public IP>
TTL:    provider-appropriate value
```

Verify propagation:

```bash
dig +short streams.example.com A
```

The returned address should match the Terraform public IP output.

## HTTPS

Caddy manages HTTPS automatically when:

```text
DNS resolves to the VM
port 80 is reachable
port 443 is reachable
SITE_ADDRESS contains the hostname
```

Set:

```dotenv
SITE_ADDRESS=streams.example.com
```

Then redeploy the frontend.

Monitor certificate issuance:

```bash
docker compose \
  --env-file deploy/compose/prod.env \
  -f deploy/compose/compose.yaml \
  -f deploy/compose/compose.prod.yaml \
  logs -f frontend
```

Successful issuance includes output similar to:

```text
certificate obtained successfully
```

## Verify TLS

From another machine:

```bash
curl -I http://streams.example.com/
```

Expected:

```text
308 Permanent Redirect
```

Then:

```bash
curl -I https://streams.example.com/
```

For an unauthenticated protected deployment:

```text
401 Unauthorized
```

Inspect the certificate:

```bash
openssl s_client \
  -connect streams.example.com:443 \
  -servername streams.example.com \
  </dev/null 2>/dev/null |
openssl x509 \
  -noout \
  -subject \
  -issuer \
  -dates
```

Do not use insecure certificate bypass flags for acceptance testing.

## Health checks

Production exposes:

```text
/healthz
/readyz
```

Test:

```bash
curl -i https://streams.example.com/healthz
curl -i https://streams.example.com/readyz
```

Expected:

```text
200 OK
```

These endpoints intentionally remain outside Basic Auth so infrastructure can probe them.

## Production Compose status

On the VM:

```bash
docker compose \
  --env-file deploy/compose/prod.env \
  -f deploy/compose/compose.yaml \
  -f deploy/compose/compose.prod.yaml \
  --profile demo \
  ps
```

A hosted demo should normally show:

```text
backend
frontend
mediamtx
test-camera
```

A non-demo production deployment normally needs only:

```text
backend
frontend
```

## Verify deployed image provenance

Check the running image references:

```bash
docker inspect compose-backend-1 \
  --format '{{.Config.Image}}'

docker inspect compose-frontend-1 \
  --format '{{.Config.Image}}'
```

A release deployment should show versioned GHCR references.

Example:

```text
ghcr.io/dataolympus/rtsp-stream-console-backend:v0.1.0
ghcr.io/dataolympus/rtsp-stream-console-frontend:v0.1.0
```

## Updating a deployment

For a new release:

```bash
./scripts/deploy-azure.sh \
  --ref <new-release> \
  --release <new-release> \
  --reuse-env
```

For the hosted demo:

```bash
./scripts/deploy-azure.sh \
  --ref <new-release> \
  --release <new-release> \
  --reuse-env \
  --demo
```

The existing production environment is preserved.

## Backend restart behavior

The current stream registry is in memory.

Replacing or restarting the backend clears registered stream records.

This means deployment updates may require streams to be registered again.

The RTSP sources themselves are not modified.

## Rollback

Because both Git and containers are versioned, rollback should deploy the earlier version explicitly.

Example:

```bash
./scripts/deploy-azure.sh \
  --ref v0.1.0 \
  --release v0.1.0 \
  --reuse-env
```

Avoid rollback strategies that depend on whatever image currently happens to be tagged `dev`.

Explicit versions make rollback deterministic.

## Network exposure

The Azure Network Security Group permits:

```text
22/tcp   restricted administrator SSH
80/tcp   public HTTP
443/tcp  public HTTPS
```

The application backend and demo RTSP server are internal.

Do not expose:

```text
8080
8554
```

directly to the Internet.

You can verify this externally:

```bash
curl \
  --connect-timeout 3 \
  http://<public-ip>:8080/healthz
```

and:

```bash
nc -vz -w 3 <public-ip> 8554
```

Both should fail or time out in the standard deployment.

## Stopping compute without destroying infrastructure

For temporary demo environments, the VM can be deallocated:

```bash
az vm deallocate \
  --resource-group rg-rtsp-stream-console-demo \
  --name vm-rtsp-stream-console-demo
```

This stops VM compute while keeping the infrastructure resources.

The Standard static Public IP remains allocated, so the DNS A record can remain unchanged.

The website will be unavailable while the VM is deallocated.

## Restart the VM

```bash
az vm start \
  --resource-group rg-rtsp-stream-console-demo \
  --name vm-rtsp-stream-console-demo
```

Then verify:

```bash
curl -i https://streams.example.com/healthz
```

Containers configured with:

```text
restart: unless-stopped
```

should normally return automatically after Docker starts.

## Static IP lifetime

The Azure public IP uses:

```text
Standard SKU
Static allocation
```

The IP remains stable while the Public IP resource exists.

Deallocating the VM does not remove it.

However:

```bash
terraform destroy
```

deletes the Public IP resource.

A later deployment may receive a different address.

DNS must then be updated.

## Terraform drift

Periodically verify infrastructure state:

```bash
cd deploy/terraform/azure
terraform plan
```

A stable environment should report:

```text
No changes. Your infrastructure matches the configuration.
```

Unexpected changes should be reviewed before running `terraform apply`.

## Cost controls

The repository does not currently provision Azure billing budgets through Terraform.

Billing policy is intentionally treated as an operator/account concern rather than an application dependency.

For hosted or temporary environments, configure an Azure Cost Management budget appropriate to the subscription billing currency.

A useful pattern is:

```text
monthly budget
50% actual-cost warning
100% actual-cost warning
```

Azure budgets are alerts, not hard spending caps.

Cost data can also be delayed, so they should not be treated as real-time shutdown mechanisms.

For temporary deployments, VM deallocation is the primary manual method for stopping compute charges while retaining the deployment address.

## Destroying the Azure environment

When the deployment is no longer required:

```bash
cd deploy/terraform/azure

terraform plan -destroy
terraform destroy
```

Review the destroy plan carefully.

This removes Terraform-managed infrastructure, including the VM and static public IP.

Application state stored only on that VM should be considered ephemeral unless backed up separately.

## Troubleshooting

### GHCR returns unauthorized

For public packages, first remove stale local credentials:

```bash
docker logout ghcr.io 2>/dev/null || true
```

Then retry:

```bash
docker pull \
  ghcr.io/dataolympus/rtsp-stream-console-backend:<tag>
```

A public release should not require `docker login`.

### HTTPS certificate is not issued

Check:

```text
DNS resolves to the correct public IP
port 80 is reachable
port 443 is reachable
SITE_ADDRESS matches the DNS hostname
Caddy is running on the deployment VM
```

Then inspect:

```bash
docker compose \
  --env-file deploy/compose/prod.env \
  -f deploy/compose/compose.yaml \
  -f deploy/compose/compose.prod.yaml \
  logs frontend
```

### Site returns 401

For a protected deployment, this is expected when credentials are not supplied.

A browser should show the Basic Auth login prompt.

### Backend is healthy but video does not appear

Check:

```text
stream state
FFmpeg process
RTSP source availability
WebSocket connection
browser developer console
backend logs
```

On the VM:

```bash
docker exec compose-backend-1 ps
```

A live stream should have a corresponding FFmpeg child process.

### Demo source unavailable

Check:

```bash
docker compose \
  --env-file deploy/compose/prod.env \
  -f deploy/compose/compose.yaml \
  -f deploy/compose/compose.prod.yaml \
  --profile demo \
  ps
```

Both:

```text
mediamtx
test-camera
```

must be running.

## Deployment principles

The deployment architecture follows several rules:

1. Terraform owns infrastructure, not application secrets.
2. Production credentials are never committed to Git.
3. Deployments use exact Git refs.
4. Production images should use explicit release tags.
5. Build work belongs in CI rather than on small production VMs.
6. Public edge traffic terminates through HTTPS.
7. Internal backend and RTSP ports remain private.
8. Demo infrastructure remains optional.
9. Temporary cloud environments should have explicit cost controls.
10. Rollback should be version-based and reproducible.
