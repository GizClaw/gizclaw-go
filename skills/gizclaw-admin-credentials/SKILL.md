---
name: gizclaw-admin-credentials
version: 1.0.0
description: "Read GizClaw provider credentials. Use for admin credentials list/get and provider-filtered credential listing."
metadata:
  requires:
    bins: ["gizclaw"]
---

# GizClaw Admin Credentials

Use this skill for provider credential reads.

## When To Use

- User asks to list or inspect provider credentials.
- User needs credentials for MiniMax tenants or other provider integrations.
- User asks whether credential body values are returned by the API.

## How To Start

1. Determine the admin context and pass `--context <name>` when known.
2. Use `list` first when the credential name is unknown.
3. Use `list --provider <provider>` when the user names a provider.
4. Use `get <name>` for a specific credential.

## Commands

```bash
<gizclaw> admin credentials list --context <admin-context>
<gizclaw> admin credentials list --provider <provider> --context <admin-context>
<gizclaw> admin credentials get <name> --context <admin-context>
```

## Behavior Notes

- This CLI resource surface is read-only: it exposes `list` and `get`.
- Use `../gizclaw-admin-resources/SKILL.md` for declarative `Credential`
  create, update, show, or delete workflows.
- The admin API returns credential `body` values in `get` and `list` responses. Do not mask them unless requested.
