# Architecture

RTSP Stream Console is a single-node streaming application that converts RTSP video sources into browser-consumable MPEG-TS streams and fans those streams out to multiple WebSocket viewers.

This document describes the current architecture, runtime ownership model, data flow, failure behavior, security boundaries, and scaling limits.

## Goals

The current architecture is designed to:

- accept arbitrary allowed RTSP sources
- expose stream lifecycle operations through an HTTP API
- process RTSP through FFmpeg
- make the output consumable by a browser
- support multiple simultaneous stream records
- share one running stream pipeline across multiple viewers
- prevent slow viewers from blocking the media pipeline
- provide predictable runtime and viewer limits
- remain simple enough to run with Docker Compose on one host

It is intentionally optimized for small self-hosted and demonstration deployments.

## High-level architecture

```text
                        +-------------------+
                        |    RTSP source    |
                        +---------+---------+
                                  |
                                  | RTSP/TCP
                                  v
                        +-------------------+
                        |      FFmpeg       |
                        | H.264 -> MPEG-TS  |
                        +---------+---------+
                                  |
                                  | stdout
                                  v
                        +-------------------+
                        |     Media pump    |
                        +---------+---------+
                                  |
                                  v
                        +-------------------+
                        |    Fan-out Hub    |
                        +----+----+----+----+
                             |    |    |
                             |    |    |
                    WebSocket|    |    |WebSocket
                             v    v    v
                          Viewer Viewer Viewer
                             |    |    |
                             v    v    v
                          mpegts.js clients
                             |    |    |
                             v    v    v
                          <video> elements
```

## Application components

### Frontend

The frontend is built with React and TypeScript.

Responsibilities include:

- registering stream records
- displaying stream state
- starting and stopping streams
- establishing WebSocket viewer connections
- passing MPEG-TS media to `mpegts.js`
- attaching playback to HTML `<video>` elements
- handling viewer-local play and pause behavior
- surfacing action and runtime errors

The browser does not connect directly to RTSP.

### Go backend

The backend provides:

- stream CRUD operations
- stream lifecycle operations
- runtime management
- FFmpeg process management
- media fan-out
- WebSocket delivery
- health and readiness endpoints
- URL validation
- resource and request limits

The application currently stores stream records in memory.

Restarting the backend therefore resets the stream registry.

### FFmpeg

Each started stream record owns an FFmpeg process.

The current pipeline is approximately:

```text
RTSP
  |
  v
FFmpeg
  |
  | H.264
  | MPEG-TS container
  v
stdout
```

FFmpeg is launched directly through Go process execution rather than through a shell.

The current implementation transcodes video to H.264 for deterministic browser compatibility.

### Media Hub

The Hub distributes one stream runtime's media to multiple WebSocket subscribers.

```text
                         Stream runtime
                               |
                               v
                            FFmpeg
                               |
                               v
                              Hub
                   +-----------+-----------+
                   |           |           |
                   v           v           v
                Viewer 1    Viewer 2    Viewer 3
```

A subscriber has a bounded buffer.

If a viewer becomes too slow, the system can disconnect that viewer instead of allowing its backpressure to stall the shared stream.

## Control plane and media plane

It is useful to separate two paths in the system.

### Control plane

```text
Browser
   |
   | HTTPS
   v
Caddy
   |
   v
Go HTTP API
   |
   +-- create stream
   +-- list streams
   +-- start stream
   +-- stop stream
   +-- delete stream
```

These requests control stream records and runtime lifecycle.

### Media plane

```text
RTSP source
    |
    v
FFmpeg
    |
    v
Media Hub
    |
    | binary MPEG-TS
    v
WebSocket
    |
    v
mpegts.js
    |
    v
<video>
```

Control traffic and media traffic therefore have different runtime characteristics.

API requests are short-lived.

WebSocket media connections can remain open for long periods.

## Stream record vs RTSP source

A stream record is an application resource.

It is not a globally deduplicated representation of an RTSP URL.

For example:

```text
Stream record A
URL = rtsp://camera/live

Stream record B
URL = rtsp://camera/live
```

If both records are started, the current implementation may run:

