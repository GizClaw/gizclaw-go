# 507 Admin Workflows

## User Story

As an admin operator, I want to read workflow documents through the CLI
so workspace definitions can be reviewed without exposing per-resource write
commands.

## Scenario

1. Start a real server and provision an admin-capable CLI context.
2. Seed a valid flowcraft workflow through the harness API.
3. Seed a second valid workflow through the harness API.
4. List workflows and verify both names appear.
5. Get one workflow and verify its `kind`.

## Covered Behaviors

- Workflow `list` and `get` work through the `workflows` namespace.
- JSON union documents round-trip through CLI output.
- The CLI resource surface is read-only; test setup uses the harness/API to
  prepare state.
