# Package map

[Documentation index](README.md) · [Repository entry](../README.md)

This map follows the current source modules, not a synchronized release.
Use the repository README for published installation versions; an unpublished
source major is not installable merely because its package appears here.
Internal packages and non-releasable fixtures are intentionally excluded.

## github.com/faustbrian/go-api-query/v5

`api-query` compiles explicitly declared API query capabilities into an immutable, storage-neutral plan. It covers field projection, relationships, typed filters, deterministic sorting, cursor or offset pagination, conservative costs, and strict transport adapters without becoming an ORM or SQL language.

| Import path | Responsibility |
| --- | --- |
| [`github.com/faustbrian/go-api-query/v5`](../compile.go) | Use for declaring bounded query capabilities and compiling authorized transport-neutral requests into immutable plans. |
| [`github.com/faustbrian/go-api-query/v5/apiquerytest`](../apiquerytest/apiquerytest.go) | Use in tests for deterministic builders, fixtures, assertions, and cross-transport conformance. |
| [`github.com/faustbrian/go-api-query/v5/cursor`](../cursor/cursor.go) | Use for bounded authenticated cursor encoding, rotation, expiry, and replay policy. |
| [`github.com/faustbrian/go-api-query/v5/adapters/http`](../adapters/http/http.go) | Strict conventional HTTP query parsing before core compilation. |
| [`github.com/faustbrian/go-api-query/v5/adapters/jsonapi`](../adapters/jsonapi/jsonapi.go) | Maps validated JSON:API query members into the transport-neutral request model. |
| [`github.com/faustbrian/go-api-query/v5/adapters/jsonrpc`](../adapters/jsonrpc/jsonrpc.go) | Bounded JSON-RPC parameters and optional OpenRPC query content. |
| [`github.com/faustbrian/go-api-query/v5/adapters/legacylist`](../adapters/legacylist/legacylist.go) | Parses the legacy Laravel-era JSON-RPC list profile and its unsigned cursors for migrated methods. |
| [`github.com/faustbrian/go-api-query/v5/adapters/postgres`](../adapters/postgres/postgres.go) | Compiles reviewed plans into allowlisted PostgreSQL statement fragments without execution. |
| [`github.com/faustbrian/go-api-query/v5/adapters/validation`](../adapters/validation/validation.go) | Translates query violations into immutable go-validation reports. |

Choose the next guide in the [documentation index](README.md)
or return to the [repository entry](../README.md) for installation and ownership.
