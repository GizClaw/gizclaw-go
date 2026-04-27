# 706 Play UI Only

## User Story

As a user looking for device-side play capabilities, I want the CLI to direct me
to the Play UI instead of exposing partially aligned `play` subcommands.

## Scenario

1. Run `gizclaw play --help`.
2. Verify the help exposes `--listen` for starting the Play UI.
3. Verify help no longer lists `register`, `config`, `ota`, or `serve` as
   subcommands.
4. Try each removed subcommand and verify it fails as an unknown command.
5. Run `gizclaw play` without `--listen` and verify it prints help rather than
   starting any device-side operation.
6. Start a real server, create an unregistered device context, then run
   `gizclaw play --listen <addr> --context <name>`.
7. Verify an admin context can read the device registration and that it is marked
   `auto_registered`.

## Covered Behaviors

- `play` remains as a UI entrypoint.
- `play register`, `play config`, `play ota`, and `play serve` are removed from
  the CLI surface.
- Running `play` without `--listen` is safe and informational.
- Running `play --listen` automatically registers the current context before the
  UI is served.
