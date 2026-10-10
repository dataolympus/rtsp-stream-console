# Security

RTSP Stream Console accepts network locations supplied by users and launches media-processing workloads on their behalf. That creates security concerns that do not exist in a static web application.

The current security model focuses on reducing risk for a small self-hosted or protected public deployment while keeping the architecture understandable and operationally simple.

This document describes the current trust boundaries, defensive controls, configuration assumptions, and known limitations.

## Security goals

The current implementation is designed to:

- prevent unrestricted server-side connections to arbitrary network destinations
- prevent RTSP credentials from being exposed through the current stream API
- avoid shell interpretation when launching FFmpeg
- bound expensive stream-processing workloads
- bound WebSocket viewer fan-out
- limit expensive API operations
- isolate internal service ports from the public Internet
- terminate public traffic through HTTPS
- keep deployment credentials outside Git and Terraform state
- prevent one slow viewer from blocking all other viewers
- provide explicit opt-in for private-network RTSP access

It is not intended to provide a complete enterprise identity, network-isolation, or DDoS-protection platform.

## Trust boundaries

A typical production deployment has the following boundaries:

```text
                         Internet
                            |
                            | 80 / 443
                            v
                    +---------------+
                    |     Caddy     |
                    | TLS + Auth    |
                    +-------+-------+
                            |
              +-------------+-------------+
              |                           |
              | HTTP API / WebSocket      | static assets
              v                           v
        +------------+               React frontend
        | Go backend |
        +------+-----+
               |
               | validated RTSP target
               v
            FFmpeg
               |
               v
          RTSP network
```

The Go backend is the main boundary between user-controlled stream configuration and server-side network/media execution.

## HTTPS edge

Production traffic terminates at Caddy.

Caddy provides:

```text
HTTP -> HTTPS redirect
automatic certificate management
TLS termination
Basic Auth
static frontend serving
HTTP API reverse proxy
WebSocket reverse proxy
```

The hosted deployment uses a trusted Let's Encrypt certificate for:

```text
streams.dataolympus.com
```

The application should not expose Basic Auth over plaintext HTTP.

## Authentication

The current application does not implement application-level user accounts.

Protected deployments use Caddy Basic Auth.

This is appropriate for:

```text
small self-hosted deployments
controlled demonstrations
review environments
internal tools behind a trusted network boundary
```

It is not intended to replace a full identity provider or multi-user authorization system.

### No default password

There is deliberately no built-in production password.

The deployment helper prompts for a password and stores only its bcrypt hash.

Example:

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

The production environment then contains a value similar to:

```dotenv
DEMO_USERNAME=reviewer
DEMO_PASSWORD_HASH='$2a$...'
```

The plaintext password is not written to the environment file.

### Credential storage

Production credentials live in:

```text
deploy/compose/prod.env
```

This file is excluded from Git.

Deployment setup also applies restrictive file permissions:

```bash
chmod 600 deploy/compose/prod.env
```

Terraform does not receive the application password, so the password hash is not intentionally written into Terraform state.

### Basic Auth headers

HTTP Basic Auth credentials are encoded, not encrypted.

Transport security therefore depends on HTTPS.

A copied `Authorization: Basic ...` header should be treated like a reusable credential and rotated if exposed.

## RTSP URL validation

RTSP URLs are user-controlled network destinations.

Without validation, accepting arbitrary RTSP URLs would create an SSRF path from the backend into networks reachable by the host.

The backend therefore validates RTSP targets before creating a stream.

### Scheme validation

Only the expected RTSP scheme is accepted.

For example:

```text
rtsp://camera.example/live
```

Other schemes are rejected.

### Host requirement

A valid hostname or address must be present.

Malformed or incomplete destinations are rejected.

### Embedded credentials

URLs containing user information are rejected.

For example:

```text
rtsp://user:password@camera.example/live
```

is not accepted.

This restriction is intentional in the current version because the stream representation is returned through the API and frontend. Accepting inline credentials would risk exposing secrets through application responses, browser state, logs, or screenshots.

A future credential-aware implementation should represent secrets separately from stream URLs.

## Address filtering

Public-oriented deployments block RTSP destinations that resolve to unsafe address classes.

The current policy rejects:

```text
loopback addresses
private-network addresses
link-local addresses
multicast addresses
unspecified addresses
```

Examples of destinations that should not be reachable in the default public configuration include:

```text
127.0.0.1
10.0.0.0/8
172.16.0.0/12
192.168.0.0/16
link-local ranges
```

The hostname is normalized before policy evaluation.

DNS resolution is performed and the target is rejected if any resolved address violates the configured policy.

## Explicit allowlist

Controlled hosts can be explicitly allowed.

The hosted demo uses:

```dotenv
RTSP_ALLOWED_HOSTS=mediamtx
```

This permits the internal demo RTSP server while keeping arbitrary private destinations blocked.

The allowlist should contain only infrastructure controlled by the deployment operator.

