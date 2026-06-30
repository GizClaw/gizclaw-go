# Play CLI

## User Story

As a user, I want `gizclaw play` to act as the Play UI launcher and not expose removed device subcommands.

## Covered Behaviors

- `play --help` exposes the UI listen flag.
- Removed `play register`, `play config`, `play ota`, and `play serve` subcommands fail.
- `play --listen` prepares the selected context before serving the UI.
