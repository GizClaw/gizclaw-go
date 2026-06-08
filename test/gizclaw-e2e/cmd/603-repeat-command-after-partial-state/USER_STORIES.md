# 603 Repeat Command After Partial State

## User Story

As a developer, I want repeated peer preparation requests for an auto-created peer
to be idempotent so retries can safely complete partial device metadata setup.

## Scenario

1. Start a real server with automatic peer creation enabled.
2. Create one device context.
3. Prepare that context through the harness API.
4. Repeat the same preparation request.
5. Verify the repeated request succeeds and updates the auto-created device info.
6. Verify the context remains usable with `gizclaw peer ping`.

## Covered Behaviors

- initial preparation succeeds
- repeating the same preparation updates auto-created metadata safely
- the context remains usable after the retry
- the scenario preserves retry coverage without restoring
  `play register` to the CLI surface
