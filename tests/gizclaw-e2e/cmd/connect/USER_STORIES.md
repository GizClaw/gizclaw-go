# Connect CLI

## User Story

As a device-side developer, I want `gizclaw connect` commands to work against the setup server with saved CLI contexts.

## Covered Behaviors

- Context-backed `connect ping` works across repeated, concurrent, reconnect, and missing-server cases.
- Public server reads and HTTP login paths work from real CLI contexts.
- Peer metadata preparation remains observable through connect/admin CLI flows.
- Invalid or missing contexts fail with user-facing errors.
