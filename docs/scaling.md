# Scaling

RTSP Stream Console currently uses a single-node architecture optimized for small self-hosted deployments, demonstrations, and engineering evaluation.

The main scaling dimensions are not identical.

The system scales differently with:

- active stream runtimes
- viewer count
- source bitrate and resolution
- transcoding cost
- network throughput
- geographic distribution

Understanding those dimensions is important before deciding how to scale the system.

## Current runtime model

The current architecture is:

```text
RTSP source
    |
    v
FFmpeg
    |
    v
Media Hub
  / | \
 /  |  \
v   v   v
WS  WS  WS
```

Each **started stream record** owns one FFmpeg process.

Multiple viewers of that stream record share the resulting media pipeline.

This means:

```text
1 stream record + 1 viewer
    -> 1 FFmpeg

1 stream record + 20 viewers
    -> 1 FFmpeg

20 started stream records
    -> up to 20 FFmpeg processes
```

Viewer count and stream-runtime count therefore scale different resources.

## Primary scaling dimensions

### Active streams

Active streams primarily affect:

```text
CPU
memory
process count
RTSP ingress bandwidth
```

because each started stream record owns an FFmpeg runtime.

A useful approximation is:

```text
transcoding load
≈
number of active streams
x
cost per FFmpeg transcode
```

The cost per stream depends heavily on:

```text
source resolution
source frame rate
source codec
target codec
encoder preset
host CPU
hardware acceleration
```

A 4K stream at high frame rate can cost substantially more CPU than a low-resolution 720p source.

For this reason:

```text
MAX_ACTIVE_STREAMS
```

is one of the most important protection mechanisms in the current architecture.

## Viewer count

Viewers primarily affect:

```text
outbound network bandwidth
WebSocket connection count
per-viewer buffering
socket/file-descriptor usage
```

They do **not** currently create one FFmpeg process each.

For example:

```text
Camera A
   |
   v
FFmpeg
   |
   v
Hub
 | | | |
 v v v v
4 viewers
```

still has one media-processing runtime.

This is one of the main advantages of the fan-out Hub.

## Bandwidth model

Suppose a stream is delivered at bitrate:

```text
B Mbps
```

and has:

```text
V viewers
```

Then approximate outbound media bandwidth is:

```text
B x V Mbps
```

For example:

```text
2 Mbps stream
x
10 viewers
=
~20 Mbps outbound
```

before protocol overhead.

With:

```text
4 streams
x
2 Mbps each
x
10 viewers each
```

the host could require approximately:

```text
80 Mbps outbound
```

again before transport overhead.

So viewer fan-out is inexpensive in CPU compared with transcoding, but it is not free in network capacity.

## RTSP ingress bandwidth

Each FFmpeg runtime independently consumes its source stream.

If the same RTSP URL is registered as three different stream records:

```text
Stream A -> rtsp://camera/live
Stream B -> rtsp://camera/live
Stream C -> rtsp://camera/live
```

and all three are started, the current implementation may create:

```text
three FFmpeg processes
three RTSP sessions
three copies of the source ingress traffic
```

The application does not currently deduplicate stream runtimes by URL.

That is an explicit current scaling boundary.

## CPU cost

The current FFmpeg pipeline transcodes video to H.264.

Conceptually:

```text
source codec
    |
    v
decode
    |
    v
raw frames
    |
    v
H.264 encode
    |
    v
MPEG-TS
```

Encoding is usually the most expensive part of the media pipeline.

The current deployment uses a low-latency, fast encoder configuration intended to reduce processing cost and latency.

Even with a fast preset, software transcoding is the dominant CPU consumer as active-stream count grows.

## Why H.264 transcoding is currently used

Always producing H.264 gives the browser pipeline a predictable media format.

This simplifies compatibility.

The trade-off is CPU.

A larger system should inspect the incoming source and avoid unnecessary transcoding when possible.

## Stream-copy optimization

If the source is already compatible with the browser delivery pipeline, FFmpeg may be able to avoid re-encoding.

Conceptually:

```text
CURRENT

H.264 source
   |
decode
   |
encode H.264
   |
MPEG-TS
```

could become:

```text
OPTIMIZED

H.264 source
   |
stream copy
   |
MPEG-TS
```

This can dramatically reduce CPU usage.

However, stream-copy cannot be assumed safely for every source.

Compatibility depends on factors such as:

```text
codec
profile
level
pixel format
timestamps
GOP structure
container behavior
browser decoder compatibility
```

A production implementation could detect compatibility and choose between:

```text
stream copy
transcode
```

rather than always using one strategy.

## Hardware acceleration

Another scaling option is hardware-assisted encoding.

Possible platforms include:

```text
NVIDIA NVENC
Intel Quick Sync Video
AMD hardware codecs
cloud GPU/video accelerators
```

