# 504 Admin Credentials

## User Story

As an admin operator, I want to read provider credentials through the CLI, so I
can inspect existing credential resources without using per-resource write
commands.

## Scenario

1. Start a real server and provision an admin-capable CLI context.
2. Seed an OpenAI credential and a MiniMax credential through the harness API.
3. List all credentials and verify both resources are visible.
4. List credentials with `--provider openai` and verify only the OpenAI
   credential is returned.
5. Get the OpenAI credential by name.

## Covered Behaviors

- `list` returns existing credentials.
- `list --provider` filters server-side credential results.
- `get` works against a named credential.
- The CLI resource surface is read-only; test setup uses the harness/API to
  prepare state.
