# 603 Repeat Command After Partial State

## User Story

As a developer, I want repeated commands to fail predictably when state already
exists so retries do not silently corrupt or duplicate resources.

## Covered Behaviors

- initial registration succeeds
- repeating the same registration reports an existing-state failure
- the context remains usable after the failed retry
