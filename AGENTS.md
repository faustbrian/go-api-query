# Engineering Policy

The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL NOT",
"SHOULD", "SHOULD NOT", "RECOMMENDED", "NOT RECOMMENDED", "MAY", and
"OPTIONAL" in this document are to be interpreted as described in BCP 14
[RFC2119] [RFC8174] when, and only when, they appear in all capitals, as
shown here.

## Scope And Authority

- This file is the canonical policy for the complete repository.
- Package policies MAY add stricter domain rules but MUST NOT weaken this file.
- `CLAUDE.md` and tool-specific files MUST point here rather than duplicate it.
- Historical `.ai/GOAL*.md` files are requirements and evidence, not proof of
  completion. Current executable evidence is REQUIRED.

## Repository Structure

- The public root module MUST live at the repository root.
- Intentional optional or test modules MAY live in explicit nested directories.
- Commands MUST live under `cmd/`; private shared code MUST live under
  `internal/`; root automation MUST live under `scripts/`.
- Public module paths MUST match their repository-relative directories beneath
  the module path declared by the root `go.mod`.
- Every module MUST be declared in `modules.json`, and every package MUST be
  declared in `packages.json`.
- Independently releasable modules MUST retain independent `go.mod` files and
  directory-prefixed semantic-version tags.
- Cross-module dependencies MUST remain acyclic and MUST use public contracts.
- Permanent `replace` directives, sibling repositories, and absolute developer
  paths are forbidden in releasable modules.

## Design

- Prefer standard-library interfaces and explicit composition over hidden
  registration, global state, reflection-driven wiring, or service locators.
- Public APIs MUST make ownership, cancellation, retries, timeouts, resource
  limits, error semantics, and concurrency behavior observable.
- Interfaces SHOULD be defined by consumers and MUST remain narrowly scoped.
- Optional integrations SHOULD be adapters or nested modules, not mandatory
  dependencies of a core package.
- Breaking protocol or specification ambiguities MUST be documented as explicit
  decisions and covered by tests.

## Safety And Concurrency

- Shared mutable state MUST have one documented synchronization owner.
- Goroutines MUST have explicit lifetime, cancellation, shutdown, and leak
  tests. Fire-and-forget goroutines are forbidden.
- Channels MUST have documented ownership and closure rules.
- Locks MUST NOT be held across caller callbacks, network IO, blocking channel
  operations, or unbounded work.
- Every external operation MUST accept or derive a bounded `context.Context`.
- Response bodies, files, rows, transactions, timers, tickers, connections,
  and temporary resources MUST be closed on every path.
- Integer conversions, sizes, offsets, recursion, decompression, and allocation
  from untrusted input MUST be bounded before allocation or conversion.
- Secrets and credentials MUST NOT appear in errors, logs, traces, snapshots,
  fixtures, mutation reports, or generated artifacts.

## Testing

- Behavioral changes MUST include meaningful tests before completion.
- Tests MUST assert outcomes, invariants, errors, cleanup, and state transitions;
  line execution without behavioral assertions is not acceptable coverage.
- Coverage and mutation results are diagnostic evidence, not universal merge
  thresholds. They MUST be selected when they exercise a material risk or a
  release boundary and MUST NOT block unrelated documentation or metadata.
- Parsers and hostile boundaries SHOULD use fuzz tests, corpus seeds, resource
  limits, and deterministic regressions when input risk warrants them.
- Concurrent code MUST pass the race detector and targeted stress or leak tests
  when concurrency behavior changes.
- Specification claims MUST be proven against official fixtures or independent
  implementations when that external conformance is part of the affected
  contract.
- Performance-sensitive changes MUST use representative benchmarks with stated
  workloads and allocation budgets.

## Proportional Assurance

Every change MUST be classified before verification:

- **Tier A** covers documentation, metadata, registration, and generated
  documentation without runtime behavior. Validate only the affected
  structure, links, examples, or generation; inspect the final diff; and use
  ordinary review when meaningful.
- **Tier B** covers internal behavior without a public contract change. Run a
  focused behavior test, affected package or module tests, applicable format
  and static checks, and one complete review. A bounded repository gate SHOULD
  run; unrelated expensive checks MAY remain scheduled.
- **Tier C** covers public APIs, lifecycle, security, persistence, and
  concurrency. Require an observable regression or characterization test,
  focused behavior, API compatibility where applicable, directly affected
  package and integration tests, direct owned reverse consumers, and one
  independent complete-diff review.
