# PostgreSQL and SQLC

[Documentation index](README.md) · [Repository entry](../README.md)

Core plans contain no SQL concepts. Import
`github.com/faustbrian/go-api-query/v5/adapters/postgres` with its default package
identifier `apiquerypostgres`. `apiquerypostgres.Compiler` is optional and only
maps public capability names to application-reviewed PostgreSQL identifiers.
Mappings are snapshotted and validated as at most three identifier segments.
Client values always become positional arguments.

Applications own the complete statement, including LIMIT/OFFSET, cursor seek
predicates, joins and include fetching. Fragments may be empty when the plan has
no selection, filter or sort; only add SQL keywords for valid nonempty fragments:

```go
parts, err := compiler.Compile(plan)
if err != nil { return err }

// The surrounding statement and joins are a reviewed application variant.
// This variant requires a nonempty projection, predicates and ordering.
sql := "SELECT " + parts.Projection +
    " FROM orders WHERE " + parts.Where +
    " ORDER BY " + parts.OrderBy
```

Never add a public name to a mapping unless its join and data exposure have been
reviewed. Bind each typed `QueryParts.Arguments` value with the driver-native
type appropriate to its `Value.Type`; do not interpolate `Value.String()`.
Mandatory constraint mappings are required and compiled before client filters.

For SQLC, prefer explicit named variants such as `ListOrdersByStatusForward`
and `ListOrdersByStatusBackward`. Dispatch only after matching the reviewed plan
shape. Keep tenant/global predicates literal in every SQLC query. Cursor seek
conditions must implement the exact ordered directions and null placement and
bind decoded positions as parameters. Fetch one extra row to determine
`has_more`; SQLC or pgx remains responsible for execution and scanning.

Choose the next guide in the [documentation index](README.md)
or return to the [repository entry](../README.md) for installation and ownership.
