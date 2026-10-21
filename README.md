# publisher-crates

[![Latest Release](https://img.shields.io/github/v/release/SemRels/publisher-crates?label=version&color=blue)](https://github.com/SemRels/publisher-crates/releases/latest)
[![CI](https://github.com/SemRels/publisher-crates/actions/workflows/ci.yml/badge.svg)](https://github.com/SemRels/publisher-crates/actions/workflows/ci.yml)
[![License](https://img.shields.io/github/license/SemRels/publisher-crates)](LICENSE)
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/SemRels/publisher-crates/badge)](https://scorecard.dev/viewer/?uri=github.com/SemRels/publisher-crates)
[![REUSE status](https://api.reuse.software/badge/github.com/SemRels/publisher-crates)](https://api.reuse.software/info/github.com/SemRels/publisher-crates)

Publishes Rust crates to crates.io by running `cargo publish` after semrel has already decided and applied the next release version.

This plugin is distributed as the standalone Go binary `semrel-plugin-publisher-crates`. Semrel executes the binary as a subprocess, provides plugin configuration through `SEMREL_PLUGIN_*` environment variables, provides release context through `SEMREL_*` environment variables, reads standard output, and treats exit code `0` as success and any non-zero exit code as failure. Install the binary in `~/.semrel/plugins/` or anywhere on your `$PATH`.

## Installation

### Binary

```bash
go install github.com/SemRels/publisher-crates/cmd/plugin@latest
```

### Docker

Pre-built, multi-platform images (linux/amd64, linux/arm64) are published to the GitHub Container Registry on every release:

```bash
docker pull ghcr.io/semrels/publisher-crates:latest
```

Images are signed with [cosign](https://github.com/sigstore/cosign) and include a full SBOM attestation. Verify the signature:

```bash
cosign verify ghcr.io/semrels/publisher-crates:latest \
  --certificate-identity-regexp 'https://github.com/SemRels/publisher-crates/.github/workflows/container.yml.*' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com
```

## Configuration

```yaml
plugins:
  - uses: "@semrel/publisher-crates"
    phase: release
    args:
      cargo_registry_token: ${CARGO_REGISTRY_TOKEN}
      cargo_workspace_member: "crate-a,crate-b"
      cargo_manifest_path: Cargo.toml
```

Workspace/package lists are processed in the order given. If you publish multiple inter-dependent workspace members, list them in the order crates.io should receive them.

## Options

Every option can be set as an entry under `args` in `.semrel.yaml`. semrel passes each entry to the plugin as the environment variable shown in the table, so you can also set that variable directly (for example in your CI environment) instead of using `args`.

| Argument | Environment variable | Required | Description | Default |
| --- | --- | --- | --- | --- |
| `cargo_registry_token` | `SEMREL_PLUGIN_CARGO_REGISTRY_TOKEN` | Yes, unless `SEMREL_DRY_RUN=true` | crates.io API token. Passed to Cargo via `CARGO_REGISTRY_TOKEN` and never placed on the command line. | _none_ |
| `cargo_workspace_member` | `SEMREL_PLUGIN_CARGO_WORKSPACE_MEMBER` | Optional | Comma-separated workspace members to publish. Each member becomes its own `cargo publish --package <name>` invocation. | Publish the default package |
| `cargo_package` | `SEMREL_PLUGIN_CARGO_PACKAGE` | Optional | Alias for `SEMREL_PLUGIN_CARGO_WORKSPACE_MEMBER`. Values from both variables are merged in order and deduplicated. | _none_ |
| `cargo_manifest_path` | `SEMREL_PLUGIN_CARGO_MANIFEST_PATH` | Optional | Manifest path passed through as `cargo publish --manifest-path <path>`. | Cargo's default manifest discovery |

## `SEMREL_*` release context used

| Variable | Description |
| --- | --- |
| `SEMREL_VERSION` | Resolved release version for the current run. |
| `SEMREL_NEXT_VERSION` | Next version computed by semrel for the release. |
| `SEMREL_DRY_RUN` | When `true`, the plugin still runs Cargo, but appends `--dry-run`. |

## Example `.semrel.yaml`

```yaml
tagPrefix: "v"
version_ceiling: "1.0.0"
ceiling_strategy: clamp
commit_changelog: false

autoCommit: false

plugins:
  - uses: "@semrel/condition-github-actions"
    phase: condition

  - uses: "@semrel/updater-cargo"
    phase: pre-tag
    args:
      file: Cargo.toml

  - uses: "@semrel/publisher-crates"
    phase: release
    args:
      cargo_registry_token: ${CARGO_REGISTRY_TOKEN}
      cargo_workspace_member: "crate-a,crate-b"
      cargo_manifest_path: Cargo.toml
```

## Behavior

- Requires `cargo` on `PATH`.
- Uses Cargo's native `publish` behavior instead of reimplementing packaging in Go.
- Stops on the first failing workspace member and reports which package failed.
- In dry-run mode, validates packaging with `cargo publish --dry-run` instead of skipping the command.

## License

Apache-2.0
