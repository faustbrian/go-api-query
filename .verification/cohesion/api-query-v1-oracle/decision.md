# Phase 3 API Query Remediation Decisions

Status: accepted; dispatch requires three matching exact-byte no-finding
reviews, the canonical freeze record, and a matching execution-ledger entry

The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL NOT",
"SHOULD", "SHOULD NOT", "RECOMMENDED", "NOT RECOMMENDED", "MAY", and
"OPTIONAL" in this document are to be interpreted as described in BCP 14
[RFC2119] [RFC8174] when, and only when, they appear in all capitals, as shown
here.

[RFC2119]: https://www.rfc-editor.org/rfc/rfc2119
[RFC8174]: https://www.rfc-editor.org/rfc/rfc8174

## Scope And Dispatch Boundary

This decision resolves the `go-api-query` row in the Phase 3 remaining adapter
and documentation audits. It authorizes one repository-owner lane composed of
the dependency-ordered atomic source, release-preparation, publication, and
schema-v3 adoption batches below. The lane:

1. adds target-oriented `adapters/http`, `adapters/jsonapi`,
   `adapters/postgres`, `adapters/jsonrpc`, and `adapters/validation` packages;
2. retains every released v1 package path with its source, type, error, wire,
   and reflection compatibility;
3. pins and verifies the published `go-validation v1.1.0` contract after the
   prerequisite no-replace reverse proof;
4. updates package documentation, examples, migration and compatibility
   guidance, changelog, manifests, API baseline, gate inventory, and
   reverse-consumer evidence atomically with the source additions, then updates
   version/date release metadata in its separately verified release-preparation
   batch; and
5. adopts the schema-v3 integration-role and delivery-evidence contract when
   its released tooling is available, without representing unfinished evidence
   as terminal.

This decision does not authorize implementation by itself. Dispatch is
prohibited until three independent exact-byte reviews cover the contract,
execution-adversary, and verification dimensions; every review has outcome
`accepted-no-findings`; the freeze record binds this document and all three
review records; and the execution ledger repeats those identities.

Dispatch additionally requires the exact ordering and drift checks in this
document. In particular, `go-validation v1.1.0` MUST be publicly resolvable and
fully verified, the unchanged Calendar predecessor MUST be green, and the
published `go-api-query v1.0.0` reverse proof against Validation v1.1.0 MUST be
green before this repository's source changes begin.

This decision does not authorize:

- removal, moving, or exported alias replacement of a released package or
  named type;
- a breaking signature, field, JSON tag, error identity, wire-shape, or
  reflection-identity change;
- reinterpretation of HTTP, JSON:API, JSON-RPC, OpenRPC, PostgreSQL, or
  Validation semantics;
- a new nested module or synchronized release with another repository;
- a root query compiler, cursor, authorization, cancellation, or persistence
  redesign;
- transport projection of context cancellation or deadline as invalid input;
- a service container, registry, bootstrap, umbrella dependency, package-global
  mutable state, hidden initialization, or background work; or
- a Phase 4 composition or middleware-order decision.

The implementation worker MUST remeasure the exact default-branch revision,
public tags, open pull requests, writer ownership, worktrees, package paths,
exported APIs, dependency graph, owned consumers, and public usage immediately
before implementation and again before release. Material drift invalidates the
affected row and requires coordinator review; it does not authorize a broader
redesign.

## Reviewed Evidence Snapshot

The public state observed on 2026-09-06 is:

