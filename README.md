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

A lightweight HTTP service for serving [age](https://age-encryption.org/)-encrypted secrets. Encrypt your files, commit the ciphertext to git, point kryptlet at the directory. Callers supply their private key in a request header; kryptlet decrypts on the fly and returns the plaintext. Wrong key returns `401`. No Vault required.

```
  ┌──────────┐                                    ┌──────────────┐
  │          │ ─── GET /v1/blob/config ──────────►│              │
  │  caller  │     Bearer: AGE-SECRET-KEY-1...    │   kryptlet   │
  │          │◄─── decrypted content ─────────────│              │
  └──────────┘                                    └──────┬───────┘
                                                         │
                                                 reads *.age files
                                                         │
                                                 ┌───────▼───────┐
                                                 │  config.age   │ ← safe to commit
                                                 │  secrets.age  │
                                                 └───────────────┘
```

Each blob can use a different key. Access control falls out of the math — a caller without the right key simply cannot decrypt.

## How it stacks up

| | kryptlet | Vault | AWS Secrets Manager |
|---|---|---|---|
| Setup time | ~5 min | 2 days and a support ticket | 20 min + IAM therapy |
| Monthly cost | $0 | $0 + your sanity | $0.40/secret + API call nickels |
| Runs on your laptop | yes | technically | lol |
| Audit log | git blame | yes | yes |
| Secret rotation | re-encrypt, push | built-in | built-in |
| Dependency | one binary | HA cluster + Consul + TLS + unsealing ritual | AWS account, VPC, IAM, endpoint policy |
| What breaks at 3am | nothing | the unseal quorum | the IAM policy you wrote at 2am |
| Vendor lock-in | age (open standard) | Vault API | very yes |

## Quick start

### 1. Encrypt a file

```bash
age-keygen -o key.txt
PUBLIC_KEY=$(grep 'public key' key.txt | awk '{print $NF}')

age -r "$PUBLIC_KEY" config.json > config.age
age -r "$PUBLIC_KEY" secrets.env > secrets.age
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

Or commit the `.age` files to git and reference them in a ConfigMap — see the [deployment guides](#deployment) below.

### 3. Query

```bash
PRIVATE_KEY=$(grep 'AGE-SECRET-KEY' key.txt)

curl -H "Authorization: Bearer $PRIVATE_KEY" \
  https://kryptlet.example.com/v1/blob/config
```

## API

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/v1/blob/{name}` | Decrypt and return blob `{name}` |
| `GET` | `/healthz` | Liveness probe |
| `GET` | `/readyz` | Readiness probe |

### Authentication

Pass the age private key in either header:

| Header | Value |
|--------|-------|
| `Authorization` | `Bearer AGE-SECRET-KEY-1...` |
| `X-Kryptlet-Key` | `AGE-SECRET-KEY-1...` |

`Authorization` is preferred — it works out of the box with standard Bearer token tooling.

### Response codes

| Code | Meaning |
|------|---------|
| `200` | Decrypted content; `Content-Type` is auto-detected |
| `401` | Missing, malformed, or wrong key for this blob |
| `404` | Blob not found |
| `500` | Internal error — check container logs |

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `KRYPTLET_ADDR` | `:8080` | Listen address |
| `KRYPTLET_BLOB_DIR` | `/etc/kryptlet/blobs` | Directory scanned for `*.age` files |

## Deployment

Detailed guides in [`docs/`](docs/):

| Guide | |
|-------|-|
| [Helm](docs/deploy-helm.md) | OCI Helm chart (recommended) |
| [Flux CD](docs/deploy-flux.md) | HelmRelease or raw Kustomize |
| [Argo CD](docs/deploy-argocd.md) | Application manifest with Helm or raw manifests |
| [Raw manifests](docs/deploy-raw.md) | Plain `kubectl apply` |

### Multiple consumers, multiple keys

Encrypt each blob with its owner's public key. Each caller can only read what was encrypted for them:

```bash
age -r "$ALICE_PUBKEY" alice-config.json > alice-config.age
age -r "$BOB_PUBKEY"   bob-secrets.env   > bob-secrets.env.age
```

Alice's key decrypts `alice-config`, gets `401` on `bob-secrets.env`. Bob's key is the inverse.

## Local development

Requires [Podman](https://podman.io/) and [just](https://github.com/casey/just):

```bash
just dev-up            # generate keypair, encrypt sample, build image, start container
just dev-fetch         # fetch the sample blob (run in a second terminal)

just dev-encrypt path/to/myfile.json
just dev-fetch myfile.json

just dev-build && just dev-run   # rebuild after code changes
```

Dev keypair: `dev/identity.txt`. Encrypted blobs: `dev/blobs/`. Both gitignored.

## Building from source

Requires Go 1.26+ and [just](https://github.com/casey/just):

```bash
git clone git@github.com:thereisnotime/kryptlet.git
cd kryptlet

just build   # → bin/kryptlet
just test    # tests with race detector
just lint    # golangci-lint
just fmt     # gofmt
```

Pre-built binaries for Linux, macOS, and Windows (amd64/arm64) are on the [releases page](https://github.com/thereisnotime/kryptlet/releases).

## Security

Report vulnerabilities via [SECURITY.md](SECURITY.md).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

Apache 2.0 — see [LICENSE](LICENSE).
