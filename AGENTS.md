# AGENTS.md

This file provides guidance to coding agents (e.g. Claude Code, claude.ai/code) when working with code in this repository.

## Repository purpose

Go module `go.bytebuilders.dev/cli` — the **ACE (AppsCode Container Engine)** command-line. Produced binary is `ace`. Used by operators to drive ACE installs and clusters from the host shell.

Subcommands (from `pkg/cmds/root.go`):
- `config` — manage local ACE config.
- `cluster` — cluster-side operations (list/select/etc.).
- `auth` — login / token management.
- `cloud-swap` — switch cloud accounts.
- `installer` — install ACE components.
- `debug` — debug introspection.
- `expose` — expose ACE services.

Plus `version` and `completion`.

## Architecture

- `cmd/ace/` — entry point.
- `pkg/cmds/` — one subdirectory per subcommand (`config`, `cluster`, `auth`, `cloud_swap`, `installer`, `debug`, `expose`).
- `pkg/config/` — local config file plumbing.
- `pkg/printer/` — output formatting.
- `license-debug-info/` — helper for surfacing license state in `ace debug`.
- `Dockerfile.in` (PROD, distroless), `Dockerfile.dbg` (debian), `Dockerfile.ubi` (Red Hat certified).
- `hack/`, `Makefile` — AppsCode build harness.
- `vendor/` — checked-in deps.

## Common commands

- `make ci` — full CI pipeline.
- `make build` / `make all-build` — host or all-platform build.
- `make fmt`, `make lint`, `make unit-tests` / `make test` — standard.
- `make verify` — codegen + module-tidy verification.
- `make container` / `make push` / `make release` — image build/publish flow.

## Conventions

- Module path is `go.bytebuilders.dev/cli` (vanity URL); imports must use that. Binary name is `ace`.
- License: `LICENSE.md`. Sign off commits (`git commit -s`); contributions follow the DCO (`DCO`, `CONTRIBUTING.md`).
- Vendor directory is checked in; keep `go mod tidy && go mod vendor` clean.
- New subcommand: drop a `pkg/cmds/<name>/` package implementing `New*` constructor and register in `pkg/cmds/root.go`.
- Output formatting goes through `pkg/printer/` — don't print directly from subcommand handlers.
- Three Dockerfiles, one binary — keep `Dockerfile.in`, `Dockerfile.dbg`, and `Dockerfile.ubi` in sync.
