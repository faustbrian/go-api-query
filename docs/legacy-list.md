# Legacy JSON-RPC list profile

[Documentation index](README.md) · [Repository entry](../README.md)

`github.com/faustbrian/go-api-query/v5/adapters/legacylist` (package
`apiquerylegacylist`) parses the list grammar that Laravel-era JSON-RPC methods
expose, so migrated services keep their wire contract without each keeping a
copy of the parser. New methods should declare a schema and use
`adapters/jsonrpc` with authenticated cursors instead.

## Contents

- [Request members](#request-members)
- [Cursors and links](#cursors-and-links)
- [Errors](#errors)
- [Authorization boundary](#authorization-boundary)
- [Decisions](#decisions)

## Request members

Declare what one method accepts in a `Profile`, then parse the raw `filters`
and `sorts` members of the params object:

```json
{
  "filters": {
    "self": [
      {"attribute": "status", "operator": "in", "value": ["delivered"]},
      {"attribute": "name", "operator": "not_like", "value": "test%", "boolean": "or"}
    ],
    "vendor": [{"attribute": "slug", "operator": "equals", "value": "posti"}]
  },
  "sorts": {"self": [{"attribute": "created_at", "direction": "desc"}]},
  "page": {"size": 25, "cursor": "eyJpZCI6MTAsIl9wb2ludHNUb05leHRJdGVtcyI6dHJ1ZX0"}
}
```

- Filters and sorts are objects keyed by resource name. Each resource holds an
  array of entries. Absent or `null` members and `null` resource arrays mean
  none.
- Operators are the fourteen long names `equals`, `not_equals`,
  `greater_than`, `greater_than_or_equal_to`, `less_than`,
  `less_than_or_equal_to`, `like`, `not_like`, `in`, `not_in`, `between`,
  `not_between`, `is_null` and `is_not_null`, matched case-sensitively.
- `boolean` is `and` or `or`; absent, empty or `null` means `and`.
- `direction` is `asc` or `desc` and has no default.
- `value` is returned as exact JSON bytes. `Filter.ValidateValue` checks the
  arity that `in`, `not_in`, `between` and `not_between` need; value typing
  stays with the method.
- `Profile.MaxBytes` bounds each raw member before decoding.

Page size, sparse `fields` and `relationships` differ between the existing
methods, so the profile leaves them to the caller.

## Cursors and links

Cursors use Laravel `CursorPaginator` encoding: one JSON object of the sort
values of the boundary item, followed by a boolean `_pointsToNextItems`, in
unpadded Base64url.

- `DecodeCursor` requires that boolean exactly once, at least one other
  parameter, unique non-empty names and nothing after the object. It keeps the
  parameter order and exact value bytes.
- `Cursor.Values` returns the values in the requested sort order only when the
  cursor holds exactly those parameters.
- `EncodeCursor` writes the parameters in order and the direction last.
- `Page.Links` applies the paginator rules. `self` re-encodes the request
  cursor. With items, `prev` points back from the first item when the request
  cursor points forward or more items remain, and `next` points forward from
  the last item when more items remain or the request cursor points backward.
  An empty `Links` means the method omits page metadata. `Links` marshals as
  `{"self", "prev", "next"}`.

## Errors

Every rejected parameter is an `*Error` wrapping `ErrInvalid`. `Kind` says what
failed (`KindShape`, `KindResource`, `KindAttribute`, `KindOperator`,
`KindBoolean`, `KindDirection`, `KindValue`), `Pointer` where, `Unknown` the
rejected client name and `Allowed` the declared alternatives. Methods map these
to their own legacy error documents. `Error()` never repeats client input.
Cursor and profile failures are `ErrInvalidCursor` and `ErrInvalidProfile`.

## Authorization boundary

A declared attribute only says the method can filter or sort by it. Parsing
grants nothing: tenant scoping, per-caller field policy and record visibility
stay in the method and its store. Cursors are neither signed nor encrypted.
Treat every decoded value as client input, bind it as a query parameter, and
never put secrets or authorization state in a cursor.

## Decisions

These choices unify the Location, Track and Postal list methods in Mono. Each
service keeps its own error documents and page-size rules.

- **Booleans are validated everywhere.** Location and Postal reject anything
  but `and` and `or`; Track passed other values through and ignored an invalid
  boolean on the first filter. The profile rejects them with `KindBoolean`.
- **Undeclared resources are rejected unless the profile opts out.** Location
  rejects them; Track and Postal read only `self` and ignore the rest, which a
  profile keeps with `IgnoreUndeclaredResources`.
- **Error precedence follows Location.** For filters, declared resources are
  checked before an undeclared one is reported; for sorts, undeclared
  resources are reported first. The alphabetically first undeclared name is
  reported so the result is deterministic.
- **`null` filters and sorts mean none.** Postal rejected `"filters": null`;
  Location and Track accepted it.
- **Cursors are strict.** The direction member is required, as in Location and
  Postal (Track defaulted it to `false`), and duplicate names are rejected
  (Postal kept the last). Value types such as Track's and Postal's strings stay
  method checks.
- **Value arity is a separate check.** Only Postal checked `in` and `between`
  arity before querying, so `ValidateValue` is opt-in.
- **Encoding uses `encoding/json` escaping.** PHP's `json_encode` escapes `/`
  and non-ASCII characters, so cursor bytes can differ from ones Laravel
  issued. Clients treat cursors as opaque and both forms decode.

Choose the next guide in the [documentation index](README.md)
or return to the [repository entry](../README.md) for installation and ownership.
