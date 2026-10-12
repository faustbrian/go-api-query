# Source documentation audit

## Scope and pre-edit decisions

Base: `1a28fcea3f3b5c7fa2cd5e1910f6537b9303969e`. Documentation-only Tier A;
source expressions, API baselines, dependencies and frozen evidence remain unchanged.
One active root module, `github.com/faustbrian/go-api-query/v4`; 14 operational
packages plus two historical verification source directories, 64 tracked Go files.

Reproduce with `git ls-files '*.go'`, `git ls-files '*go.mod'`,
`golib source-docs inventory --json`, and `golib source-docs check --json`.
The pre-edit native inventory found 196 missing exported declaration/member comments.
This is a gap inventory, not a substitute for source-contract review.

## Complete source inventory

- `.`: `apiquery_test.go`, `benchmark_test.go`, `capability_semantics_test.go`, `compile.go`, `compile_internal_test.go`, `errors.go`, `example_test.go`, `fuzz_test.go`, `jsonapi_v2_composition_test.go`, `mutation_contracts_internal_test.go`, `plan.go`, `race_test.go`, `schema.go`, `security_semantics_test.go`, `successor_compile_test.go`, `successor_differential_test.go`, `transport_conformance_test.go`, `types.go`, `value_schema_internal_test.go`.
- `.verification/cohesion/api-query-v1-final-replay`: `verify.go`, `verify_test.go`.
- `.verification/cohesion/api-query-v1-oracle`: `generate-contract.go`, `harness.go`, `verify.go`.
- `adapters/http`: `fuzz_test.go`, `http.go`, `http_test.go`.
- `adapters/jsonapi`: `jsonapi.go`, `jsonapi_test.go`.
- `adapters/jsonrpc`: `fuzz_test.go`, `jsonrpc.go`, `jsonrpc_test.go`.
- `adapters/postgres`: `postgres.go`, `postgres_test.go`.
- `adapters/validation`: `validation.go`, `validation_test.go`.
- `apiqueryhttp`: `fuzz_test.go`, `http.go`, `http_test.go`.
- `apiqueryjsonapi`: `jsonapi.go`, `jsonapi_test.go`.
- `apiquerypgx`: `internal_test.go`, `operator_test.go`, `pgx.go`, `pgx_test.go`, `postgres_integration_test.go`.
- `apiqueryrpc`: `fuzz_test.go`, `rpc.go`, `rpc_test.go`.
- `apiquerytest`: `apiquerytest.go`, `apiquerytest_test.go`.
- `apiqueryvalidation`: `internal_test.go`, `validation.go`, `validation_test.go`.
- `cursor`: `benchmark_test.go`, `cursor.go`, `cursor_test.go`, `fuzz_test.go`, `hardening_test.go`, `page.go`, `page_test.go`, `pagination_property_test.go`.
- `internal/strictjson`: `strictjson.go`, `strictjson_test.go`.

## Contracts and private rationale reviewed

- Root types, schema indexes, relationship cloning, compile subphases, overflow-safe
  cost aggregation, bounded violation collection and canonical cursor hashing:
  absent/empty selection, required execution fields, mandatory constraints,
  authorization, sort tie-breakers, copied plan accessors and sensitive values.
  Schema configuration is trusted; request bounds are not schema-size budgets.
- All five successor facades and their five deprecated implementations: shared
  sentinel identity and intentional delegation; HTTP syntax, two-pass JSON,
  JSON:API family ownership, safe validation projection and PostgreSQL fragments.
  Backend comparison/null behavior is PostgreSQL behavior, not core evaluation.
- Cursor key snapshot publication, codec clock lock/callback boundary, authenticated
  envelope and replay commit point; page helper shallow copies and callback failures.
  No callback lifecycle, goroutine, transport handler, database pool or shutdown is owned.
- Test support builders, copied requests and canonical comparison: application test
  fixtures, not an independent protocol oracle or proof of backend execution.
- All test sources: core boundary/security/capability/mutation matrices; facade
  differential, transport composition, cursor hardening/traversal/replay/rotation,
  strict JSON and builder tests. Six fuzz targets, five benchmark functions across
  root/cursor plus allocation checks, and two executable Examples are inventoried.
  Existing live PostgreSQL, race, fuzz, mutation and performance evidence is not rerun.
- Historical oracle and replay harnesses (five Go files) remain immutable, including
  generator `generate-contract.go`, external consumer module inputs and fixture pins.
  They are inventoried and structurally parsed, not re-executed or rewritten.
  No active Go build constraints, generated headers or go:generate directives were found.
  Existing analyzer directives on successor/legacy adapters are preserved exactly.