```text
FFmpeg A -> rtsp://camera/live
FFmpeg B -> rtsp://camera/live
```

This is intentional current behavior.

The system shares media between viewers of the **same started stream record**, not between all records that happen to use the same URL.

## Stream lifecycle

A stream record moves through the following primary states:

```text
created
   |
   | Start
   v
connecting
   |
   | first media arrives
   v
 live
   |
   | Stop
   v
stopping
   |
   v
stopped
```

Failures can move the stream into:

```text
error
```

Examples include:

- FFmpeg cannot start
- the RTSP source becomes unavailable
- the media pipeline fails unexpectedly

## When a stream becomes live

Starting an FFmpeg process does not immediately mean that useful media is available.

The backend therefore treats the first received media bytes as the transition point from:

```text
connecting -> live
```

This prevents the API from reporting a stream as healthy merely because the FFmpeg process exists.

## Viewer lifecycle

A viewer is independent from the stream runtime.

For a live stream:

```text
stream runtime
     |
     +------ Viewer A
     |
     +------ Viewer B
```

Pausing Viewer A does not stop the runtime:

```text
stream runtime              still live
     |
     +------ Viewer A        disconnected
     |
     +------ Viewer B        still receiving media
```

Pressing Play creates a new viewer connection.

Stopping the stream is different:

```text
Stop
 |
 v
cancel runtime
 |
 v
terminate FFmpeg
 |
 v
close media pipeline
 |
 v
stream -> stopped
```

## WebSocket fan-out

Each viewer connects to:

```text
/api/v1/streams/{id}/ws
```

The backend reserves a viewer slot before accepting the connection.

This allows the application to enforce:

```text
MAX_VIEWERS_PER_STREAM
```

For example, with a limit of four:

```text
Viewer 1    accepted
Viewer 2    accepted
Viewer 3    accepted
Viewer 4    accepted
Viewer 5    503 viewer capacity reached
```

When a viewer disconnects, its slot is released.

## Active runtime limit

The runtime manager separately enforces:

```text
MAX_ACTIVE_STREAMS
```

For example:

```text
MAX_ACTIVE_STREAMS=2

Stream A    live
Stream B    live
Stream C    start rejected
```

Stopping one runtime releases capacity for another.

Viewer count and active-stream count are therefore independent dimensions:

```text
active stream limit
    controls FFmpeg/runtime count

viewer limit
    controls WebSocket subscribers
```

## Failure behavior

The backend distinguishes user-requested shutdown from unexpected runtime failure.

Expected shutdown:

```text
user Stop
   |
   v
context cancellation
   |
   v
FFmpeg terminates
   |
   v
stopped
```

Unexpected shutdown:

```text
FFmpeg exits unexpectedly
   |
   v
runtime failure
   |
   v
error
```

This distinction prevents normal cancellation from being surfaced as an application failure.

## Production request path

```text
Internet
   |
   | 80 / 443
   v
Azure public IP
   |
   v
Network Security Group
   |
   v
Caddy
   |
   +-------- static frontend
   |
   +-------- /api/* -> backend:8080
   |
   +-------- WebSocket -> backend:8080
```

Backend port `8080` is not published to the Internet.

MediaMTX port `8554` is also internal in the hosted demo.

## Caddy

Caddy provides the production edge.

Responsibilities include:

- HTTPS termination
- automatic certificate management
- HTTP to HTTPS redirects
- Basic Auth
- serving frontend static assets
- reverse proxying API traffic
- reverse proxying WebSocket upgrades

Health and readiness endpoints can remain unauthenticated for infrastructure checks.

## Authentication

The current hosted deployment uses HTTP Basic Auth at Caddy.

The application itself does not currently implement user accounts.

This is appropriate for a small protected demo or controlled self-hosted deployment, but it is not intended as a multi-user identity system.

Credentials are generated outside Terraform.

Only the bcrypt password hash is stored in the production environment file.

## RTSP URL security boundary

Allowing a server to connect to arbitrary user-supplied URLs creates an SSRF risk.

The URL policy therefore validates RTSP destinations before stream creation.

Public-oriented defaults reject:

