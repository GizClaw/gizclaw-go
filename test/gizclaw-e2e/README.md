# GizClaw E2E

This directory keeps shared local e2e setup separate from individual test
clients. Runtime state is written under `test/gizclaw-e2e/.testbench/`:

- `.testbench/workspace`: local GizClaw server workspace used by setup.
- `.testbench/context`: `XDG_CONFIG_HOME` for generated GizClaw CLI contexts.
- `.testbench/bin`: local `gizclaw` binary built for the e2e run.

Source layout:

- `setup`: shared server config, server start script, generated client context,
  resource files, and ACL setup.
- `openaicompat`: OpenAI-compatible HTTP e2e client using the shared setup.
- `workspace`: workspace voice-path e2e client using the shared setup.

## Full E2E Run

1. Prepare `.env`:

```sh
cp test/gizclaw-e2e/setup/.env.example test/gizclaw-e2e/setup/.env
$EDITOR test/gizclaw-e2e/setup/.env
```

2. Setup for `openaicompat`.

Start the e2e server in terminal 1:

```sh
./test/gizclaw-e2e/setup/start.sh --env test/gizclaw-e2e/setup/.env
```

In terminal 2, make sure the admin context points at this server:

```sh
SERVER_PUBLIC_KEY="$(sed -n 's/^[[:space:]]*public-key:[[:space:]]*//p' \
  test/gizclaw-e2e/.testbench/context/gizclaw/e2e-client/config.yaml | head -n 1)"
gizclaw context show "${GIZCLAW_E2E_ADMIN_CONTEXT:-e2e-admin}"
```

The shown `server_public_key` should equal `$SERVER_PUBLIC_KEY`.

Apply setup resources:

```sh
./test/gizclaw-e2e/setup/apply_resources.sh --env test/gizclaw-e2e/setup/.env
```

3. Run `openaicompat` e2e.

Start the Play HTTP proxy in terminal 3:

```sh
XDG_CONFIG_HOME="$PWD/test/gizclaw-e2e/.testbench/context" \
  test/gizclaw-e2e/.testbench/bin/gizclaw play \
    --context "${GIZCLAW_E2E_CLIENT_CONTEXT:-e2e-client}" \
    --listen 127.0.0.1:8081
```

Then run the test:

```sh
go run ./test/gizclaw-e2e/openaicompat \
  --config test/gizclaw-e2e/openaicompat/config/default.example.json \
  --base-url http://127.0.0.1:8081/v1
```

4. Clean and setup again for `workspace`.

Stop the server and Play proxy from the previous run, then reset local runtime
state:

```sh
rm -rf test/gizclaw-e2e/.testbench
```

Start the server again in terminal 1:

```sh
./test/gizclaw-e2e/setup/start.sh --env test/gizclaw-e2e/setup/.env
```

Apply resources again in terminal 2:

```sh
./test/gizclaw-e2e/setup/apply_resources.sh --env test/gizclaw-e2e/setup/.env
```

5. Run `workspace` e2e:

```sh
go run ./test/gizclaw-e2e/workspace \
  -config test/gizclaw-e2e/workspace/config/doubao-realtime.example.json
```

Local compile/unit checks:

```sh
go test -count=1 ./test/gizclaw-e2e/setup ./test/gizclaw-e2e/openaicompat ./test/gizclaw-e2e/workspace
```

`setup` owns credentials, tenants, models, voices, ACL grants, and the shared
client identity. Test clients do not create provider-side resources. The
workspace test creates only its workflow/workspace pair through peer RPC.

## Context Config

`setup/start.sh` writes the e2e client context here:

```text
test/gizclaw-e2e/.testbench/context/gizclaw/e2e-client/
  config.yaml
  identity.key
```

`config.yaml` contains only normal GizClaw context fields:

```yaml
server:
  address: 127.0.0.1:9820
  public-key: ...
  cipher-mode: chacha_poly
```

`identity.key` contains the client private key. Test scenario data lives in each
client directory, for example `openaicompat/config/*.json` and
`workspace/config/*.json`. No separate `client.json` is generated.
