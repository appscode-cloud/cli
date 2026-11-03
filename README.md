# ace

`ace` is the command-line interface for [ACE (AppsCode Container Engine)](https://selfhost.appscode.com/). Use it from your shell to authenticate against an ACE API endpoint, import and manage Kubernetes clusters, expose databases, migrate data between cloud storage buckets, and collect debugging information.

## Installation

### Build from source

Requires Go and `make` (Docker is used by the build harness).

```bash
git clone git@github.com:appscode-cloud/cli.git
cd cli
make build
```

The binary is written to `bin/ace-<os>-<arch>` (for example `bin/ace-linux-amd64`). Put it on your `PATH` as `ace`:

```bash
sudo install bin/ace-linux-amd64 /usr/local/bin/ace
ace version
```

To build for every supported platform (`linux/amd64`, `linux/arm64`, `windows/amd64`, `darwin/amd64`, `darwin/arm64`), run `make all-build`.

### Container image

Images are built from `Dockerfile.in` (distroless, production), `Dockerfile.dbg` (Debian, debugging) and `Dockerfile.ubi` (Red Hat UBI, certified) and published to `ghcr.io/appscode`. Build them with `make container`.

## Quick start

```bash
# 1. Pick or create a context that points to your ACE API endpoint
ace config view
ace config set --name my-ace --endpoint https://api.example.com
ace config use --context my-ace

# 2. Log in (token, or username/password)
ace auth login --token <access-token>

# 3. Work with clusters
ace cluster list
ace cluster get --name my-cluster
```

## Configuration

The CLI stores contexts (name, API endpoint, token, cookies) in a YAML file:

- Default location: `<user-config-dir>/ace/config_v1.yaml` (on Linux, `~/.config/ace/config_v1.yaml`).
- Override with the `ACECONFIG` environment variable.
- Built-in default contexts: `bytebuilders` (`https://api.byte.builders`) and `appscodeninja` (`https://api.appscode.ninja`).

Global flags, available on every command:

| Flag        | Description                                                                |
|-------------|----------------------------------------------------------------------------|
| `--context` | Use this context instead of the current one from the configuration file.   |
| `--org`     | Use this organization instead of auto-detecting the current one.           |

### Environment variables

| Variable       | Description                                              |
|----------------|----------------------------------------------------------|
| `ACECONFIG`    | Path to the CLI configuration file.                      |
| `ACE_TOKEN`    | Access token used to call the API.                       |
| `ACE_USERNAME` | User name for basic authentication.                      |
| `ACE_PASSWORD` | Password for basic authentication.                       |

Credentials from the environment take precedence over those stored in the selected context.

## Commands

| Command                  | Description                                              |
|--------------------------|----------------------------------------------------------|
| `ace config`             | Manage CLI contexts.                                     |
| `ace auth`               | Log in to and out of an ACE API endpoint.                |
| `ace cluster`            | Manage clusters in ACE.                                  |
| `ace cloud-swap`         | Copy data from one cloud storage bucket to another.      |
| `ace installer`          | ACE installer commands.                                  |
| `ace expose`             | Expose databases.                                        |
| `ace debug`              | Debugging utilities.                                     |
| `ace version`            | Print version information.                               |
| `ace completion`         | Generate a shell completion script.                      |

Run `ace <command> --help` for the full list of flags.

### `ace config`

| Subcommand | Description                                                                 |
|------------|-----------------------------------------------------------------------------|
| `view`     | Show the current configuration (`--show-sensitive-data` reveals tokens).    |
| `set`      | Create or update a context (`--name`, `--endpoint`, `--token`).            |
| `use`      | Switch the current context (`--context`).                                   |
| `delete`   | Delete a context (`--context`).                                             |

### `ace auth`

| Subcommand | Description                                                                              |
|------------|------------------------------------------------------------------------------------------|
| `login`    | Start an authenticated session (`--token`, or `--username` and `--password`).            |
| `logout`   | End the current authenticated session.                                                   |

### `ace cluster`

All `cluster` subcommands accept `-o, --output` (`json`, `yaml` or `table`; default `table`).

| Subcommand    | Description                                                                         |
|---------------|-------------------------------------------------------------------------------------|
| `list`        | List clusters managed by ACE (`--provider` filters by provider).                    |
| `get`         | Show one cluster (`--name`).                                                        |
| `check`       | Check whether a cluster has already been imported.                                  |
| `import`      | Import a cluster into ACE.                                                          |
| `connect`     | Connect to a cluster imported by peers (`--name`, `--credential`, `--kubeconfig`).  |
| `reconfigure` | Re-install cluster components to fix common issues.                                 |
| `remove`      | Remove a cluster from ACE (`--name`, `--remove-fluxcd`, `--all-features`).          |

Cluster identification flags shared by `check` and `import`: `--provider`, `--credential`, `--id`, `--project` (GKE), `--region`, `--resource-group` (AKS) and `--kubeconfig`.

Example:

```bash
ace cluster import \
  --provider GKE --credential my-gcp-cred \
  --id my-cluster --project my-project --region us-central1 \
  --name my-cluster --display-name "My Cluster" \
  --all-features
```

`import` additionally takes `--install-fluxcd` (default `true`), `--all-features` and `--featureset key=value,...`.

### `ace cloud-swap`

Copies objects from a source bucket to a destination bucket, staging them in a local directory.

```bash
ace cloud-swap \
  --src-bucket-url s3://src-bucket?region=us-east-1 \
  --dst-bucket-url gs://dst-bucket \
  --s3proxy.endpoint <endpoint> --s3proxy.bucket <bucket> \
  --s3proxy.access-id <id> --s3proxy.secret-key <key>
```

Use `--local-backup-dir` to choose the staging directory or `--disable-local-backup` to skip it.

### `ace installer`

| Subcommand | Description                      |
|------------|----------------------------------|
| `validate` | Validate installer options.      |

### `ace expose`

| Subcommand | Description                                                                                                  |
|------------|--------------------------------------------------------------------------------------------------------------|
| `gateway`  | Expose a database through the gateway (`-t/--db-type`, `-m/--name`, `-n/--namespace`; defaults `mongodb`, `mg-test`, `demo`). |

### `ace debug`

| Subcommand   | Description                                                                                         |
|--------------|-----------------------------------------------------------------------------------------------------|
| `gateway`    | Inspect gateway resources for a database (`-t/--db-type`, `-m/--name`, `-n/--namespace`).           |
| `client-org` | Inspect (and optionally clean up) client organizations (`-m/--name`, `-d/--mode`, `-c/--cleanup`).  |
| `license`    | Show license information.                                                                           |

### Shell completion

```bash
source <(ace completion bash)   # also: zsh, fish, powershell
```

## Development

```
cmd/ace/             entry point
pkg/cmds/            one package per top-level command, wired up in root.go
pkg/config/          local configuration file handling
pkg/printer/         output formatting
license-debug-info/  license helper for `ace debug`
hack/                build, test and formatting scripts
vendor/              vendored dependencies
```

Common targets:

| Target             | Description                                   |
|--------------------|-----------------------------------------------|
| `make build`       | Build `ace` for the host platform.            |
| `make all-build`   | Build for all supported platforms.            |
| `make fmt`         | Format the code.                              |
| `make lint`        | Run linters.                                  |
| `make test`        | Run unit tests.                               |
| `make verify`      | Verify generated code and `go.mod`/`vendor`.  |
| `make ci`          | Run the full CI pipeline locally.             |
| `make container`   | Build container images.                       |

To add a command, create `pkg/cmds/<name>/` with a `NewCmd*` constructor and register it in `pkg/cmds/root.go`. Route output through `pkg/printer` rather than printing directly. After changing dependencies, run `go mod tidy && go mod vendor`.

## Contributing

Contributions are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md). Commits must be signed off (`git commit -s`) under the [Developer Certificate of Origin](DCO).

## License

Source code, binaries, Docker images and charts produced by this repository are licensed under the [AppsCode Community License 1.0.0](https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Community-1.0.0.md). See [LICENSE.md](LICENSE.md).