## Private-network deployments

Many real RTSP cameras intentionally live on private networks.

Self-hosted operators can opt into private destinations with:

```dotenv
RTSP_ALLOW_PRIVATE_NETWORKS=true
```

This changes the trust model.

It should only be enabled where users who can create stream records are already trusted to reach the camera network.

Loopback and other unsafe address classes remain separate policy concerns.

## SSRF limitations

The RTSP URL policy is a baseline SSRF mitigation.

It should not be described as complete SSRF isolation.

Current limitations include:

```text
DNS rebinding possibilities
routing changes after validation
network policy outside the application
provider-specific metadata endpoints
future protocol integrations
```

For higher-risk environments, application URL validation should be combined with infrastructure controls such as:

```text
egress firewall rules
network namespaces
dedicated worker networks
cloud network policies
explicit camera allowlists
DNS controls
```

## FFmpeg process execution

FFmpeg is launched directly from Go using process APIs.

The application does not construct a shell command such as:

```text
sh -c "ffmpeg ..."
```

Instead, executable arguments are passed directly to the process launcher.

This avoids shell expansion of user-controlled RTSP values.

Runtime execution is tied to a Go context so the application can cancel and terminate stream processes when a stream stops.

## Active-stream capacity

Starting a stream creates an expensive FFmpeg runtime.

The backend therefore enforces:

```text
MAX_ACTIVE_STREAMS
```

The public demo currently uses:

```text
MAX_ACTIVE_STREAMS=2
```

When the active runtime limit has been reached, starting another stream returns:

```text
503 Service Unavailable
```

with:

```json
{ "error": "stream capacity reached" }
```

Stopping a stream releases its runtime slot.

This control limits both resource consumption and the number of concurrent server-side media processes.

## Viewer capacity

Each live stream can also limit simultaneous WebSocket viewers with:

```text
MAX_VIEWERS_PER_STREAM
```

The public demo currently uses:

```text
MAX_VIEWERS_PER_STREAM=4
```

The backend reserves a viewer slot before accepting the WebSocket connection.

When capacity has been reached, another viewer receives:

```text
503 Service Unavailable
```

with:

```text
viewer capacity reached
```

Disconnecting a viewer releases its slot.

Viewer capacity and active-stream capacity are independent.

## Slow viewers

WebSocket subscribers use bounded buffers.

If a client cannot consume media quickly enough, the system can evict that subscriber rather than allowing its backpressure to stall the shared stream pipeline.

This protects other viewers of the same stream.

A slow or background browser tab can therefore lose its viewer connection while the backend stream itself remains live.

## Rate limiting

Expensive API operations are protected by an in-memory fixed-window rate limiter.

The current protected operations include:

```text
stream creation
stream start
```

The public deployment currently uses:

```text
EXPENSIVE_REQUESTS_PER_MINUTE=10
```

When the limit is exceeded, the API returns:

```text
429 Too Many Requests
```

with a `Retry-After` response header.

Read-only operations are not currently counted against this expensive-operation quota.

## Client identity and proxy headers

The rate limiter uses the client address as its key.

By default, the backend uses the direct remote address.

When deployed behind the trusted Caddy reverse proxy:

```dotenv
TRUST_PROXY_HEADERS=true
```

allows the application to use forwarded client information.

This option should not be enabled when arbitrary clients can connect directly to the backend.

If another proxy or CDN is later introduced ahead of Caddy, forwarded-header trust should be redesigned around an explicit trusted-proxy chain.

Blindly trusting arbitrary `X-Forwarded-For` values would allow clients to bypass per-address rate limits.

## Request-size limits

Stream-creation requests are bounded.

The current API applies a small maximum request-body size to stream creation requests.

Oversized requests are rejected instead of being read without bound.

This reduces unnecessary memory use and limits abuse of JSON request endpoints.

## WebSocket origin handling

The WebSocket implementation retains same-origin protections.

Cross-origin upgrade attempts are not broadly permitted through wildcard configuration.

The production frontend and WebSocket endpoint share the same application origin:

```text
https://streams.example.com
wss://streams.example.com
```

This simplifies browser-origin policy and authentication behavior.

## HTTP server timeouts

The backend configures HTTP server timeouts suitable for ordinary API traffic.

Header and idle timeouts help prevent clients from holding resources indefinitely during incomplete HTTP requests.

A global short `WriteTimeout` is intentionally not applied because WebSocket media connections are long-lived.

## Container resource limits

The production Compose configuration places resource ceilings on the backend container.

The hosted deployment currently uses:

```text
CPU          2
Memory       1 GiB
PID limit    128
```

These are controlled through:

```text
BACKEND_CPUS
BACKEND_MEMORY_LIMIT
BACKEND_PIDS_LIMIT
```

These limits provide a second boundary around application and FFmpeg resource use.

They do not replace host-level capacity planning.

## Network exposure

The Azure deployment exposes only the ports needed at the public edge.

Typical inbound access is:

