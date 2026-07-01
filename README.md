# provider-gitlab

[![Latest Release](https://img.shields.io/github/v/release/SemRels/provider-gitlab?label=version&color=blue)](https://github.com/SemRels/provider-gitlab/releases/latest)

Publishes the semrel release to GitLab.

This plugin is distributed as the standalone Go binary `semrel-plugin-provider-gitlab`. Semrel executes the binary as a subprocess, provides plugin configuration through `SEMREL_PLUGIN_*` environment variables, provides release context through `SEMREL_*` environment variables, reads standard output, and treats exit code `0` as success and any non-zero exit code as failure. Install the binary in `~/.semrel/plugins/` or anywhere on your `$PATH`.

## Installation

### Binary

```bash
go install github.com/SemRels/provider-gitlab/cmd/plugin@latest
```

### Docker

Pre-built, multi-platform images (linux/amd64, linux/arm64) are published to the GitHub Container Registry on every release:

```bash
docker pull ghcr.io/semrels/provider-gitlab:latest
```

Images are signed with [cosign](https://github.com/sigstore/cosign) and include a full SBOM attestation. Verify the signature:

```bash
cosign verify ghcr.io/semrels/provider-gitlab:latest \
  --certificate-identity-regexp 'https://github.com/SemRels/provider-gitlab/.github/workflows/release.yml.*' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com
```


## Configuration

```yaml
plugins:
  - name: provider-gitlab
    path: ~/.semrel/plugins/semrel-plugin-provider-gitlab
    env:
      SEMREL_PLUGIN_TOKEN: "${GITLAB_TOKEN}"
      SEMREL_PLUGIN_BASE_URL: "https://gitlab.com"
      SEMREL_PLUGIN_PROJECT_ID: "12345"
      SEMREL_PLUGIN_MILESTONE: "v{{ .Version }}"
```

## `SEMREL_PLUGIN_*` variables

| Name | Required | Description | Default |
| --- | --- | --- | --- |
| `SEMREL_PLUGIN_TOKEN` | Required | GitLab API token. | None |
| `SEMREL_PLUGIN_BASE_URL` | Optional | Base URL of the GitLab instance. | https://gitlab.com |
| `SEMREL_PLUGIN_PROJECT_ID` | Optional | GitLab project ID. Defaults from the git remote when available. | Derived from git remote |
| `SEMREL_PLUGIN_MILESTONE` | Optional | Milestone name to associate with the release. | None |

## `SEMREL_*` release context used

| Variable | Description |
| --- | --- |
| `SEMREL_TAG_NAME` | Git tag name semrel will create or publish. |
| `SEMREL_CHANGELOG` | Generated changelog text for the release. |
| `SEMREL_BRANCH` | Git branch associated with the current release run. |
| `SEMREL_DRY_RUN` | Whether semrel is running in dry-run mode. |

## Example behavior

The plugin creates a GitLab release entry for the current tag and can attach the generated changelog and milestone metadata.

## License

Apache-2.0
