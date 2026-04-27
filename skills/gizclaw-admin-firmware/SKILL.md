---
name: gizclaw-admin-firmware
version: 1.0.0
description: "Manage GizClaw firmware depots, channels, metadata, uploads, releases, and rollbacks. Use for admin firmware list/get/get-channel/put-info/upload/release/rollback."
metadata:
  requires:
    bins: ["gizclaw"]
---

# GizClaw Admin Firmware

Use this skill for firmware depot metadata, release archives, channel inspection,
release promotion, and rollback.

## When To Use

- User asks to list firmware depots or inspect a depot/channel.
- User wants to write depot metadata.
- User wants to upload a firmware release archive.
- User wants to promote or rollback firmware channels.

## How To Start

1. Determine the admin context and pass `--context <name>` when known.
2. Identify the target depot and channel from the user request or previous output.
3. For metadata, create an info JSON file and use `put-info --file`.
4. For upload, use the release archive path with `upload --file`.
5. For release/rollback, verify the target depot before running the mutating command.

## Commands

```bash
<gizclaw> admin firmware list --context <admin-context>
<gizclaw> admin firmware get <depot> --context <admin-context>
<gizclaw> admin firmware get-channel <depot> <channel> --context <admin-context>
<gizclaw> admin firmware put-info <depot> --file <info.json> --context <admin-context>
<gizclaw> admin firmware upload <depot> <channel> --file <release.tar> --context <admin-context>
<gizclaw> admin firmware release <depot> --context <admin-context>
<gizclaw> admin firmware rollback <depot> --context <admin-context>
```

## Payloads

Example depot info file:

```json
{
  "files": [
    {
      "path": "image.bin"
    }
  ]
}
```

## Behavior Notes

- `put-info` writes depot metadata.
- `upload` sends a release archive to a specific channel.
- `get-channel` inspects one channel in one depot.
- `release` promotes firmware through the server-defined flow: `testing -> beta -> stable -> rollback`.
- `rollback` promotes rollback content back to `stable`.
