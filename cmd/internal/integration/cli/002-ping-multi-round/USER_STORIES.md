# 002 Ping Multi Round

## User Story

As a developer validating local CLI stability, I want to:

1. Start one long-running server.
2. Create multiple client contexts.
3. Run several rounds of sequential and concurrent `ping` commands.

So that I can trust the CLI workflow across repeated local use, not just one successful call.

## Covered Behaviors

- one server stays healthy across multiple command rounds
- multiple contexts can be reused across rounds
- sequential and concurrent `ping` calls both keep working over time

## Isolation Rules

- This story owns its own virtual `HOME`.
- This story owns its own `XDG_CONFIG_HOME`.
- This story owns its own server workspace and runtime data.
- Runtime artifacts are temporary and cleaned by the test harness.