| Evidence | Exact observation |
| --- | --- |
| Repository | `github.com/faustbrian/go-api-query` |
| Hosted `main` | `f52a10e68221ec372e3cb1bc4b9e2b0d1b48b188` |
| Public release | annotated `v1.0.0`; tag object `ba85d7bc2a4d18e5f3a1b224c1d342a84b4c9b20`; peeled commit `b69acc7edf26bafbca4496132715b2b71cc753b2` |
| Release assets | `ALLOWED_SIGNERS`, `go-api-query-v1.0.0.cdx.json`, `go-api-query-v1.0.0.mod`, `go-api-query-v1.0.0.provenance.intoto.json`, `go-api-query-v1.0.0.zip`, `SHA256SUMS`, and `SHA256SUMS.sig` |
| Hosted pull requests | none open |
| Module layout | one releasable root module; nine declared packages; no nested module |
| Go and dependencies | Go `1.26.6`; `go-jsonapi v1.0.0`; `go-validation v1.0.0`; `pgx/v5 v5.10.0`; no permanent replacement |
| API baseline | root `apidiff` baseline at `api/v1.txt`, Git blob `ec28983aeaa00eedbfa61d2a6c5eeb67fc6c3185` |
| Current tooling | `.golib.yaml` schema 1 with Tooling v1.4.0; `modules.json` schema 2 |
| Existing adapters | `apiqueryhttp`, `apiqueryjsonapi`, `apiquerypgx`, `apiqueryrpc`, and `apiqueryvalidation`, all released in root `v1.0.0` |
| Companions | `apiquerytest` and `cursor` |
| Local checkout | clean local `main` at `c17fc8df5d8e2b893f12c21689a8e9a713d64063`, three commits ahead and seven behind its current `origin/main`; it MUST be preserved and MUST NOT be used as the implementation base |

The released adapter source and API baseline are byte-identical between the
peeled `v1.0.0` commit and hosted `main`. Their hosted-main source SHA-256
identities are:

| Path | SHA-256 |
| --- | --- |
| `apiqueryhttp/http.go` | `9c27429ca3445a88651a7547009b5d82c433994a4a16a7f4f8f0bbabec545479` |
| `apiqueryjsonapi/jsonapi.go` | `666a63e8b78aa323a9a1833320543529da90e75864bcf26f2bfe8d3025612ec5` |
| `apiquerypgx/pgx.go` | `e132e7ddcd1fcb13667521293983cb84d9132cd5060f12ce138706ce52199a19` |
| `apiqueryrpc/rpc.go` | `5cfe40c7a39e0ce18540f51a0937f678da220fa10782ad1d8942d9c446750b07` |
| `apiqueryvalidation/validation.go` | `012b4ac9003fe78d1792fa993a14582f0dde74947b31ccfd78ef3f8175fe1f2e` |

The exact owned reverse-dependency set contains one repository:

| Consumer | Hosted `main` at audit | Observed use |
| --- | --- | --- |
| `go-localized` | `18d0df5c527926454b889caaf87177f955b092f1` | root `apiquery.Value`, `Operator`, and `Predicate` in `localizedquery`; no adapter import |

The owner manifest agrees with that edge. A supplementary organization code
search found no owned consumer of any released API Query adapter path outside
the owner repository. That search is not proof that no external consumer
exists, so every released path remains a high-risk compatibility surface.

At the same observation, `go-validation/main` is
`5b0c647af78b81d3c415b144e8e094cb21690fa8`; exact-main CI run `34014361611`
and release rehearsal `34014546012` are green, but `v1.1.0` has no public tag or
release. Those green runs are prerequisites, not public-resolution evidence.

## Target-Oriented Package Map

All successors are additive packages in the existing root module. Their exact
paths, default package identifiers, and public surfaces are:

| Released package | Additive successor | Package identifier | Exported surface |
| --- | --- | --- | --- |
| `apiqueryhttp` | `adapters/http` | `apiqueryhttp` | `ErrInvalid`; `Parse(string, int) (apiquery.Request, error)` |
| `apiqueryjsonapi` | `adapters/jsonapi` | `apiqueryjsonapi` | `ErrInvalid`; `ErrUnsupported`; `FilterDecoder`; `PageDecoder`; `Config`; `FromQuery(jsonapi.Query, Config) (apiquery.Request, error)` |
| `apiquerypgx` | `adapters/postgres` | `apiquerypostgres` | `ErrInvalid`; `Mapping`; `Compiler`; `QueryParts`; `NewCompiler(Mapping) (*Compiler, error)`; `(*Compiler).Compile(*apiquery.Plan) (QueryParts, error)` |
| `apiqueryrpc` | `adapters/jsonrpc` | `apiqueryjsonrpc` | `ErrInvalid`; `Params`; `Parse([]byte, int) (Params, error)`; `(Params).Request() apiquery.Request`; `ContentDescriptor`; `OpenRPCContentDescriptor() ContentDescriptor` |
| `apiqueryvalidation` | `adapters/validation` | `apiqueryvalidation` | `Report(error, validation.Limits) validation.Report` |

