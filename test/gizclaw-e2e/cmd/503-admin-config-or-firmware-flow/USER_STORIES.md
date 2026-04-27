# 503 Admin Config Or Firmware Flow

## User Story

As an operator, I want a basic firmware and gear configuration flow through the
CLI so I can verify both depot metadata updates and the renamed gear config
commands end to end.

## Scenario

1. Start a real server from this story's workspace fixture.
2. Provision an admin-capable context.
3. Register a device context with firmware metadata.
4. List firmware depots before any update.
5. Write depot metadata with `gizclaw admin firmware put-info --file`.
6. Read the depot back with `gizclaw admin firmware get`.
7. List depots again and verify the depot is visible.
8. Run `gizclaw admin gears set-firmware-channel <pubkey> stable` and verify it
   updates the firmware channel without replacing the whole config by position.
9. Run `gizclaw admin gears put-config <pubkey> --file <config.json>` and verify
   complete config replacement works through file input.

## Covered Behaviors

- an admin context can list firmware depots
- depot metadata can be written through `put-info`
- the depot can be retrieved afterward
- `set-firmware-channel` replaces the old channel-only `put-config` usage
- `put-config --file` writes a full gear configuration document
