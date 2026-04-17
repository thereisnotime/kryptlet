# kryptlet

## GitHub Actions

Always pin actions to a full commit SHA, never use a tag or branch reference alone.
Include the version as a comment for readability:

```yaml
uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd  # v6.0.2
```

This applies to all actions added or updated — including new ones introduced during fixes or features.

## Dockerfile

Pin base images to digest SHAs for reproducible builds:

```dockerfile
FROM golang:1.22-alpine@sha256:<digest> AS builder
FROM gcr.io/distroless/static:nonroot@sha256:<digest>
```