`postgres` and `jsonrpc` are the targets; `pgx` and the abbreviated `rpc` are
released compatibility names, not successor identifiers. The other successor
identifiers intentionally retain their unambiguous domain-plus-target names.
Consumers MAY alias imports locally, but documentation and generated package
metadata MUST use the default identifiers above.

The root, `cursor`, and `apiquerytest` packages remain in place. `cursor`
retains ownership of the API Query cursor protocol and `apiquerytest` remains a
test companion; neither receives an `adapters` successor.

## Type, Error, And Reflection Identity

Every released exported named type remains owned by its released package and
retains its exact declaration, fields, JSON tags, methods, zero value, and
reflection package path. Every successor exported named type MUST be a new
named declaration owned by the successor path. Exported aliases between legacy
and successor types are prohibited.

The successor-owned named-type set is exact:

- `adapters/jsonapi.FilterDecoder`, `PageDecoder`, and `Config`;
- `adapters/postgres.Mapping`, `Compiler`, and `QueryParts`; and
- `adapters/jsonrpc.Params` and `ContentDescriptor`.

The successor JSON:API `Config` fields remain exactly `Resource string`,
`DecodeFilter FilterDecoder`, and `DecodePage PageDecoder`. The successor
PostgreSQL `Mapping` and `QueryParts` retain the released field names and field
types. The successor JSON-RPC `Params` and `ContentDescriptor` retain the
released field names, field types, and JSON tags. Methods return successor-owned
types where the released counterpart returns a legacy-owned named type.
Results owned by the root, standard library, `go-jsonapi`, or `go-validation`
retain those external result types.

At package initialization, each legacy/successor sentinel pair MUST hold the
same non-nil error identity within one build:

- both HTTP paths expose the same `ErrInvalid`;
- both JSON:API paths expose the same `ErrInvalid` and `ErrUnsupported`;
- both PostgreSQL paths expose the same `ErrInvalid`; and
- both JSON-RPC paths expose the same `ErrInvalid`.

The released sentinels are exported assignable Go variables, and their
declarations MUST remain variables for source compatibility. Reassigning an
exported sentinel is unsupported caller mutation and is outside behavioral and
cross-path identity compatibility; tests MUST restore any deliberately mutated
variable before returning and MUST NOT run such mutation in parallel. Under
supported use, each path returns its package's exported sentinel and both
variables retain their shared initialized identity. The implementation MAY
initialize successor variables from the corresponding legacy variables or a
shared private value. It MUST NOT treat consumer reassignment as supported
configuration, and callers MUST NOT be required to compare error text.

Legacy and successor types are intentionally not assignment-compatible without
explicit conversion. Reflection MUST identify each named type at its actual
public package path. Documentation MUST state this consequence and MUST NOT
describe a successor as a type alias or path rename.

## Behavioral Equivalence Contract

This batch selects no adapter behavior change. For every input, callback,
configuration, and zero/nil receiver reachable through both paths, the
successor MUST match the released path in returned root values, error category
and identity, callback count and order, defensive-copy behavior, serialization,
resource use, and panic behavior, except for the intentional named-type and
reflection-path difference above.

### HTTP

Both paths MUST preserve the positive byte bound, UTF-8 validation, strict
supported-name set, duplicate rejection, URL-decoding failure behavior,
absent-versus-explicitly-empty lists, sort direction, page parsing, sanitized
`ErrInvalid`, and absence of retained state or I/O. Neither path may accept raw
SQL, unknown query keys, duplicate values, or malformed escaping.

### JSON:API

Both paths MUST preserve `go-jsonapi` ownership of parameter syntax, the
required nonempty resource, explicit filter/page decoder authority,
`ErrUnsupported` for an unbound present family, `ErrInvalid` for invalid
configuration or callback result, declaration-order behavior, and defensive
copies of parameter-family slices and request slices. Callbacks remain
caller-owned synchronous functions; neither path retains them or calls them
concurrently.

