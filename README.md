# api-query

[![CI](https://github.com/faustbrian/go-api-query/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/faustbrian/go-api-query/actions/workflows/ci.yml)
[![CodeQL](https://img.shields.io/badge/CodeQL-required-blue)](https://github.com/faustbrian/go-api-query/actions/workflows/ci.yml)
[![Coverage](https://img.shields.io/badge/coverage-risk_selected-blue)](CONTRIBUTING.md#verification)
[![Mutation](https://img.shields.io/badge/mutation-optional_local%2Fselected_scheduled-blue)](CONTRIBUTING.md#verification)
[![Documentation](https://img.shields.io/badge/docs-checked_in_CI-blue)](docs/)
[![Go Reference](https://pkg.go.dev/badge/github.com/faustbrian/go-api-query/v5.svg)](https://pkg.go.dev/github.com/faustbrian/go-api-query/v5)
[![Release](https://img.shields.io/github/v/release/faustbrian/go-api-query?sort=semver)](https://github.com/faustbrian/go-api-query/releases)
[![Go](https://img.shields.io/badge/go-1.27.0-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

`api-query` compiles explicitly declared API query capabilities into an
immutable, storage-neutral plan. It covers field projection, relationships,
typed filters, deterministic sorting, cursor or offset pagination, conservative
costs, and strict transport adapters without becoming an ORM or SQL language.

The minimum supported toolchain is Go 1.27.0.

The v5 line uses JSONAPI v2 and Validation v3 named types. Install the
current release:

```sh
go get github.com/faustbrian/go-api-query/v5@v5.0.0
```

Integrations use the packages under `adapters/`. The
[package map](docs/api.md#adapter-packages) and
[adapter migration guide](docs/migration-adapters.md) describe the v5 move
from the removed top-level adapter paths.

For existing applications, read [adopting v5](docs/migration-adapters.md#adopting-v5)
and the earlier sections for the cursor callback migration and cross-major
type boundaries.

## Five-minute JSON-RPC quickstart

The example uses `github.com/faustbrian/go-api-query/v5/adapters/jsonrpc` as
`apiqueryjsonrpc` and `github.com/faustbrian/go-api-query/v5/adapters/validation` as
`apiqueryvalidation`.

```go
schema, err := apiquery.NewSchema(apiquery.SchemaConfig{
    Resource: "orders",
    Revision: "orders-v1",
    Fields: []apiquery.FieldDefinition{
        {Name: "id", Type: apiquery.TypeString, Required: true},
        {Name: "status", Type: apiquery.TypeString, Default: true},
    },
    Filters: []apiquery.FilterDefinition{{
        Name: "status", Type: apiquery.TypeString,
        Operators: []apiquery.Operator{apiquery.OpEqual, apiquery.OpIn},
    }},
    Sorts: []apiquery.SortDefinition{{
        Name: "id", Type: apiquery.TypeString, TieBreaker: true,
    }},
    Pagination: apiquery.PaginationDefinition{
        Cursor: true, DefaultPageSize: 25,
    },
    Bounds: apiquery.Bounds{
        MaxFields: 10, MaxFilterDepth: 3, MaxFilterNodes: 20,
        MaxValues: 40, MaxSorts: 3, MaxPageSize: 100,
        MaxCursorBytes: 2048, MaxCost: 100,
    },
})
if err != nil {
    return err // invalid server declaration
}

params, err := apiqueryjsonrpc.Parse(rawParams, schema.Bounds().MaxRequestBytes)
if err != nil {
    return err // sanitized transport error
}

plan, err := apiquery.Compile(ctx, schema, params.Request(), apiquery.CompileOptions{
    Authorize: func(ctx context.Context, capability apiquery.Capability) bool {
        return policy.Allows(ctx, capability.Kind, capability.Name)
    },
    MandatoryConstraints: []apiquery.Constraint{{
        Name: "tenant_id", Value: apiquery.StringValue(tenantID), Protected: true,
    }},
    CursorDecoder: cursorCodec,
})
if err != nil {
    if terminalErr := ctx.Err(); terminalErr != nil {
        return terminalErr // apply the application's cancellation/deadline policy
    }
    return apiqueryvalidation.Report(err, validation.DefaultLimits())
}

// Only a reviewed plan reaches an application-owned persistence adapter.
parts, err := pgCompiler.Compile(plan)
```

An explicitly empty JSON-RPC array such as `"fields": []` remains different
from an absent `fields` member. Parsing is byte-bounded and rejects exact duplicate
member names and ordinary unknown fields. Existing `encoding/json` limitations
include null-to-absence, case-insensitive field matching, repaired invalid UTF-8
and nested extra members handled by `Value.UnmarshalJSON`; see [the RPC guide](docs/openrpc.md).

## Contract boundary

Servers own every field, filter, operator, relationship, sort, bound, cost,
cursor version, authorization decision, and mandatory predicate. Clients never
supply identifiers, SQL, regular expressions, functions, joins, repositories,
or execution behavior. The core package imports no database or transport.

Start with the [complete API reference](docs/api.md),
[security model](SECURITY.md), and [cursor guide](docs/cursor.md). Transport and
adoption guides cover [HTTP](docs/http.md), [OpenRPC](docs/openrpc.md),
[JSON:API composition](docs/jsonapi.md), [SQLC](docs/sqlc.md), and
[Laravel/Cline RPC migration](docs/migration-laravel-cline.md).
Shared construction, ownership, lifecycle, and composition expectations are in
the versioned [Golib ecosystem index](https://raw.githubusercontent.com/faustbrian/go-library-tools/531e4db50fd81a7201257a7b488a0cf22d333aca/docs/ecosystem/README.md)
and its [Service edge family](https://github.com/faustbrian/go-library-tools/blob/v1.4.0/docs/ecosystem/design-language.md#package-families-and-selection).

## Local quality gates

```sh
go test ./...    # standalone tests; live PostgreSQL skips when unconfigured
make check       # bounded local route with the declared PostgreSQL fixture
make ci          # repository checks plus bounded local verification
```

Verification is provided by the pinned `go-library-tools` release declared in
`.golib.yaml`. PostgreSQL defaults to the shared task-owned fixture; set
`APIQUERY_TEST_DATABASE_URL` to use an existing test database. No package
executes queries or contacts a service at runtime.
`make check-full` preserves full assurance. See
[contributor verification](CONTRIBUTING.md#verification) for service requirements,
focused 2s fuzzing, optional mutation and relevant integration/release boundaries.

## Stability

The repository maintains the v5 line and supports Go 1.27.0. Public
compatibility rules are in
[docs/compatibility.md](docs/compatibility.md), current changes are in
[CHANGELOG.md](CHANGELOG.md). The generated `api/v5.txt` snapshot tracks the
v5 contract. Historical snapshots remain byte-exact in `api/v1.txt`,
`api/v2.txt`, `api/v3.txt` and `api/v4.txt`. Existing consumers must explicitly adopt
the new module path; named types from different majors are not interchangeable.

## Documentation

Start with the [documentation index](docs/README.md) for transport, cursor,
authorization, SQL, compatibility, and operations guidance.

## Documentation navigation

[Complete documentation map](docs/README.md#complete-documentation-map) · [Current source package map](docs/package-map.md)

Use the [versioned ecosystem entry](https://github.com/faustbrian/go-library-tools/blob/v1.4.0/docs/ecosystem) for package
selection. The [frozen compatibility cohort](https://github.com/faustbrian/go-library-tools/blob/v1.8.4/docs/ecosystem/compatibility-sets.md)
is separate from current package releases and unpublished source changes.
Adopting them does not silently update that cohort.
