# Compatibility Policy

Each independently releasable module follows semantic versioning. The root
module uses `v<version>` tags; an independently releasable nested module, when
present, uses `<module-directory>/v<version>` tags.

The current module is on its stable v5 line; the v4 line remains available.
Incompatible exported API or documented behavior changes require a new
major version. Historical v1 to v4 consumers are unaffected until they explicitly migrate; major
module suffixes identify distinct Go type contracts, not source branches.

Compatibility includes exported Go APIs, error classification, serialization,
protocol behavior, persistence schemas, environment variables, command output,
resource ownership, ordering, retry/idempotency semantics, and documented
defaults. A compile-compatible change can still be behaviorally breaking.

Specification-backed modules MUST NOT diverge from their declared standards.
Ambiguities require documented decisions and stable tests. Deprecated APIs
follow [`DEPRECATION.md`](DEPRECATION.md).

Version 5 removed the five top-level adapter packages; published v4 releases
keep them and no further v4 releases are planned.
See [`docs/migration-adapters.md`](docs/migration-adapters.md#adopting-v5).
