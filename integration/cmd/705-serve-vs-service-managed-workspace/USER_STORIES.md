# 705 Serve Vs Service Managed Workspace

## User Story

As an operator, I want foreground `gizclaw serve` and system service management
to be mutually exclusive for the same workspace, so a `serve -f` command cannot
silently interfere with a workspace installed under `gizclaw service`.

## Scenario

1. Create an isolated server workspace.
2. Read `gizclaw serve --help` and verify it describes foreground serving,
   `--force`, and the service-managed workspace boundary.
3. Write a service-management marker into the workspace.
4. Run `gizclaw serve <workspace>`.
5. Run `gizclaw serve -f <workspace>`.
6. Verify both commands fail with a message that points users back to
   `gizclaw service`.

## Covered Behaviors

- `serve` help distinguishes foreground serving from service management.
- `--force` is documented as foreground-process replacement, not system service
  control.
- Service-managed workspaces reject plain `serve`.
- Service-managed workspaces also reject `serve -f`.
