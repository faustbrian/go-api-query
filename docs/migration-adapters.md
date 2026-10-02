# Adapter package migration

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
`api/v2.txt` tracks the current major through the existing owned API generator.

## Selecting adapters within v2

The table below uses v2 module identities. Published v1 paths remain resolvable
under their existing public versions; retaining these v2 package locations
does not preserve cross-major named-type identity.

New integrations use the target-oriented adapter packages. The released paths
remain supported through the compatibility interval.

| Retained path | Preferred path | Default identifier |
| --- | --- | --- |
| `github.com/faustbrian/go-api-query/v2/apiqueryhttp` | `github.com/faustbrian/go-api-query/v2/adapters/http` | `apiqueryhttp` |
| `github.com/faustbrian/go-api-query/v2/apiqueryjsonapi` | `github.com/faustbrian/go-api-query/v2/adapters/jsonapi` | `apiqueryjsonapi` |
| `github.com/faustbrian/go-api-query/v2/apiquerypgx` | `github.com/faustbrian/go-api-query/v2/adapters/postgres` | `apiquerypostgres` |
| `github.com/faustbrian/go-api-query/v2/apiqueryrpc` | `github.com/faustbrian/go-api-query/v2/adapters/jsonrpc` | `apiqueryjsonrpc` |
| `github.com/faustbrian/go-api-query/v2/apiqueryvalidation` | `github.com/faustbrian/go-api-query/v2/adapters/validation` | `apiqueryvalidation` |

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
