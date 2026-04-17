module := "github.com/thereisnotime/kryptlet"
binary := "kryptlet"
version := `git describe --tags --always --dirty 2>/dev/null || echo "dev"`
commit  := `git rev-parse --short HEAD 2>/dev/null || echo "none"`
date    := `date -u +%Y-%m-%dT%H:%M:%SZ`
ldflags := "-s -w -X " + module + "/internal/version.Version=" + version + " -X " + module + "/internal/version.Commit=" + commit + " -X " + module + "/internal/version.Date=" + date

default:
    @just --list

build:
    mkdir -p bin
    go build -ldflags '{{ldflags}}' -o bin/{{binary}} .

build-all:
    mkdir -p bin
    GOOS=linux   GOARCH=amd64 go build -ldflags '{{ldflags}}' -o bin/{{binary}}-linux-amd64 .
    GOOS=linux   GOARCH=arm64 go build -ldflags '{{ldflags}}' -o bin/{{binary}}-linux-arm64 .
    GOOS=darwin  GOARCH=amd64 go build -ldflags '{{ldflags}}' -o bin/{{binary}}-darwin-amd64 .
    GOOS=darwin  GOARCH=arm64 go build -ldflags '{{ldflags}}' -o bin/{{binary}}-darwin-arm64 .

test:
    go test -race -shuffle=on -coverprofile=coverage.out -covermode=atomic ./...

lint:
    golangci-lint run ./...

vet:
    go vet ./...

fmt:
    gofmt -s -w .

clean:
    rm -rf bin/ dist/ coverage.out

docker-build:
    docker build -t {{binary}}:local .

# Encrypt a file with an age public key: just encrypt myfile.json AGE1...
encrypt file pubkey:
    age -r {{pubkey}} {{file}} > {{file}}.age
    @echo "Encrypted: {{file}}.age"

run:
    KRYPTLET_BLOB_DIR=./testdata/blobs go run -ldflags '{{ldflags}}' .

# ── dev (local Podman) ──────────────────────────────────────────────────────

# Generate a test age keypair — saved to dev/identity.txt (gitignored)
dev-keygen:
    mkdir -p dev/blobs
    @if [ ! -f dev/identity.txt ]; then \
        age-keygen -o dev/identity.txt && echo "Keypair written to dev/identity.txt"; \
    else \
        echo "dev/identity.txt already exists, skipping"; \
    fi
    @echo "Public key: $(grep 'public key' dev/identity.txt | awk '{print $NF}')"

# Encrypt a file into dev/blobs/ using the dev keypair: just dev-encrypt dev/sample.txt
dev-encrypt file="dev/sample.txt":
    @pubkey=$$(grep 'public key' dev/identity.txt | awk '{print $$NF}') && \
    age -r "$$pubkey" {{file}} > "dev/blobs/$$(basename {{file}}).age" && \
    chmod 644 "dev/blobs/$$(basename {{file}}).age" && \
    echo "Encrypted → dev/blobs/$$(basename {{file}}).age  (fetch as: /v1/blob/$$(basename {{file}}))"

# Build the kryptlet container image with Podman
dev-build:
    podman build -t {{binary}}:dev .

# Run kryptlet in Podman with dev blobs mounted on localhost:8080
dev-run:
    podman run --rm -it \
        -p 127.0.0.1:8080:8080 \
        -v "$(pwd)/dev/blobs:/etc/kryptlet/blobs:ro,Z" \
        -e KRYPTLET_ADDR=:8080 \
        -e KRYPTLET_BLOB_DIR=/etc/kryptlet/blobs \
        {{binary}}:dev

# Fetch a blob from the running dev instance: just dev-fetch sample.txt
dev-fetch blob="sample.txt":
    @key=$$(grep '^AGE-SECRET-KEY' dev/identity.txt | head -1) && \
    curl -sf -H "Authorization: Bearer $$key" http://localhost:8080/v1/blob/{{blob}}

# First-time setup: keygen → encrypt sample → build → run
dev-up: dev-keygen dev-encrypt dev-build dev-run
