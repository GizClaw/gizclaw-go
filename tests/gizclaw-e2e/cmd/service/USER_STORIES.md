# Service CLI

## User Story

As an operator, I want the service-managed server path and foreground serve path to stay clearly separated.

## Covered Behaviors

- `serve --help` documents explicit foreground serving.
- Plain `serve <workspace>` refuses direct starts.
- Service guidance points users to the managed server lifecycle.
