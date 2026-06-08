# 302 Client Public Retryable Reads

## User Story

As a device-side developer, I want repeated peer RPC reads to keep working so
simple polling workflows stay reliable after `play config` is removed from the
CLI surface.

## Scenario

1. Start a real server with automatic peer creation enabled.
2. Create and prepare one device context through the harness API.
3. Read peer info several times through RPC.
4. Verify `gizclaw peer ping` still succeeds after each read.

## Covered Behaviors

- one prepared context can issue repeated peer RPC reads
- repeated peer info reads and `ping` commands keep succeeding
- the scenario no longer depends on removed `play` CLI subcommands
