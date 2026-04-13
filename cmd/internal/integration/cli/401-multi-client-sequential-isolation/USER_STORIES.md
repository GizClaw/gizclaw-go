# 401 Multi Client Sequential Isolation

## User Story

As a developer, I want sequential commands across multiple contexts to stay
isolated so explicit context usage does not mutate the current selection.

## Covered Behaviors

- one current context can be set explicitly
- commands using `--context` do not change the current marker
- both contexts remain usable throughout the sequence
