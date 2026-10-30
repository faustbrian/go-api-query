# API reference

[Documentation index](README.md) · [Repository entry](../README.md)

## Schema declarations

`SchemaConfig` declares `Resource`, semantic `Revision`, fields, filters, sorts,
relationships, default sorts, allowed logical operators, pagination, and hard
`Bounds`. `NewSchema` validates and defensively copies it. Caller mutation after
construction cannot alter the schema.

`FieldDefinition` controls name, type, default response membership, required
execution membership, deprecation, and cost. Required fields may be omitted
from `Plan.ResponseFields` but always remain in `Plan.ExecutionFields`.

`RelationshipDefinition` is an explicit resource edge with nested edges and
cost. Paths use dots. Schema construction rejects duplicate names and resource
cycles; compilation enforces depth, count, and authorization on every traversed
edge.

`FilterDefinition.Protected` is descriptive metadata, not enforced authorization.
`FilterDefinition` declares one type and exact operator set plus
deprecated, nullable, empty-string, and cost behavior. Supported operators are
`eq`, `neq`, `lt`, `lte`, `gt`, `gte`, `in`, `not_in`, `between`, `is_null`,
`contains`, `starts_with`, and `ends_with`. Operators remain type-checked.
Logical `and`, `or`, and unary `not` work only when declared.

`SortDefinition` declares type, fixed null placement, cost, and whether it is
the single stable tie-breaker. Cursor schemas require exactly one tie-breaker.
Every cursor plan appends it when absent, after authorization.

`PaginationDefinition` independently enables cursor and offset modes, requires
a bounded default page size, and caps offsets. Offset mode has no cursor
stability claim.

Zero `Bounds` receive conservative nonzero defaults. Positive declarations replace
them and may be larger; negative bounds invalidate the schema. `Schema.Bounds`
returns the normalized copy so transport adapters can use `MaxRequestBytes`.
Cost is a deterministic schema weight with overflow-safe aggregation, not a
database optimizer estimate.

## Requests and values

`Request` preserves absent versus explicit empty fields, includes, sorts, and
schema revision with `Optional[T]`, constructed by `Present`. `FilterExpr` is
either one `Predicate` or one bounded logical group. `PageRequest` selects
`none`, `cursor`, or `offset`; `after` and `before` conflict.

`Value` is a closed typed representation. Constructors are `StringValue`, `IntValue`,
`UintValue`, finite `FloatValue`, `BoolValue`, UTC `TimeValue`, `BytesValue`, and
`NullValue`. Null is valid only for cursor positions, not mandatory constraints,
schema field types or typed filter values. JSON contains an explicit type and
canonical string representation when decoded successfully. Direct string and time
constructors do not validate UTF-8 or the RFC3339 year range, and compilation
does not reparse their payload text. `FloatValue` returns invalid zero Value
for NaN/infinities. `Value.UnmarshalJSON` is not a strict object-shape decoder: it
inherits member matching/null behavior and accepts unknown members.

## Compilation and plans

`Compile(ctx, schema, request, options)` returns either a reviewed immutable
`Plan` or `*Violations`. `CompileOptions.Authorize` receives only a declared
`Capability{Kind, Name}`. `MandatoryConstraints` remain a separate server-owned
list. `CursorDecoder` is required whenever an after/before token is present; the
core verifies returned direction, position count/types, and bounds. A nil
authorizer allows declared capabilities; applications needing authorization
should always provide one. `FilterDefinition.Protected` does not supply one.

Compile forwards context to callbacks but does not check cancellation itself.
Callbacks are synchronous and borrowed; concurrent callers own callback safety.
A cursor replay store can consume a token before later compile rejection, without
rollback. Authenticate and compare application cursor policy identity explicitly.
Inputs must not be mutated during compilation; `Optional` itself does not deep-copy.

Plan accessors return defensive copies: resource and revision, response and
execution fields, includes, deep-cloned filter, mandatory constraints, ordered
sorts, page request, and cost. `Canonical` returns deterministic bounded JSON
for cache keys, signatures, conformance, or equality. The size check occurs after
encoding; Compile success does not ensure Canonical success. Randomized raw cursor
tokens never enter a successful plan or canonical output; canonical pages use a
digest of authenticated `CursorState`. `Plan.Cursor` returns a defensive copy
for seek compilation. Canonical output is not an execution format and can contain
sensitive filter and protected-constraint values; store it accordingly.

## Errors

`Violations.Items` returns defensive copies with stable `Code`, `Path`, and safe
`Message`. Codes are `invalid_element`, `unsupported_operation`, `conflict`,
`authorization_rejected`, `cost_limit`, `cursor_failure`, `version_mismatch`,
and `limit_exceeded`. Paths identify request structure, never raw values.
Collectors stop at `Bounds.MaxErrors`.

Transport parsers expose one sanitized sentinel error. Cursor decoding exposes
`ErrInvalid`, `ErrExpired`, `ErrVersion`, `ErrSchema`, `ErrSort`, and `ErrReplay`
without payload content. PostgreSQL and JSON:API bridges fail with sanitized
adapter errors. The Validation adapter import path is
`github.com/faustbrian/go-api-query/v5/adapters/validation`;
`apiqueryvalidation.Report` converts query violations into an
immutable `validation` report and sanitizes unrelated errors. Applications must
branch on `ctx.Err()` and apply their cancellation or deadline policy before
projecting a compile error as validation.

## Adapter packages

- Import `github.com/faustbrian/go-api-query/v5/adapters/http` and use
  `apiqueryhttp.Parse` to strictly parse one bounded raw query string.
- Import `github.com/faustbrian/go-api-query/v5/adapters/jsonrpc` and use
  `apiqueryjsonrpc.Parse` to strictly parse bounded parameters; `Params.Request`
  returns a defensive request; `OpenRPCContentDescriptor` returns fresh maps.
- Import `github.com/faustbrian/go-api-query/v5/adapters/legacylist` only for
  migrated Laravel-era list methods: `Profile.ParseFilters` and
  `Profile.ParseSorts` read resource-keyed members, `DecodeCursor` and
  `EncodeCursor` handle unsigned paginator cursors, and `Page.Links` builds
  self/prev/next links. See the [legacy list profile](legacy-list.md).
- Import `github.com/faustbrian/go-api-query/v5/adapters/jsonapi` and use
  `apiqueryjsonapi.FromQuery` to consume a query parsed by `jsonapi`; callbacks
  explicitly own filter and pagination profile semantics.
- Import `github.com/faustbrian/go-api-query/v5/adapters/postgres` and use
  `apiquerypostgres.NewCompiler` to snapshot allowlisted mappings. `Compile` returns
  projection, where, order, and typed positional arguments, never execution.
- `cursor.NewKeyring`, `Rotate`, `NewCodec`, `Encode`, `DecodeContext`,
  `Decode`, and `BuildPage` implement the cursor protocol and response
  envelope. `ReplayStore` is the context-aware, concurrency-safe extension
  point for one-time cursor consumption.
- `apiquerytest` provides schema/request builders, `MustCompile`, violation and
  canonical assertions, an order fixture, and cross-decoder conformance suite.

Version 5 removed the former top-level `apiqueryhttp`, `apiqueryjsonapi`,
`apiquerypgx`, `apiqueryrpc`, and `apiqueryvalidation` packages; see the
[migration guide](migration-adapters.md#adopting-v5).

Choose the next guide in the [documentation index](README.md)
or return to the [repository entry](../README.md) for installation and ownership.
