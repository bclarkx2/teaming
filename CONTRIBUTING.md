# Contributing to teaming

Thank you for your interest in contributing.

## Prerequisites

- **Go 1.26** or later. The project uses `go.mod` toolchain directives, so an
  older toolchain will be rejected.
- Docker (optional, for `make docker-build` / `make docker-run`).

## Getting started

```sh
git clone https://github.com/bclarkx2/teaming.git
cd teaming
go mod download   # fetch dependencies declared in go.mod
make build        # produces bin/teaming
```

## Build and test

| Command | What it does |
|---|---|
| `make build` | Compile `bin/teaming` |
| `make test` | `go test ./...` — run the full test suite |
| `make fmt` | `go tool gofumpt -w .` — format all Go source |
| `make vet` | `go vet ./...` — run the Go static analyser |
| `make clean` | Remove `bin/` |
| `go test ./...` | Same as `make test`, usable directly |

Run `make help` to see all available targets.

## Code formatting

All Go files **must** be formatted with [gofumpt](https://github.com/mvdan/gofumpt)
before a change is merged. gofumpt is a strict superset of `gofmt` that enforces
a small set of additional style rules.

gofumpt is pinned as a `go tool` dependency in `go.mod`, so no separate
installation is required:

```sh
make fmt               # format everything
go tool gofumpt -w .   # equivalent
```

CI will fail if any `.go` file is not gofumpt-formatted. Run `make fmt` before
committing.

## Project layout

```
.
├── cmd/teaming/    # CLI entry point — main package, Cobra/Viper wiring
├── testdata/       # Sample CSVs used by tests and documentation
├── *.go            # Root package (github.com/bclarkx2/teaming) — importable logic
├── Makefile
└── go.mod / go.sum
```

The root package (`github.com/bclarkx2/teaming`) contains the importable core
logic (CSV parsing, team-assignment algorithm, etc.). `cmd/teaming` is the thin
CLI layer that wires flags/config to that logic.

Keep the CLI package free of business logic — if something is testable in
isolation, it belongs in the root package.

## Commits and pull requests

**Commit message format:**

```
type: short description (≤72 chars)

Optional longer body explaining the why.
```

Common types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`.

**Pull request checklist** (also in `.github/PULL_REQUEST_TEMPLATE.md`):

- [ ] `make fmt` run and diff is clean
- [ ] `make test` passes
- [ ] `make vet` passes
- [ ] README / docs updated if behaviour changed

Use the issue and PR templates in `.github/` — they provide the expected
structure. Link related issues in the PR body (`Closes #n`).
