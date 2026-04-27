# 508 Admin Workspaces

## User Story

As an admin operator, I want to read workspace instances and their template
references through the CLI.

## Scenario

1. Start a real server and provision an admin-capable CLI context.
2. Seed a valid workspace template through the harness API.
3. Seed a workspace from that template with parameters through the harness API.
4. List workspaces and verify the workspace appears.
5. Get the workspace by name and verify the template reference.

## Covered Behaviors

- Workspace `list` and `get` work through the `workspaces` namespace.
- Parameter maps round-trip through CLI JSON output.
- The CLI resource surface is read-only; test setup uses the harness/API to
  prepare state.
