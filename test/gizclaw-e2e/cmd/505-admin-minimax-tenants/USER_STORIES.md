# 505 Admin MiniMax Tenants

## User Story

As an admin operator configuring MiniMax voice providers, I want to read tenant
records from the CLI while credentials remain a separate resource.

## Scenario

1. Start a real server and provision an admin-capable CLI context.
2. Seed a credential and a MiniMax tenant through the harness API.
3. List tenants and verify the tenant appears.
4. Get the tenant by name and verify it references the credential.

## Covered Behaviors

- Tenant `list` and `get` work through the `minimax-tenants` namespace.
- Tenants can reference existing credential resources.
- The CLI resource surface is read-only; test setup uses the harness/API to
  prepare state.
