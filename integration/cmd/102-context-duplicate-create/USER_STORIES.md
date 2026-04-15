# 102 Context Duplicate Create

## User Story

As a developer, I want duplicate context creation to fail clearly so retries do not
overwrite an existing saved identity.

## Covered Behaviors

- the first context creation succeeds
- creating the same context name again fails
- the original context remains usable
