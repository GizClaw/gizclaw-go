# 509 Admin Declarative Resources

## User Story

As an admin operator, I want to use top-level declarative resource commands, so
I can apply resource JSON files and inspect named resources through the same CLI
surface that will later back GitOps-style workflows.

## Scenario

1. Start a real server and provision an admin-capable CLI context.
2. Write a Credential resource JSON file in the story sandbox.
3. Run `gizclaw admin apply -f <file>` and verify the resource is created.
4. Run `gizclaw admin show Credential missing` and verify missing resources
   fail with a clear not-found error.
5. Run `gizclaw admin show Credential <name>` and verify the created resource
   is returned from the generic resource lookup endpoint.
6. Update the JSON file, run `gizclaw admin apply -f <file>` again, and verify
   the resource is updated.
7. Run `gizclaw admin delete Credential <name>` and verify the generic delete
   endpoint returns the deleted resource.
8. Run `gizclaw admin show ResourceList <name>` and verify the CLI rejects the
   non-addressable container kind before making a server request.

## Covered Behaviors

- `admin apply -f` reads a resource file in a real CLI process.
- `admin show <kind> <name>` targets the generic admin resource endpoint.
- `admin delete <kind> <name>` targets the generic admin resource delete
  endpoint.
- Apply reports created and updated actions for a named resource.
- Missing generic resource lookups surface a user-visible not-found error.
- `ResourceList` is accepted by apply but rejected by show because it is not a
  named concrete resource lookup target.
