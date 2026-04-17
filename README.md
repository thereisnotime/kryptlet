# kryptlet

[![CI](https://github.com/thereisnotime/kryptlet/actions/workflows/ci.yaml/badge.svg)](https://github.com/thereisnotime/kryptlet/actions/workflows/ci.yaml)
[![Release](https://github.com/thereisnotime/kryptlet/actions/workflows/release.yaml/badge.svg)](https://github.com/thereisnotime/kryptlet/actions/workflows/release.yaml)
[![License](https://img.shields.io/github/license/thereisnotime/kryptlet)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/thereisnotime/kryptlet)](https://goreportcard.com/report/github.com/thereisnotime/kryptlet)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/thereisnotime/kryptlet/badge)](https://securityscorecards.dev/viewer/?uri=github.com/thereisnotime/kryptlet)

Cloud-native HTTP service that serves [age](https://age-encryption.org/)-encrypted files over HTTP, decrypting them on demand with the caller's key.

## How it works

```
              ┌─────────────┐   GET /v1/blob/config        ┌──────────────┐
              │   Caller    │ ───────────────────────────► │   kryptlet   │
              │             │   Authorization: Bearer <key> │              │
              │             │ ◄─────── decrypted bytes ─── │              │
              └─────────────┘                              └──────┬───────┘
                                                                  │ reads
                                                          ┌───────▼────────┐
                                                          │  config.age    │ ← safe to commit
                                                          │  secrets.age   │
                                                          └────────────────┘
```

1. Encrypt files with `age` and commit the ciphertext to git — no plaintext ever stored
2. Mount the `.age` files into kryptlet via a Kubernetes ConfigMap
3. Callers provide their age private key per-request to receive the decrypted content

The server **never stores or logs private keys**. Each key is used for a single decryption and discarded.

## Quick start

### 1. Encrypt a file

```bash
# Generate a key pair
age-keygen -o key.txt
PUBLIC_KEY=$(grep 'public key' key.txt | awk '{print $NF}')

# Encrypt any file — JSON, YAML, env file, certificate, anything
age -r $PUBLIC_KEY config.json > config.age
age -r $PUBLIC_KEY secrets.env > secrets.age
```

### 2. Mount into kryptlet

```bash
# Create ConfigMap from encrypted blobs
kubectl create configmap kryptlet-blobs \
  --from-file=config.age \
  --from-file=secrets.age \
  -n kryptlet
```

Or commit the `.age` files to git and reference them in a ConfigMap manifest — they contain only ciphertext, so it is safe.

### 3. Query

```bash
PRIVATE_KEY=$(grep 'AGE-SECRET-KEY' key.txt)

# Using Authorization header (preferred)
curl -H "Authorization: Bearer $PRIVATE_KEY" \
  https://kryptlet.example.com/v1/blob/config

# Using X-Kryptlet-Key header
curl -H "X-Kryptlet-Key: $PRIVATE_KEY" \
  https://kryptlet.example.com/v1/blob/config
```

## API reference

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/v1/blob/{name}` | Decrypt and return blob `{name}` |
| `GET` | `/healthz` | Liveness probe |
| `GET` | `/readyz` | Readiness probe |

### Authentication

Provide the age private key in either header:

| Header | Format |
|--------|--------|
| `Authorization` | `Bearer AGE-SECRET-KEY-1...` |
| `X-Kryptlet-Key` | `AGE-SECRET-KEY-1...` |

### Response codes

| Code | Meaning |
|------|---------|
| `200` | Decrypted content, `Content-Type` auto-detected |
| `401` | Missing or wrong key |
| `404` | Blob not found |
| `500` | Internal error — check container logs |

### Content-Type detection

kryptlet uses Go's `http.DetectContentType` on the decrypted bytes. JSON, plain text, and binary formats are all handled transparently — the service does not care what you encrypted.

## Configuration

| Env var | Default | Description |
|---------|---------|-------------|
| `KRYPTLET_ADDR` | `:8080` | Listen address |
| `KRYPTLET_BLOB_DIR` | `/etc/kryptlet/blobs` | Directory scanned for `*.age` files |

## Multiple blobs, multiple keys

Each blob can use a different encryption key. The server does not enforce access rules — age's asymmetric encryption does: if the caller's key doesn't match the blob's recipient, decryption fails with `401`.

```bash
# Team A can only decrypt their blobs
age -r $TEAM_A_PUBKEY team-a-config.json > team-a-config.age

# Team B can only decrypt theirs
age -r $TEAM_B_PUBKEY team-b-config.json > team-b-config.age
```

## GitOps workflow

```
encrypt locally  →  commit .age file  →  Flux/ArgoCD applies ConfigMap  →  kryptlet serves it
```

No plaintext in git. No Vault required. No cloud-specific secret store. The encrypted blobs are safe to review and diff in PRs.

## Deploying on Kubernetes

### Raw manifests

```bash
kubectl apply -f https://github.com/thereisnotime/kryptlet/releases/latest/download/install.yaml
```

### Flux + Kustomize

An example overlay is included in [`deploy/`](deploy/). Wire it up in your Flux source and patch as needed.

### Minimal example

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kryptlet
  namespace: kryptlet
spec:
  replicas: 1
  selector:
    matchLabels:
      app: kryptlet
  template:
    metadata:
      labels:
        app: kryptlet
    spec:
      containers:
        - name: kryptlet
          image: ghcr.io/thereisnotime/kryptlet:latest
          ports:
            - containerPort: 8080
          env:
            - name: KRYPTLET_BLOB_DIR
              value: /etc/kryptlet/blobs
          volumeMounts:
            - name: blobs
              mountPath: /etc/kryptlet/blobs
              readOnly: true
          livenessProbe:
            httpGet:
              path: /healthz
              port: 8080
          readinessProbe:
            httpGet:
              path: /readyz
              port: 8080
      volumes:
        - name: blobs
          configMap:
            name: kryptlet-blobs
```

## Building from source

```bash
git clone https://github.com/thereisnotime/kryptlet.git
cd kryptlet

# Build binary
just build
./bin/kryptlet

# Run tests
just test

# Lint
just lint

# Build container image
just docker-build
```

**Prerequisites:** Go 1.22+, [just](https://github.com/casey/just)

## Security

Private keys are accepted only via request headers and are never logged or stored. The blob directory is read-only at runtime. See [SECURITY.md](SECURITY.md) for reporting vulnerabilities.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

Apache 2.0 — see [LICENSE](LICENSE).
