# 705 Serve Vs Service Managed Workspace

## User Story

As an operator, I want plain `gizclaw serve` to stay protected while the explicit
foreground/server-service launch path uses one command shape:
`gizclaw serve --force <workspace>`.

## Scenario

1. Create an isolated server workspace.
2. Read `gizclaw serve --help` and verify it describes foreground serving,
   `--force`, and the service-managed workspace boundary.
3. Run `gizclaw serve <workspace>`.
4. Verify the command fails with guidance to use `gizclaw service` or explicit
   `--force`.

## Covered Behaviors

- `serve` help distinguishes foreground serving from service management.
- `--force` is documented as the explicit foreground local serve opt-in.
- Plain `serve` rejects direct starts.
- System service installation uses the same `serve --force <workspace>` command
  shape instead of a private runtime flag.
