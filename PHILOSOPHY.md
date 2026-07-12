# GoKart philosophy

GoKart removes recurring infrastructure setup while leaving applications as ordinary, user-owned Go.

## Admission test

A GoKart component is admitted only when all of these remain true:

1. It removes setup that is repeated across real applications.
2. The standard library or upstream package alone does not already provide the same useful default.
3. Callers can reach the real standard-library or upstream type when they need lower-level control.
4. Its dependency cost is isolated in a module used only by applications that select it.
5. It contains infrastructure policy, not application or domain policy.
6. It does not mirror an upstream API method-for-method.
7. An application can delete the wrapper later without redesigning its business logic.

Generated projects follow the same boundary. The generator may wire safe defaults and selected integrations, but the result is ordinary Go source owned by the user. Generated code calls upstream APIs directly when GoKart adds no durable behavior.

## Release boundary

GoKart is pre-1.0. `v0.11.0` intentionally removes wrappers that failed this test. Applications that need the former surface can remain on the historical `v0.10.3` tags while migrating.