### PostgreSQL

Both paths MUST preserve mapping validation and defensive copying, identifier
grammar and quoting, mandatory-constraint precedence, typed positional
arguments, filter and sort compilation, LIKE escaping, deterministic ordering,
nil-compiler and nil-plan rejection, sanitized `ErrInvalid`, and the prohibition
on executing SQL or owning a connection. `adapters/postgres` targets the
PostgreSQL contract; the required `pgx/v5` dependency remains a repository test
and integration dependency and MUST NOT leak a pgx type into the successor API.

### JSON-RPC And OpenRPC

Both paths MUST preserve strict bounded JSON object decoding, rejection of
unknown and duplicate members, absent-versus-explicitly-empty components,
defensive `Request` snapshots, sanitized `ErrInvalid`, and a fresh independently
mutable OpenRPC descriptor map on every call. Neither path interprets an
application method, handler, transport, or execution policy.

### Validation

Both paths MUST preserve the ordered conversion of `*apiquery.Violations` to
bounded immutable Validation findings, safe paths and codes, nil-error empty
report, sanitized `query_error` fallback, and omission of rejected values and
unsafe causes. They import the Validation root contract, not one of
Validation's transport adapters.

The Validation v1.1.0 terminal model does not authorize either API Query
adapter path to turn caller cancellation or deadline into a validation finding.
An application that has a context-terminal outcome MUST route it through its
ordinary cancellation/deadline policy before calling a validation projection.
This batch MUST include an executable example of that ordering without changing
the released `Report` signature.

## Compatibility And Deprecation

Each released package remains importable throughout the compatibility interval
and receives this exact package-documentation paragraph:

| Released package | Exact notice |
| --- | --- |
| `apiqueryhttp` | `Deprecated: use github.com/faustbrian/go-api-query/adapters/http. This package remains supported through the documented compatibility interval.` |
| `apiqueryjsonapi` | `Deprecated: use github.com/faustbrian/go-api-query/adapters/jsonapi. This package remains supported through the documented compatibility interval.` |
| `apiquerypgx` | `Deprecated: use github.com/faustbrian/go-api-query/adapters/postgres. This package remains supported through the documented compatibility interval.` |
| `apiqueryrpc` | `Deprecated: use github.com/faustbrian/go-api-query/adapters/jsonrpc. This package remains supported through the documented compatibility interval.` |
| `apiqueryvalidation` | `Deprecated: use github.com/faustbrian/go-api-query/adapters/validation. This package remains supported through the documented compatibility interval.` |

The compatibility interval is the longer of:

- 180 days after the successor release becomes publicly consumable; and
- two published stable minor releases of the root module that contain both old
  and new paths.

Time alone does not end the interval when two such minor releases have not
occurred. Removal additionally requires migration of every owned consumer,
fresh public-usage and clean external-consumer evidence, and a separately
authorized next-major release. Correctness and security fixes MUST continue on
both paths throughout the interval.

Migration is an import-path change plus explicit conversion where a consumer
stores or exchanges a package-owned named type. It is not a semantic query or
wire migration. Applications MAY migrate the five paths independently. Root,
cursor, and test-companion imports do not change.

Before publication, rollback is a normal revert of the coherent owner batch.
After publication, rollback MUST NOT delete or move the tag, retag history,
remove a successor, or silently restore a legacy-only package map. It means
keeping both paths resolvable, reverting owned consumer pins/import migrations
separately when required, and publishing a forward patch that repairs any
successor defect while retaining the selected identities.

## Documentation, Manifest, And API Scope

The source batch MUST update all affected consumer-facing surfaces atomically:

1. `README.md` gains a canonical versioned root-module installation command,
   states active stable-v1 status and Go 1.26.6 support, uses successor paths in
   the quick start, and links the complete package map and migration guide.
2. `README.md`, `docs/api.md`, `docs/http.md`, `docs/jsonapi.md`,
   `docs/openrpc.md`, and `docs/sqlc.md` use successor paths for new adoption
   and identify the legacy paths only as supported compatibility paths.
