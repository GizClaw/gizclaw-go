# 603 Repeat Command After Partial State

## User Story

As a developer, I want repeated commands to fail predictably when state already
exists so retries do not silently corrupt or duplicate resources.

## Scenario

1. Start a real server with a device registration token.
2. Create one device context.
3. Register that context through the harness API.
4. Repeat the same registration request.
5. Verify the repeated request reports an existing-state failure.
6. Verify the context remains usable with `gizclaw ping`.

## Covered Behaviors

- initial registration succeeds
- repeating the same registration reports an existing-state failure
- the context remains usable after the failed retry
- the scenario preserves duplicate-registration coverage without restoring
  `play register` to the CLI surface
