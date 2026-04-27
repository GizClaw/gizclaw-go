# 302 Client Public Retryable Reads

## User Story

As a device-side developer, I want repeated public reads to keep working so
simple polling workflows stay reliable after `play config` is removed from the
CLI surface.

## Scenario

1. Start a real server with a device registration token.
2. Create and register one device context through the harness API.
3. Read public gear configuration several times through the gear public API.
4. Verify `gizclaw ping` still succeeds after each read.

## Covered Behaviors

- one registered context can issue repeated public reads
- repeated configuration reads and `ping` commands keep succeeding
- the scenario no longer depends on removed `play` CLI subcommands
