# 506 Admin Voices

## User Story

As an admin operator curating the global voice catalog, I want to read voice
entries and filter the catalog by provider metadata.

## Scenario

1. Start a real server and provision an admin-capable CLI context.
2. Seed a manual voice for provider `main-cn` through the harness API.
3. Seed a second manual voice for provider `other-cn` through the harness API.
4. List all voices and verify both voices appear.
5. List voices with `--provider-name main-cn` and verify only the matching voice
   appears.
6. Get the first voice by id.
7. Seed a Volcengine tenant and voice, then verify the exact resource CLI command
   forms used by the Admin UI can show both the `Voice` and its `VolcTenant`.
8. Run the Volcengine tenant sync CLI command against an intentionally incomplete
   test credential and verify it fails with a clear user-facing error before any
   upstream API call is attempted.

## Covered Behaviors

- `list` and `get` work through the `voices` namespace.
- Voice list filtering by provider name is wired through the CLI.
- The generic resource CLI can show Volc-backed voices and their owning tenants.
- The Volcengine tenant `sync-voices` CLI path returns a deterministic validation
  error for incomplete credentials.
- The CLI resource surface is read-only; test setup uses the harness/API to
  prepare state.
