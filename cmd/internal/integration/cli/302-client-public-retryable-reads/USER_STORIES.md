# 302 Client Public Retryable Reads

## User Story

As a device-side developer, I want repeated public read commands to keep working so
simple polling workflows stay reliable.

## Covered Behaviors

- one registered context can issue repeated public reads
- repeated `play config` and `ping` commands keep succeeding
