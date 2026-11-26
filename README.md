# teapot-router ☕️ [![codecov](https://codecov.io/gh/mallardduck/teapot-router/graph/badge.svg?token=EJ2ZUN8J8N)](https://codecov.io/gh/mallardduck/teapot-router)

**teapot-router** adds named routes, query and header dispatch, and S3-style action context on top of
[chi](https://github.com/go-chi/chi). It stays close to chi and does not replace it.

> Routes, gently steeped.

## Install

```bash
go get github.com/mallardduck/teapot-router
```

## A taste

```go
r := teapot.New()

r.GET("/users", listUsers).Name("users.index")
r.GET("/users/{id}", showUser).Name("users.show")

path := r.MustURL("users.show", "id", "42") // "/users/42"
```

Same path, different handler per query parameter:

```go
r.Dispatch("GET", "/{bucket}", func(d *teapot.DispatchBuilder, m teapot.Matchers) {
    d.DefaultFunc(listObjects).Name("bucket.list").Action("s3:ListBucket")
    d.When(m.QueryExists("acl")).DoFunc(getBucketAcl).Name("bucket.acl").Action("s3:GetBucketAcl")
})
```

## Docs

- **Start with [chi](https://github.com/go-chi/chi)**: routing, URL parameters, and middleware work the same way here.
  Its [API docs](https://pkg.go.dev/github.com/go-chi/chi/v5) and [go-chi/docs](https://github.com/go-chi/docs) are the primary references.
- **teapot-router docs**: <https://mallardduck.github.io/teapot-router/> cover only what this package adds.
- **API reference**: [pkg.go.dev](https://pkg.go.dev/github.com/mallardduck/teapot-router).

## Status

Early development. The API may change until v1.0. Feedback and contributions are welcome. See
[CONTRIBUTING.md](CONTRIBUTING.md).

## License

MIT
