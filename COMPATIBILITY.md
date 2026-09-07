# Compatibility Policy

Each independently releasable module follows semantic versioning. The root
module uses `v<version>` tags; an independently releasable nested module, when
present, uses `<module-directory>/v<version>` tags.

The module is on its stable v1 line. Incompatible exported API or documented
behavior changes require a new major version.

Compatibility includes exported Go APIs, error classification, serialization,
protocol behavior, persistence schemas, environment variables, command output,
resource ownership, ordering, retry/idempotency semantics, and documented
defaults. A compile-compatible change can still be behaviorally breaking.

Specification-backed modules MUST NOT diverge from their declared standards.
Ambiguities require documented decisions and stable tests. Deprecated APIs
follow [`DEPRECATION.md`](DEPRECATION.md).

The released adapter packages and their preferred `adapters/` successors remain
supported together for the interval defined in
[`docs/migration-adapters.md`](docs/migration-adapters.md).
