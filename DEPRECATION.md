# Deprecation Policy

Deprecations MUST identify the replacement, reason, migration steps, and
earliest removal version. Public Go identifiers use a valid `Deprecated:` doc
paragraph and corresponding changelog entry.

At v1 and later, removal requires the longer of 180 days after a replacement is
publicly consumable and two stable minor releases containing both paths, unless
the owner authorizes an earlier major-version removal. Version 5 is such a
removal: it drops the five released adapter packages listed in
[`docs/migration-adapters.md`](docs/migration-adapters.md) and the deprecated
cursor markers below before that interval ended. Published v4 releases keep
them; no further v4 releases are planned.

Silent behavior changes, undocumented aliases, and indefinite deprecated code
are prohibited. Deprecations are checked during compatibility and release
review.

In v3 and v4, `cursor.Config.Random` and context-free `cursor.ReplayGuard`
remained source-compatible migration markers that `cursor.NewCodec` rejected
when non-nil. Version 5 removes them. Remove custom randomness so nonces use
`crypto/rand.Reader`, and migrate replay callbacks to context-aware
`cursor.ReplayStore`. Use `DecodeContext` or `DecodeCursor` for replay-enabled
decoding; `Decode` fails closed when replay storage is configured.
