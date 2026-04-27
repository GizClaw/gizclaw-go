# 301 Client Register Then Read

## User Story

As a device-side developer, I want registration metadata prepared by the test
harness and observed through the CLI to be readable afterward, so I can validate
the admin read side of an end-to-end provisioning flow after `play register` is
removed from the CLI.

## Scenario

1. Start a real GizClaw server with admin and device registration tokens.
2. Create and provision an admin context.
3. Create and register a device context with name, serial number, hardware
   metadata, depot, and firmware semver.
4. Resolve the device public key from the saved device context.
5. Run `gizclaw admin gears info <pubkey> --context admin-a`.
6. Verify the CLI output contains the device serial number, manufacturer, and
   model submitted during registration.

## Covered Behaviors

- device registration test data is prepared without relying on removed `play`
  subcommands
- an admin observer can look up the newly registered device
- the stored device info contains the submitted identifiers