```text
22/tcp     administrator SSH
80/tcp     HTTP redirect and ACME
443/tcp    HTTPS application traffic
```

SSH is restricted to the configured administrator CIDR.

The backend API port:

```text
8080
```

is not publicly published.

The demo RTSP port:

```text
8554
```

is also internal to the Docker network.

External tests against both ports are expected to time out or fail.

## Internal services

The optional demo profile runs:

```text
MediaMTX
synthetic FFmpeg test camera
```

These services are intended for development and demonstration.

A normal production installation using real RTSP infrastructure does not need the demo profile.

## Health endpoints

The production edge leaves:

```text
/healthz
/readyz
```

available without Basic Auth.

These endpoints return generic service-health information and are useful for infrastructure monitoring.

They should not expose secrets, stream URLs, credentials, or detailed internal state.

## TLS certificate storage

Caddy stores certificate state in persistent Docker volumes.

Recreating the frontend container therefore does not normally require issuing a new certificate from scratch.

The certificate state should still be treated as deployment infrastructure data.

## GitHub Container Registry

Application release images are published to:

```text
ghcr.io/dataolympus/rtsp-stream-console-backend
ghcr.io/dataolympus/rtsp-stream-console-frontend
```

The images are public so self-hosted users can pull releases without storing GitHub credentials on deployment hosts.

GitHub Actions publishes the images using repository-scoped workflow credentials.

Production deployments should prefer explicit immutable release tags instead of a moving development tag.

## Deployment separation

Terraform is responsible for infrastructure.

It manages resources such as:

```text
resource group
network
subnet
security group
public IP
network interface
virtual machine
Docker bootstrap
```

Application deployment is handled separately by:

```text
scripts/deploy-azure.sh
scripts/deploy.sh
```

This keeps runtime credentials and application configuration outside Terraform state.

## Production environment file

The production environment contains operational settings such as:

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

The file must not be committed.

## Secrets and logs

Operators should avoid exposing:

```text
plaintext Basic Auth passwords
Authorization headers
future camera credentials
private RTSP URLs containing sensitive information
production environment files
```

If a Basic Auth header or plaintext password is exposed, rotate the credential.

Logs should not intentionally contain authentication secrets.

## DDoS and volumetric attacks

Application-level controls such as rate limiting and runtime limits help with resource exhaustion caused by ordinary API usage.

They do not provide volumetric DDoS protection.

Large network floods should be handled through infrastructure or edge controls such as:

```text
cloud DDoS services
CDN or edge protection
firewall policy
network rate controls
upstream filtering
```

The Go service should not be treated as the primary defense against Internet-scale traffic attacks.

## In-memory state

Stream records are currently stored in memory.

Restarting or replacing the backend container clears them.

This has a useful security property for the demo because sensitive runtime state is not persisted automatically, but it also means the current system is not a durable configuration store.

Future persistent storage would require its own access-control, backup, retention, and secret-handling design.

## Known limitations

The current version does not provide:

```text
application-level users and roles
OIDC / OAuth / SSO
fine-grained stream authorization
durable encrypted secret storage
camera-credential management
complete DNS-rebinding protection
distributed rate limiting
distributed runtime limits
multi-node network policy
volumetric DDoS protection
audit logging
persistent security event storage
```

These are explicit current boundaries rather than implied capabilities.

## Recommended production posture

For a small Internet-accessible deployment:

```text
use HTTPS
use a strong unique Basic Auth password
keep prod.env outside Git
leave private RTSP networks disabled unless required
restrict SSH to administrator addresses
do not publish backend or RTSP ports
set conservative runtime and viewer limits
set container resource limits
use explicit GHCR release versions
keep the host and Docker runtime patched
monitor cloud cost and resource use
```

For higher-trust or larger environments, add stronger identity, network segmentation, persistent secret management, centralized observability, and dedicated media workers.

## Validated deployment behavior

The hosted deployment has been exercised against the following controls:

```text
HTTPS certificate validation                 passed
HTTP -> HTTPS redirect                       passed
Basic Auth challenge                         passed
private RTSP destination rejection           passed
embedded RTSP credentials rejection          passed
two-active-stream limit                      passed
third active stream rejection                passed
active-stream slot release                   passed
four-viewer limit                            passed
fifth viewer rejection                       passed
viewer-slot release                          passed
slow-viewer eviction                         observed
backend port 8080 externally inaccessible    passed
RTSP port 8554 externally inaccessible       passed
backend CPU limit                            applied
backend memory limit                         applied
backend PID limit                            applied
Terraform drift check                        clean
public GHCR image pull                       passed
GHCR-based Azure deployment                  passed
```

These checks validate the current demo deployment configuration. They are not a substitute for a formal penetration test or independent security review.

## Reporting security issues

Until a dedicated security reporting policy is published, avoid posting sensitive vulnerability details in a public issue.

For security-sensitive findings, contact the project maintainers privately before public disclosure.
