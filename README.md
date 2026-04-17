# kryptlet

[![CI](https://github.com/thereisnotime/kryptlet/actions/workflows/ci.yaml/badge.svg)](https://github.com/thereisnotime/kryptlet/actions/workflows/ci.yaml)
[![Image](https://github.com/thereisnotime/kryptlet/actions/workflows/image.yaml/badge.svg)](https://github.com/thereisnotime/kryptlet/actions/workflows/image.yaml)
[![Release](https://github.com/thereisnotime/kryptlet/actions/workflows/release.yaml/badge.svg)](https://github.com/thereisnotime/kryptlet/actions/workflows/release.yaml)
[![Latest Release](https://img.shields.io/github/v/release/thereisnotime/kryptlet)](https://github.com/thereisnotime/kryptlet/releases/latest)
[![codecov](https://codecov.io/gh/thereisnotime/kryptlet/branch/main/graph/badge.svg)](https://codecov.io/gh/thereisnotime/kryptlet)
[![Go Version](https://img.shields.io/github/go-mod/go-version/thereisnotime/kryptlet)](go.mod)
[![Go Report Card](https://goreportcard.com/badge/github.com/thereisnotime/kryptlet)](https://goreportcard.com/report/github.com/thereisnotime/kryptlet)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/thereisnotime/kryptlet/badge)](https://securityscorecards.dev/viewer/?uri=github.com/thereisnotime/kryptlet)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

Cloud-native HTTP service that serves [age](https://age-encryption.org/)-encrypted files over HTTP, decrypting them on demand with the caller's key.

```
              ┌─────────────┐   GET /v1/blob/config        ┌──────────────┐
              │   Caller    │ ─────────────────────────── ► │   kryptlet   │
              │             │   Authorization: Bearer <key>  │              │
              │             │ ◄ ─── decrypted bytes ──────   │              │
              └─────────────┘                               └──────┬───────┘
                                                                   │ reads
                                                           ┌───────▼────────┐
                                                           │  config.age    │ ← safe to commit
                                                           │  secrets.age   │
                                                           └────────────────┘
```

Encrypt files with `age`, commit the ciphertext to git, and let kryptlet serve them. No Vault. No cloud-specific secret store. No plaintext ever written to disk. Just a private key per request.

## Features

- **GitOps-native** — encrypted blobs are safe to store in git; review and diff them in PRs like any other file
- **Any file type** — JSON, YAML, env files, TLS certificates, binaries — kryptlet does not care what you encrypted
- **Per-blob keys** — each blob can have a different recipient key; wrong key returns `401`, not a server error
- **Zero key retention** — private keys are accepted only in request headers and are never logged, stored, or cached
- **Distroless image** — runs as uid 65532 with a read-only filesystem and all Linux capabilities dropped
- **Tiny footprint** — single static binary, ~10m CPU / 16Mi memory at idle

## Quick start

### 1. Encrypt a file

```bash
age-keygen -o key.txt
PUBLIC_KEY=$(grep 'public key' key.txt | awk '{print $NF}')

age -r "$PUBLIC_KEY" config.json  > config.age
age -r "$PUBLIC_KEY" secrets.env  > secrets.age
```

### 2. Mount into kryptlet

```bash
kubectl create namespace kryptlet

kubectl create configmap kryptlet-blobs \
  --from-file=config.age \
  --from-file=secrets.age \
  -n kryptlet

kubectl apply -f https://github.com/thereisnotime/kryptlet/releases/latest/download/kryptlet.yaml
```

Or commit the `.age` files to git (they contain only ciphertext) and reference them in a ConfigMap — see the [deployment guides](#deployment) below.

### 3. Query

```bash
PRIVATE_KEY=$(grep 'AGE-SECRET-KEY' key.txt)

curl -H "Authorization: Bearer $PRIVATE_KEY" \
  https://kryptlet.example.com/v1/blob/config
```

## API reference

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/v1/blob/{name}` | Decrypt and return blob `{name}` |
| `GET` | `/healthz` | Liveness probe — always `200 ok` |
| `GET` | `/readyz` | Readiness probe — always `200 ok` |

### Authentication

Supply the age private key in either header — `Authorization` is preferred so it aligns with standard Bearer token tooling:

| Header | Value |
|--------|-------|
| `Authorization` | `Bearer AGE-SECRET-KEY-1...` |
| `X-Kryptlet-Key` | `AGE-SECRET-KEY-1...` |

### Response codes

| Code | Meaning |
|------|---------|
| `200` | Decrypted content; `Content-Type` is auto-detected from the bytes |
| `401` | Missing key, malformed key, or wrong key for this blob |
| `404` | Blob not found in the blob directory |
| `500` | Internal error — check container logs |

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `KRYPTLET_ADDR` | `:8080` | TCP listen address |
| `KRYPTLET_BLOB_DIR` | `/etc/kryptlet/blobs` | Directory scanned for `*.age` files |

## Deployment

Detailed guides live in [`docs/`](docs/):

| Guide | Description |
|-------|-------------|
| [Helm](docs/deploy-helm.md) | Install via the OCI Helm chart (recommended) |
| [Flux CD](docs/deploy-flux.md) | GitOps with HelmRelease or raw Kustomize |
| [Argo CD](docs/deploy-argocd.md) | Application manifest with Helm or raw manifests |
| [Raw manifests](docs/deploy-raw.md) | Plain `kubectl apply` for a quick spin-up |

### Multiple blobs, multiple keys

Each blob can use a different encryption key. Access control falls out of the math — if the caller's key was not used to encrypt the blob, decryption fails:

```bash
# Team A can only decrypt their blobs
age -r "$TEAM_A_PUBKEY" team-a-config.json > team-a-config.age

# Team B can only decrypt theirs
age -r "$TEAM_B_PUBKEY" team-b-config.json > team-b-config.age
```

## Local development

Uses [Podman](https://podman.io/) and [just](https://github.com/casey/just):

```bash
# First-time setup: generate keypair, encrypt sample, build image, start container
just dev-up

# In another terminal — fetch the sample blob
just dev-fetch

# Encrypt any of your own files and fetch them
just dev-encrypt path/to/myfile.json
just dev-fetch myfile.json

# Stop the container with Ctrl-C, then rebuild after code changes
just dev-build && just dev-run
```

The dev keypair is written to `dev/identity.txt` (gitignored). Encrypted blobs land in `dev/blobs/` (also gitignored).

## Building from source

**Prerequisites:** Go 1.26+, [just](https://github.com/casey/just)

```bash
git clone git@github.com:thereisnotime/kryptlet.git
cd kryptlet

just build     # → bin/kryptlet
just test      # run tests with race detector
just lint      # golangci-lint
just fmt       # gofmt
```

Pre-built binaries for Linux, macOS, and Windows (amd64/arm64) are on the [releases page](https://github.com/thereisnotime/kryptlet/releases).

## Security

Private keys are accepted only via request headers, never written to disk, never logged, and discarded after each decryption. The blob directory is mounted read-only. The container runs as a non-root uid with no Linux capabilities.

Report vulnerabilities via [SECURITY.md](SECURITY.md).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

Apache 2.0 — see [LICENSE](LICENSE).
