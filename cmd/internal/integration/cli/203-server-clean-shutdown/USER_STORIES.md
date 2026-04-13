# 203 Server Clean Shutdown

## User Story

As a developer, I want a test server to stop cleanly and be restartable from the
same workspace without leaving unusable runtime state behind.

## Covered Behaviors

- a running server can be stopped intentionally
- client commands fail while the server is offline
- the same workspace can be started again and accept requests

## Isolation Rules

- This story owns its own virtual `HOME`
- This story owns its own `XDG_CONFIG_HOME`
- This story owns its own server workspace and runtime data
