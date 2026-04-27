# 507 Admin Workspace Templates

## User Story

As an admin operator, I want to read workflow template documents through the CLI
so workspace definitions can be reviewed without exposing per-resource write
commands.

## Scenario

1. Start a real server and provision an admin-capable CLI context.
2. Seed a valid single-agent workflow template through the harness API.
3. Seed a second valid template through the harness API.
4. List templates and verify both names appear.
5. Get one template and verify its `kind`.

## Covered Behaviors

- Template `list` and `get` work through the `workspace-templates` namespace.
- JSON union documents round-trip through CLI output.
- The CLI resource surface is read-only; test setup uses the harness/API to
  prepare state.
