# 503 Admin Config Flow

## User Story

As an operator, I want a basic peer configuration flow through the CLI so I can
verify peer config updates end to end.

## Scenario

1. Start a real server from this story's workspace fixture.
2. Provision an admin-capable context.
3. Register a device context.
4. Run `gizclaw admin peers put-config <pubkey> --file <config.json>` and verify
   complete config replacement works through file input.

## Covered Behaviors

- `put-config --file` writes a full peer configuration document
