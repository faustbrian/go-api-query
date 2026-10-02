# Threat model

Model version: 1.0.0. This model describes the published v2.0.0 cursor boundary
and its application-owned dependencies. It supplements the repository-wide
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

## Published callback contract

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

## Risk disposition and review conditions

The following are open review items, not accepted ecosystem risks or evidence
of a completed remediation:

| Boundary | Owner | Risk and current mitigation | Review condition |
| --- | --- | --- | --- |
| Replay callback and codec mutex | Repository maintainer; application owner for its callback | A slow or reentrant guard can block decoding, and request cancellation cannot terminate the wait. Authentication and payload validation precede the guard; application callbacks must be bounded, avoid reentry, atomically consume fingerprints, and expire retained state. These caller obligations do not satisfy the repository's context and no-lock-across-callback requirements. | Review before treating v2 as security-complete, changing replay storage, timeout, concurrency or topology, or publishing a callback-contract change. |
| Random reader | Repository maintainer; application owner for an override | A custom reader can undermine nonce security or block encoding. The default uses `crypto/rand.Reader`; retain it in production and do not treat test readers as safe production configuration. | Review before changing randomness injection or making stronger nonce-source or cancellation guarantees. |
| Other trusted callbacks | Application owner; repository maintainer for API guarantees | Clocks and other application collaborators are not made safe merely by bounded cursor data. Use finite, nonblocking implementations and review each collaborator's execution contract. | Review when adding external work, changing callback ownership, or claiming cancellation across application code. |

## Scope limits

Replay protection is disabled when no guard is configured. The library does
not supply durable shared replay storage, snapshot isolation, or protection
against compromised keys. This documentation records the released contract;
it does not silently substitute an unpublished `ReplayStore` API or accept the
open items above. Review this model whenever the cursor API or its dependency
ownership changes. Report vulnerabilities through [SECURITY.md](../SECURITY.md).
