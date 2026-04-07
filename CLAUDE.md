# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

A **Windows-only gRPC service** that exposes cryptographic signing via Windows certificate store (ГОСТ сертификаты, crypt32.dll) over a gRPC interface.

Designed to be reused by multiple clients: `edo-client`, `chestnyznak-client`, and future services.
Each caller passes `thumbprint` per request — the service supports multiple certificates simultaneously.

## Build & Run

```powershell
# Generate gRPC stubs (run after any .proto change)
.\scripts\generate.ps1

# Build (Windows only — crypt32.dll dependency)
go build -o sign-service.exe ./cmd/sign-service

# Run
.\sign-service.exe --config config/prod.yml

# Tests
go test ./...

# Lint
golangci-lint run
```

## Architecture

```
cmd/sign-service/main.go    Entry point: load config, build server, serve gRPC
internal/config/            YAML config via cleanenv
internal/sign/              Windows crypt32.dll syscalls — copied from edo-client, do not modify API
internal/server/            gRPC SignerServer: implements proto/signer interface
gen/signer/                 Generated protobuf + gRPC stubs (do not edit manually)
proto/signer/signer.proto   Source of truth for the gRPC contract
scripts/generate.ps1        Runs protoc to regenerate gen/
scripts/install-service.ps1 Installs as Windows service via NSSM
```

## Proto Contract

```protobuf
service Signer {
  rpc Sign            (SignRequest)    returns (SignResponse);
  rpc Verify          (VerifyRequest)  returns (VerifyResponse);
  rpc ListCertificates(Empty)          returns (CertListResponse);
}

message SignRequest {
  bytes  payload    = 1;  // raw bytes to sign
  string thumbprint = 2;  // SHA1 hex of certificate, case-insensitive
  string caller_id  = 3;  // e.g. "edo-client", "chestnyznak" — for audit log
}
```

Full schema: `proto/signer/signer.proto`.
After any change run `.\scripts\generate.ps1` and commit `gen/` together with `.proto`.

## Windows-Only Constraint

`internal/sign` uses `golang.org/x/sys/windows`. Cannot compile on Linux/macOS.
All other packages are cross-platform — server logic, proto, config.

## Configuration Fields

| Field       | Env var    | Default          | Purpose                        |
|-------------|------------|------------------|--------------------------------|
| `grpc_addr` | `GRPC_ADDR`| `0.0.0.0:50051`  | Listen address                 |
| `log_level` | `LOG_LEVEL`| `info`           | `debug` / `info` / `warn`     |
| `audit_log` | `AUDIT_LOG`| `audit.jsonl`    | Path to audit JSONL file       |

Certificates are not configured in the config file — all signing uses certificates installed in the Windows certificate store, accessed via Windows API (crypt32.dll).

## Code Notes

- Log messages and comments in Russian (consistent with edo-client)
- `internal/sign` must stay API-compatible with edo-client's copy — any fix applies to both
- Audit log format: `{"ts":"...","caller":"edo-client","thumbprint":"AB12...","payload_size":1024,"ok":true}`
- `gen/` is committed to repo — consumers can vendor it without running protoc

## Roadmap: Phase 2 Tasks ← CURRENT

> Phase 1 (interface boundary in edo-client) must be completed first.

- [ ] **2.1** Init Go module: `go mod init github.com/YOUR_ORG/sign-service`

- [ ] **2.2** Write `proto/signer/signer.proto`
  — Sign, Verify, ListCertificates
  — include `caller_id` in SignRequest for audit

- [ ] **2.3** Write `scripts/generate.ps1`
  — install `protoc-gen-go` and `protoc-gen-go-grpc` if missing
  — run protoc, output to `gen/signer/`

- [ ] **2.4** Run generation, commit `gen/` — verify stubs compile: `go build ./gen/...`

- [ ] **2.5** Copy `internal/sign/` from edo-client verbatim
  — do not refactor, just copy
  — verify: `go build ./internal/sign/` on Windows

- [ ] **2.6** Implement `internal/server/server.go`
  — `SignerServer` struct implementing generated gRPC interface
  — delegate Sign → `internal/sign`
  — write audit log entry on every Sign call (structured JSON, append to file)
  — return gRPC status codes: `codes.InvalidArgument` for bad thumbprint, `codes.Internal` for sign failure

- [x] **2.7** Implement `internal/config/config.go`
  — fields: grpc_addr, log_level, audit_log (no TLS fields — certs via Windows API)

- [x] **2.8** Implement `cmd/sign-service/main.go`
  — load config → create gRPC server → register SignerServer → serve
  — graceful shutdown on SIGINT/SIGTERM (context + grpc.GracefulStop)

- [ ] **2.9** Write `internal/server/server_test.go`
  — test Sign with StubSigner (build tag !windows) so tests run on Linux CI

- [ ] **2.10** Write `scripts/install-service.ps1`
  — NSSM install + set AppDirectory + start


- [ ] **2.11** Verify end-to-end on Windows:
  ```powershell
  .\sign-service.exe --config config/prod.yml
  # from another terminal — grpcurl or test client:
  grpcurl -plaintext localhost:50051 signer.Signer/ListCertificates
  ```

## Roadmap: Phase 3 — edo-client migration

> Starts in edo-client repo after Phase 2 is complete.

- [ ] Add `internal/signer/grpc.go` to edo-client — GRPCSigner wraps generated gRPC client
- [ ] Add `sign_service_addr` to edo-client config
- [ ] Delete `internal/sign/` from edo-client
- [ ] Update edo-client CLAUDE.md Architecture section
