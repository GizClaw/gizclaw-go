# 403 Multi Client Reconnect Race

## User Story

As a developer, I want multiple saved contexts to recover together after a server
restart so reconnect behavior is not accidentally serialized or fragile.

## Covered Behaviors

- multiple contexts are created before restart
- the server restarts with the same workspace identity
- concurrent pings from saved contexts succeed after restart
