# 104 Context Info Show

## User Story

As an operator managing more than one saved CLI context, I want to inspect the
current context and any named context after `gizclaw context list`, so I can
confirm the server address, server public key, and local identity before running
admin or device commands.

## Scenario

1. Start a real GizClaw server from this story's workspace fixture.
2. Run `gizclaw context info` before any context exists.
3. Create `alpha` against the real server and create `beta` against a second
   address.
4. Switch the current context to `beta`.
5. Run `gizclaw context info` and verify it describes `beta` as current.
6. Run `gizclaw context show alpha` and verify it describes `alpha` without
   switching the current context.
7. Run `gizclaw server-info --context alpha` to verify context-based server
   selection still works for the optional root command.
8. Run `gizclaw context show missing` and legacy `gizclaw ctx list` to verify
   both fail clearly.

## Covered Behaviors

- `context info` fails when no active context exists.
- `context info` prints the current context details as JSON.
- `context show <name>` prints a named context without changing the current one.
- `server-info --context <name>` uses the selected context and returns server
  metadata.
- The removed `ctx` alias is no longer accepted.
- Missing named contexts return a non-zero CLI result.
