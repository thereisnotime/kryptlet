# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial implementation: age-encrypted blob serving over HTTP
- `GET /v1/blob/{name}` endpoint with `Authorization: Bearer` and `X-Kryptlet-Key` support
- Auto content-type detection on decrypted content
- `/healthz` and `/readyz` probes
- Graceful shutdown on SIGINT/SIGTERM
- Configurable listen address (`KRYPTLET_ADDR`) and blob directory (`KRYPTLET_BLOB_DIR`)
- Multi-platform builds: linux/darwin/windows × amd64/arm64
- Distroless container image published to GHCR
- Cosign signing and SBOM generation on release
- OpenSSF Scorecard, gosec, govulncheck, and Trivy CI integration
