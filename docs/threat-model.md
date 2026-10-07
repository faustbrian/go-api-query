# Threat model

Model version: 3.0.0. This model distinguishes the historical v2.0.0 and
[v3.0.0](https://github.com/faustbrian/go-api-query/releases/tag/v3.0.0) cursor
contracts and their application-owned dependencies. Published
[v4.0.1](https://github.com/faustbrian/go-api-query/releases/tag/v4.0.1)
retains the v3 cursor hardening; its major change adopts JSONAPI v2 named types.
It supplements the repository-wide
[security model and threat matrix](../SECURITY.md); it is not a certification
that every ecosystem security requirement has been satisfied.

## Trust and ownership

Requests, query strings, cursor tokens, filter values, and transport parameters
are untrusted. Schemas, keys, authorization policies, mandatory constraints,
persistence mappings, clocks, and replay configuration are application-owned.
Applications remain responsible for authentication, business authorization,
database execution, key custody, and deployment limits.

The cursor codec uses the standard library's AES-GCM implementation. It
authenticates protocol version and key ID and validates schema revision, exact
ordered sorts, direction, typed positions, expiry, and policy. Configuration
bounds encoded size, position count, strings, and TTL. These controls do not
make arbitrary application callbacks bounded or cancellable.

## Published v2 callback contract

`cursor.ReplayGuard` receives an opaque SHA-256 token fingerprint and expiry.
It is optional, synchronous, and serialized by a mutex on each codec. A guard
owns atomic acceptance and expiry-based retention; separate codec instances
do not share that mutex. A successful guard result is the acceptance point.

`Codec.DecodeCursor` accepts but does not use the supplied context. The guard
has no context parameter, and a caller cannot cancel a mutex wait or a running
guard through that method. Guards must not call back into the same codec's
replay-protected decode path. The codec does not detach callbacks or start
background goroutines to make them appear cancellable.

`Config.Random` defaults to `crypto/rand.Reader`, but accepts a caller-supplied
reader. Nonce reads are synchronous and serialized on each codec. A custom
reader must provide cryptographically secure nonce bytes and bounded reads;
production callers should retain the default. Clock callbacks are also
application-owned synchronous work.

## Published v3 and v4 callback contract

The `/v3` and `/v4` modules reject non-nil `Config.Random` and `Config.ReplayGuard`.
It captures `crypto/rand.Reader` privately at construction for nonce generation.
`ReplayStore` receives the request context, opaque SHA-256 fingerprint and expiry.
It owns atomic consumption, concurrency safety and expiry-based retention.
The library authenticates and validates cursors before calling it, checks
cancellation before the call, holds no library lock and does not serialize calls.

`DecodeCursor` forwards the request context to `DecodeContext`. Context-free
`Decode` rejects replay-enabled codecs. Successful store acceptance is the
consumption commit point, including when cancellation races with callback return.
The call stays synchronous: detaching a callback that ignores cancellation would
leak work and make consumption ambiguous. Such a callback can still block the
calling goroutine; passing context does not safely preempt arbitrary caller code.

This caller-owned dependency risk requires the application owner and deployer
to enforce a deadline, validate backend cancellation and atomicity, bound retained
state and monitor latency. Review before adoption and before changes to the
backend, retry policy, timeout, retention, topology or concurrency assumptions.
These obligations bound deployment risk; they do not guarantee preemption of
uncooperative application code.

## Historical hardening and disclosure disposition

Inspected unsuffixed v1.0.0, v1.1.0 and v1.1.1, and `/v2` v2.0.0 contain
the historical callback limitations. `/v3` v3.0.0 first publishes the hardening;
`/v4` v4.0.0 and v4.0.1 retain it. Migration crosses a major module boundary,
not a same-module v2 patch.

The source review establishes lifecycle limitations and hazardous trusted
configuration capabilities, not a confirmed advisory-required vulnerability
or defensible severity. Slow replay guards present a conditional deployment
availability risk: valid authenticated tokens can reach the guard, while
forged or malformed tokens are rejected first. Custom randomness is trusted
server configuration; the secure default was already `crypto/rand.Reader`.
No vulnerability affected/fixed range or absence of historical exploitation
is claimed. Reopen disclosure triage immediately on evidence of
attacker-reachable material impact despite reasonable callback and admission
configuration.

| Boundary | Owner | Risk and current mitigation | Review condition |
| --- | --- | --- | --- |
| Historical replay callback and codec mutex | Repository maintainer; application owner for its callback | A slow or reentrant guard can block decoding, and request cancellation cannot terminate the wait. Authentication and payload validation precede the guard. Migrate to v3 or v4 context-aware storage without a library callback lock; retained legacy deployments must bound callbacks and admission, avoid reentry, atomically consume fingerprints, and expire retained state. | Review on demonstrated availability impact or changes to replay storage, timeout, concurrency or topology. |
| Random reader | Repository maintainer; application owner for an override | A custom reader can undermine nonce security or block encoding. The default uses `crypto/rand.Reader`; retain it in production and do not treat test readers as safe production configuration. | Review before changing randomness injection or making stronger nonce-source or cancellation guarantees. |
| Other trusted callbacks | Application owner; repository maintainer for API guarantees | Clocks and other application collaborators are not made safe merely by bounded cursor data. Use finite, nonblocking implementations and review each collaborator's execution contract. | Review when adding external work, changing callback ownership, or claiming cancellation across application code. |

## Current accepted collaborator residuals

The repository maintainer owns API guarantees; each adopting application owner
and deployer owns its concrete replay backend and clock. Synchronous execution
preserves an unambiguous consumption point: detaching uncooperative storage
would retain live work and create ambiguous replay outcomes. Clock callbacks
execute outside the library lock but remain caller-owned synchronous work.

Accept this residual only with finite, nonblocking clocks, request and backend
deadlines, cancellation-capable storage, atomic consumption, concurrency and
admission limits, expiry-bounded retention, latency monitoring and no unsafe
callback reentry. Review before adoption and whenever backend, timeout, retries,
retention, topology or concurrency assumptions change; reopen immediately on
demonstrated attacker-reachable impact. This acceptance is not a certification
of an application's backend, deployment or the entire ecosystem.

## Scope limits

Replay protection is disabled when no guard (v2) or store (v3/v4) is configured.
The library does not supply durable shared replay storage, snapshot isolation, or protection
against compromised keys. This documentation records the released contract;
it identifies the published v3/v4 `ReplayStore` API and its bounded library-owned
work separately from trusted collaborators. Review whenever the cursor API or its dependency
ownership changes. Report vulnerabilities through [SECURITY.md](../SECURITY.md).
