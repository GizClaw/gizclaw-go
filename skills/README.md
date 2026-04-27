# GizClaw Agent Skills

This directory contains project-level Agent Skills following the open skills layout used by `npx skills`.

## Skills


| Skill         | Description                                                                                         |
| ------------- | --------------------------------------------------------------------------------------------------- |
| `gizclaw-cli` | Route general GizClaw CLI requests to the correct command-specific skill. |
| `gizclaw-context` | Manage contexts, ping connectivity, and read server metadata. |
| `gizclaw-server` | Start foreground servers, manage system services, and edit workspace config. |
| `gizclaw-admin-gears` | Manage registered devices/gears and gear configuration. |
| `gizclaw-admin-firmware` | Manage firmware depots, channels, uploads, releases, and rollback. |
| `gizclaw-admin-resources` | Apply, show, and delete declarative admin resources. |
| `gizclaw-admin-credentials` | Read provider credentials. |
| `gizclaw-admin-minimax-tenants` | Read MiniMax tenants. |
| `gizclaw-admin-voices` | Read the global voice catalog. |
| `gizclaw-admin-workspace-templates` | Read workspace template documents. |
| `gizclaw-admin-workspaces` | Read workspace instances. |
| `gizclaw-play` | Open and explain the Play UI entrypoint. |


## Install

From the repository root:

```bash
npx skills add . --skill gizclaw-cli
npx skills add . --skill gizclaw-context
npx skills add . --skill gizclaw-server
npx skills add . --skill gizclaw-admin-gears
npx skills add . --skill gizclaw-admin-firmware
npx skills add . --skill gizclaw-admin-resources
npx skills add . --skill gizclaw-admin-credentials
npx skills add . --skill gizclaw-admin-minimax-tenants
npx skills add . --skill gizclaw-admin-voices
npx skills add . --skill gizclaw-admin-workspace-templates
npx skills add . --skill gizclaw-admin-workspaces
npx skills add . --skill gizclaw-play
```

For global installation:

```bash
npx skills add . --skill gizclaw-cli -g
npx skills add . --skill gizclaw-context -g
npx skills add . --skill gizclaw-server -g
npx skills add . --skill gizclaw-admin-gears -g
npx skills add . --skill gizclaw-admin-firmware -g
npx skills add . --skill gizclaw-admin-resources -g
npx skills add . --skill gizclaw-admin-credentials -g
npx skills add . --skill gizclaw-admin-minimax-tenants -g
npx skills add . --skill gizclaw-admin-voices -g
npx skills add . --skill gizclaw-admin-workspace-templates -g
npx skills add . --skill gizclaw-admin-workspaces -g
npx skills add . --skill gizclaw-play -g
```
