# Changelog

All notable changes follow Keep a Changelog. Public releases follow
semantic versioning.

## Unreleased

### Added

- Prepare v3.0.0 at `github.com/faustbrian/go-api-query/v3` for the cursor
  lifecycle changes below, retaining Go 1.27.0, all fourteen package locations
  and the published Validation v2.0.0 dependency. Preserve `api/v2.txt` as the
  released contract; existing v2 consumers must explicitly migrate.
- Add context-aware `cursor.DecodeContext` and concurrent `cursor.ReplayStore`
  APIs for replay-enabled request lifecycles.

### Deprecated

- Deprecate caller-controlled `cursor.Config.Random` and context-free
  `cursor.ReplayGuard` in v3; both remain migration markers, with removal no
  earlier than v4 after the deprecation interval.

### Security

- Source cursor nonces from library-selected `crypto/rand.Reader`; reject
  configured randomness and context-free replay callbacks at construction.
- Propagate cancellation through replay storage without codec-wide callback
  serialization. Store acceptance is the consumption commit point, even when
  cancellation races with callback return. Replay-enabled callers must replace
  `Decode` with `DecodeContext` or `DecodeCursor`.

### Changed

The following retained entries describe the published v2 adoption history;
the planned v3 cursor changes are described above.

- Prepare v2.0.0 at `github.com/faustbrian/go-api-query/v2`, retaining
  Go 1.27.0 and all fourteen package locations, including deprecated adapters.
  Update root and subpackage imports to include `/v2`; query algorithms,
  schemas, wire formats, cursor protocols and persistence behavior are unchanged.
