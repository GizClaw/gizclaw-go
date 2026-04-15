# 400 Multi Client Concurrent Ping

## User Story

As a developer, I want multiple client contexts to ping one long-running server at
the same time so I can catch basic coordination issues.

## Covered Behaviors

- multiple client contexts can be created for one server
- concurrent `ping` commands succeed
- the server stays healthy after concurrent requests
