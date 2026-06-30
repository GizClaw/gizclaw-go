# GizClaw E2E

This directory contains the manual/setup-driven GizClaw e2e suites. These tests
depend on a prepared local e2e server, a shared business resource set, and generated
CLI config homes. Go test files in this tree require the `gizclaw_e2e` build tag
so they are not pulled into ordinary `go test ./...` runs.

## Modules

- `testdata/`: committed config/resource data plus ignored generated runtime files.
- `setup/`: lifecycle scripts for building the CLI, starting services,
  resetting shared data, granting the default client view, and stopping
  services.
- `client/`: typed client and protocol-level e2e tests.
- `cmd/`: user-facing `gizclaw` CLI command e2e tests.
- `ui/`: browser UI e2e tests for Admin UI, Play UI, and cross-surface smoke
  checks.

## Standard Flow

1. Copy `tests/gizclaw-e2e/.env.example` to `tests/gizclaw-e2e/.env`, then fill
   provider credential values. The same file may also override context roles
   when running against existing local or remote dev contexts. Runtime
   addresses, resource names, resource IDs, model IDs, voice IDs, and e2e
   identity keys are committed fixtures, not env values.

2. Run the full ordered giznet e2e gate:

```sh
./tests/gizclaw-e2e/run_giznet_tests.sh
```

The script selects `testdata/config-home-giznet`, builds the e2e CLI, resets
server data, runs non-UI packages one at a time, starts the Admin UI and Play
UI, then runs UI packages one at a time. It excludes human-review cases, which
require separate interactive audio review. It stops e2e services on success or
failure.

To run both transport gates sequentially:

```sh
./tests/gizclaw-e2e/run_tests_all.sh
```

For debugging, the same flow can be run manually.

Human-review cases are not part of the default e2e gate. Run them explicitly:

```sh
./tests/gizclaw-e2e/run_human_review_tests.sh
```

3. Build the e2e CLI binary:

```sh
./tests/gizclaw-e2e/setup/build.sh
```

4. Start the local e2e server:

```sh
./tests/gizclaw-e2e/setup/start-server.sh
```

5. Clear and initialize server resources:

```sh
./tests/gizclaw-e2e/setup/reset_data.sh
```

To let another peer public key use the default shared client view, apply a
`PeerConfig` for that key:

```sh
./tests/gizclaw-e2e/setup/apply_client_view.sh <peer-public-key>
```

`reset_data.sh` only rebuilds resource state: provider tenants, models,
workflows, workspaces, firmware metadata, ACL rows, and social graph
resources. It does not call provider sync operations. It must not seed runtime
history, message records, replay audio, or other non-resource state.

6. Run client tests that create runtime state. These should run before any UI
   test that expects conversations, history entries, replay data, or social
   message state to already exist:

```sh
go test -tags gizclaw_e2e -count=1 -skip '^(TestHumanReview|TestServerSocialRPCHumanReview)$' ./tests/gizclaw-e2e/client/admin
go test -tags gizclaw_e2e -count=1 -skip '^(TestHumanReview|TestServerSocialRPCHumanReview)$' ./tests/gizclaw-e2e/client/chat
go test -tags gizclaw_e2e -count=1 -skip '^(TestHumanReview|TestServerSocialRPCHumanReview)$' ./tests/gizclaw-e2e/client/rpc
go test -tags gizclaw_e2e -count=1 -skip '^(TestHumanReview|TestServerSocialRPCHumanReview)$' ./tests/gizclaw-e2e/client/social
```

7. Run CLI story tests against the same setup-created server and resource
   catalog:

```sh
go test -tags gizclaw_e2e -count=1 ./tests/gizclaw-e2e/cmd/connect
```

### Giznet Transport Parity

The committed setup server exposes both giznet transports:

- `9820/udp`: Noise over UDP
- `9820/tcp`: public HTTP API and WebRTC signaling
- `9821/udp` and `9821/tcp`: WebRTC ICE

For #90 transport parity, run the same client suites twice with separate config
homes and a data reset between runs. The default config home is
`testdata/config-home-giznet`; the WebRTC config home is
`testdata/config-home-webrtc`.

Run these transport parity suites sequentially against the shared setup server.
The shared `testdata/server-workspace` and committed context homes are mutable
during e2e execution, so concurrent parity runs must use fully isolated server
instances: a separate workspace directory, config home, public API port, Noise
UDP port, and WebRTC ICE port for each run.

```sh
./tests/gizclaw-e2e/run_giznet_tests.sh
./tests/gizclaw-e2e/run_webrtc_tests.sh
```

The harness reads transport from the selected config home's `gear1` context,
then logs the selected transport and dial address when it binds an alias,
creates a `cmd/connect` context, or opens a direct `gizcli.Client` connection.

