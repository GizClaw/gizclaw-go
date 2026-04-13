# 202 Server Workspace Isolation

## User Story

As a developer, I want separate server workspaces to remain isolated so local test
or demo environments do not accidentally share identity or runtime state.

## Covered Behaviors

- separate workspaces generate different server public keys
- each workspace can serve requests independently
- a client context for one workspace can connect without affecting the other

## Isolation Rules

- This story owns its own virtual `HOME`
- This story owns its own `XDG_CONFIG_HOME`
- This story uses separate server workspaces inside its sandbox
