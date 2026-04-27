# 300 Client Public Read Sequence

## User Story

As a device-side developer, I want to register one client context and then read
the public gear configuration path without relying on removed `play` CLI
subcommands.

## Scenario

1. Start a real server with a device registration token.
2. Create one device context.
3. Register that context through the harness API using the device token.
4. Read the device configuration through the gear public API.
5. Verify the same context still answers `gizclaw ping`.

## Covered Behaviors

- one client context can register with the server without `play register`
- the public configuration read path succeeds after registration
- the same context still answers `ping`
