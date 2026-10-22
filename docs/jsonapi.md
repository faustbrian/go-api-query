# JSON:API composition

[Documentation index](README.md) · [Repository entry](../README.md)

This project does not define JSON:API query syntax. Parse and validate JSON:API
parameters with `github.com/faustbrian/go-jsonapi/v2`; its names, extensions,
profiles, and recommendations remain authoritative. Import
`github.com/faustbrian/go-api-query/v5/adapters/jsonapi` with its default package
identifier `apiqueryjsonapi`, then pass the parsed `jsonapi.Query` to
`apiqueryjsonapi.FromQuery`.

Sparse fieldsets select only the configured resource; includes retain their parsed
paths. Filter
and page parameter families require explicit `FilterDecoder` and `PageDecoder`
callbacks because JSON:API does not assign one universal semantic grammar to
them. An unconfigured present family returns `ErrUnsupported`; callback errors
become sanitized `ErrInvalid`. Family maps and value slices are copied before
callbacks. Returned filter expressions remain borrowed by the request; do not
mutate them during compilation. Callbacks run synchronously without a context
parameter or panic recovery; the application owns concurrency and lifetime.

Do not feed a conventional HTTP query directly to this bridge and do not use it
to override JSON:API member naming or extension negotiation. After composition,
compile the ordinary transport-neutral request and compare its canonical plan
with equivalent JSON-RPC/HTTP requests in conformance tests.

Choose the next guide in the [documentation index](README.md)
or return to the [repository entry](../README.md) for installation and ownership.
