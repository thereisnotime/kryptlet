# Contributing

## Setup

```bash
git clone https://github.com/thereisnotime/kryptlet.git
cd kryptlet
go mod download
just build
```

## Running tests

```bash
just test
```

Tests require no external dependencies. Coverage must stay at or above 80%.

## Running CI locally

```bash
just lint       # golangci-lint
just vet        # go vet
just test       # go test -race -shuffle=on
```

For security scanning:

```bash
go install github.com/securego/gosec/v2/cmd/gosec@v2.25.0
gosec ./...

go install golang.org/x/vuln/cmd/govulncheck@v1.1.4
govulncheck ./...
```

## Project layout

```
internal/
  crypto/    age decryption
  store/     reads .age files from a directory
  handler/   HTTP request handling
  server/    server setup and graceful shutdown
  version/   build-time version injection
```

## Coding standards

- Follow [Effective Go](https://go.dev/doc/effective_go) and [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Error messages: lowercase, no trailing punctuation
- No magic numbers — use named constants
- Keep `main.go` thin — all logic lives in `internal/`

## Pull requests

1. Fork and create a feature branch
2. Write or update tests (keep coverage above 80%)
3. `just fmt && just vet && just lint && just test`
4. Open a PR against `main`

## GitHub Actions

All actions must be pinned to a full commit SHA with the version as a comment:

```yaml
uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd  # v6.0.2
```