Instead of:

```text
CPU -> software H.264 encoder
```

the architecture becomes:

```text
CPU
 |
 v
hardware encoder
```

This can significantly increase the number of simultaneous streams per host.

It introduces additional operational complexity:

```text
device drivers
container device access
hardware scheduling
provider-specific VM types
codec capability differences
```

For the current small deployment, CPU-based transcoding keeps the system portable.

## Memory scaling

Memory usage includes:

```text
Go backend
FFmpeg processes
WebSocket buffers
media chunks
container/runtime overhead
```

Because every active stream creates another FFmpeg process, memory generally grows with active-stream count.

Viewer buffering also contributes, but buffers are intentionally bounded.

The system should not accumulate unlimited media for a slow subscriber.

## Slow consumers

Each viewer has a bounded media queue.

If a viewer cannot consume data quickly enough:

```text
producer
   |
   v
bounded queue
   |
   X slow viewer
```

the viewer can be disconnected.

The alternative would be:

```text
slow viewer
   |
backpressure
   |
media Hub
   |
all viewers stall
```

which would allow one unhealthy client to degrade every subscriber.

Slow-subscriber eviction therefore protects both reliability and scalability.

## Current resource controls

The current deployment provides:

```text
MAX_ACTIVE_STREAMS
MAX_VIEWERS_PER_STREAM
EXPENSIVE_REQUESTS_PER_MINUTE

BACKEND_CPUS
BACKEND_MEMORY_LIMIT
BACKEND_PIDS_LIMIT
```

These are intentionally independent controls.

For example:

```text
MAX_ACTIVE_STREAMS=2
```

limits processing runtimes.

While:

```text
MAX_VIEWERS_PER_STREAM=4
```

limits fan-out per runtime.

And Docker resource settings bound the entire backend container.

## Scaling one node vertically

The simplest scaling strategy is a larger VM.

For example:

```text
2 CPU
   |
   v
4 CPU
   |
   v
8 CPU
```

This increases the number of FFmpeg runtimes that may be supported.

Vertical scaling is appropriate when:

```text
stream count remains modest
deployment simplicity matters
single-node availability is acceptable
```

It eventually reaches economic and operational limits.

## When vertical scaling stops being enough

A single-node architecture becomes unsuitable when requirements include:

```text
many simultaneous cameras
many concurrent transcodes
high availability
multiple geographic regions
large viewer counts
rolling worker maintenance
failure isolation
```

At that point, stream processing should move out of the API process.

## Worker separation

A natural evolution is:

```text
                 +----------------+
                 |   API service  |
                 +-------+--------+
                         |
                         v
                 shared state/queue
                         |
            +------------+------------+
            |            |            |
            v            v            v
         worker 1     worker 2     worker 3
            |            |            |
         FFmpeg        FFmpeg       FFmpeg
```

The API becomes responsible for control-plane operations.

Workers become responsible for media processing.

Benefits include:

```text
independent worker scaling
better failure isolation
different hardware pools
rolling maintenance
stream placement
```

This would require replacing the current in-memory coordination model.

## Shared state

Once multiple backend or worker instances exist, local process memory is no longer sufficient.

Distributed deployments would need shared coordination for data such as:

```text
stream definitions
desired state
runtime ownership
worker leases
viewer routing
health
```

Possible technologies could include:

```text
PostgreSQL
Redis
message queues
Kubernetes APIs/custom resources
distributed coordination services
```

The exact choice depends on the target deployment model.

## Stream ownership

A distributed system needs exactly one intended runtime owner for each stream.

For example:

```text
Stream A -> worker 2
Stream B -> worker 1
Stream C -> worker 3
```

Without ownership coordination, two workers could accidentally process the same logical stream.

Typical mechanisms include:

```text
leases
distributed locks
queue assignment
scheduler ownership
```

## Viewer routing

Once streams run on different workers, WebSocket viewers must reach the worker that owns the stream.

Possible models include:

### Reverse-proxy routing

```text
viewer
  |
  v
gateway
  |
  +---- stream A -> worker 2
  +---- stream B -> worker 1
```

### Media broker

Workers publish media into a shared distribution layer.

```text
worker
  |
  v
media broker
  |
  v
viewer gateways
```

### Dedicated media protocols

Replace custom MPEG-TS/WebSocket fan-out with infrastructure specifically designed for media distribution.

## WebRTC evolution

WebRTC becomes attractive when requirements include:

```text
very low latency
interactive monitoring
browser-native real-time transport
adaptive network behavior
```

At scale, the topology would normally use an SFU:

```text
camera pipeline
      |
      v
     SFU
   /  |  \
  v   v   v
viewer viewer viewer
```

An SFU forwards encoded media rather than independently transcoding it for every viewer.

This is a stronger fit for interactive real-time viewing than a generic WebSocket fan-out layer.

