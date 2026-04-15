# 500 Admin Context Provision

## User Story

As an operator, I want admin commands to stay unavailable until an admin context is
explicitly provisioned, and then become usable.

## Covered Behaviors

- a plain saved context cannot use admin APIs
- registering the same context with an admin token enables admin commands
