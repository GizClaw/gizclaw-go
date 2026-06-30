# Context CLI

## User Story

As a developer, I want `gizclaw context` commands to manage saved server contexts predictably.

## Covered Behaviors

- Context creation, current selection, listing, info, and show commands operate on the configured `XDG_CONFIG_HOME`.
- Duplicate creation fails without overwriting saved identities.
- Separate config homes stay isolated while targeting the same server.
- Removed legacy aliases fail clearly.