## Missing documentation versus existing limitations

Repair member comments and useful ownership/validation rationale. Reconcile current
claims only: positive bounds replace defaults rather than only narrow them; null is
not a mandatory-constraint value; the Protected filter flag does not enforce policy;
Compile does not poll cancellation; Canonical checks size after encoding and can contain
sensitive predicates. Constructor strings/time values are not fully revalidated by Compile.

Existing wire limitations are not specification interpretations: Value.UnmarshalJSON
uses encoding/json member matching and accepts extra typed-value members; RPC null
collapses pointer presence and top-level null becomes empty parameters. JSON decoding
can repair invalid UTF-8 and custom unmarshaling controls nested unknown-member handling.
The internal decoder's second pass may partially populate its target on failure.
No runtime repair or conformance success is claimed for these behaviors.

Replay acceptance can occur despite already-collected violations or before later
core validation rejects a request; no rollback of external consumption is supplied. Policy strings are authenticated but an expected
application policy identity is not compared by the codec. Key retirement does not
invalidate an operation which already copied a key. SQL fragments omit pagination,
seek predicates, joins and include execution; application statement construction owns them.

Unknown rationale: why FilterDefinition.Protected is retained as unused metadata;
whether future wire decoders should reject null, case aliases and nested extra members;
whether replay should be deferred until the full compile outcome. These are separate
behavioral decisions, not prose-pass authority or invented guarantees.

## Selected checks

Pinned Go 1.27.2 in one serialized session: gofmt structure, all tracked comment-free
AST/signature/literal/token identity, exact compiler/analyzer directives, all 14 operational
package GoDoc renders and parsed historical comments; native docs target and two existing
Examples. Optional implicit semicolons immediately before closing braces may differ only
when canonical field GoDoc expands a one-line declaration; report raw differences explicitly.
`.golib.yaml` selects apidiff `api/v4.txt`, not full GoDoc snapshots; keep all four API
baseline files byte-exact. No live database or unchanged runtime campaign is applicable.

Also select shared source inventory/check, affected local links and current maintained
prose, marker/generation inspection, pinned added-content/new-file secrets, Git whitespace,
ordinary complete final-diff review and explicit Intent/ACK safe-wrapper commit.

Results are recorded below after execution; task-owned caches and evidence are removed
on every exit. Remote main/CI/delivery remain coordinator-owned.

## Results

- Go 1.27.2 `gofmt -l` over all 64 tracked Go files: clean after formatting
  only changed source. The disposable native parser/scanner compared every file
  with the recorded base: exact comment-free AST/signatures, raw tokens/literals
  including implicit semicolons, and analyzer/compiler directives all equal.
  No optional-semicolon normalization was needed. All 432 comment groups parsed
  and rendered, including frozen sources without executing their harnesses.
- Native `go doc -all` for each of the 14 operational packages rendered and was
  inspected; `make -f verification/package.mk docs` passed. Apidiff mode means
  no generated full-prose snapshot applies. All four API baselines, module and
  dependency inputs, fixtures and historical authorities remain unchanged.
- `go test -run=^Example -count=1 -timeout=2m . ./adapters/validation` passed
  both existing Examples. No permanent test or example expressions changed.
- Seven disposable read-only decoder observations passed: RPC top-level null,
  null field, case alias, invalid UTF-8 repair, Value null payload/extra member,
  nested Value extra member through RPC, and strictjson partial target population
  on unknown-field failure. These demonstrate existing limitations, not repaired
  behavior or protocol conformance. The probe source was removed immediately.
- Shared `source-docs check --json`: zero findings after useful member contracts.
  Affected local links, generation/marker inspection, pinned Gitleaks v8.30.1
  added/staged-content scan, Git whitespace and complete ordinary Tier A review
  are the final non-Go checks; results are included in the delivery handoff.
- The bounded session's exact GOCACHE, GOMODCACHE, GOTMPDIR, helper and rendered
  output were removed and absence verified. Read-only downloaded module directory
  permissions were made writable only inside the owned cache for cleanup.
  Shared tooling, original checkout and the detached task worktree are preserved.

Unverified: live PostgreSQL execution, race/fuzz/mutation/performance campaigns,
external protocol qualification, collaborator deployments and hosted CI/main
delivery. These are not new claims of this documentation-only source pass.