8. For browser UI tests, start the matching UI surface after the needed client
   tests have created runtime state:

```sh
./tests/gizclaw-e2e/setup/start-admin-ui.sh
./tests/gizclaw-e2e/setup/start-play-ui.sh
```

Then run the relevant UI suite:

```sh
go test -tags gizclaw_e2e -count=1 ./tests/gizclaw-e2e/ui/admin/...
go test -tags gizclaw_e2e -count=1 ./tests/gizclaw-e2e/ui/play/...
go test -tags gizclaw_e2e -count=1 ./tests/gizclaw-e2e/ui/smoke/...
```

9. Stop e2e services when finished:

```sh
./tests/gizclaw-e2e/setup/stop.sh
```

The full e2e run is intentionally ordered. Setup creates resource state, client
tests exercise the server and create runtime state, and UI tests verify the
browser surfaces against the resulting server state. Do not make UI tests depend
on setup-created runtime records.

## Test Data

`testdata/resources` is the business resource set used by client,
cmd, and UI tests. It is organized by resource domain instead of by test
surface:

```text
resources/
  00-credentials/
  01-tenants/
  03-models/
  04-workflows/
  05-workspaces/
  06-firmwares/
  07-gameplay/
  08-voices/
  09-social/
  90-acl/
  assets/
```

Resource fixture filenames use a local numeric prefix inside each resource
domain directory, for example `00-credentials/00-openai.yaml` or
`04-workflows/06-flowcraft-chat.yaml`. The directory prefix controls
cross-resource apply order, and the file prefix controls order within that
resource domain. `gizclaw admin apply` accepts JSON and YAML, but committed e2e
resource fixtures should use `.yaml`.

Only credential-like provider values should be environment placeholders, such as
`${GIZCLAW_E2E_OPENAI_API_KEY}`. Values are supplied by
`tests/gizclaw-e2e/.env` during setup. `reset_data.sh init/reset` fails before
starting setup when any required provider credential is missing. Do not commit
real provider keys, tokens, app secrets, or access keys. Stable e2e identity key
pairs are committed config fixtures, not env values.

`~/Work/haivivi/env` can be used as a private source for local provider values.
For example, Volc/Doubao maps `bytedance_ark_token` to
`GIZCLAW_E2E_VOLC_ARK_API_KEY`, and maps `bytedance_speech_app_id`,
`bytedance_speech_access_token`, and `bytedance_speech_search_api_key` to the
matching `GIZCLAW_E2E_DOUBAO_*` values. MiniMax maps `minimax_cn_key` /
`minimax_cn_group_id` and
`minimax_global_key` / `minimax_global_group_id` to the matching
`GIZCLAW_E2E_MINIMAX_*` values in `.env`. Qwen should be represented by the
DashScope provider (`GIZCLAW_E2E_DASHSCOPE_API_KEY`) when a DashScope/Tongyi
credential is available.

Generated runtime data under `testdata/server-workspace/data/` and generated
binaries under `testdata/bin/` stay ignored.

## Resource Set

`setup/reset_data.sh init` creates a resource set that looks like a small real
deployment: provider tenants, model rows, voice rows, workflows, workspaces,
firmware entries, ACL policy bindings, and social graph rows. Client, CLI, and
UI tests should be written around this business resource set instead of adding
private per-test or UI-specific resource groups. Tests may still create and delete
`mutation-*` resources for mutation coverage.

Stable business resource IDs:

- Workflow: `flowcraft-support`
- Run-control workflow: `chatroom-direct`
- Chatroom workflow: `family-circle-chatroom`
- Workspace: `support-desk-workspace`
- Run-control workspace: `direct-chatroom-workspace`
- Family chatroom workspace: `family-circle-chatroom-workspace`
- Model: `openai-gpt-4o-mini`
- Gameplay system task models: `reward-claim`, `pet-action` (Volc/Doubao credentials required)
- Credential: `openai-main-credential`
- MiniMax voice metadata row: `minimax-narrator-clone`
- Volc voice metadata row: `volc-tenant:volc-main:zh_female_vv_mars_bigtts`
- Pet species: `rabbit`
- Badge: `founder`
- Firmware: `devkit-firmware-main`
- Firmware channel/artifact: `stable` / `main`
- Mutation-safe names: `mutation-flowcraft-workflow`, `mutation-flowcraft-workspace`,
  `mutation-openai-model`, `mutation-openai-credential`

Bulk fake resource prefixes:

- `flowcraft-scenario-000` through `flowcraft-scenario-119`
- `workspace-scenario-000` through `workspace-scenario-119`
- `fake-openai-chat-000` through `fake-openai-chat-079`
- `fake-openai-credential-000` through `fake-openai-credential-049`
- `devkit-firmware-000` through `devkit-firmware-079`

