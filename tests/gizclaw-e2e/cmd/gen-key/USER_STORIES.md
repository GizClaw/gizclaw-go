# Gen Key CLI

## User Story

As a developer, I want `gizclaw gen-key` to emit a valid private key for local contexts and server fixtures.

## Covered Behaviors

- `gizclaw gen-key` succeeds without arguments.
- The printed value is a valid GizClaw private key that can derive a key pair.
- Extra positional arguments fail.
