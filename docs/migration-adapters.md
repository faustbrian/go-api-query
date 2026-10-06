# Adapter package migration

## Adopting v4

This root source prepares unpublished v4. Stay on public APIQuery v3.0.0 until
the v4 release and clean-public-consumer boundary have passed. Source stays on
main at the repository root; all package locations and both JSONAPI adapter
variants are retained without version-specific source directories.

The dependency selects published `github.com/faustbrian/go-jsonapi/v2@v2.0.0`
from qualified main source `41700010871799526a9dcfd7e96affe98bdecbba`.
Its source tree is unchanged from the reviewed development dependency. The
generated `api/v4.txt` records the development contract. APIQuery v4 independent
review, hosted CI, public release and clean-public-consumer verification remain
prerequisites for v4 adoption; selecting stable JSONAPI v2 does not satisfy them.

After those boundaries pass, replace `/v3` with `/v4` in root and subpackage
imports and select `github.com/faustbrian/go-jsonapi/v2`. Both
`adapters/jsonapi` and `apiqueryjsonapi` now accept JSONAPI v2 `Query` values.
Their `FilterDecoder` and `PageDecoder` parameters use the v2
`ParameterFamily`; v1 and v2 named types are not interchangeable. Preserve
application-owned filter and page semantics, callback copy boundaries and
sentinel handling. The adapter does not silently add a cursor profile decoder.

Applications choosing JSONAPI v2 cursor pagination must provide a positive,
finite `CursorPaginationConfig.MaxSize`; unbounded v1 configuration is no
longer supported. Published APIQuery v3 snapshots remain unchanged. Direct
Localized query adapters and maintained Tools composition need explicit
adoption of the versioned public types rather than implicit source replacement.

## Adopting v3

The retained v3 migration introduced the published v3.0.0 root module. Require
`github.com/faustbrian/go-api-query/v3@v3.0.0` and replace `/v2` with `/v3`
in root and subpackage imports. All fourteen package locations remain at the
repository root on main; Go 1.27.0 and Validation v2.0.0 remain unchanged.
API Query v2 and v3 named types are distinct; migrate each composition boundary
explicitly. This does not migrate Localized or other owned consumers.

Remove non-nil `cursor.Config.Random` and migrate `Config.ReplayGuard` to
`Config.ReplayStore`. The store receives the request context, opaque fingerprint
and expiry; it owns atomic consumption, concurrency safety and bounded retention.
Use `DecodeContext` or `DecodeCursor` when replay storage is configured because
`Decode` now fails closed in that configuration. Observed cancellation before
the store call prevents invocation; store acceptance remains the commit point even when
cancellation races with return. A store which ignores context can still block
the synchronous caller. See [cursor ownership](cursor.md) and
the [versioned threat model](threat-model.md).

Cursor wire formats, query algorithms and Validation v2 adoption are otherwise
unchanged. The published snapshots `api/v1.txt`, `api/v2.txt` and `api/v3.txt`
are preserved.

## Adopting v2

After v2.0.0 is publicly available, require
`github.com/faustbrian/go-api-query/v2@v2.0.0` and insert `/v2` immediately
after `go-api-query` in every root and subpackage import. All fourteen package
locations remain, including the five retained deprecated adapters. Source
stays at the repository root on main; no version source directory is needed.
Go 1.27.0 remains the minimum.

Use `github.com/faustbrian/go-validation/v2@v2.0.0` when constructing limits
or consuming reports from either Validation adapter. Their public `Report`
functions accept Validation v2 `Limits` and return Validation v2 `Report`.
API Query v1/v2 and Validation v1/v2 types, interface method signatures and
reflection paths have distinct Go identities; do not mix them at a boundary.
Existing v1 consumers, including Localized, may keep their published v1 pins.
Localized adoption requires a separate explicit public-contract migration;
this release does not imply that it has already occurred.

Query algorithms, request schemas, plans, canonical output, wire formats,
cursor protocols and persistence semantics are unchanged by this identity
migration. Re-run application composition fixtures when changing imports.
Validation v2's private default diagnostics and composition limits remain
governed by its own published migration contract, not by API Query error prose.

The exact released v1.1.1 export snapshot remains in `api/v1.txt`.
`api/v2.txt` preserves that released major's export snapshot.

## Selecting adapters within v3

The table below uses published v3 module identities. Published v1 and v2 paths remain
resolvable under their existing public versions; retaining these v3 package locations
does not preserve cross-major named-type identity.

New integrations use the target-oriented adapter packages. The released paths
remain supported through the compatibility interval.

| Retained path | Preferred path | Default identifier |
| --- | --- | --- |
| `github.com/faustbrian/go-api-query/v3/apiqueryhttp` | `github.com/faustbrian/go-api-query/v3/adapters/http` | `apiqueryhttp` |
| `github.com/faustbrian/go-api-query/v3/apiqueryjsonapi` | `github.com/faustbrian/go-api-query/v3/adapters/jsonapi` | `apiqueryjsonapi` |
| `github.com/faustbrian/go-api-query/v3/apiquerypgx` | `github.com/faustbrian/go-api-query/v3/adapters/postgres` | `apiquerypostgres` |
| `github.com/faustbrian/go-api-query/v3/apiqueryrpc` | `github.com/faustbrian/go-api-query/v3/adapters/jsonrpc` | `apiqueryjsonrpc` |
| `github.com/faustbrian/go-api-query/v3/apiqueryvalidation` | `github.com/faustbrian/go-api-query/v3/adapters/validation` | `apiqueryvalidation` |

The old and new paths have equivalent supported behavior and share their
initialized sentinel error identities. Successor `Config`, decoder, `Mapping`,
`Compiler`, `QueryParts`, `Params`, and `ContentDescriptor` declarations are
new named types owned by their new packages, not aliases. Their reflection
paths therefore differ, and stored or exchanged package-owned values require
explicit field-by-field conversion. Within this major, selecting a successor
adapter does not change root `apiquery`, `cursor`, or `apiquerytest` imports.

Applications may migrate the five adapters independently. Update one import,
add explicit conversion only where a package-owned named value crosses an API
boundary, and rerun that consumer's compile and behavioral checks. Query wire
formats, schemas, plans, and persistence semantics do not migrate.

The compatibility interval ends only after both 180 days from the first public
successor release and two stable minor releases containing both paths. Removal
also requires migration of every owned consumer, fresh public-usage and clean
external-consumer evidence, and a separately authorized next-major release.
Correctness and security fixes continue on both paths throughout the interval.

Before publication, rollback is a normal revert of the coherent change. After
publication, keep both paths resolvable, roll back consumer pins or imports
independently when necessary, and publish a forward patch for successor defects.
Do not delete, move, or replace the published release tag.
