# 103 Context Isolation Between Story Homes

## User Story

As a developer, I want different story sandboxes to keep separate client context
stores so one test cannot pollute another test's virtual home.

## Covered Behaviors

- two harnesses can target the same server from different homes
- each home sees only its own saved contexts
- both homes can still use their own contexts successfully
