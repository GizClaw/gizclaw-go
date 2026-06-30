# Serve CLI

## User Story

As a developer, I want `gizclaw serve` to boot, stop, restart, and validate workspace configs through the real CLI.

## Covered Behaviors

- Foreground serve honors workspace networking, identity, and store configuration.
- Server identity and saved contexts survive restarts.
- Separate workspaces remain isolated.
- Direct serve without `--force` fails with service guidance.
