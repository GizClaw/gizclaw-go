# 500 Admin Context Provision

## User Story

As an operator, I want a saved context to be provisioned with an admin
registration token and then verified with an admin command, so I know the context
can be used for later control-plane workflows.

## Scenario

1. Start a real server with an admin registration token.
2. Create a saved CLI context pointing at that server.
3. Register the context through the test harness using the admin token.
4. Run `gizclaw admin gears list --context admin-a`.
5. Verify the admin command can connect and succeeds after provisioning.

## Covered Behaviors

- provisioning a context with an admin registration token enables admin command
  access.
- the scenario uses the harness API registration path instead of the removed
  `play register` CLI.