- Adopt public `github.com/faustbrian/go-validation/v2` v2.0.0. Both
  `adapters/validation.Report` and retained `apiqueryvalidation.Report` now
  accept Validation v2 `Limits` and return its `Report`; v1 and v2 named types
  are not interchangeable. Existing Localized v1 consumers remain on API Query
  v1 until an explicit future consumer migration. See the
  [migration guide](docs/migration-adapters.md#adopting-v2).

## 1.1.1 - 2026-10-02

### Changed

- Require Go 1.27.0 instead of Go 1.26.6 for module adoption.
- Update the PostgreSQL integration-test dependency to pgx v5.11.0;
  query compiler and adapter behavior remain unchanged.
- Refresh the shared CI workflow while retaining the pinned tooling contract.

### Fixed

- Route context cancellation and deadline outcomes before Validation
  projection in the canonical quickstart and API guidance.
- Retire the terminal v1.1.0 legacy replay from active repository gates so
  unrelated documentation delivery does not regenerate historical evidence.

## 1.1.0 - 2026-09-08

### Added

- Add preferred HTTP, JSON:API, PostgreSQL, JSON-RPC, and Validation packages
  under `adapters/` with behavior matching the released compatibility paths
  ([ae89ed7fbf](https://github.com/faustbrian/go-api-query/commit/ae89ed7fbff1a17cec321fe69230934fe5fcb497)).

### Deprecated

- Deprecate the five released adapter paths in favor of their target-oriented
  successors while retaining both through the documented compatibility interval.

### Changed

- Share initialized adapter sentinel identities across released and successor
  paths, and select `go-validation` v1.1.0 for the Validation contract.
- Describe stable-v1 installation, adapter selection, named-type conversion,
  independent migration, and post-publication rollback
  ([3d1476e477](https://github.com/faustbrian/go-api-query/commit/3d1476e4774c1eccb7a8e06640696c1849570c11)).

- Replace copied repository verification tooling with the pinned
  `go-library-tools` v1.0.13 contract while preserving the package API baseline,
  mutation checkpoints, and PostgreSQL integration evidence
  ([f3047a824a](https://github.com/faustbrian/go-api-query/commit/f3047a824a5e7ab91ad9d1c56ee3dedfa452eda5),
  [9e39ea5dae](https://github.com/faustbrian/go-api-query/commit/9e39ea5dae937f0e931b454e6a96a6c5f1af0d31),
  [b12293bcc7](https://github.com/faustbrian/go-api-query/commit/b12293bcc7d86f909dfefc9dd35192180f04ec88)).
- Adopt the `go-library-tools` v1.4.0 schema-v2 cohesion contract without
  changing the package API or runtime behavior
  ([21c17e8d69](https://github.com/faustbrian/go-api-query/commit/21c17e8d69e0dbbfe519095d060b61f5542cbf0e)).
- Pin reusable CI to the immutable v1.4.0 W14-enforcement workflow and retain
  the complete required repository contract.
- Reconcile owned v1.0.0 dependency checksums with their transparency-log
  authenticated public module archives without changing dependency versions
  ([5154bd33e9](https://github.com/faustbrian/go-api-query/commit/5154bd33e9d6e50e8f286439a3975db6776f85d0)).

### Fixed

- Preserve released PostgreSQL compiler behavior for a zero-value successor
  when compiling a valid plan with no execution fields, constraints, filters,
  or sorts
  ([d59d2b2f25](https://github.com/faustbrian/go-api-query/commit/d59d2b2f259f013cfbc7e7091eec0246ca6a71f5)).
- Adapt the frozen replay harness at execution time so its descriptor utility
  resolves portably without changing the historical oracle artifact
  ([052707e6fd](https://github.com/faustbrian/go-api-query/commit/052707e6fd856f63aca9f719fba3a0c19bc0100e)).

### Maintenance

- Record, harden, and rebind the immutable legacy replay and mutation evidence
  used to verify the release
  ([197a09945a](https://github.com/faustbrian/go-api-query/commit/197a09945a0bbd7217803d2f3545d90560ef21d6),
  [402b896bec](https://github.com/faustbrian/go-api-query/commit/402b896becd3d16fe106705802c92a7b35bd3a7f),
  [6a4cade8ff](https://github.com/faustbrian/go-api-query/commit/6a4cade8ff6fc9ff500c5f7579223392a1db618c),
  [b4c5295661](https://github.com/faustbrian/go-api-query/commit/b4c5295661a57458044021a9139026ff2bf82056),
  [f2882dfc84](https://github.com/faustbrian/go-api-query/commit/f2882dfc847ff30f7d123266e5e0449c6937cddc),
  [0d8dd604a5](https://github.com/faustbrian/go-api-query/commit/0d8dd604a5442e7315d64d507084eb90a613adb4),
  [fcba93080b](https://github.com/faustbrian/go-api-query/commit/fcba93080b3d6ce58d2bc79209f488ffcc7792a7),
  [9278645735](https://github.com/faustbrian/go-api-query/commit/92786457355822b646d9a5afd784d98fd90fba1f),
  [c9fadab14e](https://github.com/faustbrian/go-api-query/commit/c9fadab14e8a8c48c3b7fe62ac8e7898d25a292f),
  [70e2913546](https://github.com/faustbrian/go-api-query/commit/70e2913546bb9d34c2bd71ce8e589f5bdbd2c589),
  [4278073716](https://github.com/faustbrian/go-api-query/commit/42780737166c91d0f69be182b096b5e8a4b71144),
  [14881eb050](https://github.com/faustbrian/go-api-query/commit/14881eb050595b5ca2f6937d6004f7bcca298814)).

### Documentation

- Replace stale pre-release tooling references with the shared repository
  verification commands and package-owned documentation.
- Link package documentation to the immutable v1.4.0 ecosystem index and
  Service Edge family guidance.

## 1.0.0 - 2026-08-25

### Changed

- Exclude intentional nested modules from root local-proxy archives so local,
  bootstrap, CI, and public module checksums describe the same source
  boundary.

- Track the pinned documentation-tool lockfile so clean CI checkouts install
  the exact validated cspell dependency.

- Reconcile standalone dependency checksums against deterministic current
  module archives so CI, local verification, and release consumers resolve
  identical content.

- Harden standalone documentation validation with deterministic spelling and
  link checks, package-specific documentation gates, and repository-local
  contributor guidance.

### Documentation

- Link the package README to package-owned documentation.

### Distribution

- Include the canonical MIT licence in the independently published module.

### Changed

- Publish the module from its standalone `github.com/faustbrian/go-api-query` identity while preserving its documented API and behavior.
- Upgrade `golang.org/x/text` to v0.41.0 so the dependency graph no longer
  contains GO-2026-5970.
- Pin owned dependencies to published source revisions so clean consumers can
  resolve the module before the first tags.
- Execute API compatibility tooling against the isolated module graph so owned
  dependency source changes cannot conflict with release checksums.
- Normalized standalone module metadata against the canonical owned dependency
  graph, including complete checksums for clean consumer resolution.
- Removed package-local quality-tool dependencies now that repository tooling
  is versioned and executed exclusively by the root command surface.
- Run API compatibility with an explicitly pinned tool so isolated release
  checks do not depend on undeclared host-installed Go tools.
- Use the repository-pinned current `apidiff` revision for the canonical API
  compatibility gate.

### Added

- Immutable typed schemas, requests, plans, canonical JSON, structured errors,
  authorization hooks, mandatory constraints, and conservative query costs.
- Bounded field selection, relationship paths, typed filter expressions,
  deterministic sorts, cursor and offset page requests.
- Authenticated encrypted versioned cursors, rotation, replay hooks, nullable
  positions, and stable response page envelopes.
- Strict HTTP and JSON-RPC parsers, OpenRPC descriptors, authoritative
  `jsonapi` composition, cross-transport conformance, and `validation`
  reporting.
- Safe PostgreSQL primitives, SQLC guidance, test builders, canonical
  conformance helpers, and real PostgreSQL safety tests.
- Exact production coverage, race, fuzz, mutation, vulnerability, compatibility,
  documentation, benchmark, and release automation.

### Security

- Hostile input suites cover injection, authorization, tenant isolation,
  traversal, schema probing, cursor tampering/replay, Unicode, and resource
  exhaustion.
