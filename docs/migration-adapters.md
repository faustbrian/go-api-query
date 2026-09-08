# Adapter package migration

New integrations use the target-oriented adapter packages. The released paths
remain supported through the compatibility interval.

| Released path | Preferred path | Default identifier |
| --- | --- | --- |
| `github.com/faustbrian/go-api-query/apiqueryhttp` | `github.com/faustbrian/go-api-query/adapters/http` | `apiqueryhttp` |
| `github.com/faustbrian/go-api-query/apiqueryjsonapi` | `github.com/faustbrian/go-api-query/adapters/jsonapi` | `apiqueryjsonapi` |
| `github.com/faustbrian/go-api-query/apiquerypgx` | `github.com/faustbrian/go-api-query/adapters/postgres` | `apiquerypostgres` |
| `github.com/faustbrian/go-api-query/apiqueryrpc` | `github.com/faustbrian/go-api-query/adapters/jsonrpc` | `apiqueryjsonrpc` |
| `github.com/faustbrian/go-api-query/apiqueryvalidation` | `github.com/faustbrian/go-api-query/adapters/validation` | `apiqueryvalidation` |

The old and new paths have equivalent supported behavior and share their
initialized sentinel error identities. Successor `Config`, decoder, `Mapping`,
`Compiler`, `QueryParts`, `Params`, and `ContentDescriptor` declarations are
new named types owned by their new packages, not aliases. Their reflection
paths therefore differ, and stored or exchanged package-owned values require
explicit field-by-field conversion. Root `apiquery`, `cursor`, and
`apiquerytest` imports do not change.

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