- loopback destinations
- private network destinations
- link-local destinations
- multicast destinations
- unspecified destinations
- RTSP URLs containing embedded credentials

Explicitly controlled hostnames can be allowlisted.

Self-hosted deployments can opt into private RTSP networks when their cameras intentionally live on private infrastructure.

This validation is a baseline SSRF defense.

It is not a complete defense against all DNS rebinding or network-routing attacks.

## Resource boundaries

Several independent controls protect the single-node deployment.

```text
MAX_ACTIVE_STREAMS
    bounds concurrent FFmpeg runtimes

MAX_VIEWERS_PER_STREAM
    bounds WebSocket viewers per stream

EXPENSIVE_REQUESTS_PER_MINUTE
    limits create/start operations

BACKEND_CPUS
    Docker CPU ceiling

BACKEND_MEMORY_LIMIT
    Docker memory ceiling

BACKEND_PIDS_LIMIT
    Docker process ceiling
```

HTTP header and idle timeouts provide additional server-side protection.

## Deployment architecture

The deployment system separates infrastructure provisioning from application deployment.

```text
                     GitHub
                        |
               GitHub Actions
                        |
                        v
                       GHCR
                  backend/frontend
                        |
                        |
Terraform               |
   |                    |
   v                    |
Azure VM <--------------+
   |
   v
deploy-azure.sh
   |
   v
deploy.sh
   |
   v
Docker Compose
```

### Terraform owns

- Resource Group
- network
- subnet
- Network Security Group
- static public IP
- network interface
- Linux VM
- Docker bootstrap

### Deployment scripts own

- application Git revision
- production environment
- authentication configuration
- GHCR image version
- Docker Compose lifecycle

This prevents application credentials from being written into Terraform state.

## Source mode vs release mode

The deployment helper supports two artifact paths.

### Source mode

```text
repository source
      |
      v
docker compose build
      |
      v
local images
```

Useful for development.

### Release mode

```text
GitHub Actions
      |
      v
GHCR
      |
      v
docker compose pull
      |
      v
versioned images
```

This is the preferred production path.

Release mode avoids compiling Go and frontend assets on the deployment VM.

## Local demo profile

The Docker Compose demo profile adds:

```text
MediaMTX
   |
   +-- synthetic FFmpeg camera
```

The synthetic source publishes:

```text
rtsp://mediamtx:8554/camera-1
```

This allows the complete system to be exercised without external RTSP infrastructure.

A normal production deployment does not require the demo profile.

## Current scaling boundary

The current implementation is intentionally single-node.

State is local to one backend process.

Each started stream record runs one FFmpeg process.

This means primary resource pressure grows with:

```text
number of streams
x
source resolution / bitrate
x
transcoding cost
```

Viewer fan-out adds network and WebSocket pressure, but does not create one FFmpeg process per viewer.

## Likely scaling evolution

Larger deployments would likely separate stream processing from the API.

One possible direction:

```text
                    API / control plane
                           |
                           v
                     shared state
                           |
             +-------------+-------------+
             |             |             |
             v             v             v
         worker A       worker B       worker C
             |             |             |
          FFmpeg         FFmpeg         FFmpeg
```

For high viewer counts, media delivery could evolve toward:

```text
WebRTC + SFU
```

for interactive low-latency viewing, or:

```text
HLS / LL-HLS + CDN
```

for large-scale one-to-many distribution.

Hardware transcoding or H.264 stream-copy could reduce CPU cost where source compatibility permits.

## Current non-goals

The current version does not attempt to provide:

- durable distributed stream state
- URL-level runtime deduplication
- user/account management
- multi-node scheduling
- DVR or recording
- video analytics
- camera discovery
- large-scale CDN delivery
- full protection against volumetric DDoS attacks

These are intentionally outside the current scope.

## Architectural invariants

Several properties should remain true as the project evolves:

1. A slow viewer must not stall every other viewer.
2. Viewer pause must not stop a shared stream runtime.
3. Stream stop must terminate its runtime.
4. Public deployments must not accept unrestricted RTSP destinations.
5. Application credentials must not be committed to Git.
6. Infrastructure provisioning and application secrets should remain separate.
7. Production deployments should be reproducible from explicit versions.
