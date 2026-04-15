# 300 Client Public Read Sequence

## User Story

As a device-side developer, I want to register one client context and then read its
public configuration through the CLI.

## Covered Behaviors

- one client context can register with the server
- `play config` succeeds after registration
- the same context still answers `ping`
