# 701 Wrong Server Public Key

## User Story

As a developer, I want a saved context with the wrong server public key to fail
loudly so bad configuration is immediately visible.

## Covered Behaviors

- a context can be saved with an incorrect server key
- `ping` fails instead of silently connecting
