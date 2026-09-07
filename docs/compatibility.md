# Compatibility policy

On the stable v1 line, semantic versioning applies to exported Go APIs and documented query
behavior. The `api/v1.txt` export baseline blocks incompatible Go API changes.

The five released adapter paths remain supported while new adoption uses the
five target-oriented `adapters/` paths. Their compatibility interval is the
longer of 180 days after the successors become publicly consumable and two
published stable minor releases containing both old and new paths. Removal also
requires all owned consumers to migrate, fresh public and clean-consumer proof,
and a separately authorized next-major release. See
[adapter migration](migration-adapters.md).

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

Deprecated declarations fail compilation rather than silently changing
meaning. Keep the old schema revision available for an announced window or
return `version_mismatch` and migration guidance at the application layer.
