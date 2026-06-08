# 300 Client Public Read Sequence

## User Story

As a device-side developer, I want to prepare one client context and then read
the peer-facing RPC path without relying on removed `play` CLI subcommands.

## Scenario

1. Start a real server with automatic peer creation enabled.
2. Create one device context.
3. Prepare that context through the harness API using the device context.
4. Read the stored device info through peer RPC.
5. Verify the same context still answers `gizclaw peer ping`.

## Covered Behaviors

- one client context can be prepared without `play register`
- the peer RPC info read path succeeds after preparation
- the same context still answers `ping`
