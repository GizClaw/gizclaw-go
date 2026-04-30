---
name: gizclaw-play
version: 1.0.0
description: "Open and explain the GizClaw Play UI entrypoint. Use for gizclaw play --listen and when users ask about device-side play registration, config, OTA, or reverse API flows."
metadata:
  requires:
    bins: ["gizclaw"]
---

# GizClaw Play

Use this skill for the Play UI.

## When To Use

- User asks to open the Play UI.
- User asks where device-side registration, config, OTA, or reverse API flows should be handled.
- User asks about `gizclaw play`.

## How To Start

1. Pick a listen address or use the address requested by the user.
2. Make sure the desired CLI context is current, or pass `--context <name>`.
3. Run `play --listen <addr>` in the background.
4. Monitor startup output.
5. Tell the user the local UI address.

## Command

```bash
<gizclaw> play --listen 127.0.0.1:8081
```

## Behavior Notes

- `play` is UI-only.
- `play --listen` automatically registers the current context before serving the UI. If the context is already registered, it continues normally.
- The automatic registration stores the gear as `auto_registered` and active.
- Device-side registration, config, OTA, and reverse API workflows belong in the Play UI.
- Do not try to use CLI subcommands for those Play workflows.
