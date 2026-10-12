# Contributing

## Before Editing

1. Read [`AGENTS.md`](AGENTS.md) and the affected package's docs.
2. Run `go test ./...` or the affected package's narrower tests.
3. Identify owned dependencies and reverse dependants in `modules.json`.
4. Preserve unrelated work and generated/corpus provenance.

## Changes

Keep commits focused and conventional. Update every affected changelog with
the behavior and migration impact. Public API changes require compatibility
evidence and documentation. Specification behavior requires a decision record,
fixture coverage, and interoperability evidence.

New direct dependencies and dependency updates must follow the
[dependency governance policy](AGENTS.md#dependencies-and-supply-chain). Package-local
update bots are forbidden; the root policy owns every module and action update.

Repository verification is provided by the released `go-library-tools` version
and checksum in `.golib.yaml`. The Makefile is a thin local entry point;
package-owned PostgreSQL verification remains in `verification/package.mk`,
while the API baseline and mutation checkpoints remain repository-owned.

Specification-backed changes must follow the
[specification governance contract](AGENTS.md#design), update
the affected stable decision entries, and complete the Specification Decisions
section of the pull request template. An unresolved interpretation or stale
source pin is release-blocking; peer behavior cannot silently select policy.

Mutation is optional local or scheduled assurance, not a required PR status.
Confirmed behavioral or security defects still require a focused regression
and repair; retain reviewed narrow equivalent/invalid-mutant dispositions.

Do not add package-local workflows, permanent replacements, machine-specific
paths, bypass flags, broad mutation exclusions, or aggregate quality metrics
that hide a failing package.

## Verification

Standalone `go test ./...` runs deterministic tests, fuzz seeds and saved corpus
regressions without sibling checkouts or ecosystem ledgers. With no
`APIQUERY_TEST_DATABASE_URL`, the live PostgreSQL test is skipped; this is not
database integration evidence. Set that variable only to an owned test database
when selecting live integration.

With the checksum-pinned public verifier selected by `.golib.yaml`, use:

```bash
make inventory
make check
make ci
```

`check` and `ci` select bounded local verification. The manifest's PostgreSQL
service requirement still applies to these Make routes: the verifier supplies
its task-owned fixture, so they are not Docker/service-free alternatives to
direct Go tests. No build or test tags are declared. One existing root race
assertion checks shared schema/plan canonical results and defensive field copies
with its original 32 workers and 100 iterations, once with a two-minute timeout.
Ordinary CI consumes the public verifier without source bootstrapping.

Select one target for short active fuzz exploration. Optional local mutation
selects one package; neither command is an all-change prerequisite:

```bash
GOMAXPROCS=2 go test -run '^$' -fuzz '^FuzzCompileFilterExpression$' -fuzztime=2s -parallel=1 -timeout=2m .
golib mutation --module . --package ./cursor
```

Authorization, cursor, transport, SQL and public API changes retain affected
security, race, compatibility, live PostgreSQL and direct-adapter/consumer
checks. The full PostgreSQL operation checks both legacy and preferred compilers
for injection resistance and stable cursor ordering; preserve its fixture and
connection cleanup ownership.

`make check-full` preserves all full operations, including six native 5s fuzz
budgets, PostgreSQL integration, documentation and benchmarks. Relevant release
qualification remains `golib release dry-run --module .`, without a sibling
catalog release prerequisite. Report unavailable, skipped and failed boundaries
honestly; a local pass is not hosted-CI, live integration or public-release proof.

## Adding A Module

Follow [repository structure policy](AGENTS.md#repository-structure). New modules
require an explicit purpose, ownership boundary, dependency review, package
catalog entry, full quality gates, documentation, changelog, license, security
policy, compatibility plan, and release dry-run.

## Source Documentation

Install the checksum-pinned public verifier declared in `.golib.yaml`.
`make source-docs-check` validates objective Go comment, marker, link and
generated-producer structure; `make check`, `make check-full` and `make ci`
require it before their existing checks. Shared CI runs the same strict
contract and includes its result in Required. This is structural enforcement,
not a substitute for source-truth review or behavioral verification.