3. `docs/README.md` links package selection, migration, compatibility,
   security, support, FAQ, troubleshooting, changelog, API, examples, and the
   versioned ecosystem family/index without inventing Phase 4 compositions.
4. A migration section or dedicated document records the exact five-path map,
   default identifiers, sentinel sharing, named-type/reflection distinction,
   compatibility interval, independent migration, and rollback.
5. `docs/compatibility.md`, `COMPATIBILITY.md`, and `DEPRECATION.md` describe
   current stable-v1 policy rather than future-v1 status and bind the interval.
6. The five legacy package entry points receive the exact notices above; every
   successor has useful package documentation covering ownership, errors,
   mutability, concurrency, and non-goals.
7. `CHANGELOG.md` stops claiming that semantic versioning begins in the future
   and records the five additive successors, shared error identities,
   deprecations, Validation v1.1.0 dependency, documentation corrections, and
   any schema/tooling adoption without overstating terminal delivery.
8. `packages.json` and the `modules.json` package inventory add all five
   successors with `kind: adapter`, their exact directories, identifiers, and
   import paths. The five legacy entries remain public production packages.
9. `modules.json` package selection and adapter metadata list successors as the
   preferred paths and legacy paths as retained compatibility paths;
   `apiquerytest` and `cursor` remain companions.
10. `go.mod` and `go.sum` select publicly resolved `go-validation v1.1.0` and
    retain the existing `go-jsonapi` and pgx dependencies unless independent
    dependency review proves a required update.
11. `.golib.yaml` adds successor HTTP and JSON-RPC fuzz targets and retains the
    existing parser fuzz targets until legacy removal. PostgreSQL verification
    exercises both paths against the same plan and database behavior.
12. Before baseline replacement, the final code is checked against the
    immutable released baseline whose v1.0.0 and hosted-main Git blob is
    `ec28983aeaa00eedbfa61d2a6c5eeb67fc6c3185`, byte size is 25,021, and
    SHA-256 is
    `b8eac531590599de50da2ed042591db7edf07c6a438e900085aca53ed6354fc5`.
    The repository-pinned `golib api update` command may then regenerate
    `api/v1.txt`; a deterministic semantic delta from the immutable baseline
    MUST contain only declarations at the five selected successor paths, and a
    following `golib api check` MUST pass against the final tree. The regenerated
    baseline and same-tree check are current-state evidence, not historical
    compatibility evidence.

The root module's next version is `v1.1.0` only if remeasurement confirms that
tag is still absent and is the next valid stable minor. The manifest version and
dated changelog heading MUST be changed in a separate release-preparation batch
after source implementation, hardening, and review are green. A drifted or
occupied tag requires a coordinator amendment; the worker MUST NOT silently
choose a different version.

## Schema-v3 Adoption

Schema-v3 adoption is required for this repository's final Cohesion terminal
state, but unavailable tooling MUST NOT block the independent source and
hardening dimensions. Until the released schema-v3 command can create and
verify repository-owned receipts, delivery remains truthfully `in-progress`.

The v3 manifest MUST replace authored legacy adapter/companion arrays with
these exact integration roles:

| Target | Role | Rationale |
| --- | --- | --- |
| each of the five successor import paths | `adapter` | canonical target-oriented integration path |
| each of the five released compatibility import paths | `adapter` | retained adapter path during the compatibility interval |
| `github.com/faustbrian/go-api-query/apiquerytest` | `companion` | deterministic builders, fixtures, assertions, and cross-transport conformance |
| `github.com/faustbrian/go-api-query/cursor` | `companion` | package-owned cursor protocol and page-envelope support |

No API Query package qualifies as `domain-owned` under this decision. In
particular, `adapters/postgres` compiles allowlisted PostgreSQL primitives but
does not own a persisted semantic model, migrations, transactions, or backend
lifecycle.

Every authored v3 role MUST carry the authorization identity required by the
frozen schema-v3 contract. The role decision, delivery goal, applicable-input
manifests, implementation/hardening/release receipts, complete-input
fingerprints, and catalog projection MUST be generated and verified by the
released schema-v3 tooling. Schema-v2 fields MAY remain only until that
migration is executable; they MUST NOT be hand-translated into a false v3
terminal claim.

