# 201 Server Identity Persistence

## User Story

As a developer reusing one server workspace, I want the server identity to persist
across restarts so saved contexts do not become stale.

## Covered Behaviors

- a workspace-generated server identity is persisted
- restarting the same workspace preserves the public key
- a context created after restart still targets the same server identity

## Isolation Rules

- This story owns its own virtual `HOME`
- This story owns its own `XDG_CONFIG_HOME`
- This story owns its own server workspace and runtime data