- **Tier D** covers public releases and ecosystem milestones. Bind immutable
  release or milestone inputs once and run only the relevant compatibility,
  composition, consumer, and aggregate checks.

Race, fuzz, mutation, leak, performance, conformance, external-service,
clean-consumer, release-rehearsal, and aggregate fleet checks MUST run only when
they exercise a material risk or the applicable Tier D boundary. They MUST NOT
block an unrelated change merely because the check exists.

One complete independent review is the default for a meaningful Tier C or Tier
D batch. Additional reviews MUST each cover a distinct named high-risk domain.
A fixed review count and cryptographically bound review records are prohibited
as routine requirements.

## Required Commands

- `make inventory` validates repository and package manifests.
- `make check` runs the exact contract for every repository module.
- `make ci` runs the complete repository contract.
- Pull requests MUST run fast checks selected for the affected assurance tier.
  Aggregate, scheduled, and release workflows MUST own broad or expensive
  checks that are not required for the pull request's material risks.
- Local commands and CI SHOULD share scripts and thresholds for the same gate.
- A missing dependency required by a selected gate MUST fail that gate; an
  unselected gate MUST NOT fail unrelated work merely because its tool,
  service, profile, mutant, or report is absent.
- NilAway is advisory; its findings MUST remain visible and tracked against a
  no-regression baseline.

## Evidence Validity And Reuse

- Evidence validity MUST follow the inputs that materially affect the selected
  gate, not a branch name or repository-history shape alone.
- Commit hashes and tool versions MAY be recorded for traceability, but MUST
  NOT create recursive provenance requirements for mutable plans, prose,
  progress ledgers, or evidence already bound to an immutable commit and CI
  run.
- Unchanged immutable inputs SHOULD reuse existing evidence. An unrelated edit,
  squash, rebase, or metadata-only commit MUST NOT trigger an expensive rerun
  when the relevant inputs and risk boundary are unchanged.
- After a change, rerun only the gates, modules, packages, and direct reverse
  consumers affected by that change and its assurance tier.
- Evidence MUST NOT be reused when the relevant input identity cannot be
  established. The affected gate then requires fresh execution.
- Tooling and evidence schemas MUST remain backward-compatible with supported
  repository versions. Optional fields or new tooling MUST NOT force a fleet
  migration, block unrelated source work, or trigger a module release.

## CI And Workflows

- `.github/workflows/ci.yml` is the only owned GitHub Actions workflow.
- Package-local workflows MUST NOT be added.
- Actions and external tools MUST be pinned to immutable versions.
- Every selected module MUST have an attributable result and evidence artifact.
- The stable required job MUST fail for failed, cancelled, skipped, or missing
  module results.
- Required checks MUST NOT use `continue-on-error`, `|| true`, permissive
  thresholds, or warning substitutions.

## Dependencies And Supply Chain

- Dependencies MUST be necessary, maintained, license-compatible, and pinned to
  reviewed current versions.
- Standard-library functionality MUST NOT be wrapped merely to create an owned
  abstraction; wrappers require a stable policy or portability boundary.
- Generated code and vendored corpora MUST record source, version, checksum,
  license, generation command, and update procedure.
- Vulnerability, secret, license, SBOM, provenance, and clean-consumer checks
  are release gates.

## Documentation

- Public identifiers MUST have useful Go documentation describing semantics,
  invariants, ownership, errors, concurrency, and caveats where relevant.
- Comments MUST explain why a constraint or non-obvious implementation exists;
  they MUST NOT narrate obvious syntax.
- Every public module MUST provide a quick start, API reference, examples,
  adoption guidance, tradeoffs, security notes, FAQ, and release notes.
- Documentation and examples MUST compile and be checked in CI.

## Changelogs

- Every user-visible change MUST update the affected module `CHANGELOG.md` in
  the same commit.
- Entries MUST describe behavior and migration impact, not internal activity.
- Changes to multiple modules MUST update every affected changelog.
- Unreleased entries MUST NOT be silently rewritten or removed.
- Generated, dependency, security, compatibility, and deprecation changes are
  user-visible and require entries.

## Completion

- Run the narrowest affected gates during development and only the risk-selected
  release gates before declaring completion.
- Re-run affected gates after the final source, test, dependency, documentation,
  workflow, or generated-file change.
- Report exact commands and results. A skipped, blocked, stale, or warning-only
  selected gate is not a pass; an unselected unrelated gate is outside the
  claim rather than a failure.
