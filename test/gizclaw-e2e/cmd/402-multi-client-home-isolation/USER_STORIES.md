# 402 Multi Client Home Isolation

## User Story

As a developer, I want separate client homes to keep their own context stores even
when they connect to the same server.

## Covered Behaviors

- two virtual homes can target the same server
- each home sees only its own contexts
- commands from either home still work against the shared server
