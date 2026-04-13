# 100 Context Create Use List

## User Story

As a developer working against one Giztoy server, I want to:

1. Create multiple CLI contexts for the same server.
2. List the available contexts in a predictable order.
3. Switch the current context explicitly.

So that I can manage multiple local client identities without guessing which one is active.

## Covered Behaviors

- `giztoy context create <name>` creates isolated client identities.
- `giztoy context list` prints sorted context names.
- `giztoy context list` marks the current context with `*`.
- `giztoy context use <name>` switches the current context.
- Explicit `giztoy ping --context <name>` works for each created context.

## Isolation Rules

- This story owns its own virtual `HOME`.
- This story owns its own `XDG_CONFIG_HOME`.
- This story owns its own server workspace and runtime data.
- Runtime artifacts are temporary and cleaned by the test harness.