## HLS and LL-HLS evolution

For large one-to-many viewing, another direction is:

```text
RTSP
 |
 v
transcoder
 |
 v
HLS segments
 |
 v
object storage / origin
 |
 v
CDN
 |
 +---- viewer
 +---- viewer
 +---- viewer
```

This removes viewer delivery load from the application server.

HLS trades additional latency for extremely scalable distribution.

LL-HLS can reduce that latency while retaining CDN compatibility.

## WebRTC vs HLS

A rough decision model:

```text
Need very low latency?
       |
      yes
       |
       v
WebRTC / SFU

Need very large broadcast audience?
       |
      yes
       |
       v
HLS / LL-HLS + CDN
```

These are not mutually exclusive.

A platform can support different delivery modes for different workloads.

## Geographic scaling

Camera location matters.

If cameras are in one region and processing is in another:

```text
camera
   |
long-distance RTSP
   |
cloud worker
```

the system becomes sensitive to:

```text
latency
packet loss
network reliability
egress cost
```

A larger architecture could place media workers close to camera networks:

```text
camera site A -> regional worker A
camera site B -> regional worker B
camera site C -> regional worker C
```

while keeping the control plane centralized.

## Edge deployments

RTSP commonly appears in local camera networks.

Running processing near the camera can provide:

```text
lower source latency
less WAN traffic
better resilience during Internet disruption
private-network camera access
```

An edge worker could publish processed media or metadata back to a central platform.

## Recording

Recording changes the storage model significantly.

Current streams are transient.

Adding recording introduces:

```text
storage capacity
retention policy
segment indexing
object storage
encryption
access control
lifecycle management
```

Recording should therefore be treated as a separate subsystem rather than simply writing the current stream to disk indefinitely.

## Observability at scale

A larger deployment needs metrics beyond container health.

Useful measurements include:

```text
active streams
stream start latency
stream failure count
FFmpeg restart count
viewer count
viewer disconnect count
slow-viewer eviction count
media bytes in
media bytes out
per-stream bitrate
FFmpeg CPU usage
FFmpeg memory usage
WebSocket duration
```

Worker-level metrics would be needed for scheduling decisions.

## Autoscaling signals

CPU alone may not be the best scaling metric.

Potential signals include:

```text
active stream runtimes per worker
total encoding CPU
viewer network throughput
pending stream assignments
memory pressure
```

A scheduler could place streams based on measured transcoding cost instead of treating every stream as identical.

## Current single-node sizing

There is no universal formula for streams per VM because FFmpeg cost varies significantly with input characteristics.

Capacity should be measured using representative sources.

A practical test is:

```text
1. start one representative stream
2. measure CPU and memory
3. increment active stream count
4. observe latency and process stability
5. retain safety headroom
6. configure MAX_ACTIVE_STREAMS below saturation
```

Do not size production capacity from synthetic assumptions alone.

## Viewer sizing

Viewer capacity testing should likewise use realistic bitrate and connection duration.

Useful checks include:

```text
network throughput
socket count
memory per subscriber
slow-client behavior
disconnect/reconnect rate
```

Because viewers share the transcoder, network capacity can become the limiting factor before CPU for a single popular stream.

## Failure domains

The current single-node system has one primary failure domain:

```text
VM
 |
 +-- backend
 +-- frontend
 +-- FFmpeg runtimes
```

If the VM fails, all streams fail.

Multi-node availability requires:

```text
replicated control plane
external shared state
worker health detection
runtime reassignment
edge/load-balancer routing
```

Those capabilities are outside the current version.

## Cost scaling

Infrastructure cost grows differently depending on workload.

### More active streams

Usually increases:

```text
CPU cost
memory cost
possibly GPU cost
RTSP ingress
```

### More viewers

Usually increases:

```text
network egress
connection capacity
```

### More regions

Usually increases:

```text
compute duplication
networking
operational complexity
cross-region traffic
```

Optimization should therefore begin by identifying the dominant cost dimension.

## Recommended evolution path

A reasonable progression from the current design is:

```text
Stage 1
single VM
software transcoding
WebSocket fan-out

        |
        v

Stage 2
stream-copy where possible
larger or optimized VM
hardware acceleration where useful

        |
        v

Stage 3
separate API and media workers
shared runtime coordination
worker scheduling

        |
        v

Stage 4
WebRTC/SFU for low-latency fan-out
or
HLS/CDN for broadcast fan-out

        |
        v

Stage 5
multi-region / edge workers
central control plane
```

The architecture should evolve only when workload characteristics justify the added complexity.

## Design principle

The central scaling principle is:

```text
Do expensive media processing once,
then share the result whenever possible.
```

The current Hub already applies this principle within one stream record.

Future versions can extend it across workers, protocols, and regions.
