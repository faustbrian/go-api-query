# OpenRPC and JSON-RPC

[Documentation index](README.md) · [Repository entry](../README.md)

Import `github.com/faustbrian/go-api-query/v5/adapters/jsonrpc` with its default
package identifier `apiqueryjsonrpc`. `apiqueryjsonrpc.Parse` accepts
bounded JSON parameters, normally using `schema.Bounds().MaxRequestBytes`.
Exact duplicate names, ordinary unknown fields, trailing data, malformed JSON,
excess nesting and excess bytes fail with `ErrInvalid`. `Params` uses pointers
so explicit zero/empty values differ from absence; `Params.Request` deep-copies
slices and filters. Existing wire limitations are not conformance guarantees:
top-level null becomes empty parameters, null optional members become absent,
field matching is case-insensitive, and invalid UTF-8 can be repaired by
`encoding/json`. `Value.UnmarshalJSON` accepts nested extra members. Applications
requiring stricter wire shape must enforce it before accepting parameters.

`OpenRPCContentDescriptor` returns a minimal optional content descriptor named
`query`, with `additionalProperties: false`. The returned map is caller-owned.
Applications should enrich its filter, sort, and page schemas from their static
published contract, not by reflecting domain models or exposing authorization-
specific hidden capabilities.

An RPC method should parse bounded transport input, compile with request policy,
translate `Violations` to its stable application error envelope, execute an
application-owned adapter, and use `cursor.Page` for response boundaries.
OpenRPC generation must not compile or execute a request.

Choose the next guide in the [documentation index](README.md)
or return to the [repository entry](../README.md) for installation and ownership.
