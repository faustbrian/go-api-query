# Compatibility policy

[Documentation index](README.md) · [Repository entry](../README.md)

Semantic versioning applies to exported Go APIs and documented query behavior.
The v5 export baseline is `api/v5.txt`; earlier published baselines are retained.
`api/v3.txt` preserves the released v3 API byte-for-byte.
`api/v2.txt` preserves the
released v2 API byte-for-byte; `api/v1.txt` preserves the
byte-exact released v1.1.1 API from commit
`47773a680c06821214acc7eac92d2d2be7c85ca5`; it is historical evidence rather
than the current-major check baseline. See [adopting v3](migration-adapters.md#adopting-v3)
for the retained cursor migration, [adopting v4](migration-adapters.md#adopting-v4)
for the JSONAPI v2 named-type boundary, and [adopting v5](migration-adapters.md#adopting-v5)
for the adapter paths. V5 selects Validation v3.

Version 5 removed the five top-level adapter paths in favour of `adapters/`
before their announced compatibility interval ended. Published v4 releases
keep them and remain available; no further v4 releases are planned. See
[adapter migration](migration-adapters.md#adopting-v5).

The following are public contracts and require migration planning: capability
names, types, operators, default fields, required execution fields, relationship
paths, default or tie-breaker sorts, null placement, page defaults and limits,
costs, schema revisions, cursor versions/policies, canonical plan bytes, error
codes and paths, transport names, and page envelope JSON.

Adding an optional capability can be backward compatible, but changing defaults
or cost rejection can alter existing requests. Removing or renaming anything,
tightening a bound below observed use, changing canonical output, or changing a
sort requires a schema revision. Cursor-incompatible changes also require a new
cursor protocol version or rejection of old tokens.

Deprecated client-selected fields and filters fail compilation rather than
silently changing meaning. Required execution fields are server dependencies
and remain independent of client deprecation/authorization. Keep the old schema
revision available for an announced window or
return `version_mismatch` and migration guidance at the application layer.

Choose the next guide in the [documentation index](README.md)
or return to the [repository entry](../README.md) for installation and ownership.
