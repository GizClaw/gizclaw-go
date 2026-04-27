---
name: gizclaw-admin-workspaces
version: 1.0.0
description: "Read GizClaw workspace instances. Use for admin workspaces list/get."
metadata:
  requires:
    bins: ["gizclaw"]
---

# GizClaw Admin Workspaces

Use this skill for workspace instance reads.

## When To Use

- User asks to list or inspect workspaces.
- User wants to verify a workspace template reference or parameters.

## How To Start

1. Determine the admin context and pass `--context <name>` when known.
2. Use `list` first when the workspace name is unknown.
3. Use `get <name>` to verify template reference and parameters.

## Commands

```bash
<gizclaw> admin workspaces list --context <admin-context>
<gizclaw> admin workspaces get <name> --context <admin-context>
```

## Behavior Notes

- This CLI resource surface is read-only: it exposes `list` and `get`.
- Use `../gizclaw-admin-resources/SKILL.md` for declarative `Workspace`
  create, update, show, or delete workflows.
- `workspace_template_name` references an existing workspace template.
- `parameters` is an arbitrary JSON object returned by `get`/`list` when present.