## Immutable Released-Behavior Oracle

Before any production edit, an external version-neutral harness MUST execute
the public legacy APIs from the peeled `go-api-query v1.0.0` source. It MUST use
the published module without `replace` or `go.work`, canonicalize every result
deterministically, and record:

- the tag object and peeled commit above;
- the exact harness source and input SHA-256;
- the Go toolchain identity;
- one canonical output artifact and SHA-256; and
- an independently reviewed closed manifest binding all those identities.

The harness matrix MUST cover the complete supported behavior promised here:
HTTP success and every invalid class; JSON:API family presence, callback
count/order, callback panic, nil result, error, and copy isolation; PostgreSQL
mapping validation, defensive copies, nil cases, every operator, exact
fragments/arguments, and real database outcomes; JSON-RPC strict decode,
absence/empty, defensive request copies, and fresh descriptor maps; Validation
nil, structured, bounded/truncated, generic-error, path, and copy behavior; and
every released sentinel's normal-use identity and text. Resources are recorded
as absent for stateless paths rather than inferred from final code.

The same harness and inputs MUST then execute the final legacy paths through a
task-owned no-replace local module proxy built from the exact final source tree.
Its canonical output MUST be byte-identical to the frozen v1.0.0 output. The
successor differential matrix MUST separately prove that each new path matches
that independent historical output except for the selected named-type and
reflection-path differences. Final-tree legacy/successor equality alone is not
evidence of released-v1 preservation.

## TDD Red And Acceptance Matrix

Before production edits, the worker MUST add compile-level acceptance tests for
all five absent successor paths and observe the expected package-not-found
failure on the exact hosted-main base. A missing package is only additive API
red evidence; it MUST NOT be described as a behavioral failure.

After adding the smallest declarations needed to compile, the worker MUST run
behavioral differential tests before sharing or moving implementation. Any
legacy/successor divergence is red and remains red until the shared/private
implementation is corrected. Source-text, registration-only, or manifest-only
assertions are not behavioral proof.

| Contract | Required red observation | Required green proof |
| --- | --- | --- |
| Five successor imports | package does not exist | every package declaration has the exact default identifier, and old/new imports compile together when explicitly aliased in one test file |
| Sentinel sharing | successor sentinel absent or initially distinct | direct equality and `errors.Is` succeed across every initialized legacy/successor pair under supported use |
| Named-type ownership | successor type absent or aliased | reflection reports each successor path and each legacy path independently |
| HTTP equivalence | successor absent or result differs | complete released success/failure matrix, exact request snapshots, exact sentinel, and matching fuzz invariant |
| JSON:API equivalence | successor absent or result differs | all presence, callback, error, copy, and nil-result cases match; callbacks receive independent family copies |
| PostgreSQL equivalence | successor absent or result differs | mappings, fragments, typed arguments, error identities, nil cases, mutation isolation, and real PostgreSQL result behavior match |
| JSON-RPC equivalence | successor absent or result differs | strict decode, absence/empty distinction, defensive request, descriptor equality, independently mutable maps, and fuzz invariant match |
| Validation equivalence | successor absent or result differs | nil, structured, bounded/truncated, and generic-error reports are equal and immutable under Validation v1.1.0 |
| Terminal projection order | cancellation is projected as invalid input in the example | example branches on context identity before validation projection |
| Legacy compatibility | immutable v1.0.0 baseline rejects a changed released declaration | final code passes tag-vs-final compatibility before update; the old/new baseline delta contains only the specified successor additions |
| Manifest and docs | package selection omits or prefers legacy paths | exact old/new map, roles, default identifiers, install/version, interval, and links are consistent |

Test cases MUST cover zero values and nil receivers for every exported successor
type whose legacy counterpart permits them. Mutable maps, slices, callback
inputs, returned requests, query arguments, and descriptor maps MUST be mutated
after construction or return to prove the frozen copy/ownership contract.

Successor HTTP and JSON-RPC public parsing surfaces MUST have their own fuzz
entry points. The worker MAY reuse corpus seeds and shared private behavior, but
each public entry point MUST be attributable in the gate. PostgreSQL parity MUST
exercise the real repository-owned PostgreSQL gate, not only string snapshots.

