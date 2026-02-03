# teapot-router ☕️ [![codecov](https://codecov.io/gh/mallardduck/teapot-router/graph/badge.svg?token=EJ2ZUN8J8N)](https://codecov.io/gh/mallardduck/teapot-router)

**teapot-router** is an expressive routing layer built on top of [`chi`](https://github.com/go-chi/chi), inspired by
Laravel's router.

It adds features like **named routes**, **query-based routing**, and **S3 action context injection**, while staying
lightweight and Go-idiomatic.

> Routes, gently steeped.

---

## Features

* 🍃 Built on `chi` — fast, minimal, battle-tested
* 🏷 **Named routes** with reverse URL generation
* 🧭 Laravel-inspired fluent routing API
* 🔍 **Query parameter multiplexing** — same path routes to different handlers based on query params
* ☁️ **S3 action context injection** — ideal for S3-compatible APIs
* 🧩 Fully compatible with `chi` middleware

---

## Installation

```bash
go get github.com/mallardduck/teapot-router
```

---

## Basic Usage

```go
import (
"net/http"

teapot "github.com/mallardduck/teapot-router"
)

func main() {
r := teapot.New()

r.GET("/users", listUsers).Name("users.index")
r.GET("/users/{id}", showUser).Name("users.show")

http.ListenAndServe(":3000", r)
}
```

For more usage details see: [docs/USAGE.md](docs/USAGE.md)

---

## Philosophy

* Prefer **clarity over cleverness**
* Stay close to `chi`, don't replace it
* Add expressiveness *without* becoming a framework
* Make S3-style APIs readable at a glance

---

## Status

⚠️ **Early development**
APIs may change until v1.0. Feedback and contributions are welcome.

---

## License

MIT ☕