The committed firmware metadata is applied through ResourceList YAML, but the
downloadable firmware payload is a real tar fixture at
`testdata/assets/firmware/devkit-firmware-main.tar`. During init,
`reset_data.sh` uploads that tar with:

```sh
gizclaw admin firmwares upload-artifact devkit-firmware-main \
  --channel stable \
  -f testdata/assets/firmware/devkit-firmware-main.tar
```

Provider-independent resource rows use schema-valid committed metadata, but the
full e2e resource catalog also includes real provider rows. Required provider
credential values must be present in `.env`; otherwise `reset_data.sh init/reset`
fails fast and no partial e2e setup should be treated as valid. `reward-claim`
and `pet-action` are Volc/Doubao-backed gameplay system task model rows.
`client/admin` owns provider voice sync verification and should run before chat
voice tests.

Workspace history is runtime data. `family-circle-chatroom-workspace` is a normal
chatroom workspace target. `reset_data.sh` must not seed
history entries or audio directly; social and workspace e2e cases should create
history by running the relevant client workflows.

## Config Homes

`testdata/config-home-giznet` and `testdata/config-home-webrtc` are committed
`XDG_CONFIG_HOME` roots for transport parity. Each contains the normal
`gizclaw/` config layout and committed client `identity.key` fixtures. Context
config files must store the server `public-key` directly; do not point contexts
at the server `identity.key`, because that file is the server private key.

Committed fixture contexts are stable identities and should be used when a test
depends on a known role or ACL state. Each config home uses the same role names:

- `admin`: setup resource initialization, admin API/client tests, and Admin UI.
- `gear1`: ordinary client, workspace, RPC, chat, Play UI, and social peer A.
- `gear2`: secondary gear identity for two-peer social tests.

`reset_data.sh` uses `admin` to apply resource fixtures and registers `gear1`
before applying resource fixtures. Tests that create sandbox contexts at runtime
are responsible for registering those peers before using role-gated services.

Context overrides in `.env` select identities inside the selected config home:

- `GIZCLAW_E2E_CONFIG_HOME`: config home root. It defaults to
  `tests/gizclaw-e2e/testdata/config-home-giznet`; use
  `tests/gizclaw-e2e/testdata/config-home-webrtc` for WebRTC parity.
- `GIZCLAW_E2E_ADMIN_CONTEXT`: setup resource initialization, admin API/client
  tests, and Admin UI. The committed context is `admin`.
- `GIZCLAW_E2E_GEAR1_CONTEXT`: primary gear identity. The committed context is
  `gear1`.
- `GIZCLAW_E2E_GEAR2_CONTEXT`: secondary gear identity. The committed context is
  `gear2`.

Transport is not configured with an environment variable. It comes from the
selected config home. Most `cmd/*` story tests still create isolated sandbox
contexts; those sandbox contexts inherit transport from `gear1`.

## Client Tests

`client/admin` contains typed Admin HTTP API contract coverage using the
generated `adminservice` client. It verifies Swagger-defined request/response
schemas, pagination, binary upload/download where the current Admin API exposes
it, provider voice sync prerequisites for chat tests, and selected
mutation-safe paths against the shared setup server.

`client/rpc` contains typed RPC coverage. Test files should be grouped by RPC
module prefix, and individual methods should be split by `Test...` functions.

`client/chat` contains workspace-backed voice conversation and history cases as
ordinary `_test.go` files. It should not use a custom `main.go -case ...`
dispatcher.

`client/social` contains friend and friend-group behavior. These tests are
client-driven, not CLI-story driven, and should cover relation changes,
workspace ACL visibility, message rounds, `workspace.history.updated`, history
list/get cursor behavior, and history replay.

## CLI Tests

`cmd` tests run the real `gizclaw` binary from `testdata/bin/gizclaw` through
Go `os/exec`. They should not use `go run` and should not shortcut through
typed clients.

The `cmd` layout mirrors the real CLI command hierarchy: `root`, `gen-key`,
`context`, `serve`, `service`, `migrate`, `connect`, `admin`, and `play`. Each
command directory has one `USER_STORIES.md` plus focused `_test.go` files.

`cmd/play` tests the `gizclaw play` command. Browser Play UI behavior belongs
under `ui/play`.

## UI Tests

`ui/admin` and `ui/play` are browser UI tests organized by visible page or major
surface. Missing business resources should be added under the matching
`testdata/resources/<NN-domain>/` directory and initialized by
`setup/reset_data.sh`.

`ui/smoke` contains cross-surface checks, such as opening both Admin UI and Play
UI against the same shared test service.
