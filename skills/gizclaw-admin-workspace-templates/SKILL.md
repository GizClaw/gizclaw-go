---
name: gizclaw-admin-workspace-templates
version: 1.0.0
description: "Read GizClaw workspace template documents. Use for admin workspace-templates list/get."
metadata:
  requires:
    bins: ["gizclaw"]
---

# GizClaw Admin Workspace Templates

Use this skill for workspace template document reads.

## When To Use

- User asks to inspect or list workspace templates.
- User is preparing templates for workspace creation.

## How To Start

1. Determine the admin context and pass `--context <name>` when known.
2. Use `list` first when the template name is unknown.
3. Use `get <name>` to inspect the full template document.

## Commands

```bash
<gizclaw> admin workspace-templates list --context <admin-context>
<gizclaw> admin workspace-templates get <name> --context <admin-context>
```

## Behavior Notes

- This CLI resource surface is read-only: it exposes `list` and `get`.
- Use `../gizclaw-admin-resources/SKILL.md` for declarative
  `WorkspaceTemplate` create, update, show, or delete workflows.
- Workspaces reference templates by `workspace_template_name`.
