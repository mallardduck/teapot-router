# V1 Scope

This document is the single source of planned scope for teapot-router v1.0. It replaces the earlier `PLAN.md`,
which is no longer in the repository.

## What teapot-router is

A routing layer on top of [chi](https://github.com/go-chi/chi): named routes, query and header dispatch, route
composition, and route introspection. It stays a router. It does not implement any protocol's behavior.

## In scope for V1

- **Registration:** `GET`/`POST`/`PUT`/`DELETE`/`HEAD`/`PATCH`/`OPTIONS`, `Handle`, `Func()` variants,
  `RegisterExternal` (phantom routes).
- **Named routes:** `Name`, `URL`, `MustURL`, wildcard substitution.
- **Query, header, and subdomain dispatch:** `Query`, `QueryValue`, `Dispatch` with `Default`/`When`, and `Matchers`
  (including `HostHasSubdomain`).
- **Composition:** `Group`, `NamedGroup`, `SubRouter`, `Mount`, `MountNamed`, with late propagation.
- **Middleware:** `Use`, `With`, `MiddlewareGroup`, and chi middleware compatibility.
- **Resources:** `Resource` with a controller value and functional options (`Only`, `Except`, `Middleware`, `Path`, `Param`, `Names`, `StoreVerb`).
- **Action context:** `Action`, `GetAction`, `GetRouteName`, `URLParam`, `URLParams`.
- **Production:** `Finalize`, `IsFinalized`, `NoopHandler`.
- **Introspection:** `Routes`, `AggregateRoutes`, table/JSON/compact formatters, and the list HTTP handlers.
- **`pkg/urlbuilder`:** absolute URL building, scheme detection, subdomain helpers.
- **`pkg/dispatch`:** standalone query, header, and host dispatcher, usable without teapot.

## Out of scope for V1

- **Protocol behavior.** teapot-router does not implement any API's semantics. The S3 files in `examples/routes-cli`
  are mock showcases that demonstrate how the router is meant to be used. Their handlers are intentionally empty, and
  they are not an S3 implementation.
- Authentication, authorization, storage, and request parsing.

## Planned for V1, not yet built

None. Everything in the V1 scope is built. Open decisions are listed under "API review" below.

## API review before v1.0

Breaking changes must land before the v1.0.0 tag. Open questions:

- **Decided:** `Default` and `Do` take `http.Handler` in both `pkg/teapot` and `pkg/dispatch`. `FuncDefault` and
  `FuncDo` take plain functions in both. This is a breaking change for `pkg/dispatch` callers who passed function
  literals to `Default` or `Do`; they now call the `Func` variants.
- **Decided:** `Resource` takes a controller value and functional options. `ResourceHandlers` and `ResourceHandlerBuilder` are removed. Routes are registered inside a group per resource, so middleware is scoped to that resource.
- `FuncRouter` and the `Func*` variants overlap with `Handle`.

Record the decision for each item here when it is made.

## Decisions recorded

1. **S3 scope:** V1 is router-only. The S3 example is a mock route table, not an implementation.
2. **Missing planning docs:** `PLAN.md`, `STRUCTURE.md`, `SUMMARY.md`, and the other removed planning documents are
   not restored. This file replaces `PLAN.md`.
