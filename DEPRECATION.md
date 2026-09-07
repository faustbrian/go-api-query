# Deprecation Policy

Deprecations MUST identify the replacement, reason, migration steps, and
earliest removal version. Public Go identifiers use a valid `Deprecated:` doc
paragraph and corresponding changelog entry.

At v1 and later, removal requires the longer of 180 days after a replacement is
publicly consumable and two stable minor releases containing both paths. The
five released adapter packages and their preferred replacements are listed in
[`docs/migration-adapters.md`](docs/migration-adapters.md). Removal additionally
requires every owned consumer to migrate, fresh public-usage and clean-consumer
evidence, and a separately authorized next-major release. Security or
correctness fixes continue on both paths throughout the interval.

Silent behavior changes, undocumented aliases, and indefinite deprecated code
are prohibited. Deprecations are checked during compatibility and release
review.
