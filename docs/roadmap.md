# Roadmap

RTSP Stream Console is an open-source platform for operating video sources reliably, understanding failures, and making incident-related video evidence available to other systems.

This roadmap covers the shared product foundation through **v0.5.0: Operational Integration**. It does not prescribe a particular application domain. Features and integrations beyond this scope should be prioritized using demonstrated user needs and operational evidence.

- **Current baseline:** [v0.1.0](https://github.com/dataolympus/rtsp-stream-console/releases/tag/v0.1.0), a self-hostable RTSP viewing and management console.
- **Active milestone:** [v0.2.0: Reliable Video Operations](https://github.com/dataolympus/rtsp-stream-console/milestone/1).
- **Work tracking:** [GitHub issues](https://github.com/dataolympus/rtsp-stream-console/issues).

The milestones below describe intended capabilities, **not features already available**. Version scope and release gates may be refined as implementation and testing reveal new constraints. There are no fixed release dates.

## Product responsibility

The project aims to provide a dependable path from a configured video source to actionable operational information and portable evidence:

**Connect → Observe → Recover → Capture → Verify → Deliver**

The shared platform should:

- Manage explicitly permitted video sources and their desired state.
- Distinguish source health, media-worker health, and viewer delivery.
- Diagnose interruptions and recover predictably when recovery is possible.
- Capture bounded video evidence associated with events.
- Preserve relevant timestamps, provenance, and known gaps.
- Make artifacts usable outside the console through documented interfaces.
- Be secure, observable, testable, and supportable in its declared deployment environments.

The project is **not** intended to replace mature media servers, codecs, object stores, inference runtimes, or workflow engines. Where practical, it should integrate existing components behind documented interfaces. Control-plane responsibilities should remain distinct from media processing and storage.

## Milestone overview

| Milestone                             | Primary outcome                                                                                  | Release gate                                                                                           |
| ------------------------------------- | ------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------ |
| **v0.2.0: Reliable Video Operations** | Persistent and secure sources, source health, recovery, and diagnostics                          | A permitted source can fail, recover, and survive application restart predictably.                     |
| **v0.3.0: Incident Evidence**         | Timestamped events, bounded capture, retention, and incident review                              | An event produces retrievable evidence, or an explicit record of why capture was incomplete.           |
| **v0.4.0: Portable Evidence**         | Versioned manifests, integrity checks, export APIs, and storage adapters                         | An independent consumer can validate and use exported evidence without the console.                    |
| **v0.5.0: Operational Integration**   | Repeatable deployment, monitoring, capacity limits, recovery procedures, and tested integrations | An independent operator can deploy, upgrade, monitor, recover, and troubleshoot a documented workload. |

Each milestone builds on the guarantees of the previous one. A milestone is complete only when its acceptance tests, documentation, and release gate have been met.

## v0.2.0: Reliable Video Operations

**Goal:** Make the current viewing console a dependable service for managing real RTSP sources.

### Planned capabilities

- **Persistent source configuration:** Keep source definitions and desired state across backend restarts. Separate persistent configuration from ephemeral worker and viewer sessions. Define restart and migration behavior.
- **Approved source connectivity:** Support explicitly allowed RTSP destinations, including configured private-network sources, without broadly disabling destination protections. Test DNS and address resolution, network restrictions, and server-side request forgery defenses.
- **Credential safety:** Keep RTSP credentials out of routine API responses, UI displays, and logs. Define secure storage and access to source secrets.
- **Health model:** Report source connectivity, usable media, worker state, and viewer delivery as different conditions. Avoid treating a running process as proof of healthy video.
- **Bounded recovery:** Define timeouts, retries, exponential backoff, cancellation, and resource cleanup. Intentional stops must not trigger automatic restarts.
- **Actionable diagnostics:** Identify connection, authentication, timeout, codec, decoding, and delivery failures where they can be distinguished reliably. Preserve a bounded history of state transitions.
- **Operational visibility:** Publish documented per-source metrics and useful diagnostic information without exposing secrets or high-cardinality identifiers unnecessarily.
- **Regression coverage:** Use synthetic sources and fault injection to test disconnection, recovery, capacity limits, restart, and unavailable sources. Document at least one representative real-camera configuration.

### Release gate

A reviewer can register an approved source, inspect its health, reproduce a failure, understand the reported condition, observe bounded recovery, and restart the backend without losing the source configuration. Failed and partial operations are surfaced clearly. Secret exposure and unauthorized source access are covered by automated regression tests. The supported operating limits are documented.

Implementation and issue-level progress are tracked in the [v0.2.0 GitHub milestone](https://github.com/dataolympus/rtsp-stream-console/milestone/1).

## v0.3.0: Incident Evidence

**Goal:** Move from live monitoring to reproducible investigation of past events.

### Planned capabilities

- **Event ingestion:** Introduce a versioned event contract with event ID, associated source, event type, source-reported occurrence time where available, and server receipt time. Define validation, idempotency, and error behavior.
- **Bounded recording:** Capture an event-related video interval with configurable limits. Evaluate existing media-server recording facilities before adding custom media processing. Pre-event capture requires an active buffer or recording before the event arrives.
- **Retention and storage limits:** Define disk usage, retention policy, cleanup, capture failure behavior, and recovery of partial or interrupted recordings.
- **Incident records:** Link events, sources, media segments, capture requests, outcomes, and errors through stable identifiers.
- **Review workflow:** Allow an operator to locate an incident, inspect its timeline, replay available footage, and identify unavailable intervals.
- **Time semantics:** Distinguish reported event time, event receipt time, recording time, and any available media timestamps. Document clock assumptions and synchronization uncertainty; do not imply frame-level alignment without evidence.
- **Reliability tests:** Cover duplicate events, late events, unavailable video, disk pressure, process restarts, interrupted writes, and incomplete capture windows.

### Release gate

A controlled source event can be submitted and found later through the API and UI. The recorded interval is retrievable, and the incident record makes any missing portion, timestamp uncertainty, or capture failure explicit. Storage consumption remains bounded under the configured retention policy.

## v0.4.0: Portable Evidence

**Goal:** Make incident evidence independently consumable and verifiable.

### Planned capabilities

- **Versioned evidence manifest:** Define source and incident references, capture windows, clock information, media metadata, artifact paths, processing versions, and known gaps. Publish a schema and compatibility rules.
- **Artifact integrity:** Produce content digests and provide an independent validation command or library. A digest detects changes relative to a recorded value; it does not prove camera authenticity or the truth of an event description.
- **Export interface:** Support a portable local package and a documented programmatic export API with bounded, retryable operations.
- **Storage adapters:** Support a local filesystem backend and an S3-compatible object-storage path, with clear configuration, permissions, failure, and retention behavior.
- **Independent consumption:** Provide a small reference reader and example workflow to inspect a manifest, validate artifacts, and access its media and event records without a running console.
- **Compatibility tests:** Verify schema evolution, round-trip export/import, corrupt or missing artifacts, interrupted transfers, and validation failures.

### Release gate

A separate process can download or receive an evidence package, validate its declared files, parse its versioned manifest, and read its video and event data without accessing the original application. Known gaps, clock assumptions, and failed exports remain visible rather than being silently omitted.

## v0.5.0: Operational Integration

**Goal:** Deliver a supportable, operationally integrated system with a clearly defined deployment and capacity envelope.

### Planned capabilities

- **Repeatable deployment:** Maintain a tested single-deployment reference installation using Docker Compose. Document prerequisites, configuration, TLS/authentication boundaries, installation, health checks, and clean removal.
- **Upgrade and recovery:** Test configuration and data migrations, backups, restores, image-version pinning, upgrades, and rollback procedures. Explicitly document operations that cannot be rolled back safely.
- **Operational monitoring:** Provide stable metrics, structured logs, health/readiness behavior, dashboards or alert examples, and incident troubleshooting procedures. Keep source, worker, storage, and downstream-delivery signals distinct.
- **Capacity and resource controls:** Benchmark representative codecs, source counts, viewer counts, recording load, CPU, memory, disk, and network use. Publish test conditions, supported limits, saturation behavior, and reproducible benchmark scripts.
- **Security operations:** Document access control, secret rotation, destination allowlists, network isolation, data retention, and safe configuration defaults. Exercise relevant security tests in CI.
- **Deployment portability:** Avoid cloud-specific assumptions in application interfaces. Evaluate a tested Kubernetes deployment or Helm chart when it reduces operational effort; do not require a custom operator without a demonstrated reconciliation problem.
- **Integration contract:** Stabilize the documented API, event format, metrics, and artifact interfaces sufficiently for at least one independent integration test and a non-browser consumer.
- **Release and support tooling:** Maintain reproducible builds, versioned images, dependency/security scanning, compatibility notes, release procedures, smoke tests, and troubleshooting/runbook documentation.

### Release gate

An engineer who did not develop the project can follow the documentation to deploy a supported configuration, register and operate sources, capture and export evidence, monitor resource usage, respond to an injected failure, restore from backup, and complete a tested upgrade. The published limits and unsupported configurations are explicit. No claim of high availability, horizontal scaling, multi-cloud support, or guaranteed data completeness is made without corresponding tests.

Additional cloud-specific Terraform modules, distributed scheduling, high availability, and custom Kubernetes controllers are **not automatic release requirements**. They should be scoped separately when a concrete workload and test plan justify them.

## Engineering and release standards

The following apply to every milestone:

1. **Tests first where practical:** Add a failing test for a reported bug or defined behavior, implement the smallest correct change, then refactor and rerun relevant tests.
2. **Explicit failure behavior:** Test unavailable sources, cancellation, resource exhaustion, partial results, and recovery, not only successful operation.
3. **Security by default:** Protect credentials, restrict destinations, bound resource consumption, and avoid exposing sensitive data in metrics or logs.
4. **Observable claims:** Record test conditions and metrics for performance or reliability claims. Prefer repeatable measurements over unqualified scale statements.
5. **Documented contracts:** Version externally consumed APIs and schemas, describe breaking changes, and provide migration instructions when necessary.
6. **Operational documentation:** Update configuration examples, troubleshooting guidance, and known limitations with each user-visible change.
7. **Verified release:** Require passing automated checks, reviewed changes, documented release notes, and smoke tests for the supported deployment path.

## Maintaining this roadmap

- GitHub milestones and issues track accepted implementation work. This document records the broader sequence, boundaries, and release criteria.
- The active [v0.2.0 milestone](https://github.com/dataolympus/rtsp-stream-console/milestone/1) provides the detailed issue backlog. Later milestone sections are proposed scope, not a claim that their issues or GitHub milestones already exist.
- New tasks should describe an operator problem, expected behavior, acceptance criteria, and a reproducible test where possible.
- Move tasks between releases when dependencies, security concerns, or evidence require it. Do not keep a feature solely to satisfy a predetermined version number.
- Avoid treating additional adapters, cloud providers, protocols, or integrations as success measures on their own. Favor tested reliability and user value.

For current implementation and operating details, see the [README](../README.md), [architecture](architecture.md), [deployment](deployment.md), and [security](security.md) documentation.
