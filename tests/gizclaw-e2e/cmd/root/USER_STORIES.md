# Root CLI

## User Story

As a developer, I want the root `gizclaw` command to expose the supported command groups and keep compatibility for legacy single-dash long flags.

## Covered Behaviors

- `gizclaw --help` lists the top-level command groups.
- Legacy single-dash long flags are normalized before command dispatch.