## Verification And Final Review

The source batch is not implementation- or hardening-complete until the exact
final input satisfies:

- the immutable released-behavior oracle and final legacy replay above;
- focused differential tests for every old/new pair;
- exact 100% statement coverage for every production package without aggregate
  masking;
- 100% mutation efficacy and mutant coverage with zero viable survivors;
- race checks and deterministic stress where callbacks or shared values are
  exercised;
- both legacy and successor fuzz targets;
- the real PostgreSQL integration proof for both compiler paths;
- immutable-baseline API compatibility, deterministic baseline-delta,
  documentation, manifest, cohesion, dependency, static, vulnerability,
  secret, license, SBOM, provenance, and task-owned no-replace clean-consumer
  gates;
- one complete `make check` and repository-wide `make ci`; and
- three independent no-finding final implementation reviews covering contract,
  execution/security, and verification/documentation after the last edit.

All Go commands MUST use task-owned disposable `GOCACHE`, `GOMODCACHE`,
`GOTMPDIR`, and `TMPDIR`, and all test-owned PostgreSQL containers, volumes,
caches, mutation workspaces, and generated scratch artifacts MUST be removed
after evidence is recorded. One heavy local gate runs at a time; hosted CI may
overlap unrelated local work.

Implementation and hardening evidence MUST prove:

- the immutable v1.0.0 API baseline and behavior oracle reject any removed or
  changed released import, declaration, field, tag, supported-use error
  identity, behavior, or reflection identity;
- every successor owns its named types and exposes the exact package identifier;
- every sentinel pair has one initialized identity under supported use and no
  error classification uses text;
- every mutable input/output retains the released copy or ownership behavior;
- both parser successors retain hostile-input bounds and strict decoding;
- both PostgreSQL paths remain statement-only compilers with no connection or
  transaction ownership and no raw identifier acceptance;
- Validation v1.1.0 context terminals are not projected as invalid query input;
- the dependency graph remains acyclic and the root module gains no unnecessary
  mandatory runtime dependency;
- catalogs and documentation prefer successors without hiding supported legacy
  paths; and
- no framework runtime, global registry, hidden initialization, background
  goroutine, or umbrella dependency was introduced.

Evidence reuse is allowed only when the complete applicable input fingerprint
is unchanged under repository policy. A history-only change MUST NOT cause an
unrelated expensive rerun, and a changed package, test, dependency, tool,
fixture, manifest, or gate configuration invalidates its affected evidence.

## Publication And Reverse-Consumer Order

The exact order is:

1. Publish and fully verify `go-validation v1.1.0`: signed tag object and peel,
   seven release assets and checksums, signatures, SBOM, provenance, module
   proxy, checksum database, package documentation, and one clean external
   consumer importing the root plus all five Validation successors without
   `replace` or `go.work`.
2. Verify published `go-calendar v1.0.0` in the frozen clean no-replace
   predecessor scenario.
3. In a clean external module, select published `go-api-query v1.0.0` and
   `go-validation v1.1.0` without replacement, import and exercise
   `apiqueryvalidation`, and run the applicable API Query root/adapter tests.
   This distinct proof satisfies Validation's frozen reverse-consumer order and
   MUST precede edits to API Query.
4. Implement and verify this decision from the then-current exact hosted
   `go-api-query/main`, preserving the divergent local `main` and using one
   repository writer.
5. Merge the reviewed source batch, run exact-main CI, prepare `v1.1.0` in a
   separate coherent release-metadata batch, run exact-main CI and the root
   release rehearsal, and publish only through the serialized publication slot.
6. Verify the API Query tag object and peel, seven assets and checksums,
   signature, SBOM, provenance, proxy, SumDB, pkg.go.dev package resolution for
   root plus all five successors, and a clean external consumer importing old
   and new paths together without replacement.
7. Verify `go-localized` at its exact authoritative revision against published
   API Query v1.1.0. Because its observed code uses only root types, no import
   migration is expected; any source or manifest change is a separate owner
   batch and release decision.
