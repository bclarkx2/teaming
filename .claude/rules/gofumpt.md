# Rule: gofumpt formatting required for all Go files

Any Go file (`*.go`) created or modified in this repository **must** be formatted
with gofumpt before the change is considered complete.

## How to run

Format a single file:

```sh
go tool gofumpt -w <file>
```

Format the whole repository (preferred):

```sh
make fmt
# equivalent to: go tool gofumpt -w .
```

## Notes

- gofumpt is pinned as a `go tool` dependency in `go.mod`, so `go tool gofumpt`
  always uses the version the project depends on — no separate install needed.
- In this environment the `go` binary lives at `$HOME/go-sdk/go/bin/go` if it
  is not already on your `PATH`. Adjust accordingly, e.g.:
  ```sh
  $HOME/go-sdk/go/bin/go tool gofumpt -w .
  ```
- gofumpt is a strict superset of `gofmt`. Running it satisfies both formatters.
- Do not skip formatting to save time — CI will catch it, and the diff noise
  makes reviews harder.
