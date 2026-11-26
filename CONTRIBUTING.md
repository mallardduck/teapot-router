# Contributing

Thanks for helping with teapot-router.

## Setup

Requires Go 1.25 or later.

```bash
git clone https://github.com/mallardduck/teapot-router.git
cd teapot-router
go mod download
go test ./...
```

## Layout

- `pkg/teapot`: the router. Public API.
- `pkg/dispatch`: standalone query, header, and host dispatcher.
- `pkg/urlbuilder`: absolute URLs and subdomain helpers.
- `internal/core`: implementation details. Free to change.
- `tests/`: integration tests and benchmarks (`tests/teapot`).
- `examples/routes-cli`: a mock S3 route table.

Unit tests sit next to the code they cover.

## Before you open a PR

```bash
go fmt ./...
go vet ./...
go test ./...
```

Run benchmarks if you touch hot paths:

```bash
go test -bench=. -benchmem ./tests/teapot/
```

CI also runs golangci-lint, using `.golangci.yml`.

## Conventions

- **Commits** use conventional format: `feat:`, `fix:`, `refactor:`, `test:`, `docs:`, `perf:`, `chore:`.
- **Public API** (`pkg/`) must have godoc comments on exported symbols. Keep it small. After v1.0 it must stay backward
  compatible.
- **Tests** come with new behavior. Keep one PR to one feature or fix.

## Docs

The user docs site lives on the `gh-pages` branch, not on `main`. Update it in the same change when a feature or
behavior changes. chi covers general routing and middleware, so the site only documents what teapot-router adds.

## Questions

Open an issue.