8. Recheck the owned dependency graph and supplementary public usage, then
   verify every newly discovered direct or transitive consumer in topological
   order.
9. Adopt and verify schema-v3 roles and delivery receipts when released tooling
   makes that contract executable. This step controls Cohesion terminal status;
   it does not rewrite the already published API identities.

API Query, International, and Temporal MAY execute their independent
post-Validation lanes concurrently after steps 1 through 3's shared
predecessors are satisfied. Localized MUST wait until API Query and
International are green. Publication remains serialized even when source and
hosted verification overlap.

If `go-api-query v1.1.0` exists before step 5, if Validation or Calendar proof
fails, if the legacy source/API identities drift, if a second writer owns the
repository, or if a newly discovered consumer changes the compatibility
decision, dispatch MUST stop for coordinator amendment.

## Completion Conditions

This decision's implementation and hardening dimensions are complete only
when:

- all five successors exist at the exact paths and package identifiers;
- each successor exposes its exact selected API and owns every named type;
- every legacy package remains source-, behavior-, error-, wire-, and
  reflection-compatible and carries the exact deprecation notice;
- the final legacy paths match the independent released-v1 oracle and the new
  paths pass the complete differential matrix;
- Validation v1.1.0, API baseline, manifests, docs, examples, changelog, gate
  inventory, and reverse-consumer evidence are coherent;
- immutable-baseline API comparison, deterministic allowed-additions delta,
  local aggregate verification, task-owned no-replace clean-consumer proof,
  and final implementation reviews are attributable to the exact source input;
  and
- every task-owned resource is removed.

The release dimension is independently complete only when the reviewed source
and release-preparation batches are merged, exact-main CI and rehearsal are
green, the signed release and seven assets are verified, proxy, SumDB, and
pkg.go.dev resolve the exact release, the public old/new clean consumer is
green, and the owned reverse-consumer proof is attributable to the exact
published identities. A nonterminal release dimension does not invalidate
completed implementation or hardening evidence.

The repository's Cohesion row becomes terminal only after schema-v3 roles,
goal state, receipts, fingerprints, and catalog projection are also valid. A
green source release MUST NOT be reported as terminal schema-v3 delivery.

## Freeze Record Contract

The three review records are closed UTF-8 JSON objects at:

- `.ai/cohesion/phase3/API_QUERY_REMEDIATION_CONTRACT_REVIEW.json`;
- `.ai/cohesion/phase3/API_QUERY_REMEDIATION_EXECUTION_REVIEW.json`; and
- `.ai/cohesion/phase3/API_QUERY_REMEDIATION_VERIFICATION_REVIEW.json`.

Each record contains exactly: `schema` equal to
`api-query-remediation-review-v1`; `dimension` equal to `contract`,
`execution-adversary`, or `verification` according to its path;
`decision_path` equal to this path; `decision_sha256` equal to this file's
lowercase exact-byte SHA-256; `reviewer` as one non-empty stable identity
different from the other two reviewers and the author; `outcome` equal to
`accepted-no-findings`; and an empty `findings` array. No record contains a
timestamp-dependent semantic field.

The canonical freeze record is a closed UTF-8 JSON object at
`.ai/cohesion/phase3/API_QUERY_REMEDIATION_FREEZE.json`. It contains exactly:
`schema` equal to `api-query-remediation-freeze-v1`; `decision_path` and
`decision_sha256`; `reviews`, an array in contract, execution-adversary,
verification order whose entries each contain exactly `dimension`, `reviewer`,
`review_record_path`, `review_record_sha256`, and `outcome`; and top-level
`outcome` equal to `accepted-no-findings`.

The freeze writer MUST parse every JSON record with duplicate-member rejection,
verify every exact-byte digest and closed member set, verify three distinct
reviewer identities and dimensions, and reject any nonempty finding set or
nonmatching outcome. The execution ledger MUST repeat the decision path and
digest, freeze path and digest, all three review paths and digests, reviewer
identities, ordered dimensions, accepted outcomes, exact dispatch prerequisites,
and the rule that schema-v3 evidence controls terminal status. Any later
decision-byte change invalidates all reviews and the freeze.
